package llmutil

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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

// audioMIMEToExt maps audio MIME types to file extensions.
var audioMIMEToExt = map[string]string{
	"audio/mpeg": ".mp3",
	"audio/mp3":  ".mp3",
	"audio/wav":  ".wav",
	"audio/ogg":  ".ogg",
	"audio/webm": ".webm",
}

// ExtractContentSavingAudio is like ExtractContent but saves audio InlineData
// to outputDir instead of encoding as a data URI. Returns the file path for
// audio parts. Image InlineData is still returned as data URIs.
// If outputDir is empty, audio falls back to data URI behaviour.
func ExtractContentSavingAudio(resp *adkmodel.LLMResponse, outputDir string) string {
	if resp == nil || resp.Content == nil {
		return ""
	}
	var parts []string
	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			parts = append(parts, p.Text)
			continue
		}
		if p.InlineData == nil || len(p.InlineData.Data) == 0 {
			continue
		}
		mime := p.InlineData.MIMEType
		ext, isAudio := audioMIMEToExt[mime]
		if isAudio && outputDir != "" {
			path, err := saveToFile(p.InlineData.Data, outputDir, ext)
			if err != nil {
				parts = append(parts, fmt.Sprintf("data:%s;base64,%s", mime,
					base64.StdEncoding.EncodeToString(p.InlineData.Data)))
			} else {
				parts = append(parts, path)
			}
			continue
		}
		parts = append(parts, fmt.Sprintf("data:%s;base64,%s", mime,
			base64.StdEncoding.EncodeToString(p.InlineData.Data)))
	}
	return strings.Join(parts, "\n")
}

// saveToFile writes data to a uniquely named file in dir and returns the path.
func saveToFile(data []byte, dir, ext string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	name := filepath.Join(dir, uuid.New().String()+ext)
	if err := os.WriteFile(name, data, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return name, nil
}
