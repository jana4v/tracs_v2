package models

import "time"

// UDTMRow is a single row in the user-defined telemetry table.
type UDTMRow struct {
	RowIndex    int       `json:"row_index"`
	Mnemonic    string    `json:"mnemonic"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // "ANALOG" or "BINARY"
	Unit        string    `json:"unit"`
	Value       string    `json:"value"`
	LastUpdated time.Time `json:"last_updated"`
}

// UserTelemetry is the current UD_TM document stored per project in user_telemetry table.
type UserTelemetry struct {
	Project   string    `json:"project"`
	Rows      []UDTMRow `json:"rows"`
	Version   int       `json:"version"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// UserTelemetryVersion is a historical snapshot of UD_TM rows in user_telemetry_versions table.
type UserTelemetryVersion struct {
	Project       string    `json:"project"`
	Version       int       `json:"version"`
	Rows          []UDTMRow `json:"rows"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	ChangeMessage string    `json:"change_message"`
}
