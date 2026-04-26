// Package parsers provides pure, stateless file-parsing functions for telemetry
// upload formats. No HTTP, database, or logging dependencies are introduced here,
// making every function in this package trivially unit-testable.
//
// Supported formats
//
//   - .xlsx — Excel workbooks produced by the spacecraft catalogue tool.
//   - .out  — Fixed-width ASCII export from TM catalogue systems.
//
// The gateway handler is responsible for decoding the Base64 payload, writing a temp
// file, and calling ParseXLSX / ParseOUT. It receives a []map[string]any that it can
// pass directly to the upsert layer.
package parsers

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ── column index → field name mapping for .xlsx uploads ──────────────────────

var xlsxColumnMap = map[int]string{
	1:  "slNo",
	2:  "subsystem",
	3:  "cdbPidNo",
	4:  "cdbMnemonic",
	5:  "description",
	6:  "type",
	7:  "processingType",
	8:  "noOfWords",
	9:  "channelNo",
	10: "frameNo",
	11: "startBit",
	12: "endBit",
	13: "samplingRate",
	14: "dwellAddress",
	15: "range",
	16: "resolutionA1",
	17: "offsetA0",
	18: "unit",
	19: "remarks",
	20: "packageId",
	21: "wordNo",
	22: "digitalStatus",
	23: "condSrc",
	24: "condSts",
	25: "gcoMnemonic",
	26: "pidScope",
	27: "authenticationStage",
	28: "lutRef",
	29: "qualificationLimit",
	30: "storageLimit",
	31: "pidAddress",
	32: "pt",
	33: "descUpdate",
}

// ── .out format type → normalised type string ─────────────────────────────────

var outTypeMap = map[string]string{
	"STATUS":     "BINARY",
	"EUCN":       "ANALOG",
	"EUCN-16B":   "ANALOG",
	"TMCD-16B":   "ANALOG",
	"TMCD":       "ANALOG",
	"TMCH-16B":   "ANALOG",
	"TMCH":       "ANALOG",
	"AEXP":       "ANALOG",
	"LKPTBL":     "ANALOG",
	"EUCN-HRMIN": "ANALOG",
	"EUCN-1750-": "ANALOG",
}

var outPrimaryRowPattern = regexp.MustCompile(`^[A-Z]{3}\d{5}`)
var outDigitalStatePattern = regexp.MustCompile(`^\d{2}:`)

// outHeaderLines is the number of fixed header lines that precede data records
// in a .out TM catalogue export file. Lines 0–6 are skipped unconditionally.
const outHeaderLines = 7

// ── Public API ────────────────────────────────────────────────────────────────

// ParseXLSX reads an Excel (.xlsx) telemetry definition file and returns a slice
// of record maps, one per data row, keyed by the field names in xlsxColumnMap.
func ParseXLSX(path string) ([]map[string]any, error) {
	xf, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer xf.Close()

	sourceFile := filepath.Base(path)
	records := make([]map[string]any, 0)

	for _, sheet := range xf.GetSheetList() {
		if sheet == "Information" || sheet == "Spacecraft Information" || strings.HasPrefix(sheet, "LUT_TBL_INFO_") {
			continue
		}

		rows, err := xf.GetRows(sheet)
		if err != nil {
			return nil, err
		}
		if len(rows) < 2 {
			continue
		}

		for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
			row := rows[rowIdx]
			pid := ""
			if len(row) >= 3 {
				pid = safeString(row[2])
			}
			if pid == "" {
				continue
			}

			rec := make(map[string]any)
			for colIdx, field := range xlsxColumnMap {
				if colIdx-1 >= len(row) {
					continue
				}
				raw := safeString(row[colIdx-1])
				switch field {
				case "type":
					rec[field] = mapTypeCode(raw)
				case "samplingRate", "resolutionA1", "offsetA0":
					rec[field] = tryParseFloat(raw)
				case "range":
					rec[field] = parseRange(raw)
				case "qualificationLimit", "storageLimit":
					rec[field] = parseRange(raw)
				default:
					rec[field] = raw
				}
			}

			rec["sourceSheet"] = sheet
			rec["sourceFile"] = sourceFile

			recType := toString(rec["type"])
			recRange := rec["range"]
			recDigitalStatus := toString(rec["digitalStatus"])
			if recType == "BINARY" && isRangeEmpty(recRange) && recDigitalStatus != "" {
				rec["range"] = parseDigitalRange(recDigitalStatus)
			}

			records = append(records, rec)
		}
	}

	return records, nil
}

// ParseOUT reads a text .out catalogue file and returns a slice of record maps.
func ParseOUT(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	groups := make([][]string, 0)
	for i := outHeaderLines; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if outPrimaryRowPattern.MatchString(line) {
			groups = append(groups, []string{line})
		} else if len(groups) > 0 {
			groups[len(groups)-1] = append(groups[len(groups)-1], line)
		}
	}

	sourceFile := filepath.Base(path)
	records := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		rec := ParseOUTGroup(group, sourceFile)
		if rec != nil {
			records = append(records, rec)
		}
	}

	return records, nil
}

// ParseOUTGroup parses a single OUT record group (primary line + optional continuation
// lines) and returns a field map. Returns nil if the group is malformed.
func ParseOUTGroup(group []string, sourceFile string) map[string]any {
	if len(group) == 0 {
		return nil
	}
	tokens := strings.Fields(group[0])
	if len(tokens) < 4 {
		return nil
	}

	cdbPidNo := tokens[0]
	cdbMnemonic := tokens[1]
	processingType := tokens[2]
	subsystem := tokens[3]
	mappedType := outTypeMap[processingType]
	if mappedType == "" {
		mappedType = "ANALOG"
	}

	highLimit := ""
	lowLimit := ""
	tolerance := ""
	unitValue := ""
	resolutionA1 := ""
	offsetA0 := ""
	lutRef := ""
	description := ""
	digitalStates := make([]string, 0)

	calIdx := -1
	for i, tok := range tokens {
		if tok == "SIMPLE" || tok == "NONE" {
			calIdx = i
			break
		}
	}

	if calIdx != -1 {
		calType := tokens[calIdx]
		rest := tokens[calIdx+1:]
		after := make([]string, 0)
		if calType == "SIMPLE" && len(rest) >= 4 {
			highLimit = tryParseFloatOut(rest[0])
			lowLimit = tryParseFloatOut(rest[1])
			tolerance = tryParseFloatOut(rest[2])
			unitValue = rest[3]
			after = append(after, rest[4:]...)
		} else if calType == "NONE" && len(rest) >= 2 {
			tolerance = tryParseFloatOut(rest[0])
			unitValue = rest[1]
			after = append(after, rest[2:]...)
		}

		switch processingType {
		case "STATUS":
			for _, tok := range after {
				if outDigitalStatePattern.MatchString(tok) && !strings.HasPrefix(tok, "%") {
					digitalStates = append(digitalStates, tok)
				}
			}
		case "AEXP":
			parts := make([]string, 0)
			for _, tok := range after {
				if !strings.HasPrefix(tok, "%") {
					parts = append(parts, tok)
				}
			}
			description = strings.Join(parts, " ")
		case "LKPTBL":
			parts := make([]string, 0)
			for _, tok := range after {
				if strings.HasPrefix(tok, "%") {
					continue
				}
				if regexp.MustCompile(`^\d+\.\d+$`).MatchString(tok) {
					continue
				}
				parts = append(parts, tok)
			}
			if len(parts) > 0 {
				lutRef = parts[len(parts)-1]
			}
		default:
			numericAfter := make([]string, 0)
			for _, tok := range after {
				if !strings.HasPrefix(tok, "%") {
					numericAfter = append(numericAfter, tok)
				}
			}
			if len(numericAfter) >= 2 {
				resolutionA1 = tryParseFloatOut(numericAfter[0])
				offsetA0 = tryParseFloatOut(numericAfter[1])
			} else if len(numericAfter) == 1 {
				resolutionA1 = tryParseFloatOut(numericAfter[0])
			}
		}
	}

	for _, line := range group[1:] {
		for _, tok := range strings.Fields(line) {
			if outDigitalStatePattern.MatchString(tok) && !strings.HasPrefix(tok, "%") {
				digitalStates = append(digitalStates, tok)
			}
		}
	}

	var rangeValue any = ""
	digitalStatus := ""
	if mappedType == "BINARY" && len(digitalStates) > 0 {
		labels := make([]string, 0, len(digitalStates))
		for _, state := range digitalStates {
			parts := strings.SplitN(state, ":", 2)
			if len(parts) == 2 {
				labels = append(labels, parts[1])
			}
		}
		rangeValue = labels
		digitalStatus = strings.Join(digitalStates, ";")
	} else if mappedType == "ANALOG" {
		high := parseFloat(highLimit)
		low := parseFloat(lowLimit)
		if high != nil && low != nil {
			rangeValue = []float64{*low, *high}
		}
	}

	return map[string]any{
		"cdbPidNo":       cdbPidNo,
		"cdbMnemonic":    cdbMnemonic,
		"type":           mappedType,
		"processingType": processingType,
		"subsystem":      subsystem,
		"sourceSheet":    subsystem,
		"sourceFile":     sourceFile,
		"range":          rangeValue,
		"digitalStatus":  digitalStatus,
		"tolerance":      tolerance,
		"resolutionA1":   resolutionA1,
		"offsetA0":       offsetA0,
		"unit":           unitValue,
		"lutRef":         lutRef,
		"description":    description,
	}
}

// NormalizeRangeLikeValue coerces v into a clean slice value suitable for the
// "range" or "limits" field stored in SQLite. Returns []any{} for empty/nil input.
func NormalizeRangeLikeValue(v any) any {
	if v == nil {
		return []any{}
	}
	if arr, ok := v.([]float64); ok {
		out := make([]float64, len(arr))
		copy(out, arr)
		return out
	}
	if arr, ok := v.([]string); ok {
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		if out == nil {
			return []string{}
		}
		return out
	}
	if arr, ok := v.([]any); ok {
		out := make([]any, 0, len(arr))
		for _, item := range arr {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, item)
			}
		}
		if out == nil {
			return []any{}
		}
		return out
	}

	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" || strings.EqualFold(s, "<nil>") || s == "[]" {
		return []any{}
	}

	separators := []string{" to ", "~", " - ", ":"}
	for _, sep := range separators {
		if strings.Contains(s, sep) {
			parts := strings.SplitN(s, sep, 2)
			if len(parts) != 2 {
				continue
			}
			lo, errLo := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			hi, errHi := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if errLo == nil && errHi == nil {
				return []float64{lo, hi}
			}
		}
	}

	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';'
	})
	if len(parts) == 0 {
		return []string{s}
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

// CloneRangeLikeValue performs a shallow copy of a range-like value so that
// the caller can store independent copies in "range" and "limits" fields.
func CloneRangeLikeValue(v any) any {
	v = NormalizeRangeLikeValue(v)
	if v == nil {
		return []any{}
	}
	if arr, ok := v.([]float64); ok {
		out := make([]float64, len(arr))
		copy(out, arr)
		return out
	}
	if arr, ok := v.([]string); ok {
		out := make([]string, len(arr))
		copy(out, arr)
		return out
	}
	if arr, ok := v.([]any); ok {
		out := make([]any, len(arr))
		copy(out, arr)
		return out
	}
	return []string{toString(v)}
}

// DeriveExpectedValue infers the expected_value from a telemetry record. For BINARY
// mnemonics with no explicit expected_value, returns the first non-empty range label.
func DeriveExpectedValue(record map[string]any) string {
	explicit := strings.TrimSpace(toString(record["expected_value"]))
	if explicit != "" {
		return explicit
	}

	if !strings.EqualFold(strings.TrimSpace(toString(record["type"])), "BINARY") {
		return ""
	}

	rangeValue := NormalizeRangeLikeValue(recordOrEmpty(record, "range"))
	switch vals := rangeValue.(type) {
	case []string:
		for _, v := range vals {
			v = strings.TrimSpace(v)
			if v != "" {
				return v
			}
		}
	case []any:
		for _, v := range vals {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" {
				return s
			}
		}
	case []float64:
		if len(vals) > 0 {
			return strconv.FormatFloat(vals[0], 'f', -1, 64)
		}
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", rangeValue))
		if s != "" && s != "[]" {
			return s
		}
	}

	return ""
}

// ── private helpers (self-contained; not exported) ────────────────────────────

func mapTypeCode(v string) string {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "A":
		return "ANALOG"
	case "B":
		return "BINARY"
	case "D":
		return "DECIMAL"
	default:
		return strings.TrimSpace(v)
	}
}

func tryParseFloat(v string) any {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}
	if strings.Contains(s, ",") {
		parts := strings.SplitN(s, ",", 2)
		s = strings.TrimSpace(parts[0])
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return v
	}
	return f
}

func parseRange(v string) any {
	s := strings.TrimSpace(v)
	if s == "" {
		return []any{}
	}
	separators := []string{" to ", "~", " - ", ":"}
	for _, sep := range separators {
		if strings.Contains(s, sep) {
			parts := strings.SplitN(s, sep, 2)
			if len(parts) != 2 {
				continue
			}
			lo, errLo := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			hi, errHi := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if errLo == nil && errHi == nil {
				return []float64{lo, hi}
			}
		}
	}
	return NormalizeRangeLikeValue(s)
}

func parseDigitalRange(digitalStatus string) []string {
	parts := strings.Split(digitalStatus, ";")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		idx := strings.Index(s, ":")
		if idx >= 0 {
			label := strings.TrimSpace(s[idx+1:])
			if label != "" {
				labels = append(labels, label)
			}
			continue
		}
		labels = append(labels, s)
	}
	return labels
}

func isRangeEmpty(v any) bool {
	if v == nil {
		return true
	}
	s := strings.TrimSpace(toString(v))
	if s == "" || s == "[]" {
		return true
	}
	if arr, ok := v.([]string); ok {
		return len(arr) == 0
	}
	if arr, ok := v.([]float64); ok {
		return len(arr) == 0
	}
	if arr, ok := v.([]any); ok {
		return len(arr) == 0
	}
	return false
}

func parseFloat(v string) *float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return nil
	}
	return &f
}

func tryParseFloatOut(v string) string {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return v
	}
	if math.Mod(f, 1.0) == 0 {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%v", f)
}

func safeString(v any) string {
	if v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if strings.EqualFold(s, "<nil>") {
		return ""
	}
	return s
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func recordOrEmpty(rec map[string]any, key string) any {
	v, ok := rec[key]
	if !ok {
		return ""
	}
	return v
}
