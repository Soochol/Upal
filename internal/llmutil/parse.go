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

	// Find the first '{' that isn't part of '{{' (template syntax).
	// We can't use '{"' as the needle because pretty-printed JSON has
	// '{' on its own line followed by a newline, not a quote.
	start := -1
	for i := 0; i < len(content); i++ {
		if content[i] == '{' {
			if i+1 < len(content) && content[i+1] == '{' {
				i++ // skip '{{' pair
				continue
			}
			start = i
			break
		}
	}

	if start < 0 {
		return "", fmt.Errorf("no JSON object found in text")
	}

	return content[start:], nil
}
