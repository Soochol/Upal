package skills

// Provider abstracts read access to skill and prompt content.
// Registry satisfies this interface implicitly.
type Provider interface {
	Get(name string) string
	GetPrompt(name string) string
}
