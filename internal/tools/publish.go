// internal/tools/publish.go
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type PublishTool struct {
	outputDir string
}

func NewPublishTool(outputDir string) *PublishTool {
	return &PublishTool{outputDir: outputDir}
}

func (p *PublishTool) Name() string { return "publish" }
func (p *PublishTool) Description() string {
	return "Publish content to various channels. Supports 'markdown_file' (save to local file) and 'webhook' (POST to external URL). Extensible for future platform integrations."
}

func (p *PublishTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"channel": map[string]any{
				"type":        "string",
				"enum":        []any{"markdown_file", "webhook"},
				"description": "Publishing channel",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to publish",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Title of the content (optional)",
			},
			"metadata": map[string]any{
				"type":        "object",
				"description": "Channel-specific metadata. For webhook: { webhook_url: string }",
			},
		},
		"required": []any{"channel", "content"},
	}
}

func (p *PublishTool) Execute(ctx context.Context, input any) (any, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input: expected object")
	}

	channel, _ := args["channel"].(string)
	content, _ := args["content"].(string)
	title, _ := args["title"].(string)
	metadata, _ := args["metadata"].(map[string]any)

	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	switch channel {
	case "markdown_file":
		return p.publishMarkdown(title, content, metadata)
	case "webhook":
		return p.publishWebhook(ctx, title, content, metadata)
	default:
		return nil, fmt.Errorf("unknown channel %q: supported channels are markdown_file, webhook", channel)
	}
}

func (p *PublishTool) publishMarkdown(title, content string, metadata map[string]any) (any, error) {
	if err := os.MkdirAll(p.outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	slug := slugify(title)
	if slug == "" {
		slug = "untitled"
	}
	filename := fmt.Sprintf("%s-%s.md", time.Now().Format("2006-01-02"), slug)
	path := filepath.Join(p.outputDir, filename)

	var buf strings.Builder
	if title != "" {
		buf.WriteString("---\n")
		buf.WriteString(fmt.Sprintf("title: %q\n", title))
		buf.WriteString(fmt.Sprintf("date: %s\n", time.Now().Format(time.RFC3339)))
		if metadata != nil {
			for k, v := range metadata {
				buf.WriteString(fmt.Sprintf("%s: %v\n", k, v))
			}
		}
		buf.WriteString("---\n\n")
	}
	buf.WriteString(content)

	if err := os.WriteFile(path, []byte(buf.String()), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return map[string]any{
		"status": "published",
		"path":   path,
	}, nil
}

func (p *PublishTool) publishWebhook(ctx context.Context, title, content string, metadata map[string]any) (any, error) {
	webhookURL := ""
	if metadata != nil {
		webhookURL, _ = metadata["webhook_url"].(string)
	}
	if webhookURL == "" {
		return nil, fmt.Errorf("metadata.webhook_url is required for webhook channel")
	}

	payload, _ := json.Marshal(map[string]any{
		"title":   title,
		"content": content,
	})

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return map[string]any{
		"status":      "published",
		"status_code": resp.StatusCode,
	}, nil
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlphaNum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
