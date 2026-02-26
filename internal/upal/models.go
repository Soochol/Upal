package upal

// ModelCategory classifies models into groups with shared configuration options.
type ModelCategory string

const (
	ModelCategoryText  ModelCategory = "text"
	ModelCategoryImage ModelCategory = "image"
	ModelCategoryTTS   ModelCategory = "tts"
)

// ModelTier classifies models by capability level for prompt-based selection.
type ModelTier string

const (
	ModelTierHigh ModelTier = "high"
	ModelTierMid  ModelTier = "mid"
	ModelTierLow  ModelTier = "low"
)

// OptionSchema describes a single configurable option for a model category.
type OptionSchema struct {
	Key     string         `json:"key"`
	Label   string         `json:"label"`
	Type    string         `json:"type"` // "slider", "number", "select"
	Min     *float64       `json:"min,omitempty"`
	Max     *float64       `json:"max,omitempty"`
	Step    *float64       `json:"step,omitempty"`
	Default any            `json:"default,omitempty"`
	Choices []OptionChoice `json:"choices,omitempty"`
}

// OptionChoice represents a single choice in a select-type option.
type OptionChoice struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

// ModelInfo describes a single model available in the system.
type ModelInfo struct {
	ID            string         `json:"id"`
	Provider      string         `json:"provider"`
	Name          string         `json:"name"`
	Category      ModelCategory  `json:"category"`
	Tier          ModelTier      `json:"tier,omitempty"`
	Hint          string         `json:"hint,omitempty"` // one-line capability hint for LLM selection
	Options       []OptionSchema `json:"options"`
	SupportsTools bool           `json:"supportsTools"`
	IsDefault     bool           `json:"isDefault,omitempty"`
}

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
