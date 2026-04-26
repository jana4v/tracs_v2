package parsers_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mainframe/tm-system/gateway/internal/parsers"
	"github.com/xuri/excelize/v2"
)

// ── NormalizeRangeLikeValue ───────────────────────────────────────────────────

func TestNormalizeRangeLikeValue_Nil(t *testing.T) {
	got := parsers.NormalizeRangeLikeValue(nil)
	arr, ok := got.([]any)
	if !ok || len(arr) != 0 {
		t.Fatalf("expected empty []any, got %T %v", got, got)
	}
}

func TestNormalizeRangeLikeValue_Float64Slice(t *testing.T) {
	in := []float64{0.5, 100.0}
	got := parsers.NormalizeRangeLikeValue(in)
	out, ok := got.([]float64)
	if !ok || len(out) != 2 || out[0] != 0.5 || out[1] != 100.0 {
		t.Fatalf("unexpected %T %v", got, got)
	}
}

func TestNormalizeRangeLikeValue_StringSlice_FiltersEmpty(t *testing.T) {
	in := []string{"OFF", "", "ON", "  "}
	got := parsers.NormalizeRangeLikeValue(in)
	out, ok := got.([]string)
	if !ok || len(out) != 2 || out[0] != "OFF" || out[1] != "ON" {
		t.Fatalf("unexpected %T %v", got, got)
	}
}

func TestNormalizeRangeLikeValue_StringWithSeparator(t *testing.T) {
	// "0 to 100" should parse into []float64{0, 100}
	got := parsers.NormalizeRangeLikeValue("0 to 100")
	out, ok := got.([]float64)
	if !ok || len(out) != 2 || out[0] != 0 || out[1] != 100 {
		t.Fatalf("unexpected %T %v", got, got)
	}
}

func TestNormalizeRangeLikeValue_CSV(t *testing.T) {
	got := parsers.NormalizeRangeLikeValue("OFF,ON,STANDBY")
	out, ok := got.([]string)
	if !ok || len(out) != 3 {
		t.Fatalf("unexpected %T %v", got, got)
	}
}

// ── CloneRangeLikeValue ───────────────────────────────────────────────────────

func TestCloneRangeLikeValue_IndependentFloat64(t *testing.T) {
	in := []float64{1, 2}
	clone := parsers.CloneRangeLikeValue(in)
	out := clone.([]float64)
	out[0] = 999
	if in[0] == 999 {
		t.Fatal("clone shares underlying array with original")
	}
}

func TestCloneRangeLikeValue_IndependentString(t *testing.T) {
	in := []string{"A", "B"}
	clone := parsers.CloneRangeLikeValue(in)
	out := clone.([]string)
	out[0] = "X"
	if in[0] == "X" {
		t.Fatal("clone shares underlying array with original")
	}
}

// ── DeriveExpectedValue ───────────────────────────────────────────────────────

func TestDeriveExpectedValue_ExplicitTakesPriority(t *testing.T) {
	rec := map[string]any{
		"expected_value": "NOMINAL",
		"type":           "BINARY",
		"range":          []string{"OFF", "ON"},
	}
	if got := parsers.DeriveExpectedValue(rec); got != "NOMINAL" {
		t.Fatalf("expected NOMINAL, got %q", got)
	}
}

func TestDeriveExpectedValue_AnalogAlwaysEmpty(t *testing.T) {
	rec := map[string]any{
		"type":  "ANALOG",
		"range": []float64{0, 100},
	}
	if got := parsers.DeriveExpectedValue(rec); got != "" {
		t.Fatalf("expected empty for ANALOG, got %q", got)
	}
}

func TestDeriveExpectedValue_BinaryFirstLabel(t *testing.T) {
	rec := map[string]any{
		"type":  "BINARY",
		"range": []string{"OFF", "ON"},
	}
	if got := parsers.DeriveExpectedValue(rec); got != "OFF" {
		t.Fatalf("expected OFF, got %q", got)
	}
}

func TestDeriveExpectedValue_NilRange(t *testing.T) {
	rec := map[string]any{
		"type": "BINARY",
	}
	// nil range → no derived expected value
	if got := parsers.DeriveExpectedValue(rec); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

// ── ParseOUTGroup ─────────────────────────────────────────────────────────────

func TestParseOUTGroup_Nil_OnEmptyGroup(t *testing.T) {
	if got := parsers.ParseOUTGroup(nil, "test.out"); got != nil {
		t.Fatalf("expected nil for empty group, got %v", got)
	}
}

func TestParseOUTGroup_Nil_OnTooFewTokens(t *testing.T) {
	// only 2 tokens — should return nil
	if got := parsers.ParseOUTGroup([]string{"ABC12345 TMP1"}, "x.out"); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestParseOUTGroup_BinaryWithDigitalStates(t *testing.T) {
	line := "ABC12345 TMP1 STATUS PAYLOAD NONE 0.1 % 00:OFF 01:ON"
	got := parsers.ParseOUTGroup([]string{line}, "x.out")
	if got == nil {
		t.Fatal("expected non-nil record")
	}
	if got["type"] != "BINARY" {
		t.Fatalf("expected BINARY type, got %v", got["type"])
	}
	ranges, ok := got["range"].([]string)
	if !ok || len(ranges) == 0 {
		t.Fatalf("expected range labels, got %T %v", got["range"], got["range"])
	}
}

func TestParseOUTGroup_AnalogWithSimple(t *testing.T) {
	line := "ABC12345 TMP1 EUCN PAYLOAD SIMPLE 100 0 0.5 degC"
	got := parsers.ParseOUTGroup([]string{line}, "x.out")
	if got == nil {
		t.Fatal("expected non-nil record")
	}
	if got["type"] != "ANALOG" {
		t.Fatalf("expected ANALOG, got %v", got["type"])
	}
	ranges, ok := got["range"].([]float64)
	if !ok || len(ranges) != 2 {
		t.Fatalf("expected [low, high] float64 slice, got %T %v", got["range"], got["range"])
	}
	// SIMPLE order: high, low → stored as [low, high]
	if ranges[0] != 0 || ranges[1] != 100 {
		t.Fatalf("expected [0, 100], got %v", ranges)
	}
}

// ── ParseOUT (file round-trip) ────────────────────────────────────────────────

func TestParseOUT_SkipsFirst7Lines(t *testing.T) {
	// Lines 0-6 are header; data starts at line 7
	content := ""
	for i := 0; i < 7; i++ {
		content += fmt.Sprintf("header line %d\n", i)
	}
	// valid primary row
	content += "ABC12345 MN1 EUCN PAYLOAD SIMPLE 100 0 0.5 degC\n"

	path := writeTempFile(t, content, ".out")
	records, err := parsers.ParseOUT(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0]["cdbPidNo"] != "ABC12345" {
		t.Fatalf("unexpected cdbPidNo: %v", records[0]["cdbPidNo"])
	}
}

func TestParseOUT_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "", ".out")
	records, err := parsers.ParseOUT(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestParseOUT_InvalidPath(t *testing.T) {
	_, err := parsers.ParseOUT("/does/not/exist.out")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// ── ParseXLSX ────────────────────────────────────────────────────────────────

func TestParseXLSX_InvalidPath(t *testing.T) {
	_, err := parsers.ParseXLSX("/no/such/file.xlsx")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseXLSX_EmptySheet(t *testing.T) {
	// Only a header row — ParseXLSX skips rowIdx 0, so no records produced.
	path := writeTempXLSX(t, map[string][][]string{
		"TM": {{"slNo", "subsystem", "cdbPidNo", "cdbMnemonic", "description", "type"}},
	})
	records, err := parsers.ParseXLSX(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestParseXLSX_SkipsInformationSheet(t *testing.T) {
	header := xlsxRow(6, nil)
	dataRow := xlsxRow(6, map[int]string{2: "ABC12345", 5: "A"})
	path := writeTempXLSX(t, map[string][][]string{
		"Information": {header, dataRow},
	})
	records, err := parsers.ParseXLSX(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records from Information sheet, got %d", len(records))
	}
}

func TestParseXLSX_SkipsLUTSheet(t *testing.T) {
	header := xlsxRow(6, nil)
	dataRow := xlsxRow(6, map[int]string{2: "ABC12345", 5: "A"})
	path := writeTempXLSX(t, map[string][][]string{
		"LUT_TBL_INFO_TEMP": {header, dataRow},
	})
	records, err := parsers.ParseXLSX(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records from LUT sheet, got %d", len(records))
	}
}

func TestParseXLSX_SkipsRowsWithEmptyPid(t *testing.T) {
	header := xlsxRow(6, nil)
	// col index 2 (cdbPidNo) deliberately left empty
	emptyPid := xlsxRow(6, map[int]string{5: "A"})
	path := writeTempXLSX(t, map[string][][]string{
		"TM": {header, emptyPid},
	})
	records, err := parsers.ParseXLSX(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records (empty pid skipped), got %d", len(records))
	}
}

func TestParseXLSX_AnalogRecord(t *testing.T) {
	// col indices (0-based): 2=cdbPidNo, 5=type "A", 14=range "0 to 100"
	header := xlsxRow(15, nil)
	dataRow := xlsxRow(15, map[int]string{2: "TMP12345", 5: "A", 14: "0 to 100"})
	path := writeTempXLSX(t, map[string][][]string{
		"TM": {header, dataRow},
	})
	records, err := parsers.ParseXLSX(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	rec := records[0]
	if rec["type"] != "ANALOG" {
		t.Errorf("expected type ANALOG, got %v", rec["type"])
	}
	rng, ok := rec["range"].([]float64)
	if !ok || len(rng) != 2 {
		t.Fatalf("expected []float64 range, got %T %v", rec["range"], rec["range"])
	}
	if rng[0] != 0 || rng[1] != 100 {
		t.Errorf("expected [0, 100], got %v", rng)
	}
	if rec["cdbPidNo"] != "TMP12345" {
		t.Errorf("expected cdbPidNo TMP12345, got %v", rec["cdbPidNo"])
	}
	if rec["sourceSheet"] != "TM" {
		t.Errorf("expected sourceSheet TM, got %v", rec["sourceSheet"])
	}
}

func TestParseXLSX_BinaryRecord(t *testing.T) {
	// col 14 (0-based) = range field; comma-separated → []string
	header := xlsxRow(15, nil)
	dataRow := xlsxRow(15, map[int]string{2: "TMP12345", 5: "B", 14: "OFF,ON"})
	path := writeTempXLSX(t, map[string][][]string{
		"TM": {header, dataRow},
	})
	records, err := parsers.ParseXLSX(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	rec := records[0]
	if rec["type"] != "BINARY" {
		t.Errorf("expected type BINARY, got %v", rec["type"])
	}
	rng, ok := rec["range"].([]string)
	if !ok || len(rng) != 2 {
		t.Fatalf("expected []string range with 2 labels, got %T %v", rec["range"], rec["range"])
	}
	if rng[0] != "OFF" || rng[1] != "ON" {
		t.Errorf("expected [OFF ON], got %v", rng)
	}
}

func TestParseXLSX_BinaryDigitalStatusFallback(t *testing.T) {
	// type=B, range col empty → digitalStatus col (0-based index 21) used as fallback
	header := xlsxRow(22, nil)
	dataRow := xlsxRow(22, map[int]string{2: "TMP12345", 5: "B", 14: "", 21: "00:OFF;01:ON"})
	path := writeTempXLSX(t, map[string][][]string{
		"TM": {header, dataRow},
	})
	records, err := parsers.ParseXLSX(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	rng, ok := records[0]["range"].([]string)
	if !ok || len(rng) != 2 {
		t.Fatalf("expected digitalStatus-derived range, got %T %v", records[0]["range"], records[0]["range"])
	}
	if rng[0] != "OFF" || rng[1] != "ON" {
		t.Errorf("expected [OFF ON], got %v", rng)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// writeTempXLSX creates a temporary .xlsx file with the given sheets and rows,
// where each row is a []string of cell values. Row 0 in each sheet is the header.
func writeTempXLSX(t *testing.T, sheets map[string][][]string) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()

	renamed := false
	for sheet, rows := range sheets {
		if !renamed {
			// excelize always starts with a default "Sheet1"; rename it.
			if err := f.SetSheetName("Sheet1", sheet); err != nil {
				t.Fatalf("rename sheet: %v", err)
			}
			renamed = true
		} else {
			if _, err := f.NewSheet(sheet); err != nil {
				t.Fatalf("new sheet %q: %v", sheet, err)
			}
		}
		for rowIdx, row := range rows {
			for colIdx, val := range row {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err := f.SetCellValue(sheet, cell, val); err != nil {
					t.Fatalf("set cell %s: %v", cell, err)
				}
			}
		}
	}

	tmp, err := os.CreateTemp("", "parsers_test_*.xlsx")
	if err != nil {
		t.Fatalf("create temp xlsx: %v", err)
	}
	path := filepath.Clean(tmp.Name())
	tmp.Close()
	os.Remove(path) // SaveAs will (re)create it

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save xlsx: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })
	return path
}

// xlsxRow returns a string slice of the given length with values set at the
// specified 0-based column indices.
func xlsxRow(length int, cols map[int]string) []string {
	row := make([]string, length)
	for i, v := range cols {
		if i < length {
			row[i] = v
		}
	}
	return row
}

func writeTempFile(t *testing.T, content, ext string) string {
	t.Helper()
	f, err := os.CreateTemp("", "parsers_test_*"+ext)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	path := filepath.Clean(f.Name())
	t.Cleanup(func() { os.Remove(path) })
	return path
}
