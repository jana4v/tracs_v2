package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FlexStringID is a plain string alias retained for backwards compatibility
// with existing code that references this type.
type FlexStringID = string

// FlexAnySlice is a plain slice alias retained for backwards compatibility.
type FlexAnySlice = []interface{}

// FlexFloat64 is a float64 that can be unmarshalled from either a JSON number
// or a quoted string (e.g. "0.5" or 0.5). This is necessary because the TM
// parser stores tolerance values as strings in the SQLite data column.
type FlexFloat64 float64

// UnmarshalJSON implements json.Unmarshaler, accepting both numeric and
// string-encoded float values.
func (f *FlexFloat64) UnmarshalJSON(b []byte) error {
	var n float64
	if err := json.Unmarshal(b, &n); err == nil {
		*f = FlexFloat64(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*f = 0
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		*f = 0
		return nil
	}
	*f = FlexFloat64(n)
	return nil
}

// TmMnemonic represents a telemetry mnemonic definition stored in the
// tm_mnemonics SQLite table. Fields map 1-to-1 with the JSON document stored
// in the `data` column.
type TmMnemonic struct {
	ID               string        `json:"_id"`
	Subsystem        string        `json:"subsystem"`
	Type             string        `json:"type"`             // "BINARY" or "ANALOG"
	ProcessingType   string        `json:"processingType"`   // e.g. "STATUS", "EUCN-16B"
	Range            []interface{} `json:"range"`            // ANALOG: [min, max]; BINARY: list of states
	Limits           []interface{} `json:"limits,omitempty"` // Editable limit pair used by Update TM DB
	ExpectedValue    string        `json:"expected_value,omitempty"`
	Tolerance        FlexFloat64   `json:"tolerance"`     // For ANALOG comparison/storage threshold
	Unit             string        `json:"unit"`          // e.g. "INT", "V"
	DigitalStatus    string        `json:"digitalStatus"` // Metadata only
	CdbMnemonic      string        `json:"cdbMnemonic"`   // Human-readable mnemonic name
	SourceFile       string        `json:"sourceFile"`    // Audit metadata
	IgnoreLimitCheck bool          `json:"ignore_limit_check"`
	IgnoreChangeDet  bool          `json:"ignore_change_detection"`
	IgnoreChainCmp   bool          `json:"ignore_chain_comparision"`
	EnableComparison bool          `json:"enable_comparison"` // Skip chain comparison if false
	EnableLimit      bool          `json:"enable_limit"`      // Skip limit monitoring if false
	EnableStorage    bool          `json:"enable_storage"`    // Skip InfluxDB storage if false
}

// IsAnalog returns true if the mnemonic type is ANALOG.
func (m *TmMnemonic) IsAnalog() bool {
	return strings.EqualFold(strings.TrimSpace(m.Type), "ANALOG")
}

// IsBinary returns true if the mnemonic type is BINARY.
func (m *TmMnemonic) IsBinary() bool {
	return strings.EqualFold(strings.TrimSpace(m.Type), "BINARY")
}

// GetRangeStr returns the Range field as []string, converting from any type.
func (m *TmMnemonic) GetRangeStr() []string {
	if m.Range == nil {
		return nil
	}
	result := make([]string, len(m.Range))
	for i, v := range m.Range {
		switch val := v.(type) {
		case string:
			result[i] = val
		case float64:
			result[i] = strconv.FormatFloat(val, 'f', -1, 64)
		default:
			result[i] = fmt.Sprintf("%v", val)
		}
	}
	return result
}
