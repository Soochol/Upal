// Package skills provides a registry of embedded skill files and base prompts
// that guide LLM behavior. Skill files (nodes/*.md and stages/*.md) are loaded
// on demand via get_skill(); prompt files (prompts/*.md) are always loaded and
// used as base system prompts. Both support {{include name}} syntax for shared
// fragments in _frameworks/.
package skills

import (
	"embed"
	"io/fs"
	"regexp"
	"strings"
)

//go:embed nodes/*.md stages/*.md tools/*.md prompts/*.md _frameworks/*.md
var embedded embed.FS

var includePattern = regexp.MustCompile(`\{\{include\s+([\w-]+)\}\}`)

// Registry holds parsed and resolved skill content and base prompts indexed by name.
type Registry struct {
	skills  map[string]string // top-level *.md — on-demand skill docs
	prompts map[string]string // prompts/*.md — base system prompts
}

// New creates a Registry by loading all embedded files and resolving
// {{include}} references from _frameworks/.
func New() *Registry {
	r := &Registry{
		skills:  make(map[string]string),
		prompts: make(map[string]string),
	}

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

	// Load skill files from nodes/, stages/, and tools/ subdirectories.
	for _, dir := range []string{"nodes", "stages", "tools"} {
		entries, _ := fs.ReadDir(embedded, dir)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			data, err := fs.ReadFile(embedded, dir+"/"+e.Name())
			if err != nil {
				continue
			}
			name, body := parseFrontmatter(string(data))
			if name == "" {
				name = strings.TrimSuffix(e.Name(), ".md")
			}
			r.skills[name] = resolveIncludes(body, frameworks)
		}
	}

	// Load prompt files from prompts/ subdirectory.
	promptEntries, _ := fs.ReadDir(embedded, "prompts")
	for _, e := range promptEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := fs.ReadFile(embedded, "prompts/"+e.Name())
		if err != nil {
			continue
		}
		name, body := parseFrontmatter(string(data))
		if name == "" {
			name = strings.TrimSuffix(e.Name(), ".md")
		}
		r.prompts[name] = resolveIncludes(body, frameworks)
	}

	return r
}

// Get returns the resolved content of a skill by name.
// Returns an empty string if the skill does not exist.
func (r *Registry) Get(name string) string {
	return r.skills[name]
}

// GetPrompt returns the resolved content of a base prompt by name.
// Returns an empty string if the prompt does not exist.
func (r *Registry) GetPrompt(name string) string {
	return r.prompts[name]
}

// Names returns all registered skill names (on-demand skill docs only, not prompts).
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.skills))
	for k := range r.skills {
		names = append(names, k)
	}
	return names
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
