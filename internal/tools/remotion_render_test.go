package tools

import (
	"context"
	"testing"
)

func TestRemotion_Name(t *testing.T) {
	tool := &RemotionRenderTool{}
	if tool.Name() != "remotion_render" {
		t.Errorf("expected 'remotion_render', got %q", tool.Name())
	}
}

func TestRemotion_InputValidation(t *testing.T) {
	tool := &RemotionRenderTool{}

	t.Run("missing composition_code", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"audio_path": "/tmp/test.mp3",
		})
		if err == nil {
			t.Error("expected error for missing composition_code")
		}
	})

	t.Run("empty composition_code", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{
			"composition_code": "",
		})
		if err == nil {
			t.Error("expected error for empty composition_code")
		}
	})
}

func TestRemotion_Schema(t *testing.T) {
	tool := &RemotionRenderTool{}
	schema := tool.InputSchema()
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("missing properties")
	}
	for _, field := range []string{"composition_code", "audio_path", "duration_sec", "fps"} {
		if _, ok := props[field]; !ok {
			t.Errorf("missing property %q", field)
		}
	}
}
