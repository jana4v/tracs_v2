package models

// TcParameter describes a single parameter of a TC command.
type TcParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// TcMnemonic represents a telecommand definition stored in the tc_mnemonics SQLite table.
type TcMnemonic struct {
	Command     string        `json:"command"`
	FullRef     string        `json:"-"`      // computed: "TC.<command>"
	Description string        `json:"description"`
	Parameters  []TcParameter `json:"parameters"`
	Subsystem   string        `json:"subsystem"`
	Category    string        `json:"category"`
}
