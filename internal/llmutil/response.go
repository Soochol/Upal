package llmutil

import (
	adkmodel "google.golang.org/adk/model"
)

// ExtractText concatenates all text parts from an LLMResponse into a single string.
// Returns an empty string if the response or its content is nil.
func ExtractText(resp *adkmodel.LLMResponse) string {
	if resp == nil || resp.Content == nil {
		return ""
	}
	var text string
	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			text += p.Text
		}
	}
	return text
}
