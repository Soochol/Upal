// internal/tools/rss_feed.go
package tools

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

type RSSFeedTool struct{}

func (r *RSSFeedTool) Name() string { return "fetch_rss" }
func (r *RSSFeedTool) Description() string {
	return "Fetch and parse an RSS, Atom, or JSON feed. Returns structured items with title, link, published date, summary, and author. No LLM cost — pure code parsing."
}

func (r *RSSFeedTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "URL of the RSS/Atom/JSON feed to fetch",
			},
			"max_items": map[string]any{
				"type":        "number",
				"description": "Maximum number of items to return (default: all)",
			},
			"since_date": map[string]any{
				"type":        "string",
				"description": "Only return items published after this ISO 8601 date (e.g. 2026-02-20T00:00:00Z)",
			},
		},
		"required": []any{"url"},
	}
}

func (r *RSSFeedTool) Execute(ctx context.Context, input any) (any, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input: expected object")
	}

	url, _ := args["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	maxItems := 0
	if v, ok := args["max_items"].(float64); ok && v > 0 {
		maxItems = int(v)
	}

	var sinceDate time.Time
	if v, ok := args["since_date"].(string); ok && v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, fmt.Errorf("invalid since_date format (use RFC3339): %w", err)
		}
		sinceDate = parsed
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fp := gofeed.NewParser()
	fp.Client = &http.Client{Timeout: 30 * time.Second}

	feed, err := fp.ParseURLWithContext(url, reqCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch/parse feed: %w", err)
	}

	var items []map[string]any
	for _, item := range feed.Items {
		// When since_date is set, exclude items with no parseable date and items older than the cutoff.
		if !sinceDate.IsZero() && (item.PublishedParsed == nil || item.PublishedParsed.Before(sinceDate)) {
			continue
		}

		published := ""
		if item.PublishedParsed != nil {
			published = item.PublishedParsed.Format(time.RFC3339)
		} else if item.Published != "" {
			published = item.Published
		}

		author := ""
		if item.Author != nil {
			author = item.Author.Name
		}

		items = append(items, map[string]any{
			"title":     item.Title,
			"link":      item.Link,
			"published": published,
			"summary":   item.Description,
			"author":    author,
		})

		if maxItems > 0 && len(items) >= maxItems {
			break
		}
	}

	return map[string]any{
		"items":      items,
		"feed_title": feed.Title,
		"feed_url":   feed.Link,
		"item_count": len(items),
	}, nil
}
