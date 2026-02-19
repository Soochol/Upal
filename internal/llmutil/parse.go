package llmutil

import (
	"fmt"
	"strings"
)

// StripMarkdownJSON extracts JSON from an LLM response that may contain
// markdown code fences or leading text. It trims whitespace, strips ```json
// and ``` fences, and finds the first '{' to start parsing from.
// Returns an error if no '{' is found in the text.
func StripMarkdownJSON(text string) (string, error) {
	content := strings.TrimSpace(text)
	// Strip markdown code fences if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// Skip to the first '{"' in case there is leading text.
	// Using '{"' instead of '{' avoids false matches on {{template}} syntax
	// that LLMs may include in explanatory text.
	if idx := strings.Index(content, `{"`); idx > 0 {
		content = content[idx:]
	}

	if !strings.Contains(content, "{") {
		return "", fmt.Errorf("no JSON object found in text")
	}

	return content, nil
}
