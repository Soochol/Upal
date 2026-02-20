// Package skills provides a registry of embedded skill files that guide LLM
// behavior when generating node configurations. Each skill is a Markdown file
// with YAML frontmatter (name + description) and a body containing instructions.
//
// Shared fragments in _frameworks/ can be included via {{include name}} syntax.
package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

//go:embed *.md _frameworks/*.md
var embedded embed.FS

var includePattern = regexp.MustCompile(`\{\{include\s+([\w-]+)\}\}`)

// Registry holds parsed and resolved skill content indexed by name.
type Registry struct {
	skills map[string]string
}

// New creates a Registry by loading all embedded skill files and resolving
// {{include}} references from _frameworks/.
func New() *Registry {
	r := &Registry{skills: make(map[string]string)}

	// Load framework fragments first (used for include resolution).
	frameworks := make(map[string]string)
	frameworkEntries, _ := fs.ReadDir(embedded, "_frameworks")
	for _, e := range frameworkEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := fs.ReadFile(embedded, "_frameworks/"+e.Name())
		if err != nil {
			continue
		}
		name, body := parseFrontmatter(string(data))
		if name == "" {
			name = strings.TrimSuffix(e.Name(), ".md")
		}
		frameworks[name] = body
	}

	// Load top-level skill files.
	entries, _ := fs.ReadDir(embedded, ".")
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := fs.ReadFile(embedded, e.Name())
		if err != nil {
			continue
		}
		name, body := parseFrontmatter(string(data))
		if name == "" {
			name = strings.TrimSuffix(e.Name(), ".md")
		}
		resolved := resolveIncludes(body, frameworks)
		r.skills[name] = resolved
	}

	return r
}

// Get returns the resolved content of a skill by name.
// Returns an empty string if the skill does not exist.
func (r *Registry) Get(name string) string {
	return r.skills[name]
}

// Names returns all registered skill names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.skills))
	for k := range r.skills {
		names = append(names, k)
	}
	return names
}

// MustGet returns the resolved content of a skill or panics if not found.
func (r *Registry) MustGet(name string) string {
	s, ok := r.skills[name]
	if !ok {
		panic(fmt.Sprintf("skill %q not found", name))
	}
	return s
}

// parseFrontmatter splits YAML frontmatter from the body.
// Frontmatter is delimited by --- lines. Returns (name, body).
func parseFrontmatter(content string) (string, string) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return "", content
	}

	// Find the closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", content
	}

	frontmatter := rest[:idx]
	body := strings.TrimSpace(rest[idx+4:])

	// Extract name from frontmatter (simple line-based parsing, no YAML dep).
	var name string
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			name = strings.Trim(name, `"'`)
		}
	}

	return name, body
}

// resolveIncludes replaces {{include name}} markers with framework content.
func resolveIncludes(content string, frameworks map[string]string) string {
	return includePattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := includePattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		if replacement, ok := frameworks[sub[1]]; ok {
			return replacement
		}
		return match // leave unresolved includes as-is
	})
}
