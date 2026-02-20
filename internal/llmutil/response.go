package llmutil

import (
	"encoding/base64"
	"fmt"
	"strings"

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

// ExtractContent extracts all content from an LLMResponse, including images.
// Text parts are concatenated as-is. InlineData parts (images) are converted
// to data URI strings (e.g., "data:image/png;base64,...").
// Multiple parts are joined with newlines.
func ExtractContent(resp *adkmodel.LLMResponse) string {
	if resp == nil || resp.Content == nil {
		return ""
	}
	var parts []string
	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			parts = append(parts, p.Text)
		}
		if p.InlineData != nil && len(p.InlineData.Data) > 0 {
			dataURI := fmt.Sprintf("data:%s;base64,%s",
				p.InlineData.MIMEType,
				base64.StdEncoding.EncodeToString(p.InlineData.Data))
			parts = append(parts, dataURI)
		}
	}
	return strings.Join(parts, "\n")
}
