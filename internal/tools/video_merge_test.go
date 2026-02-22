package tools

import (
	"context"
	"testing"
)

func TestVideoMergeTool_Name(t *testing.T) {
	tool := &VideoMergeTool{}
	if tool.Name() != "video_merge" {
		t.Errorf("expected 'video_merge', got %q", tool.Name())
	}
}

func TestVideoMergeTool_InputValidation(t *testing.T) {
	tool := &VideoMergeTool{}

	t.Run("missing inputs", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"mode": "mux_audio",
		})
		if err == nil {
			t.Error("expected error for missing inputs")
		}
	})

	t.Run("invalid input type", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"inputs": "not-an-array",
		})
		if err == nil {
			t.Error("expected error for invalid inputs type")
		}
	})

	t.Run("empty inputs array", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"inputs": []any{},
		})
		if err == nil {
			t.Error("expected error for empty inputs")
		}
	})
}

func TestVideoMergeTool_Schema(t *testing.T) {
	tool := &VideoMergeTool{}
	schema := tool.InputSchema()
	if schema["type"] != "object" {
		t.Errorf("expected object schema")
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("missing properties")
	}
	if _, ok := props["inputs"]; !ok {
		t.Error("missing 'inputs' property")
	}
	if _, ok := props["mode"]; !ok {
		t.Error("missing 'mode' property")
	}
}
