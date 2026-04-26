package models

// SpasdacsDiagram is a SPASDACS mimic diagram stored in the spasdacs SQLite table.
type SpasdacsDiagram struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Description      string      `json:"description"`
	ModelData        interface{} `json:"modelData"`
	BackgroundColor  string      `json:"backgroundColor"`
	CreatedAt        string      `json:"createdAt"`
	UpdatedAt        string      `json:"updatedAt"`
	AutoViewInclude  *bool       `json:"autoViewInclude,omitempty"`
	AutoViewDuration *int        `json:"autoViewDuration,omitempty"`
}

// SpasdacsMeta is the list-view representation (no ModelData) returned by GET /diagrams.
type SpasdacsMeta struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
	AutoViewInclude  *bool  `json:"autoViewInclude,omitempty"`
	AutoViewDuration *int   `json:"autoViewDuration,omitempty"`
}
