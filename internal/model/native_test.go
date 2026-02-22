package model_test

import (
	"testing"

	"google.golang.org/genai"

	"github.com/soochol/upal/internal/model"
)

func TestLookupNativeTool_Known(t *testing.T) {
	spec, ok := model.LookupNativeTool("web_search")
	if !ok {
		t.Fatal("web_search should be registered")
	}
	if spec == nil || spec.GoogleSearch == nil {
		t.Fatal("expected GoogleSearch spec")
	}
}

func TestLookupNativeTool_Unknown(t *testing.T) {
	_, ok := model.LookupNativeTool("unknown_tool")
	if ok {
		t.Fatal("unknown_tool should not be found")
	}
}

func TestRegisterNativeToolSpec_Custom(t *testing.T) {
	model.RegisterNativeToolSpec("test_tool_xyz", &genai.Tool{})
	spec, ok := model.LookupNativeTool("test_tool_xyz")
	if !ok || spec == nil {
		t.Fatal("custom tool should be findable after registration")
	}
}
