package model_test

import (
	"testing"

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
	_, ok := model.LookupNativeTool("unknown_tool_xyz")
	if ok {
		t.Fatal("unknown_tool_xyz should not be found")
	}
}
