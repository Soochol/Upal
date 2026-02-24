package upal

// ModelSummary is a lightweight model descriptor for generation contexts.
type ModelSummary struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Tier     string `json:"tier,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// ToolSummary is a lightweight tool descriptor for generation contexts.
type ToolSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
