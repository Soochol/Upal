package model

import (
	"sync"

	"google.golang.org/genai"
)

// NativeToolProvider is an optional interface that LLM implementations can
// implement to declare provider-specific native tools.
//
// When llm_builder processes tool names from node config, it checks if the
// resolved LLM implements this interface and delegates native tool spec
// construction to the model. This avoids hardcoded provider mappings in the
// builder layer.
//
// A model that does not implement this interface, or returns (nil, false) for a
// given name, will cause that tool name to fall through to the custom tool
// registry instead.
type NativeToolProvider interface {
	// NativeTool returns the genai.Tool representation for the given tool name
	// if it is a provider-managed native tool, or (nil, false) if unsupported.
	NativeTool(name string) (*genai.Tool, bool)
}

// nativeToolSpecs maps well-known Upal tool names to their genai.Tool spec.
// All LLM implementations share this registry — adding a new native tool
// requires editing only this file, not each model implementation.
var (
	nativeToolSpecsMu sync.RWMutex
	nativeToolSpecs   = map[string]*genai.Tool{}
)

// RegisterNativeToolSpec registers a genai.Tool spec for a native tool name.
func RegisterNativeToolSpec(name string, spec *genai.Tool) {
	nativeToolSpecsMu.Lock()
	defer nativeToolSpecsMu.Unlock()
	nativeToolSpecs[name] = spec
}

// LookupNativeTool returns the genai.Tool spec for the given tool name.
func LookupNativeTool(name string) (*genai.Tool, bool) {
	nativeToolSpecsMu.RLock()
	defer nativeToolSpecsMu.RUnlock()
	spec, ok := nativeToolSpecs[name]
	return spec, ok
}

func init() {
	RegisterNativeToolSpec("web_search", &genai.Tool{GoogleSearch: &genai.GoogleSearch{}})
}
