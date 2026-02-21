// internal/tools/rss_feed_test.go
package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Article One</title>
      <link>https://example.com/1</link>
      <description>First article summary</description>
      <pubDate>Mon, 20 Feb 2026 09:00:00 GMT</pubDate>
      <author>Alice</author>
    </item>
    <item>
      <title>Article Two</title>
      <link>https://example.com/2</link>
      <description>Second article summary</description>
      <pubDate>Sun, 19 Feb 2026 09:00:00 GMT</pubDate>
      <author>Bob</author>
    </item>
  </channel>
</rss>`

func TestRSSFeedTool_Execute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(testRSSFeed))
	}))
	defer srv.Close()

	tool := &RSSFeedTool{}

	if tool.Name() != "fetch_rss" {
		t.Fatalf("expected name fetch_rss, got %s", tool.Name())
	}

	result, err := tool.Execute(context.Background(), map[string]any{
		"url": srv.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	items, ok := res["items"].([]map[string]any)
	if !ok {
		t.Fatalf("expected items array, got %T", res["items"])
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0]["title"] != "Article One" {
		t.Errorf("expected title 'Article One', got %v", items[0]["title"])
	}
	if res["feed_title"] != "Test Feed" {
		t.Errorf("expected feed_title 'Test Feed', got %v", res["feed_title"])
	}
}

func TestRSSFeedTool_MaxItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(testRSSFeed))
	}))
	defer srv.Close()

	tool := &RSSFeedTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"url":       srv.URL,
		"max_items": float64(1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res := result.(map[string]any)
	items := res["items"].([]map[string]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item with max_items=1, got %d", len(items))
	}
}

func TestRSSFeedTool_InvalidURL(t *testing.T) {
	tool := &RSSFeedTool{}
	_, err := tool.Execute(context.Background(), map[string]any{
		"url": "http://localhost:1/nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}
