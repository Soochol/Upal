package skills

// Provider abstracts read access to skill content.
// Registry satisfies this interface implicitly.
type Provider interface {
	Get(name string) string
}
