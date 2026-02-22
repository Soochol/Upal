// internal/agents/output_extract.go
package agents

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// outputExtractConfig holds the parsed output_extract config for an agent node.
type outputExtractConfig struct {
	mode string // "json" | "tagged"
	key  string // json mode: top-level JSON key
	tag  string // tagged mode: XML tag name
}

// parseOutputExtract reads output_extract from the node Config map.
// Returns nil if absent, mode is unrecognised, or required field is empty.
func parseOutputExtract(cfg map[string]any) *outputExtractConfig {
	raw, ok := cfg["output_extract"].(map[string]any)
	if !ok {
		return nil
	}
	mode, _ := raw["mode"].(string)
	switch mode {
	case "json":
		key, _ := raw["key"].(string)
		if key == "" {
			return nil
		}
		return &outputExtractConfig{mode: "json", key: key}
	case "tagged":
		tag, _ := raw["tag"].(string)
		if tag == "" {
			return nil
		}
		return &outputExtractConfig{mode: "tagged", tag: tag}
	default:
		return nil
	}
}

// systemPromptAppend returns the instruction text to append to the system prompt.
func (oe *outputExtractConfig) systemPromptAppend() string {
	switch oe.mode {
	case "json":
		return fmt.Sprintf(
			"\n\nRespond ONLY with valid JSON in this exact format:\n{\"%s\": <your output here>}\nDo not include any other text outside the JSON object.",
			oe.key,
		)
	case "tagged":
		return fmt.Sprintf(
			"\n\nWrap your final output in <%s> tags:\n<%s>your output here</%s>\nDo not include any other text outside the tags.",
			oe.tag, oe.tag, oe.tag,
		)
	}
	return ""
}

// applyOutputExtract extracts the artifact from raw LLM output using the configured strategy.
// Returns raw unchanged if cfg is nil or extraction fails (safe fallback).
func applyOutputExtract(cfg *outputExtractConfig, raw string) string {
	if cfg == nil {
		return raw
	}
	switch cfg.mode {
	case "json":
		if v, err := extractJSONKey(raw, cfg.key); err == nil {
			return v
		}
	case "tagged":
		if v := extractTagged(raw, cfg.tag); v != "" {
			return v
		}
	}
	return raw
}

// extractJSONKey finds the first JSON object in s and returns the string value for key.
// Uses json.Decoder so string values containing braces are handled correctly.
func extractJSONKey(s, key string) (string, error) {
	start := strings.Index(s, "{")
	if start < 0 {
		return "", fmt.Errorf("no JSON object found")
	}
	dec := json.NewDecoder(strings.NewReader(s[start:]))
	var obj map[string]json.RawMessage
	if err := dec.Decode(&obj); err != nil {
		return "", fmt.Errorf("JSON parse failed: %w", err)
	}
	rawVal, ok := obj[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in JSON", key)
	}
	var str string
	if err := json.Unmarshal(rawVal, &str); err == nil {
		return str, nil
	}
	return strings.TrimSpace(string(rawVal)), nil
}

// extractTagged extracts the content inside <tag>...</tag> (including newlines).
// Returns empty string if tags are not found.
func extractTagged(s, tag string) string {
	pattern := fmt.Sprintf(`(?s)<%s>(.*?)</%s>`, regexp.QuoteMeta(tag), regexp.QuoteMeta(tag))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}
