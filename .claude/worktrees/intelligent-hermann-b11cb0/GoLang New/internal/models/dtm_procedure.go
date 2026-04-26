package models

import "time"

// DTMProcedureRow is a single row in the derived telemetry procedure table.
// Each row defines a mnemonic and a Julia/DSL procedure that computes its value.
// The procedure runs every second; the result is written to DTM_MAP and TM_MAP.
//
// The procedure must assign the computed value to a Julia variable named "result".
// Example:
//
//	val1 = parse(Float64, TM.PAYLOAD.ACM05521)
//	val2 = parse(Float64, TM.PAYLOAD.ACM05522)
//	result = (val1 + val2) / 2
type DTMProcedureRow struct {
	RowIndex    int     `json:"row_index"`
	Mnemonic    string  `json:"mnemonic"`
	Procedure   string  `json:"procedure"`   // Julia/DSL code; must set "result"
	Description string  `json:"description"`
	Type        string  `json:"type"`        // "ANALOG" or "DIGITAL"
	Unit        string  `json:"unit"`
	Range       string  `json:"range"`       // ANALOG: "min:max" e.g. "-5:5"; DIGITAL: comma-separated states e.g. "on,off"
	Tolerance   float64 `json:"tolerance"`   // numeric tolerance for limit checking
	Enabled     bool    `json:"enabled"`
}

// DTMProcedures is the current DTM procedure document stored per project in dtm_procedures table.
type DTMProcedures struct {
	Project   string           `json:"project"`
	Rows      []DTMProcedureRow `json:"rows"`
	Version   int              `json:"version"`
	CreatedBy string           `json:"created_by"`
	CreatedAt time.Time        `json:"created_at"`
}
