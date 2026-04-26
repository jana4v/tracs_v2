package models

// ScoCommand represents an SCO command definition stored in the sco_commands SQLite table.
type ScoCommand struct {
	Command     string `json:"command"`
	FullRef     string `json:"-"`      // computed: "SCO.<command>"
	Description string `json:"description"`
	Subsystem   string `json:"subsystem"`
	Category    string `json:"category"`
}
