package provider

import "testing"

func TestParseModelID(t *testing.T) {
	tests := []struct {
		input    string
		provider string
		model    string
		wantErr  bool
	}{
		{"openai/gpt-4o", "openai", "gpt-4o", false},
		{"ollama/llama3.2", "ollama", "llama3.2", false},
		{"anthropic/claude-sonnet-4-20250514", "anthropic", "claude-sonnet-4-20250514", false},
		{"invalid", "", "", true},
		{"", "", "", true},
	}
	for _, tt := range tests {
		p, m, err := ParseModelID(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseModelID(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			continue
		}
		if p != tt.provider || m != tt.model {
			t.Errorf("ParseModelID(%q): got (%q,%q), want (%q,%q)", tt.input, p, m, tt.provider, tt.model)
		}
	}
}

func TestMessage_Roles(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "You are helpful."},
		{Role: RoleUser, Content: "Hello"},
		{Role: RoleAssistant, Content: "Hi there!"},
	}
	if msgs[0].Role != RoleSystem {
		t.Errorf("role: got %q, want system", msgs[0].Role)
	}
}
