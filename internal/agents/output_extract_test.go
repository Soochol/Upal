// internal/agents/output_extract_test.go
package agents

import (
	"strings"
	"testing"
)

func TestApplyOutputExtract_Nil(t *testing.T) {
	result := applyOutputExtract(nil, "raw text")
	if result != "raw text" {
		t.Fatalf("expected passthrough, got %q", result)
	}
}

func TestApplyOutputExtract_JSON(t *testing.T) {
	cfg := &outputExtractConfig{mode: "json", key: "result"}
	raw := `{"result": "hello world"}`
	got := applyOutputExtract(cfg, raw)
	if got != "hello world" {
		t.Fatalf("want %q, got %q", "hello world", got)
	}
}

func TestApplyOutputExtract_JSON_EmbeddedInText(t *testing.T) {
	cfg := &outputExtractConfig{mode: "json", key: "result"}
	raw := "Sure! Here you go:\n{\"result\": \"fan chant text\"}"
	got := applyOutputExtract(cfg, raw)
	if got != "fan chant text" {
		t.Fatalf("want %q, got %q", "fan chant text", got)
	}
}

func TestApplyOutputExtract_JSON_Fallback(t *testing.T) {
	cfg := &outputExtractConfig{mode: "json", key: "result"}
	raw := "No JSON here at all"
	got := applyOutputExtract(cfg, raw)
	if got != raw {
		t.Fatalf("should fallback to raw, got %q", got)
	}
}

func TestApplyOutputExtract_Tagged(t *testing.T) {
	cfg := &outputExtractConfig{mode: "tagged", tag: "artifact"}
	raw := "Thinking...\n<artifact>the final content</artifact>"
	got := applyOutputExtract(cfg, raw)
	if got != "the final content" {
		t.Fatalf("want %q, got %q", "the final content", got)
	}
}

func TestApplyOutputExtract_Tagged_Multiline(t *testing.T) {
	cfg := &outputExtractConfig{mode: "tagged", tag: "artifact"}
	raw := "<artifact>\nline1\nline2\n</artifact>"
	got := applyOutputExtract(cfg, raw)
	want := "\nline1\nline2\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestApplyOutputExtract_Tagged_Fallback(t *testing.T) {
	cfg := &outputExtractConfig{mode: "tagged", tag: "artifact"}
	raw := "No tags here"
	got := applyOutputExtract(cfg, raw)
	if got != raw {
		t.Fatalf("should fallback to raw, got %q", got)
	}
}

func TestOutputExtractSystemPrompt_JSON(t *testing.T) {
	cfg := &outputExtractConfig{mode: "json", key: "result"}
	prompt := cfg.systemPromptAppend()
	if !strings.Contains(prompt, `"result"`) {
		t.Fatalf("system prompt should mention key name, got: %s", prompt)
	}
}

func TestOutputExtractSystemPrompt_Tagged(t *testing.T) {
	cfg := &outputExtractConfig{mode: "tagged", tag: "output"}
	prompt := cfg.systemPromptAppend()
	if !strings.Contains(prompt, "<output>") {
		t.Fatalf("system prompt should mention tag, got: %s", prompt)
	}
}

func TestParseOutputExtract_JSON(t *testing.T) {
	cfg := map[string]any{
		"output_extract": map[string]any{
			"mode": "json",
			"key":  "answer",
		},
	}
	oe := parseOutputExtract(cfg)
	if oe == nil || oe.mode != "json" || oe.key != "answer" {
		t.Fatalf("unexpected: %+v", oe)
	}
}

func TestParseOutputExtract_Tagged(t *testing.T) {
	cfg := map[string]any{
		"output_extract": map[string]any{
			"mode": "tagged",
			"tag":  "artifact",
		},
	}
	oe := parseOutputExtract(cfg)
	if oe == nil || oe.mode != "tagged" || oe.tag != "artifact" {
		t.Fatalf("unexpected: %+v", oe)
	}
}

func TestParseOutputExtract_Missing(t *testing.T) {
	oe := parseOutputExtract(map[string]any{"model": "test"})
	if oe != nil {
		t.Fatalf("expected nil, got %+v", oe)
	}
}

func TestParseOutputExtract_EmptyKey(t *testing.T) {
	cfg := map[string]any{
		"output_extract": map[string]any{
			"mode": "json",
			"key":  "",
		},
	}
	oe := parseOutputExtract(cfg)
	if oe != nil {
		t.Fatalf("should return nil for empty key, got %+v", oe)
	}
}

func TestParseOutputExtract_UnknownMode(t *testing.T) {
	cfg := map[string]any{
		"output_extract": map[string]any{
			"mode": "regex",
			"key":  "result",
		},
	}
	oe := parseOutputExtract(cfg)
	if oe != nil {
		t.Fatalf("should return nil for unknown mode, got %+v", oe)
	}
}

func TestParseOutputExtract_EmptyTag(t *testing.T) {
	cfg := map[string]any{
		"output_extract": map[string]any{
			"mode": "tagged",
			"tag":  "",
		},
	}
	oe := parseOutputExtract(cfg)
	if oe != nil {
		t.Fatalf("should return nil for empty tag, got %+v", oe)
	}
}

func TestApplyOutputExtract_JSON_BraceInStringValue(t *testing.T) {
	cfg := &outputExtractConfig{mode: "json", key: "result"}
	// Value contains an unbalanced open brace — must not break extraction
	raw := `{"result": "if (x > 0) {return x"}`
	got := applyOutputExtract(cfg, raw)
	if got != "if (x > 0) {return x" {
		t.Fatalf("want %q, got %q", "if (x > 0) {return x", got)
	}
}

func TestApplyOutputExtract_JSON_TrailingBrace(t *testing.T) {
	cfg := &outputExtractConfig{mode: "json", key: "result"}
	raw := `{"result": "hello"} and some text with a brace}`
	got := applyOutputExtract(cfg, raw)
	if got != "hello" {
		t.Fatalf("want %q, got %q", "hello", got)
	}
}
