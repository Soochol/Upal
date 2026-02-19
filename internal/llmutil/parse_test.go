package llmutil

import (
	"testing"
)

func TestStripMarkdownJSON_CleanJSON(t *testing.T) {
	input := `{"name": "test", "value": 42}`
	got, err := StripMarkdownJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestStripMarkdownJSON_JSONFenced(t *testing.T) {
	input := "```json\n{\"name\": \"test\"}\n```"
	got, err := StripMarkdownJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"name": "test"}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStripMarkdownJSON_LeadingText(t *testing.T) {
	input := "Here is the result:\n{\"name\": \"test\"}"
	got, err := StripMarkdownJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"name": "test"}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStripMarkdownJSON_GenericFence(t *testing.T) {
	input := "```\n{\"key\": \"value\"}\n```"
	got, err := StripMarkdownJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"key": "value"}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStripMarkdownJSON_NoJSON(t *testing.T) {
	input := "This is just plain text with no JSON object."
	_, err := StripMarkdownJSON(input)
	if err == nil {
		t.Fatal("expected error for text with no JSON")
	}
}

func TestStripMarkdownJSON_LeadingTemplateText(t *testing.T) {
	// Ensures {{template}} syntax in leading text doesn't cause false matches.
	input := "{{node_id}} template reference.\n\n{\"name\": \"workflow\"}"
	got, err := StripMarkdownJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"name": "workflow"}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
