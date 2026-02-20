package skills

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
	if len(r.skills) == 0 {
		t.Fatal("registry has no skills loaded")
	}
}

func TestGetNodeSkills(t *testing.T) {
	r := New()

	expectedSkills := []string{
		"agent-node",
		"input-node",
		"output-node",
	}

	for _, name := range expectedSkills {
		content := r.Get(name)
		if content == "" {
			t.Errorf("skill %q not found or empty", name)
		}
	}
}

func TestGetReturnsEmptyForMissing(t *testing.T) {
	r := New()
	if got := r.Get("nonexistent-skill"); got != "" {
		t.Errorf("Get(nonexistent) = %q, want empty string", got)
	}
}

func TestMustGetPanicsForMissing(t *testing.T) {
	r := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet(nonexistent) did not panic")
		}
	}()
	r.MustGet("nonexistent-skill")
}

func TestIncludeResolution(t *testing.T) {
	r := New()

	// agent-node.md uses {{include system-prompt}} and {{include prompt-framework}}.
	// After resolution, the content should contain text from the frameworks,
	// NOT the raw {{include ...}} directives.
	agentSkill := r.Get("agent-node")
	if agentSkill == "" {
		t.Fatal("agent-node skill not found")
	}

	if strings.Contains(agentSkill, "{{include system-prompt}}") {
		t.Error("agent-node still contains unresolved {{include system-prompt}}")
	}
	if strings.Contains(agentSkill, "{{include prompt-framework}}") {
		t.Error("agent-node still contains unresolved {{include prompt-framework}}")
	}

	// The resolved content should contain signature text from each framework.
	if !strings.Contains(agentSkill, "SYSTEM PROMPT FRAMEWORK") {
		t.Error("agent-node missing system-prompt content (expected 'SYSTEM PROMPT FRAMEWORK')")
	}
	if !strings.Contains(agentSkill, "USER PROMPT FRAMEWORK") {
		t.Error("agent-node missing prompt-framework content (expected 'USER PROMPT FRAMEWORK')")
	}
}

func TestFrontmatterStripped(t *testing.T) {
	r := New()

	// Skill content should NOT contain YAML frontmatter delimiters.
	for _, name := range r.Names() {
		content := r.Get(name)
		if strings.HasPrefix(strings.TrimSpace(content), "---") {
			t.Errorf("skill %q still contains frontmatter", name)
		}
	}
}

func TestNames(t *testing.T) {
	r := New()
	names := r.Names()
	if len(names) < 3 {
		t.Errorf("Names() returned %d skills, want at least 3", len(names))
	}

	// Check that all expected names are present.
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for _, expected := range []string{"agent-node", "input-node", "output-node"} {
		if !nameSet[expected] {
			t.Errorf("Names() missing %q", expected)
		}
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantBody string
	}{
		{
			input:    "---\nname: test-skill\ndescription: A test\n---\n\nBody here",
			wantName: "test-skill",
			wantBody: "Body here",
		},
		{
			input:    "No frontmatter content",
			wantName: "",
			wantBody: "No frontmatter content",
		},
		{
			input:    "---\nname: \"quoted\"\n---\n\nQuoted body",
			wantName: "quoted",
			wantBody: "Quoted body",
		},
	}

	for _, tt := range tests {
		name, body := parseFrontmatter(tt.input)
		if name != tt.wantName {
			t.Errorf("parseFrontmatter(%q): name = %q, want %q", tt.input, name, tt.wantName)
		}
		if body != tt.wantBody {
			t.Errorf("parseFrontmatter(%q): body = %q, want %q", tt.input, body, tt.wantBody)
		}
	}
}

func TestResolveIncludes(t *testing.T) {
	frameworks := map[string]string{
		"greeting": "Hello, World!",
		"farewell": "Goodbye!",
	}

	tests := []struct {
		input string
		want  string
	}{
		{"Start {{include greeting}} End", "Start Hello, World! End"},
		{"{{include greeting}} and {{include farewell}}", "Hello, World! and Goodbye!"},
		{"{{include unknown}}", "{{include unknown}}"}, // unresolved stays as-is
		{"No includes here", "No includes here"},
	}

	for _, tt := range tests {
		got := resolveIncludes(tt.input, frameworks)
		if got != tt.want {
			t.Errorf("resolveIncludes(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
