# Pipeline Automation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Pipeline orchestration layer with 3 new tools (fetch_rss, content_store, publish) and full CRUD + execution UI for chaining Workflows, Schedules, and Approvals.

**Architecture:** Pipeline is a new top-level entity (like Workflow) with ordered Stages. Each Stage has a type (workflow/approval/schedule/trigger/transform) and a pluggable StageExecutor. The PipelineRunner iterates through stages sequentially, persisting state between runs for approval waits and schedule sleeps. Three new tools handle data collection (RSS), persistence (key-value store), and delivery (multi-channel publish) — all without LLM cost.

**Tech Stack:** Go 1.23 (Chi router, gofeed), React 19 (TypeScript, Zustand, Tailwind, shadcn/ui, lucide-react)

**Design Doc:** `docs/plans/2026-02-21-pipeline-automation-design.md`

---

## Phase 1: New Tools

### Task 1: fetch_rss Tool

**Files:**
- Create: `internal/tools/rss_feed.go`
- Create: `internal/tools/rss_feed_test.go`

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tools/... -v -race -run TestRSSFeed`
Expected: FAIL — `RSSFeedTool` undefined

**Step 3: Write implementation**

```go
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

func (r *RSSFeedTool) Name() string        { return "fetch_rss" }
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
		if !sinceDate.IsZero() && item.PublishedParsed != nil && item.PublishedParsed.Before(sinceDate) {
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tools/... -v -race -run TestRSSFeed`
Expected: PASS

**Step 5: Add gofeed dependency**

Run: `go get github.com/mmcdole/gofeed`

**Step 6: Commit**

```bash
git add internal/tools/rss_feed.go internal/tools/rss_feed_test.go go.mod go.sum
git commit -m "feat: add fetch_rss tool for RSS/Atom/JSON feed parsing"
```

---

### Task 2: content_store Tool

**Files:**
- Create: `internal/tools/content_store.go`
- Create: `internal/tools/content_store_test.go`

**Step 1: Write the failing test**

```go
// internal/tools/content_store_test.go
package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestContentStoreTool_SetAndGet(t *testing.T) {
	dir := t.TempDir()
	tool := NewContentStoreTool(filepath.Join(dir, "store.json"))

	// Set a value
	_, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
		"key":    "seen:https://example.com/1",
		"value":  "2026-02-21T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// Get the value
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "get",
		"key":    "seen:https://example.com/1",
	})
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	res := result.(map[string]any)
	if res["value"] != "2026-02-21T09:00:00Z" {
		t.Errorf("expected stored value, got %v", res["value"])
	}
}

func TestContentStoreTool_ListByPrefix(t *testing.T) {
	dir := t.TempDir()
	tool := NewContentStoreTool(filepath.Join(dir, "store.json"))

	ctx := context.Background()
	tool.Execute(ctx, map[string]any{"action": "set", "key": "seen:url1", "value": "1"})
	tool.Execute(ctx, map[string]any{"action": "set", "key": "seen:url2", "value": "2"})
	tool.Execute(ctx, map[string]any{"action": "set", "key": "other:key", "value": "3"})

	result, err := tool.Execute(ctx, map[string]any{
		"action": "list",
		"prefix": "seen:",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	res := result.(map[string]any)
	keys := res["keys"].([]string)
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys with prefix 'seen:', got %d", len(keys))
	}
}

func TestContentStoreTool_Delete(t *testing.T) {
	dir := t.TempDir()
	tool := NewContentStoreTool(filepath.Join(dir, "store.json"))

	ctx := context.Background()
	tool.Execute(ctx, map[string]any{"action": "set", "key": "k1", "value": "v1"})
	tool.Execute(ctx, map[string]any{"action": "delete", "key": "k1"})

	result, err := tool.Execute(ctx, map[string]any{"action": "get", "key": "k1"})
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	res := result.(map[string]any)
	if res["value"] != nil {
		t.Errorf("expected nil after delete, got %v", res["value"])
	}
}

func TestContentStoreTool_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	// Write with first instance
	tool1 := NewContentStoreTool(path)
	tool1.Execute(context.Background(), map[string]any{
		"action": "set", "key": "persist", "value": "yes",
	})

	// Read with new instance (same file)
	tool2 := NewContentStoreTool(path)
	result, _ := tool2.Execute(context.Background(), map[string]any{
		"action": "get", "key": "persist",
	})
	res := result.(map[string]any)
	if res["value"] != "yes" {
		t.Errorf("expected persisted value 'yes', got %v", res["value"])
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("store file should exist on disk")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tools/... -v -race -run TestContentStore`
Expected: FAIL — `NewContentStoreTool` undefined

**Step 3: Write implementation**

```go
// internal/tools/content_store.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type ContentStoreTool struct {
	mu   sync.RWMutex
	path string
	data map[string]string
}

func NewContentStoreTool(path string) *ContentStoreTool {
	t := &ContentStoreTool{
		path: path,
		data: make(map[string]string),
	}
	t.load()
	return t
}

func (c *ContentStoreTool) Name() string { return "content_store" }
func (c *ContentStoreTool) Description() string {
	return "Persistent key-value store for tracking state across pipeline runs. Use for deduplication (seen URLs), timestamps (last collection), counters, and any data that must survive between executions."
}

func (c *ContentStoreTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []any{"get", "set", "list", "delete"},
				"description": "Operation to perform",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "Key to get, set, or delete",
			},
			"value": map[string]any{
				"type":        "string",
				"description": "Value to store (required for 'set' action)",
			},
			"prefix": map[string]any{
				"type":        "string",
				"description": "Key prefix filter for 'list' action",
			},
		},
		"required": []any{"action"},
	}
}

func (c *ContentStoreTool) Execute(_ context.Context, input any) (any, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input: expected object")
	}

	action, _ := args["action"].(string)
	key, _ := args["key"].(string)

	switch action {
	case "get":
		if key == "" {
			return nil, fmt.Errorf("key is required for get")
		}
		c.mu.RLock()
		val, exists := c.data[key]
		c.mu.RUnlock()
		if !exists {
			return map[string]any{"value": nil, "found": false}, nil
		}
		return map[string]any{"value": val, "found": true}, nil

	case "set":
		if key == "" {
			return nil, fmt.Errorf("key is required for set")
		}
		value, _ := args["value"].(string)
		c.mu.Lock()
		c.data[key] = value
		err := c.save()
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("failed to persist: %w", err)
		}
		return map[string]any{"status": "ok"}, nil

	case "list":
		prefix, _ := args["prefix"].(string)
		c.mu.RLock()
		var keys []string
		for k := range c.data {
			if prefix == "" || strings.HasPrefix(k, prefix) {
				keys = append(keys, k)
			}
		}
		c.mu.RUnlock()
		sort.Strings(keys)
		return map[string]any{"keys": keys, "count": len(keys)}, nil

	case "delete":
		if key == "" {
			return nil, fmt.Errorf("key is required for delete")
		}
		c.mu.Lock()
		delete(c.data, key)
		err := c.save()
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("failed to persist: %w", err)
		}
		return map[string]any{"status": "ok"}, nil

	default:
		return nil, fmt.Errorf("unknown action %q: use get, set, list, or delete", action)
	}
}

func (c *ContentStoreTool) load() {
	raw, err := os.ReadFile(c.path)
	if err != nil {
		return // file doesn't exist yet — start empty
	}
	json.Unmarshal(raw, &c.data)
}

func (c *ContentStoreTool) save() error {
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	raw, err := json.Marshal(c.data)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, raw, 0644)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tools/... -v -race -run TestContentStore`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/content_store.go internal/tools/content_store_test.go
git commit -m "feat: add content_store tool for persistent key-value storage"
```

---

### Task 3: publish Tool

**Files:**
- Create: `internal/tools/publish.go`
- Create: `internal/tools/publish_test.go`

**Step 1: Write the failing test**

```go
// internal/tools/publish_test.go
package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishTool_MarkdownFile(t *testing.T) {
	dir := t.TempDir()
	tool := NewPublishTool(dir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"channel":  "markdown_file",
		"title":    "Test Article",
		"content":  "# Hello World\n\nThis is a test.",
		"metadata": map[string]any{"tags": "test,demo"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res := result.(map[string]any)
	if res["status"] != "published" {
		t.Errorf("expected status 'published', got %v", res["status"])
	}

	path := res["path"].(string)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !contains(string(content), "# Hello World") {
		t.Error("output file should contain the content")
	}
}

func TestPublishTool_Webhook(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tool := NewPublishTool(t.TempDir())
	result, err := tool.Execute(context.Background(), map[string]any{
		"channel": "webhook",
		"content": "Hello from pipeline",
		"title":   "Test",
		"metadata": map[string]any{
			"webhook_url": srv.URL,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res := result.(map[string]any)
	if res["status"] != "published" {
		t.Errorf("expected status 'published', got %v", res["status"])
	}
	if received["content"] != "Hello from pipeline" {
		t.Errorf("webhook should receive content, got %v", received["content"])
	}
}

func TestPublishTool_UnknownChannel(t *testing.T) {
	tool := NewPublishTool(t.TempDir())
	_, err := tool.Execute(context.Background(), map[string]any{
		"channel": "fax_machine",
		"content": "test",
	})
	if err == nil {
		t.Fatal("expected error for unknown channel")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}
func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tools/... -v -race -run TestPublishTool`
Expected: FAIL — `NewPublishTool` undefined

**Step 3: Write implementation**

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tools/... -v -race -run TestPublishTool`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/publish.go internal/tools/publish_test.go
git commit -m "feat: add publish tool for markdown file and webhook channels"
```

---

### Task 4: Register Tools in main.go

**Files:**
- Modify: `cmd/upal/main.go` (around line 108-112 where tools are registered)

**Step 1: Add tool registrations**

After the existing tool registrations, add:

```go
toolReg.Register(&tools.RSSFeedTool{})
toolReg.Register(tools.NewContentStoreTool(filepath.Join(dataDir, "content_store.json")))
toolReg.Register(tools.NewPublishTool(filepath.Join(dataDir, "published")))
```

Where `dataDir` is resolved near the top of `serve()`:

```go
dataDir := "data"
if err := os.MkdirAll(dataDir, 0755); err != nil {
    log.Fatalf("failed to create data directory: %v", err)
}
```

Add imports: `"os"`, `"path/filepath"`

**Step 2: Verify it compiles**

Run: `go build ./cmd/upal/...`
Expected: BUILD SUCCESS

**Step 3: Run existing tests to check for regressions**

Run: `go test ./internal/tools/... -v -race`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat: register fetch_rss, content_store, publish tools in server"
```

---

## Phase 2: Pipeline Backend

### Task 5: Pipeline Types

**Files:**
- Modify: `internal/upal/types.go`

**Step 1: Add Pipeline type definitions**

Append to `internal/upal/types.go` after existing types:

```go
// Pipeline orchestrates a sequence of Stages (workflows, approvals, schedules).
type Pipeline struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Stages      []Stage   `json:"stages"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Stage is a single step in a Pipeline.
type Stage struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Type      string      `json:"type"` // "workflow", "approval", "schedule", "trigger", "transform"
	Config    StageConfig `json:"config"`
	DependsOn []string    `json:"depends_on,omitempty"`
}

// StageConfig holds type-specific configuration for a Stage.
type StageConfig struct {
	// Workflow stage
	WorkflowName string            `json:"workflow_name,omitempty"`
	InputMapping map[string]string  `json:"input_mapping,omitempty"`

	// Approval stage
	Message      string `json:"message,omitempty"`
	ConnectionID string `json:"connection_id,omitempty"`
	Timeout      int    `json:"timeout,omitempty"`

	// Schedule stage
	Cron     string `json:"cron,omitempty"`
	Timezone string `json:"timezone,omitempty"`

	// Trigger stage
	TriggerID string `json:"trigger_id,omitempty"`

	// Transform stage
	Expression string `json:"expression,omitempty"`
}

// PipelineRun tracks a single execution of a Pipeline.
type PipelineRun struct {
	ID           string                  `json:"id"`
	PipelineID   string                  `json:"pipeline_id"`
	Status       string                  `json:"status"` // pending, running, waiting, completed, failed
	CurrentStage string                  `json:"current_stage,omitempty"`
	StageResults map[string]*StageResult `json:"stage_results,omitempty"`
	StartedAt    time.Time               `json:"started_at"`
	CompletedAt  *time.Time              `json:"completed_at,omitempty"`
}

// StageResult is the output of a completed Stage.
type StageResult struct {
	StageID     string         `json:"stage_id"`
	Status      string         `json:"status"` // pending, running, waiting, completed, skipped, failed
	Output      map[string]any `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}
```

Ensure `"time"` is in the import block (it should be already).

**Step 2: Verify it compiles**

Run: `go build ./internal/upal/...`
Expected: BUILD SUCCESS

**Step 3: Commit**

```bash
git add internal/upal/types.go
git commit -m "feat: add Pipeline, Stage, PipelineRun, StageResult types"
```

---

### Task 6: Pipeline Repository

**Files:**
- Create: `internal/repository/pipeline.go`
- Create: `internal/repository/pipeline_memory.go`
- Create: `internal/repository/pipeline_memory_test.go`

**Step 1: Write the repository interface**

```go
// internal/repository/pipeline.go
package repository

import (
	"context"

	"github.com/anthropics/upal/internal/upal"
)

type PipelineRepository interface {
	Create(ctx context.Context, p *upal.Pipeline) error
	Get(ctx context.Context, id string) (*upal.Pipeline, error)
	List(ctx context.Context) ([]*upal.Pipeline, error)
	Update(ctx context.Context, p *upal.Pipeline) error
	Delete(ctx context.Context, id string) error
}

type PipelineRunRepository interface {
	Create(ctx context.Context, run *upal.PipelineRun) error
	Get(ctx context.Context, id string) (*upal.PipelineRun, error)
	ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error)
	Update(ctx context.Context, run *upal.PipelineRun) error
}
```

**Step 2: Write failing tests for memory implementation**

```go
// internal/repository/pipeline_memory_test.go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/upal/internal/upal"
)

func TestMemoryPipelineRepo_CRUD(t *testing.T) {
	repo := NewMemoryPipelineRepository()
	ctx := context.Background()

	p := &upal.Pipeline{
		ID:   "pipe-test1",
		Name: "Test Pipeline",
		Stages: []upal.Stage{
			{ID: "s1", Name: "Collect", Type: "workflow"},
		},
		CreatedAt: time.Now(),
	}

	// Create
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, "pipe-test1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Name != "Test Pipeline" {
		t.Errorf("expected name 'Test Pipeline', got %q", got.Name)
	}

	// List
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(list))
	}

	// Update
	p.Name = "Updated"
	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	got, _ = repo.Get(ctx, "pipe-test1")
	if got.Name != "Updated" {
		t.Errorf("expected updated name, got %q", got.Name)
	}

	// Delete
	if err := repo.Delete(ctx, "pipe-test1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	_, err = repo.Get(ctx, "pipe-test1")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestMemoryPipelineRepo_DuplicateCreate(t *testing.T) {
	repo := NewMemoryPipelineRepository()
	ctx := context.Background()
	p := &upal.Pipeline{ID: "pipe-dup", Name: "Dup"}
	repo.Create(ctx, p)
	if err := repo.Create(ctx, p); err == nil {
		t.Error("expected error on duplicate create")
	}
}

func TestMemoryPipelineRunRepo_CRUD(t *testing.T) {
	repo := NewMemoryPipelineRunRepository()
	ctx := context.Background()

	run := &upal.PipelineRun{
		ID:         "prun-1",
		PipelineID: "pipe-1",
		Status:     "running",
		StartedAt:  time.Now(),
	}

	if err := repo.Create(ctx, run); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, err := repo.Get(ctx, "prun-1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Status != "running" {
		t.Errorf("expected status 'running', got %q", got.Status)
	}

	runs, err := repo.ListByPipeline(ctx, "pipe-1")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	run.Status = "completed"
	repo.Update(ctx, run)
	got, _ = repo.Get(ctx, "prun-1")
	if got.Status != "completed" {
		t.Errorf("expected updated status, got %q", got.Status)
	}
}
```

**Step 3: Run tests to verify they fail**

Run: `go test ./internal/repository/... -v -race -run TestMemoryPipeline`
Expected: FAIL — undefined symbols

**Step 4: Write memory implementations**

```go
// internal/repository/pipeline_memory.go
package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/anthropics/upal/internal/upal"
)

// MemoryPipelineRepository implements PipelineRepository in-memory.
type MemoryPipelineRepository struct {
	mu        sync.RWMutex
	pipelines map[string]*upal.Pipeline
}

func NewMemoryPipelineRepository() *MemoryPipelineRepository {
	return &MemoryPipelineRepository{pipelines: make(map[string]*upal.Pipeline)}
}

func (r *MemoryPipelineRepository) Create(_ context.Context, p *upal.Pipeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.pipelines[p.ID]; exists {
		return fmt.Errorf("pipeline %q already exists", p.ID)
	}
	r.pipelines[p.ID] = p
	return nil
}

func (r *MemoryPipelineRepository) Get(_ context.Context, id string) (*upal.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.pipelines[id]
	if !ok {
		return nil, fmt.Errorf("pipeline %q not found", id)
	}
	return p, nil
}

func (r *MemoryPipelineRepository) List(_ context.Context) ([]*upal.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*upal.Pipeline, 0, len(r.pipelines))
	for _, p := range r.pipelines {
		out = append(out, p)
	}
	return out, nil
}

func (r *MemoryPipelineRepository) Update(_ context.Context, p *upal.Pipeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.pipelines[p.ID]; !exists {
		return fmt.Errorf("pipeline %q not found", p.ID)
	}
	r.pipelines[p.ID] = p
	return nil
}

func (r *MemoryPipelineRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.pipelines[id]; !exists {
		return fmt.Errorf("pipeline %q not found", id)
	}
	delete(r.pipelines, id)
	return nil
}

// MemoryPipelineRunRepository implements PipelineRunRepository in-memory.
type MemoryPipelineRunRepository struct {
	mu   sync.RWMutex
	runs map[string]*upal.PipelineRun
}

func NewMemoryPipelineRunRepository() *MemoryPipelineRunRepository {
	return &MemoryPipelineRunRepository{runs: make(map[string]*upal.PipelineRun)}
}

func (r *MemoryPipelineRunRepository) Create(_ context.Context, run *upal.PipelineRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[run.ID] = run
	return nil
}

func (r *MemoryPipelineRunRepository) Get(_ context.Context, id string) (*upal.PipelineRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[id]
	if !ok {
		return nil, fmt.Errorf("pipeline run %q not found", id)
	}
	return run, nil
}

func (r *MemoryPipelineRunRepository) ListByPipeline(_ context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*upal.PipelineRun
	for _, run := range r.runs {
		if run.PipelineID == pipelineID {
			out = append(out, run)
		}
	}
	return out, nil
}

func (r *MemoryPipelineRunRepository) Update(_ context.Context, run *upal.PipelineRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.runs[run.ID]; !exists {
		return fmt.Errorf("pipeline run %q not found", run.ID)
	}
	r.runs[run.ID] = run
	return nil
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/repository/... -v -race -run TestMemoryPipeline`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/repository/pipeline.go internal/repository/pipeline_memory.go internal/repository/pipeline_memory_test.go
git commit -m "feat: add Pipeline and PipelineRun repository interfaces with in-memory implementations"
```

---

### Task 7: Pipeline Service

**Files:**
- Create: `internal/services/pipeline_service.go`
- Create: `internal/services/pipeline_service_test.go`

**Step 1: Write the failing test**

```go
// internal/services/pipeline_service_test.go
package services

import (
	"context"
	"testing"

	"github.com/anthropics/upal/internal/repository"
	"github.com/anthropics/upal/internal/upal"
)

func TestPipelineService_CreateAndGet(t *testing.T) {
	svc := NewPipelineService(
		repository.NewMemoryPipelineRepository(),
		repository.NewMemoryPipelineRunRepository(),
	)
	ctx := context.Background()

	p := &upal.Pipeline{
		Name: "Test Pipeline",
		Stages: []upal.Stage{
			{Name: "Collect", Type: "workflow", Config: upal.StageConfig{WorkflowName: "rss-collect"}},
			{Name: "Approve", Type: "approval", Config: upal.StageConfig{Message: "Pick a topic"}},
		},
	}

	if err := svc.Create(ctx, p); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if p.ID == "" {
		t.Error("expected ID to be generated")
	}
	if len(p.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(p.Stages))
	}
	if p.Stages[0].ID == "" {
		t.Error("expected stage IDs to be generated")
	}

	got, err := svc.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Name != "Test Pipeline" {
		t.Errorf("expected name 'Test Pipeline', got %q", got.Name)
	}
}

func TestPipelineService_List(t *testing.T) {
	svc := NewPipelineService(
		repository.NewMemoryPipelineRepository(),
		repository.NewMemoryPipelineRunRepository(),
	)
	ctx := context.Background()

	svc.Create(ctx, &upal.Pipeline{Name: "A"})
	svc.Create(ctx, &upal.Pipeline{Name: "B"})

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(list))
	}
}

func TestPipelineService_Delete(t *testing.T) {
	svc := NewPipelineService(
		repository.NewMemoryPipelineRepository(),
		repository.NewMemoryPipelineRunRepository(),
	)
	ctx := context.Background()

	p := &upal.Pipeline{Name: "ToDelete"}
	svc.Create(ctx, p)

	if err := svc.Delete(ctx, p.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err := svc.Get(ctx, p.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/... -v -race -run TestPipelineService`
Expected: FAIL — `NewPipelineService` undefined

**Step 3: Write implementation**

```go
// internal/services/pipeline_service.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/upal/internal/repository"
	"github.com/anthropics/upal/internal/upal"
)

type PipelineService struct {
	repo    repository.PipelineRepository
	runRepo repository.PipelineRunRepository
}

func NewPipelineService(repo repository.PipelineRepository, runRepo repository.PipelineRunRepository) *PipelineService {
	return &PipelineService{repo: repo, runRepo: runRepo}
}

func (s *PipelineService) Create(ctx context.Context, p *upal.Pipeline) error {
	if p.Name == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if p.ID == "" {
		p.ID = upal.GenerateID("pipe")
	}
	for i := range p.Stages {
		if p.Stages[i].ID == "" {
			p.Stages[i].ID = fmt.Sprintf("stage-%d", i+1)
		}
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	return s.repo.Create(ctx, p)
}

func (s *PipelineService) Get(ctx context.Context, id string) (*upal.Pipeline, error) {
	return s.repo.Get(ctx, id)
}

func (s *PipelineService) List(ctx context.Context) ([]*upal.Pipeline, error) {
	return s.repo.List(ctx)
}

func (s *PipelineService) Update(ctx context.Context, p *upal.Pipeline) error {
	p.UpdatedAt = time.Now()
	return s.repo.Update(ctx, p)
}

func (s *PipelineService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *PipelineService) GetRun(ctx context.Context, runID string) (*upal.PipelineRun, error) {
	return s.runRepo.Get(ctx, runID)
}

func (s *PipelineService) ListRuns(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	return s.runRepo.ListByPipeline(ctx, pipelineID)
}

func (s *PipelineService) CreateRun(ctx context.Context, run *upal.PipelineRun) error {
	if run.ID == "" {
		run.ID = upal.GenerateID("prun")
	}
	run.StartedAt = time.Now()
	return s.runRepo.Create(ctx, run)
}

func (s *PipelineService) UpdateRun(ctx context.Context, run *upal.PipelineRun) error {
	return s.runRepo.Update(ctx, run)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/... -v -race -run TestPipelineService`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/services/pipeline_service.go internal/services/pipeline_service_test.go
git commit -m "feat: add PipelineService for CRUD and run management"
```

---

### Task 8: PipelineRunner & StageExecutor Interface

**Files:**
- Create: `internal/services/pipeline_runner.go`
- Create: `internal/services/pipeline_runner_test.go`
- Create: `internal/services/stage_workflow.go`
- Create: `internal/services/stage_approval.go`
- Create: `internal/services/stage_transform.go`

**Step 1: Write the failing test**

```go
// internal/services/pipeline_runner_test.go
package services

import (
	"context"
	"testing"

	"github.com/anthropics/upal/internal/repository"
	"github.com/anthropics/upal/internal/upal"
)

// mockStageExecutor records calls and returns canned results.
type mockStageExecutor struct {
	stageType string
	calls     []string
	output    map[string]any
	err       error
}

func (m *mockStageExecutor) Type() string { return m.stageType }
func (m *mockStageExecutor) Execute(_ context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	m.calls = append(m.calls, stage.ID)
	if m.err != nil {
		return nil, m.err
	}
	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "completed",
		Output:  m.output,
	}, nil
}

func TestPipelineRunner_ExecuteSequential(t *testing.T) {
	runRepo := repository.NewMemoryPipelineRunRepository()
	wfExec := &mockStageExecutor{stageType: "workflow", output: map[string]any{"result": "ok"}}
	transformExec := &mockStageExecutor{stageType: "transform", output: map[string]any{"transformed": true}}

	runner := NewPipelineRunner(runRepo)
	runner.RegisterExecutor(wfExec)
	runner.RegisterExecutor(transformExec)

	pipeline := &upal.Pipeline{
		ID:   "pipe-1",
		Name: "Test",
		Stages: []upal.Stage{
			{ID: "s1", Name: "Collect", Type: "workflow"},
			{ID: "s2", Name: "Transform", Type: "transform"},
		},
	}

	run, err := runner.Start(context.Background(), pipeline)
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if run.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", run.Status)
	}
	if len(wfExec.calls) != 1 || wfExec.calls[0] != "s1" {
		t.Errorf("expected workflow executor called with s1, got %v", wfExec.calls)
	}
	if len(transformExec.calls) != 1 || transformExec.calls[0] != "s2" {
		t.Errorf("expected transform executor called with s2, got %v", transformExec.calls)
	}
}

func TestPipelineRunner_StageFailure(t *testing.T) {
	runRepo := repository.NewMemoryPipelineRunRepository()
	failExec := &mockStageExecutor{
		stageType: "workflow",
		err:       context.DeadlineExceeded,
	}

	runner := NewPipelineRunner(runRepo)
	runner.RegisterExecutor(failExec)

	pipeline := &upal.Pipeline{
		ID:   "pipe-2",
		Name: "Fail Test",
		Stages: []upal.Stage{
			{ID: "s1", Name: "Broken", Type: "workflow"},
		},
	}

	run, err := runner.Start(context.Background(), pipeline)
	if err == nil {
		t.Fatal("expected error from failed stage")
	}
	if run.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", run.Status)
	}
}

func TestPipelineRunner_UnknownStageType(t *testing.T) {
	runner := NewPipelineRunner(repository.NewMemoryPipelineRunRepository())

	pipeline := &upal.Pipeline{
		ID:   "pipe-3",
		Name: "Unknown",
		Stages: []upal.Stage{
			{ID: "s1", Type: "quantum_computer"},
		},
	}

	_, err := runner.Start(context.Background(), pipeline)
	if err == nil {
		t.Fatal("expected error for unknown stage type")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/... -v -race -run TestPipelineRunner`
Expected: FAIL — undefined symbols

**Step 3: Write PipelineRunner implementation**

```go
// internal/services/pipeline_runner.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/upal/internal/repository"
	"github.com/anthropics/upal/internal/upal"
)

// StageExecutor is the interface for executing a pipeline stage.
// Implement this interface to add new stage types.
type StageExecutor interface {
	Type() string
	Execute(ctx context.Context, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error)
}

// PipelineRunner orchestrates sequential execution of pipeline stages.
type PipelineRunner struct {
	executors map[string]StageExecutor
	runRepo   repository.PipelineRunRepository
}

func NewPipelineRunner(runRepo repository.PipelineRunRepository) *PipelineRunner {
	return &PipelineRunner{
		executors: make(map[string]StageExecutor),
		runRepo:   runRepo,
	}
}

func (r *PipelineRunner) RegisterExecutor(exec StageExecutor) {
	r.executors[exec.Type()] = exec
}

func (r *PipelineRunner) Start(ctx context.Context, pipeline *upal.Pipeline) (*upal.PipelineRun, error) {
	run := &upal.PipelineRun{
		ID:           upal.GenerateID("prun"),
		PipelineID:   pipeline.ID,
		Status:       "running",
		StageResults: make(map[string]*upal.StageResult),
		StartedAt:    time.Now(),
	}
	r.runRepo.Create(ctx, run)

	var prevResult *upal.StageResult

	for _, stage := range pipeline.Stages {
		executor, ok := r.executors[stage.Type]
		if !ok {
			run.Status = "failed"
			r.runRepo.Update(ctx, run)
			return run, fmt.Errorf("no executor registered for stage type %q", stage.Type)
		}

		run.CurrentStage = stage.ID
		stageResult := &upal.StageResult{
			StageID:   stage.ID,
			Status:    "running",
			StartedAt: time.Now(),
		}
		run.StageResults[stage.ID] = stageResult
		r.runRepo.Update(ctx, run)

		result, err := executor.Execute(ctx, stage, prevResult)
		if err != nil {
			now := time.Now()
			stageResult.Status = "failed"
			stageResult.Error = err.Error()
			stageResult.CompletedAt = &now
			run.Status = "failed"
			run.CompletedAt = &now
			r.runRepo.Update(ctx, run)
			return run, fmt.Errorf("stage %q failed: %w", stage.ID, err)
		}

		if result.Status == "waiting" {
			run.Status = "waiting"
			run.StageResults[stage.ID] = result
			r.runRepo.Update(ctx, run)
			return run, nil
		}

		now := time.Now()
		result.CompletedAt = &now
		run.StageResults[stage.ID] = result
		r.runRepo.Update(ctx, run)

		prevResult = result
	}

	now := time.Now()
	run.Status = "completed"
	run.CompletedAt = &now
	r.runRepo.Update(ctx, run)

	return run, nil
}
```

**Step 4: Write stub stage executors**

```go
// internal/services/stage_workflow.go
package services

import (
	"context"
	"fmt"

	"github.com/anthropics/upal/internal/upal"
)

// WorkflowStageExecutor runs a workflow by name.
type WorkflowStageExecutor struct {
	workflowSvc *WorkflowService
}

func NewWorkflowStageExecutor(workflowSvc *WorkflowService) *WorkflowStageExecutor {
	return &WorkflowStageExecutor{workflowSvc: workflowSvc}
}

func (e *WorkflowStageExecutor) Type() string { return "workflow" }

func (e *WorkflowStageExecutor) Execute(ctx context.Context, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error) {
	wfName := stage.Config.WorkflowName
	if wfName == "" {
		return nil, fmt.Errorf("workflow_name is required for workflow stage")
	}

	// Build inputs from input mapping + previous stage output
	inputs := make(map[string]any)
	if prevResult != nil && stage.Config.InputMapping != nil {
		for destKey, srcExpr := range stage.Config.InputMapping {
			if val, ok := prevResult.Output[srcExpr]; ok {
				inputs[destKey] = val
			}
		}
	}

	// Look up and run the workflow
	wf, err := e.workflowSvc.repo.Get(ctx, wfName)
	if err != nil {
		return nil, fmt.Errorf("workflow %q not found: %w", wfName, err)
	}

	eventCh, resultCh, err := e.workflowSvc.Run(ctx, wf, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow %q: %w", wfName, err)
	}

	// Drain events
	for range eventCh {
	}

	result := <-resultCh

	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "completed",
		Output:  result.State,
	}, nil
}
```

```go
// internal/services/stage_approval.go
package services

import (
	"context"
	"time"

	"github.com/anthropics/upal/internal/upal"
)

// ApprovalStageExecutor pauses the pipeline and waits for user approval.
type ApprovalStageExecutor struct {
	waitHandles map[string]chan map[string]any // runID+stageID → response channel
}

func NewApprovalStageExecutor() *ApprovalStageExecutor {
	return &ApprovalStageExecutor{
		waitHandles: make(map[string]chan map[string]any),
	}
}

func (e *ApprovalStageExecutor) Type() string { return "approval" }

func (e *ApprovalStageExecutor) Execute(_ context.Context, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error) {
	// Return a "waiting" result — the PipelineRunner will persist this
	// and the API layer handles resume via Approve/Reject endpoints.
	return &upal.StageResult{
		StageID:   stage.ID,
		Status:    "waiting",
		Output:    map[string]any{"message": stage.Config.Message},
		StartedAt: time.Now(),
	}, nil
}
```

```go
// internal/services/stage_transform.go
package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/upal/internal/upal"
)

// TransformStageExecutor processes previous stage output without LLM.
type TransformStageExecutor struct{}

func (e *TransformStageExecutor) Type() string { return "transform" }

func (e *TransformStageExecutor) Execute(_ context.Context, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error) {
	output := make(map[string]any)

	if prevResult != nil {
		// Pass through previous output, applying input mapping if configured
		if stage.Config.InputMapping != nil {
			for destKey, srcKey := range stage.Config.InputMapping {
				if val, ok := prevResult.Output[srcKey]; ok {
					output[destKey] = val
				}
			}
		} else {
			// Pass through all
			for k, v := range prevResult.Output {
				output[k] = v
			}
		}
	}

	// If expression is set, evaluate it (basic JSON parse for now)
	if stage.Config.Expression != "" {
		var parsed any
		if err := json.Unmarshal([]byte(stage.Config.Expression), &parsed); err != nil {
			return nil, fmt.Errorf("transform expression error: %w", err)
		}
		output["expression_result"] = parsed
	}

	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "completed",
		Output:  output,
	}, nil
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/services/... -v -race -run TestPipelineRunner`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/services/pipeline_runner.go internal/services/pipeline_runner_test.go \
       internal/services/stage_workflow.go internal/services/stage_approval.go internal/services/stage_transform.go
git commit -m "feat: add PipelineRunner with StageExecutor interface and workflow/approval/transform executors"
```

---

### Task 9: Pipeline API Handlers

**Files:**
- Create: `internal/api/pipelines.go`
- Modify: `internal/api/server.go` (add fields + routes)

**Step 1: Write API handlers**

```go
// internal/api/pipelines.go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/anthropics/upal/internal/upal"
)

func (s *Server) createPipeline(w http.ResponseWriter, r *http.Request) {
	var p upal.Pipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if p.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if err := s.pipelineSvc.Create(r.Context(), &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	pipelines, err := s.pipelineSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pipelines == nil {
		pipelines = []*upal.Pipeline{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pipelines)
}

func (s *Server) getPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.pipelineSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func (s *Server) updatePipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var p upal.Pipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	p.ID = id
	if err := s.pipelineSvc.Update(r.Context(), &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func (s *Server) deletePipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.pipelineSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) startPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.pipelineSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	run, err := s.pipelineRunner.Start(r.Context(), p)
	if err != nil && run == nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if run.Status == "waiting" {
		w.WriteHeader(http.StatusAccepted)
	} else if run.Status == "failed" {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(run)
}

func (s *Server) listPipelineRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	runs, err := s.pipelineSvc.ListRuns(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if runs == nil {
		runs = []*upal.PipelineRun{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

func (s *Server) approvePipelineStage(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement resume logic — look up waiting run, advance to next stage
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

func (s *Server) rejectPipelineStage(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement reject logic — mark run as failed or idle
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}
```

**Step 2: Add fields to Server struct and routes**

In `internal/api/server.go`, add to the Server struct:

```go
pipelineSvc    *services.PipelineService
pipelineRunner *services.PipelineRunner
```

Add setter:

```go
func (s *Server) SetPipelineService(svc *services.PipelineService) {
	s.pipelineSvc = svc
}

func (s *Server) SetPipelineRunner(runner *services.PipelineRunner) {
	s.pipelineRunner = runner
}
```

Add routes in the Handler() method inside the `/api` route group:

```go
r.Route("/pipelines", func(r chi.Router) {
    r.Post("/", s.createPipeline)
    r.Get("/", s.listPipelines)
    r.Get("/{id}", s.getPipeline)
    r.Put("/{id}", s.updatePipeline)
    r.Delete("/{id}", s.deletePipeline)
    r.Post("/{id}/start", s.startPipeline)
    r.Get("/{id}/runs", s.listPipelineRuns)
    r.Post("/{id}/stages/{stageId}/approve", s.approvePipelineStage)
    r.Post("/{id}/stages/{stageId}/reject", s.rejectPipelineStage)
})
```

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: BUILD SUCCESS

**Step 4: Commit**

```bash
git add internal/api/pipelines.go internal/api/server.go
git commit -m "feat: add Pipeline REST API handlers and routes"
```

---

### Task 10: Wire Pipeline Backend in main.go

**Files:**
- Modify: `cmd/upal/main.go`

**Step 1: Add pipeline wiring**

After existing service initialization in `serve()`, add:

```go
// Pipeline
pipelineRepo := repository.NewMemoryPipelineRepository()
pipelineRunRepo := repository.NewMemoryPipelineRunRepository()
pipelineSvc := services.NewPipelineService(pipelineRepo, pipelineRunRepo)
pipelineRunner := services.NewPipelineRunner(pipelineRunRepo)
pipelineRunner.RegisterExecutor(services.NewWorkflowStageExecutor(workflowSvc))
pipelineRunner.RegisterExecutor(services.NewApprovalStageExecutor())
pipelineRunner.RegisterExecutor(&services.TransformStageExecutor{})

srv.SetPipelineService(pipelineSvc)
srv.SetPipelineRunner(pipelineRunner)
```

**Step 2: Verify it compiles and starts**

Run: `go build ./cmd/upal/...`
Expected: BUILD SUCCESS

**Step 3: Run all tests**

Run: `go test ./... -v -race -count=1`
Expected: ALL PASS (or only pre-existing failures)

**Step 4: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat: wire pipeline service and runner in server startup"
```

---

## Phase 3: Pipeline Frontend

### Task 11: Pipeline API Types & Client

**Files:**
- Modify: `web/src/lib/api/types.ts`
- Create: `web/src/lib/api/pipelines.ts`
- Modify: `web/src/lib/api/index.ts` (add export)

**Step 1: Add TypeScript types**

Append to `web/src/lib/api/types.ts`:

```typescript
// Pipeline
export type Pipeline = {
  id: string
  name: string
  description?: string
  stages: Stage[]
  created_at: string
  updated_at: string
}

export type Stage = {
  id: string
  name: string
  type: 'workflow' | 'approval' | 'schedule' | 'trigger' | 'transform'
  config: StageConfig
  depends_on?: string[]
}

export type StageConfig = {
  workflow_name?: string
  input_mapping?: Record<string, string>
  message?: string
  connection_id?: string
  timeout?: number
  cron?: string
  timezone?: string
  trigger_id?: string
  expression?: string
}

export type PipelineRun = {
  id: string
  pipeline_id: string
  status: 'pending' | 'running' | 'waiting' | 'completed' | 'failed'
  current_stage?: string
  stage_results?: Record<string, StageResult>
  started_at: string
  completed_at?: string
}

export type StageResult = {
  stage_id: string
  status: 'pending' | 'running' | 'waiting' | 'completed' | 'skipped' | 'failed'
  output?: Record<string, unknown>
  error?: string
  started_at: string
  completed_at?: string
}
```

**Step 2: Create API client**

```typescript
// web/src/lib/api/pipelines.ts
import { apiFetch } from './client'
import type { Pipeline, PipelineRun } from './types'

const API_BASE = '/api'

export async function fetchPipelines(): Promise<Pipeline[]> {
  return apiFetch<Pipeline[]>(`${API_BASE}/pipelines`)
}

export async function fetchPipeline(id: string): Promise<Pipeline> {
  return apiFetch<Pipeline>(`${API_BASE}/pipelines/${encodeURIComponent(id)}`)
}

export async function createPipeline(data: Partial<Pipeline>): Promise<Pipeline> {
  return apiFetch<Pipeline>(`${API_BASE}/pipelines`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function updatePipeline(id: string, data: Partial<Pipeline>): Promise<Pipeline> {
  return apiFetch<Pipeline>(`${API_BASE}/pipelines/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function deletePipeline(id: string): Promise<void> {
  return apiFetch<void>(`${API_BASE}/pipelines/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export async function startPipeline(id: string): Promise<PipelineRun> {
  return apiFetch<PipelineRun>(`${API_BASE}/pipelines/${encodeURIComponent(id)}/start`, {
    method: 'POST',
  })
}

export async function fetchPipelineRuns(id: string): Promise<PipelineRun[]> {
  return apiFetch<PipelineRun[]>(`${API_BASE}/pipelines/${encodeURIComponent(id)}/runs`)
}

export async function approvePipelineStage(pipelineId: string, stageId: string): Promise<void> {
  return apiFetch<void>(
    `${API_BASE}/pipelines/${encodeURIComponent(pipelineId)}/stages/${encodeURIComponent(stageId)}/approve`,
    { method: 'POST' }
  )
}

export async function rejectPipelineStage(pipelineId: string, stageId: string): Promise<void> {
  return apiFetch<void>(
    `${API_BASE}/pipelines/${encodeURIComponent(pipelineId)}/stages/${encodeURIComponent(stageId)}/reject`,
    { method: 'POST' }
  )
}
```

**Step 3: Add to barrel export in `web/src/lib/api/index.ts`**

Add: `export * from './pipelines'`

**Step 4: Type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add web/src/lib/api/types.ts web/src/lib/api/pipelines.ts web/src/lib/api/index.ts
git commit -m "feat: add Pipeline API types and client functions"
```

---

### Task 12: Pipelines Page (List View)

**Files:**
- Create: `web/src/pages/Pipelines.tsx`
- Modify: `web/src/App.tsx` (add route)
- Modify: `web/src/components/Header.tsx` (add nav link)

**Step 1: Create the Pipelines page**

```tsx
// web/src/pages/Pipelines.tsx
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Plus, Play, Square, Loader2, Trash2, Clock, CheckCircle2,
  AlertCircle, PauseCircle, GitBranch,
} from 'lucide-react'
import { Header } from '@/components/Header'
import { fetchPipelines, createPipeline, deletePipeline, startPipeline } from '@/lib/api'
import type { Pipeline, PipelineRun } from '@/lib/api/types'

const stageTypeIcons: Record<string, string> = {
  workflow: '▶',
  approval: '⏸',
  schedule: '⏰',
  trigger: '⚡',
  transform: '🔄',
}

const statusConfig: Record<string, { icon: typeof CheckCircle2; color: string; label: string }> = {
  idle: { icon: Clock, color: 'text-muted-foreground', label: 'Idle' },
  running: { icon: Loader2, color: 'text-info', label: 'Running' },
  waiting: { icon: PauseCircle, color: 'text-warning', label: 'Waiting' },
  completed: { icon: CheckCircle2, color: 'text-success', label: 'Completed' },
  failed: { icon: AlertCircle, color: 'text-destructive', label: 'Failed' },
}

export default function Pipelines() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

  const reload = async () => {
    try {
      const data = await fetchPipelines()
      setPipelines(data)
    } catch {
      // silent
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { reload() }, [])

  const handleCreate = async () => {
    try {
      const p = await createPipeline({
        name: 'New Pipeline',
        stages: [],
      })
      navigate(`/pipelines/${p.id}`)
    } catch {
      // silent
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deletePipeline(id)
      reload()
    } catch {
      // silent
    }
  }

  const handleStart = async (id: string) => {
    try {
      await startPipeline(id)
      reload()
    } catch {
      // silent
    }
  }

  return (
    <div className="flex flex-col h-screen bg-background">
      <Header />
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto px-6 py-6 space-y-6">
          <div className="flex items-center justify-between">
            <h1 className="text-xl font-semibold">Pipelines</h1>
            <button
              onClick={handleCreate}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <Plus className="h-4 w-4" />
              New Pipeline
            </button>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : pipelines.length === 0 ? (
            <div className="text-center py-16 text-muted-foreground">
              <GitBranch className="h-10 w-10 mx-auto mb-3 opacity-40" />
              <p className="text-sm">No pipelines yet</p>
              <p className="text-xs mt-1">Create one to chain workflows with approvals and schedules</p>
            </div>
          ) : (
            <div className="space-y-3">
              {pipelines.map((p) => (
                <div
                  key={p.id}
                  className="border rounded-xl p-4 hover:border-foreground/20 transition-colors cursor-pointer"
                  onClick={() => navigate(`/pipelines/${p.id}`)}
                >
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-medium">{p.name}</h3>
                    <div className="flex items-center gap-1">
                      <button
                        onClick={(e) => { e.stopPropagation(); handleStart(p.id) }}
                        className="p-1.5 rounded-md hover:bg-muted transition-colors"
                        title="Start"
                      >
                        <Play className="h-3.5 w-3.5" />
                      </button>
                      <button
                        onClick={(e) => { e.stopPropagation(); handleDelete(p.id) }}
                        className="p-1.5 rounded-md hover:bg-muted transition-colors text-muted-foreground hover:text-destructive"
                        title="Delete"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>

                  {p.description && (
                    <p className="text-xs text-muted-foreground mb-2">{p.description}</p>
                  )}

                  {p.stages.length > 0 && (
                    <div className="flex items-center gap-1 text-xs text-muted-foreground">
                      {p.stages.map((stage, i) => (
                        <span key={stage.id} className="flex items-center gap-1">
                          {i > 0 && <span className="text-border">→</span>}
                          <span className="px-1.5 py-0.5 rounded bg-muted">
                            {stageTypeIcons[stage.type] || '○'} {stage.name || stage.type}
                          </span>
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
```

**Step 2: Add route in App.tsx**

Add import: `const Pipelines = lazy(() => import('./pages/Pipelines'))`

Add route: `<Route path="/pipelines" element={<Pipelines />} />`
Add route: `<Route path="/pipelines/:id" element={<Pipelines />} />`

**Step 3: Add nav link in Header.tsx**

Add to the nav links array:

```typescript
{ to: "/pipelines", label: "Pipelines" },
```

**Step 4: Type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add web/src/pages/Pipelines.tsx web/src/App.tsx web/src/components/Header.tsx
git commit -m "feat: add Pipelines page with list view and navigation"
```

---

### Task 13: Pipeline Editor Components

**Files:**
- Create: `web/src/components/pipelines/StageCard.tsx`
- Create: `web/src/components/pipelines/PipelineEditor.tsx`
- Modify: `web/src/pages/Pipelines.tsx` (add editor view when `:id` param present)

**Step 1: Create StageCard component**

```tsx
// web/src/components/pipelines/StageCard.tsx
import { Trash2, GripVertical } from 'lucide-react'
import type { Stage } from '@/lib/api/types'

type Props = {
  stage: Stage
  index: number
  isActive?: boolean
  onChange: (stage: Stage) => void
  onDelete: () => void
}

const stageTypeLabels: Record<string, string> = {
  workflow: 'Workflow',
  approval: 'Approval',
  schedule: 'Schedule',
  trigger: 'Trigger',
  transform: 'Transform',
}

const stageTypeBg: Record<string, string> = {
  workflow: 'border-l-info',
  approval: 'border-l-warning',
  schedule: 'border-l-success',
  trigger: 'border-l-[oklch(0.7_0.15_30)]',
  transform: 'border-l-muted-foreground',
}

export function StageCard({ stage, index, isActive, onChange, onDelete }: Props) {
  return (
    <div className={`border rounded-lg border-l-4 ${stageTypeBg[stage.type] || ''} ${isActive ? 'ring-2 ring-primary' : ''}`}>
      <div className="flex items-center justify-between px-3 py-2">
        <div className="flex items-center gap-2">
          <GripVertical className="h-3.5 w-3.5 text-muted-foreground/50 cursor-grab" />
          <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
            {stageTypeLabels[stage.type] || stage.type}
          </span>
        </div>
        <button
          onClick={onDelete}
          className="p-1 rounded hover:bg-muted transition-colors text-muted-foreground hover:text-destructive"
        >
          <Trash2 className="h-3 w-3" />
        </button>
      </div>

      <div className="px-3 pb-3 space-y-2">
        <input
          type="text"
          value={stage.name}
          onChange={(e) => onChange({ ...stage, name: e.target.value })}
          placeholder="Stage name"
          className="w-full text-sm font-medium bg-transparent border-none outline-none placeholder:text-muted-foreground/50"
        />

        {stage.type === 'workflow' && (
          <input
            type="text"
            value={stage.config.workflow_name || ''}
            onChange={(e) => onChange({ ...stage, config: { ...stage.config, workflow_name: e.target.value } })}
            placeholder="Workflow name"
            className="w-full text-xs bg-muted/50 rounded px-2 py-1.5 outline-none"
          />
        )}

        {stage.type === 'approval' && (
          <textarea
            value={stage.config.message || ''}
            onChange={(e) => onChange({ ...stage, config: { ...stage.config, message: e.target.value } })}
            placeholder="Approval message"
            rows={2}
            className="w-full text-xs bg-muted/50 rounded px-2 py-1.5 outline-none resize-none"
          />
        )}

        {stage.type === 'schedule' && (
          <input
            type="text"
            value={stage.config.cron || ''}
            onChange={(e) => onChange({ ...stage, config: { ...stage.config, cron: e.target.value } })}
            placeholder="Cron expression (e.g. 0 9 * * *)"
            className="w-full text-xs bg-muted/50 rounded px-2 py-1.5 outline-none font-mono"
          />
        )}
      </div>
    </div>
  )
}
```

**Step 2: Create PipelineEditor component**

```tsx
// web/src/components/pipelines/PipelineEditor.tsx
import { useState } from 'react'
import { Plus, ArrowLeft, Save } from 'lucide-react'
import { StageCard } from './StageCard'
import type { Pipeline, Stage, StageConfig } from '@/lib/api/types'

type Props = {
  pipeline: Pipeline
  onSave: (pipeline: Pipeline) => void
  onBack: () => void
}

const newStageDefaults: Record<string, Partial<Stage>> = {
  workflow: { type: 'workflow', config: {} },
  approval: { type: 'approval', config: { timeout: 3600 } },
  schedule: { type: 'schedule', config: {} },
  transform: { type: 'transform', config: {} },
}

export function PipelineEditor({ pipeline, onSave, onBack }: Props) {
  const [draft, setDraft] = useState<Pipeline>({ ...pipeline })

  const updateStage = (index: number, stage: Stage) => {
    const stages = [...draft.stages]
    stages[index] = stage
    setDraft({ ...draft, stages })
  }

  const deleteStage = (index: number) => {
    const stages = draft.stages.filter((_, i) => i !== index)
    setDraft({ ...draft, stages })
  }

  const addStage = (type: string) => {
    const defaults = newStageDefaults[type] || { type, config: {} }
    const stage: Stage = {
      id: `stage-${draft.stages.length + 1}`,
      name: '',
      type: type as Stage['type'],
      config: defaults.config as StageConfig,
    }
    setDraft({ ...draft, stages: [...draft.stages, stage] })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={onBack} className="p-1.5 rounded-md hover:bg-muted transition-colors">
            <ArrowLeft className="h-4 w-4" />
          </button>
          <input
            type="text"
            value={draft.name}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            className="text-lg font-semibold bg-transparent border-none outline-none"
            placeholder="Pipeline name"
          />
        </div>
        <button
          onClick={() => onSave(draft)}
          className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Save className="h-3.5 w-3.5" />
          Save
        </button>
      </div>

      {draft.description !== undefined && (
        <input
          type="text"
          value={draft.description || ''}
          onChange={(e) => setDraft({ ...draft, description: e.target.value })}
          placeholder="Pipeline description (optional)"
          className="w-full text-sm text-muted-foreground bg-transparent border-none outline-none"
        />
      )}

      <div className="space-y-2">
        {draft.stages.map((stage, i) => (
          <div key={stage.id}>
            {i > 0 && (
              <div className="flex justify-center py-1">
                <div className="w-px h-4 bg-border" />
              </div>
            )}
            <StageCard
              stage={stage}
              index={i}
              onChange={(s) => updateStage(i, s)}
              onDelete={() => deleteStage(i)}
            />
          </div>
        ))}
      </div>

      <div className="flex items-center gap-2 pt-2">
        <span className="text-xs text-muted-foreground">Add stage:</span>
        {Object.keys(newStageDefaults).map((type) => (
          <button
            key={type}
            onClick={() => addStage(type)}
            className="flex items-center gap-1 px-2 py-1 text-xs rounded-md border hover:bg-muted transition-colors"
          >
            <Plus className="h-3 w-3" />
            {type}
          </button>
        ))}
      </div>
    </div>
  )
}
```

**Step 3: Update Pipelines page to show editor when `:id` param is present**

In `web/src/pages/Pipelines.tsx`, add `useParams` and conditional rendering:

```tsx
import { useParams } from 'react-router-dom'
import { fetchPipeline, updatePipeline } from '@/lib/api'
import { PipelineEditor } from '@/components/pipelines/PipelineEditor'

// Inside component:
const { id } = useParams<{ id: string }>()
const [selected, setSelected] = useState<Pipeline | null>(null)

useEffect(() => {
  if (id) {
    fetchPipeline(id).then(setSelected).catch(() => navigate('/pipelines'))
  } else {
    setSelected(null)
  }
}, [id])

const handleSave = async (p: Pipeline) => {
  await updatePipeline(p.id, p)
  reload()
}

// In the render: if selected, show PipelineEditor; otherwise show list
```

**Step 4: Type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add web/src/components/pipelines/StageCard.tsx web/src/components/pipelines/PipelineEditor.tsx web/src/pages/Pipelines.tsx
git commit -m "feat: add Pipeline editor with StageCard components"
```

---

### Task 14: Pipeline Run History & Approval UI

**Files:**
- Create: `web/src/components/pipelines/PipelineRunHistory.tsx`
- Modify: `web/src/pages/Pipelines.tsx` (add run history section to editor view)

**Step 1: Create PipelineRunHistory component**

```tsx
// web/src/components/pipelines/PipelineRunHistory.tsx
import { useState, useEffect } from 'react'
import { CheckCircle2, XCircle, Loader2, PauseCircle, Clock } from 'lucide-react'
import { fetchPipelineRuns, approvePipelineStage, rejectPipelineStage } from '@/lib/api'
import type { Pipeline, PipelineRun, StageResult } from '@/lib/api/types'

type Props = {
  pipeline: Pipeline
}

const statusIcons: Record<string, typeof CheckCircle2> = {
  pending: Clock,
  running: Loader2,
  waiting: PauseCircle,
  completed: CheckCircle2,
  failed: XCircle,
  skipped: Clock,
}

const statusColors: Record<string, string> = {
  pending: 'text-muted-foreground',
  running: 'text-info',
  waiting: 'text-warning',
  completed: 'text-success',
  failed: 'text-destructive',
  skipped: 'text-muted-foreground/50',
}

export function PipelineRunHistory({ pipeline }: Props) {
  const [runs, setRuns] = useState<PipelineRun[]>([])
  const [loading, setLoading] = useState(true)

  const reload = async () => {
    try {
      const data = await fetchPipelineRuns(pipeline.id)
      setRuns(data)
    } catch {
      // silent
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { reload() }, [pipeline.id])

  const handleApprove = async (run: PipelineRun, stageId: string) => {
    await approvePipelineStage(pipeline.id, stageId)
    reload()
  }

  const handleReject = async (run: PipelineRun, stageId: string) => {
    await rejectPipelineStage(pipeline.id, stageId)
    reload()
  }

  if (loading) {
    return <Loader2 className="h-4 w-4 animate-spin text-muted-foreground mx-auto" />
  }

  if (runs.length === 0) {
    return (
      <p className="text-xs text-muted-foreground text-center py-4">
        No runs yet. Start the pipeline to see history.
      </p>
    )
  }

  return (
    <div className="space-y-2">
      <h3 className="text-sm font-medium">Run History</h3>
      {runs.map((run) => {
        const Icon = statusIcons[run.status] || Clock
        return (
          <div key={run.id} className="border rounded-lg p-3">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <Icon className={`h-3.5 w-3.5 ${statusColors[run.status]} ${run.status === 'running' ? 'animate-spin' : ''}`} />
                <span className="text-xs font-medium">{run.id.slice(0, 12)}</span>
                <span className="text-[10px] text-muted-foreground">
                  {new Date(run.started_at).toLocaleString()}
                </span>
              </div>
              <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-medium ${statusColors[run.status]}`}>
                {run.status}
              </span>
            </div>

            <div className="flex items-center gap-1">
              {pipeline.stages.map((stage, i) => {
                const result = run.stage_results?.[stage.id]
                const stageStatus = result?.status || 'pending'
                const StageIcon = statusIcons[stageStatus] || Clock

                return (
                  <div key={stage.id} className="flex items-center gap-1">
                    {i > 0 && <span className="text-border text-xs">→</span>}
                    <div className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-xs ${
                      stageStatus === 'waiting' ? 'bg-warning/10' : 'bg-muted'
                    }`}>
                      <StageIcon className={`h-3 w-3 ${statusColors[stageStatus]} ${stageStatus === 'running' ? 'animate-spin' : ''}`} />
                      <span className="truncate max-w-[80px]">{stage.name || stage.type}</span>
                    </div>
                  </div>
                )
              })}
            </div>

            {run.status === 'waiting' && run.current_stage && (
              <div className="flex items-center gap-2 mt-2 pt-2 border-t">
                <span className="text-xs text-muted-foreground">Awaiting approval:</span>
                <button
                  onClick={() => handleApprove(run, run.current_stage!)}
                  className="px-2 py-0.5 text-xs font-medium rounded bg-success/10 text-success hover:bg-success/20 transition-colors"
                >
                  Approve
                </button>
                <button
                  onClick={() => handleReject(run, run.current_stage!)}
                  className="px-2 py-0.5 text-xs font-medium rounded bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors"
                >
                  Reject
                </button>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
```

**Step 2: Add run history to the editor view in Pipelines.tsx**

Below the `<PipelineEditor>`, add:

```tsx
{selected && <PipelineRunHistory pipeline={selected} />}
```

**Step 3: Type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add web/src/components/pipelines/PipelineRunHistory.tsx web/src/pages/Pipelines.tsx
git commit -m "feat: add Pipeline run history with approval/reject UI"
```

---

### Task 15: Frontend Type-Check & Build Verification

**Step 1: Run TypeScript type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No type errors

**Step 2: Run ESLint**

Run: `cd web && npm run lint`
Expected: No lint errors (or only pre-existing ones)

**Step 3: Run Vite build**

Run: `cd web && npm run build`
Expected: BUILD SUCCESS

**Step 4: Commit any fixes needed**

```bash
git add -A web/src/
git commit -m "fix: resolve any type-check or build issues in pipeline frontend"
```

---

### Task 16: Full Backend Test Suite

**Step 1: Run all Go tests**

Run: `go test ./... -v -race -count=1`
Expected: ALL PASS (or only pre-existing failures unrelated to pipeline changes)

**Step 2: Build the full binary**

Run: `make build`
Expected: BUILD SUCCESS

**Step 3: Commit if any fixes were needed**

```bash
git add -A
git commit -m "fix: resolve any test failures from pipeline integration"
```

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| **Phase 1** | Tasks 1-4 | 3 new tools (fetch_rss, content_store, publish) + registration |
| **Phase 2** | Tasks 5-10 | Pipeline types, repository, service, runner, API, wiring |
| **Phase 3** | Tasks 11-15 | Frontend types, API client, page, editor, run history, approval UI |
| **Phase 4** | Task 16 | Full verification |

**Total: 16 tasks across 4 phases**

Phase 4 (templates & extensions) from the design doc is deferred — it builds on top of this foundation and can be planned separately once the core pipeline system is verified working end-to-end.
