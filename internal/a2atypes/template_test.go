package a2atypes

import "testing"

func TestResolveTemplate_TextArtifact(t *testing.T) {
	artifacts := map[string][]Artifact{
		"research": {{Parts: []Part{TextPart("AI is transformative")}, Index: 0}},
	}
	result := ResolveTemplate("Based on: {{research}}", artifacts)
	if result != "Based on: AI is transformative" {
		t.Errorf("got %q", result)
	}
}

func TestResolveTemplate_DataArtifact(t *testing.T) {
	artifacts := map[string][]Artifact{
		"research": {{Parts: []Part{DataPart(map[string]any{"key": "val"}, "application/json")}, Index: 0}},
	}
	result := ResolveTemplate("Data: {{research.data}}", artifacts)
	if result != `Data: {"key":"val"}` {
		t.Errorf("got %q", result)
	}
}

func TestResolveTemplate_MissingNode(t *testing.T) {
	artifacts := map[string][]Artifact{}
	result := ResolveTemplate("Hello {{missing}}", artifacts)
	if result != "Hello {{missing}}" {
		t.Errorf("got %q", result)
	}
}

func TestResolveTemplate_MultipleRefs(t *testing.T) {
	artifacts := map[string][]Artifact{
		"input":  {{Parts: []Part{TextPart("topic A")}, Index: 0}},
		"output": {{Parts: []Part{TextPart("result B")}, Index: 0}},
	}
	result := ResolveTemplate("{{input}} then {{output}}", artifacts)
	if result != "topic A then result B" {
		t.Errorf("got %q", result)
	}
}

func TestResolveTemplate_NoTemplates(t *testing.T) {
	result := ResolveTemplate("no templates here", nil)
	if result != "no templates here" {
		t.Errorf("got %q", result)
	}
}

func TestResolveTemplate_DataFallback(t *testing.T) {
	// When no text part exists, should fall back to data
	artifacts := map[string][]Artifact{
		"node": {{Parts: []Part{DataPart(map[string]any{"x": 1}, "application/json")}, Index: 0}},
	}
	result := ResolveTemplate("Result: {{node}}", artifacts)
	if result != `Result: {"x":1}` {
		t.Errorf("got %q", result)
	}
}
