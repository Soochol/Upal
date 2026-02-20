package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// maxTextOutput caps how much extracted text we return to the LLM.
const maxTextOutput = 100 * 1024 // 100 KB

// GetWebpageTool fetches a URL and extracts readable text content.
type GetWebpageTool struct{}

func (g *GetWebpageTool) Name() string { return "get_webpage" }

func (g *GetWebpageTool) Description() string {
	return "Fetch a webpage URL and extract its readable text content. Returns the page title and clean text with HTML tags removed."
}

func (g *GetWebpageTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch and extract text from",
			},
		},
		"required": []any{"url"},
	}
}

func (g *GetWebpageTool) Execute(ctx context.Context, input any) (any, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input: expected object")
	}

	url, _ := args["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Upal/1.0 (webpage reader)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Cap raw HTML input to 1MB.
	limited := io.LimitReader(resp.Body, 1024*1024)

	title, text, err := extractText(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	if len(text) > maxTextOutput {
		text = text[:maxTextOutput] + "\n... [truncated at 100KB]"
	}

	return map[string]any{
		"title": title,
		"text":  text,
		"url":   url,
	}, nil
}

// skipTags are HTML elements whose text content should be excluded.
var skipTags = map[string]bool{
	"script":   true,
	"style":    true,
	"noscript": true,
	"svg":      true,
}

// extractText walks an HTML token stream and returns (title, bodyText).
func extractText(r io.Reader) (string, string, error) {
	tokenizer := html.NewTokenizer(r)
	var (
		title       strings.Builder
		text        strings.Builder
		inTitle     bool
		skipDepth   int
		lastWasText bool
	)

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return strings.TrimSpace(title.String()), strings.TrimSpace(text.String()), nil
			}
			// Treat parse errors as end-of-input; return what we have.
			return strings.TrimSpace(title.String()), strings.TrimSpace(text.String()), nil

		case html.StartTagToken:
			tn, _ := tokenizer.TagName()
			tag := string(tn)
			if tag == "title" {
				inTitle = true
			}
			if skipTags[tag] {
				skipDepth++
			}
			if isBlockTag(tag) && lastWasText {
				text.WriteString("\n")
				lastWasText = false
			}

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			tag := string(tn)
			if tag == "title" {
				inTitle = false
			}
			if skipTags[tag] && skipDepth > 0 {
				skipDepth--
			}

		case html.TextToken:
			content := strings.TrimSpace(string(tokenizer.Text()))
			if content == "" {
				continue
			}
			if inTitle {
				title.WriteString(content)
			}
			if skipDepth == 0 {
				if lastWasText {
					text.WriteString(" ")
				}
				text.WriteString(content)
				lastWasText = true
			}
		}
	}
}

func isBlockTag(tag string) bool {
	switch tag {
	case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6",
		"li", "br", "hr", "blockquote", "pre", "article",
		"section", "header", "footer", "nav", "main", "tr":
		return true
	}
	return false
}
