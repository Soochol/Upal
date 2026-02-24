# Google Trends + Social Trends Source Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Google Trends와 Social Trends(Bluesky+Mastodon) 파이프라인 소스를 실제 동작하도록 구현

**Architecture:** Google Trends는 공식 RSS 피드로 매핑하여 기존 rssFetcher 재사용. Twitter/X는 Social Trends로 리네임하고 Bluesky + Mastodon 공개 API를 호출하는 새 socialFetcher 구현. 프론트엔드 타입 시스템에서 `twitter` → `social`로 전환.

**Tech Stack:** Go (net/http, encoding/json), React/TypeScript, Bluesky AT Protocol API, Mastodon API

**Design doc:** `docs/plans/2026-02-24-pipeline-source-trends-social-design.md`

---

### Task 1: Backend — CollectSource와 PipelineSource에 필드 추가

**Files:**
- Modify: `internal/upal/pipeline.go:35-46` (CollectSource struct)
- Modify: `internal/upal/pipeline.go:91-106` (PipelineSource struct)

**Step 1: Add Keywords, Accounts, and Geo to CollectSource**

In `internal/upal/pipeline.go`, add three fields to `CollectSource` after `ScrapeLimit`:

```go
Keywords []string `json:"keywords,omitempty"` // Social: search keywords
Accounts []string `json:"accounts,omitempty"` // Social: follow account handles
Geo      string   `json:"geo,omitempty"`      // Google Trends: country code
```

**Step 2: Add Accounts and Geo to PipelineSource**

In `internal/upal/pipeline.go`, add two fields to `PipelineSource` after `Keywords`:

```go
Accounts []string `json:"accounts,omitempty"` // social: follow account handles (e.g. "alice.bsky.social", "user@mastodon.social")
Geo      string   `json:"geo,omitempty"`      // google_trends: country code (e.g. "US", "KR")
```

**Step 3: Run tests to verify no breakage**

Run: `go test ./internal/... -v -race -count=1 2>&1 | tail -20`
Expected: All existing tests pass (JSONB, no schema change needed)

**Step 4: Commit**

```bash
git add internal/upal/pipeline.go
git commit -m "feat: add Keywords/Geo fields to CollectSource and PipelineSource"
```

---

### Task 2: Backend — Google Trends RSS 매핑 in mapPipelineSources

**Files:**
- Modify: `internal/services/content_collector.go:842-844`

**Step 1: Write test for mapPipelineSources google_trends mapping**

Since `mapPipelineSources` is unexported, test it indirectly via `ContentCollector` or add a test helper. The simplest approach: test it end-to-end by adding a unit test that calls `CollectStageExecutor.Execute` with a mock fetcher whose type is `"rss"` and verify that google_trends sources get routed there.

Add to `internal/services/content_collector_test.go` (create if needed):

```go
package services

import (
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestMapPipelineSources_GoogleTrends(t *testing.T) {
	sources := []upal.PipelineSource{
		{
			ID:       "gt1",
			Type:     "google_trends",
			Keywords: []string{"AI", "LLM"},
			Geo:      "KR",
			Limit:    10,
		},
	}
	result := mapPipelineSources(sources, false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source, got %d", len(result))
	}
	cs := result[0].collectSource
	if cs.Type != "rss" {
		t.Errorf("expected type=rss, got %q", cs.Type)
	}
	if cs.URL != "https://trends.google.com/trending/rss?geo=KR" {
		t.Errorf("unexpected URL: %s", cs.URL)
	}
	if cs.Limit != 10 {
		t.Errorf("expected limit=10, got %d", cs.Limit)
	}
}

func TestMapPipelineSources_GoogleTrends_DefaultGeo(t *testing.T) {
	sources := []upal.PipelineSource{
		{ID: "gt2", Type: "google_trends"},
	}
	result := mapPipelineSources(sources, false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source, got %d", len(result))
	}
	if result[0].collectSource.URL != "https://trends.google.com/trending/rss?geo=US" {
		t.Errorf("expected default geo=US, got URL: %s", result[0].collectSource.URL)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -v -race -run TestMapPipelineSources_GoogleTrends`
Expected: FAIL — google_trends still skipped, returns 0 mapped sources

**Step 3: Implement google_trends mapping**

In `internal/services/content_collector.go`, replace lines 842-844:

```go
case "google_trends", "twitter":
	log.Printf("content_collector: skipping unsupported source type %q (source %s)", ps.Type, ps.ID)
	continue
```

With:

```go
case "google_trends":
	cs.Type = "rss"
	geo := ps.Geo
	if geo == "" {
		geo = "US"
	}
	cs.URL = fmt.Sprintf("https://trends.google.com/trending/rss?geo=%s", geo)
	cs.Limit = ps.Limit
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -v -race -run TestMapPipelineSources_GoogleTrends`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/content_collector.go internal/services/content_collector_test.go
git commit -m "feat: implement Google Trends source via RSS feed mapping"
```

---

### Task 3: Backend — socialFetcher 구현 (트렌딩 + 키워드 + 계정 피드)

**Files:**
- Create: `internal/services/fetcher_social.go`
- Modify: `internal/services/stage_collect.go:32-41` (register new fetcher)

**Step 1: Write test for socialFetcher**

Create `internal/services/fetcher_social_test.go`:

```go
package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

func TestSocialFetcher_Type(t *testing.T) {
	f := services.NewSocialFetcher(http.DefaultClient)
	if f.Type() != "social" {
		t.Errorf("expected type=social, got %q", f.Type())
	}
}

func TestSocialFetcher_Fetch(t *testing.T) {
	// Mock Bluesky trending
	bskyTrending := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"topics": []map[string]any{
				{"topic": "AI", "link": "https://bsky.app/search?q=AI"},
				{"topic": "Go", "link": "https://bsky.app/search?q=Go"},
			},
		})
	}))
	defer bskyTrending.Close()

	// Mock Bluesky search
	bskySearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"posts": []map[string]any{
				{
					"uri":    "at://did:plc:abc/app.bsky.feed.post/123",
					"author": map[string]any{"handle": "alice.bsky.social"},
					"record": map[string]any{"text": "AI is transforming everything"},
				},
			},
		})
	}))
	defer bskySearch.Close()

	// Mock Mastodon trends/tags
	mastoTags := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "ai", "url": "https://mastodon.social/tags/ai"},
			{"name": "golang", "url": "https://mastodon.social/tags/golang"},
		})
	}))
	defer mastoTags.Close()

	// Mock Mastodon trends/links
	mastoLinks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"title": "AI News Article", "url": "https://example.com/ai-news", "description": "Latest AI developments"},
		})
	}))
	defer mastoLinks.Close()

	f := services.NewSocialFetcherWithEndpoints(http.DefaultClient, services.SocialEndpoints{
		BlueskyTrending:   bskyTrending.URL,
		BlueskySearch:     bskySearch.URL,
		BlueskyAuthorFeed: bskySearch.URL, // reuse mock
		MastodonTags:      mastoTags.URL,
		MastodonLinks:     mastoLinks.URL,
		MastodonBase:      mastoTags.URL, // reuse mock
	})

	src := upal.CollectSource{
		ID:       "social-1",
		Type:     "social",
		Keywords: []string{"AI"},
		Limit:    10,
	}

	text, data, err := f.Fetch(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Error("expected non-empty text")
	}
	if !strings.Contains(text, "Bluesky") {
		t.Error("expected text to contain Bluesky section")
	}
	if !strings.Contains(text, "Mastodon") {
		t.Error("expected text to contain Mastodon section")
	}

	items, ok := data.([]map[string]any)
	if !ok {
		t.Fatalf("expected data to be []map[string]any, got %T", data)
	}
	if len(items) == 0 {
		t.Error("expected at least one item in data")
	}
}

func TestSocialFetcher_AccountFeeds(t *testing.T) {
	// Mock Bluesky author feed
	bskyAuthorFeed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"feed": []map[string]any{
				{
					"post": map[string]any{
						"uri":    "at://did:plc:xyz/app.bsky.feed.post/456",
						"author": map[string]any{"handle": "bob.bsky.social"},
						"record": map[string]any{"text": "Check out my new project"},
					},
				},
			},
		})
	}))
	defer bskyAuthorFeed.Close()

	// Mock Mastodon account lookup
	mastoBase := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "lookup") {
			json.NewEncoder(w).Encode(map[string]any{"id": "12345"})
		} else if strings.Contains(r.URL.Path, "statuses") {
			json.NewEncoder(w).Encode([]map[string]any{
				{"content": "<p>Hello from Mastodon</p>", "url": "https://mastodon.social/@alice/789", "created_at": "2026-02-24T10:00:00Z"},
			})
		}
	}))
	defer mastoBase.Close()

	bskyTrending := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"topics": []any{}})
	}))
	defer bskyTrending.Close()
	mastoTags := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{})
	}))
	defer mastoTags.Close()
	mastoLinks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{})
	}))
	defer mastoLinks.Close()

	f := services.NewSocialFetcherWithEndpoints(http.DefaultClient, services.SocialEndpoints{
		BlueskyTrending:   bskyTrending.URL,
		BlueskySearch:     bskyTrending.URL,
		BlueskyAuthorFeed: bskyAuthorFeed.URL,
		MastodonTags:      mastoTags.URL,
		MastodonLinks:     mastoLinks.URL,
		MastodonBase:      mastoBase.URL,
	})

	src := upal.CollectSource{
		ID:       "social-feed",
		Type:     "social",
		Accounts: []string{"bob.bsky.social", "alice@mastodon.social"},
		Limit:    10,
	}

	text, data, err := f.Fetch(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "bob.bsky.social") {
		t.Error("expected Bluesky account feed in output")
	}
	if !strings.Contains(text, "alice@mastodon.social") {
		t.Error("expected Mastodon account feed in output")
	}

	items, ok := data.([]map[string]any)
	if !ok {
		t.Fatalf("expected data to be []map[string]any, got %T", data)
	}
	if len(items) == 0 {
		t.Error("expected at least one item from account feeds")
	}
}

func TestSocialFetcher_NoKeywords(t *testing.T) {
	// With no keywords, should still return trending topics
	bskyTrending := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"topics": []map[string]any{
				{"topic": "Trending1", "link": "https://bsky.app/search?q=Trending1"},
			},
		})
	}))
	defer bskyTrending.Close()

	bskySearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"posts": []any{}})
	}))
	defer bskySearch.Close()

	mastoTags := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "trending", "url": "https://mastodon.social/tags/trending"},
		})
	}))
	defer mastoTags.Close()

	mastoLinks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{})
	}))
	defer mastoLinks.Close()

	f := services.NewSocialFetcherWithEndpoints(http.DefaultClient, services.SocialEndpoints{
		BlueskyTrending:   bskyTrending.URL,
		BlueskySearch:     bskySearch.URL,
		BlueskyAuthorFeed: bskySearch.URL,
		MastodonTags:      mastoTags.URL,
		MastodonLinks:     mastoLinks.URL,
		MastodonBase:      mastoTags.URL,
	})

	src := upal.CollectSource{ID: "social-2", Type: "social", Limit: 10}
	text, _, err := f.Fetch(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Trending1") {
		t.Error("expected trending topics in output")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -v -race -run TestSocialFetcher`
Expected: FAIL — compilation error, socialFetcher doesn't exist yet

**Step 3: Implement socialFetcher**

Create `internal/services/fetcher_social.go`:

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/soochol/upal/internal/upal"
)

const (
	defaultBlueskyTrending   = "https://public.api.bsky.app/xrpc/app.bsky.unspecced.getTrendingTopics"
	defaultBlueskySearch     = "https://public.api.bsky.app/xrpc/app.bsky.feed.searchPosts"
	defaultBlueskyAuthorFeed = "https://public.api.bsky.app/xrpc/app.bsky.feed.getAuthorFeed"
	defaultMastodonTags      = "https://mastodon.social/api/v1/trends/tags"
	defaultMastodonLinks     = "https://mastodon.social/api/v1/trends/links"
	defaultMastodonBase      = "https://mastodon.social"
)

// SocialEndpoints holds configurable API URLs for testing.
type SocialEndpoints struct {
	BlueskyTrending   string
	BlueskySearch     string
	BlueskyAuthorFeed string
	MastodonTags      string
	MastodonLinks     string
	MastodonBase      string // base URL for account lookup + statuses
}

type socialFetcher struct {
	client    *http.Client
	endpoints SocialEndpoints
}

// NewSocialFetcher creates a socialFetcher with default production endpoints.
func NewSocialFetcher(client *http.Client) *socialFetcher {
	return &socialFetcher{
		client: client,
		endpoints: SocialEndpoints{
			BlueskyTrending:   defaultBlueskyTrending,
			BlueskySearch:     defaultBlueskySearch,
			BlueskyAuthorFeed: defaultBlueskyAuthorFeed,
			MastodonTags:      defaultMastodonTags,
			MastodonLinks:     defaultMastodonLinks,
			MastodonBase:      defaultMastodonBase,
		},
	}
}

// NewSocialFetcherWithEndpoints creates a socialFetcher with custom endpoints (for testing).
func NewSocialFetcherWithEndpoints(client *http.Client, ep SocialEndpoints) *socialFetcher {
	return &socialFetcher{client: client, endpoints: ep}
}

func (f *socialFetcher) Type() string { return "social" }

func (f *socialFetcher) Fetch(ctx context.Context, src upal.CollectSource) (string, any, error) {
	limit := src.Limit
	if limit <= 0 {
		limit = 20
	}

	var sb strings.Builder
	var items []map[string]any

	fmt.Fprintf(&sb, "=== Social Trends: %s ===\n\n", src.ID)

	// --- Bluesky Trending ---
	sb.WriteString("## Bluesky Trending Topics\n\n")
	bskyTopics, err := f.fetchBlueskyTrending(ctx)
	if err != nil {
		fmt.Fprintf(&sb, "[error] Bluesky trending: %v\n\n", err)
	} else {
		for _, t := range bskyTopics {
			fmt.Fprintf(&sb, "- %s\n", t["topic"])
			items = append(items, map[string]any{
				"title":        t["topic"],
				"url":          t["link"],
				"fetched_from": "bluesky_trending",
			})
		}
		sb.WriteString("\n")
	}

	// --- Bluesky keyword search ---
	for _, kw := range src.Keywords {
		posts, err := f.fetchBlueskySearch(ctx, kw, limit)
		if err != nil {
			fmt.Fprintf(&sb, "[error] Bluesky search %q: %v\n", kw, err)
			continue
		}
		if len(posts) > 0 {
			fmt.Fprintf(&sb, "### Bluesky: %q (%d posts)\n\n", kw, len(posts))
			for _, p := range posts {
				author, _ := p["author"].(string)
				text, _ := p["text"].(string)
				uri, _ := p["uri"].(string)
				if len(text) > 300 {
					text = text[:300] + "…"
				}
				fmt.Fprintf(&sb, "@%s: %s\n%s\n\n", author, text, uri)
				items = append(items, map[string]any{
					"title":        fmt.Sprintf("@%s: %s", author, truncate(text, 100)),
					"url":          uri,
					"content":      text,
					"fetched_from": "bluesky_search",
				})
			}
		}
	}

	// --- Account feeds ---
	// Route accounts: contains "@" → Mastodon, otherwise → Bluesky
	var bskyAccounts, mastoAccounts []string
	for _, acct := range src.Accounts {
		if strings.Contains(acct, "@") {
			mastoAccounts = append(mastoAccounts, acct)
		} else {
			bskyAccounts = append(bskyAccounts, acct)
		}
	}

	if len(bskyAccounts) > 0 {
		sb.WriteString("## Bluesky Account Feeds\n\n")
		for _, handle := range bskyAccounts {
			posts, err := f.fetchBlueskyAuthorFeed(ctx, handle, limit)
			if err != nil {
				fmt.Fprintf(&sb, "[error] Bluesky feed @%s: %v\n\n", handle, err)
				continue
			}
			fmt.Fprintf(&sb, "### @%s (%d posts)\n\n", handle, len(posts))
			for _, p := range posts {
				text, _ := p["text"].(string)
				uri, _ := p["uri"].(string)
				if len(text) > 300 {
					text = text[:300] + "…"
				}
				fmt.Fprintf(&sb, "%s\n%s\n\n", text, uri)
				items = append(items, map[string]any{
					"title":        fmt.Sprintf("@%s: %s", handle, truncate(text, 100)),
					"url":          uri,
					"content":      text,
					"fetched_from": "bluesky_feed",
				})
			}
		}
	}

	if len(mastoAccounts) > 0 {
		sb.WriteString("## Mastodon Account Feeds\n\n")
		for _, acct := range mastoAccounts {
			posts, err := f.fetchMastodonAccountStatuses(ctx, acct, limit)
			if err != nil {
				fmt.Fprintf(&sb, "[error] Mastodon feed %s: %v\n\n", acct, err)
				continue
			}
			fmt.Fprintf(&sb, "### %s (%d posts)\n\n", acct, len(posts))
			for _, p := range posts {
				content, _ := p["content"].(string)
				postURL, _ := p["url"].(string)
				if len(content) > 300 {
					content = content[:300] + "…"
				}
				fmt.Fprintf(&sb, "%s\n%s\n\n", content, postURL)
				items = append(items, map[string]any{
					"title":        fmt.Sprintf("%s: %s", acct, truncate(content, 100)),
					"url":          postURL,
					"content":      content,
					"fetched_from": "mastodon_feed",
				})
			}
		}
	}

	// --- Mastodon Trending ---
	sb.WriteString("## Mastodon Trending\n\n")
	mastoTags, err := f.fetchMastodonTags(ctx)
	if err != nil {
		fmt.Fprintf(&sb, "[error] Mastodon tags: %v\n\n", err)
	} else {
		sb.WriteString("### Trending Tags\n")
		for _, tag := range mastoTags {
			fmt.Fprintf(&sb, "- #%s\n", tag["name"])
			items = append(items, map[string]any{
				"title":        "#" + fmt.Sprint(tag["name"]),
				"url":          tag["url"],
				"fetched_from": "mastodon_tags",
			})
		}
		sb.WriteString("\n")
	}

	mastoLinks, err := f.fetchMastodonLinks(ctx)
	if err != nil {
		fmt.Fprintf(&sb, "[error] Mastodon links: %v\n\n", err)
	} else if len(mastoLinks) > 0 {
		sb.WriteString("### Trending Links\n")
		for _, link := range mastoLinks {
			title := fmt.Sprint(link["title"])
			linkURL := fmt.Sprint(link["url"])
			desc := fmt.Sprint(link["description"])
			if len(desc) > 200 {
				desc = desc[:200] + "…"
			}
			fmt.Fprintf(&sb, "- %s\n  %s\n  %s\n\n", title, linkURL, desc)
			items = append(items, map[string]any{
				"title":        title,
				"url":          linkURL,
				"content":      desc,
				"fetched_from": "mastodon_links",
			})
		}
	}

	// Apply limit
	if len(items) > limit {
		items = items[:limit]
	}

	return sb.String(), items, nil
}

func (f *socialFetcher) fetchJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}
	return json.Unmarshal(body, out)
}

func (f *socialFetcher) fetchBlueskyTrending(ctx context.Context) ([]map[string]string, error) {
	var resp struct {
		Topics []struct {
			Topic string `json:"topic"`
			Link  string `json:"link"`
		} `json:"topics"`
	}
	if err := f.fetchJSON(ctx, f.endpoints.BlueskyTrending, &resp); err != nil {
		return nil, err
	}
	var out []map[string]string
	for _, t := range resp.Topics {
		out = append(out, map[string]string{"topic": t.Topic, "link": t.Link})
	}
	return out, nil
}

func (f *socialFetcher) fetchBlueskySearch(ctx context.Context, query string, limit int) ([]map[string]any, error) {
	u := fmt.Sprintf("%s?q=%s&limit=%d", f.endpoints.BlueskySearch, url.QueryEscape(query), min(limit, 25))
	var resp struct {
		Posts []struct {
			URI    string `json:"uri"`
			Author struct {
				Handle string `json:"handle"`
			} `json:"author"`
			Record struct {
				Text string `json:"text"`
			} `json:"record"`
		} `json:"posts"`
	}
	if err := f.fetchJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, p := range resp.Posts {
		out = append(out, map[string]any{
			"uri":    p.URI,
			"author": p.Author.Handle,
			"text":   p.Record.Text,
		})
	}
	return out, nil
}

// fetchBlueskyAuthorFeed fetches recent posts from a Bluesky account.
// API: GET /xrpc/app.bsky.feed.getAuthorFeed?actor=handle&limit=N
func (f *socialFetcher) fetchBlueskyAuthorFeed(ctx context.Context, handle string, limit int) ([]map[string]any, error) {
	u := fmt.Sprintf("%s?actor=%s&limit=%d", f.endpoints.BlueskyAuthorFeed, url.QueryEscape(handle), min(limit, 30))
	var resp struct {
		Feed []struct {
			Post struct {
				URI    string `json:"uri"`
				Author struct {
					Handle string `json:"handle"`
				} `json:"author"`
				Record struct {
					Text string `json:"text"`
				} `json:"record"`
			} `json:"post"`
		} `json:"feed"`
	}
	if err := f.fetchJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, item := range resp.Feed {
		out = append(out, map[string]any{
			"uri":    item.Post.URI,
			"author": item.Post.Author.Handle,
			"text":   item.Post.Record.Text,
		})
	}
	return out, nil
}

// fetchMastodonAccountStatuses fetches recent posts from a Mastodon account.
// Format: "user@instance" — looks up account ID first, then fetches statuses.
func (f *socialFetcher) fetchMastodonAccountStatuses(ctx context.Context, acct string, limit int) ([]map[string]any, error) {
	// Step 1: lookup account ID
	lookupURL := fmt.Sprintf("%s/api/v1/accounts/lookup?acct=%s", f.endpoints.MastodonBase, url.QueryEscape(acct))
	var account struct {
		ID string `json:"id"`
	}
	if err := f.fetchJSON(ctx, lookupURL, &account); err != nil {
		return nil, fmt.Errorf("account lookup %s: %w", acct, err)
	}

	// Step 2: fetch statuses
	statusURL := fmt.Sprintf("%s/api/v1/accounts/%s/statuses?limit=%d&exclude_replies=true", f.endpoints.MastodonBase, account.ID, min(limit, 40))
	var statuses []struct {
		Content   string `json:"content"`
		URL       string `json:"url"`
		CreatedAt string `json:"created_at"`
	}
	if err := f.fetchJSON(ctx, statusURL, &statuses); err != nil {
		return nil, fmt.Errorf("statuses %s: %w", acct, err)
	}

	var out []map[string]any
	for _, s := range statuses {
		out = append(out, map[string]any{
			"content": s.Content,
			"url":     s.URL,
		})
	}
	return out, nil
}

func (f *socialFetcher) fetchMastodonTags(ctx context.Context) ([]map[string]any, error) {
	var tags []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := f.fetchJSON(ctx, f.endpoints.MastodonTags, &tags); err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, t := range tags {
		out = append(out, map[string]any{"name": t.Name, "url": t.URL})
	}
	return out, nil
}

func (f *socialFetcher) fetchMastodonLinks(ctx context.Context) ([]map[string]any, error) {
	var links []struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Description string `json:"description"`
	}
	if err := f.fetchJSON(ctx, f.endpoints.MastodonLinks, &links); err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, l := range links {
		out = append(out, map[string]any{"title": l.Title, "url": l.URL, "description": l.Description})
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -v -race -run TestSocialFetcher`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/fetcher_social.go internal/services/fetcher_social_test.go
git commit -m "feat: implement socialFetcher for Bluesky + Mastodon trends"
```

---

### Task 4: Backend — socialFetcher 등록 및 mapPipelineSources에 social 매핑

**Files:**
- Modify: `internal/services/stage_collect.go:32-41` (NewCollectStageExecutor)
- Modify: `internal/services/content_collector.go:842-844` (mapPipelineSources remaining twitter case)

**Step 1: Write test for social/twitter mapping**

Add to `internal/services/content_collector_test.go`:

```go
func TestMapPipelineSources_Social(t *testing.T) {
	sources := []upal.PipelineSource{
		{
			ID:       "soc1",
			Type:     "social",
			Keywords: []string{"AI", "startup"},
			Limit:    15,
		},
	}
	result := mapPipelineSources(sources, false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source, got %d", len(result))
	}
	cs := result[0].collectSource
	if cs.Type != "social" {
		t.Errorf("expected type=social, got %q", cs.Type)
	}
	if len(cs.Keywords) != 2 || cs.Keywords[0] != "AI" {
		t.Errorf("expected keywords=[AI startup], got %v", cs.Keywords)
	}
}

func TestMapPipelineSources_TwitterFallback(t *testing.T) {
	sources := []upal.PipelineSource{
		{ID: "tw1", Type: "twitter", Keywords: []string{"Go"}},
	}
	result := mapPipelineSources(sources, false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source (twitter fallback), got %d", len(result))
	}
	if result[0].collectSource.Type != "social" {
		t.Errorf("expected twitter to fallback to social, got %q", result[0].collectSource.Type)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -v -race -run TestMapPipelineSources_Social`
Expected: FAIL — "social" and "twitter" types are still skipped

**Step 3: Implement mapping and registration**

In `internal/services/content_collector.go`, add `case "social", "twitter"` after the google_trends case (where the old twitter skip was):

```go
case "social", "twitter":
	cs.Type = "social"
	cs.Keywords = ps.Keywords
	cs.Accounts = ps.Accounts
	cs.Limit = ps.Limit
```

In `internal/services/stage_collect.go`, in `NewCollectStageExecutor()`, add after the scrape fetcher registration:

```go
e.RegisterFetcher(NewSocialFetcher(client))
```

**Step 4: Run all tests**

Run: `go test ./internal/services/ -v -race -run "TestMapPipelineSources|TestSocialFetcher|TestCollect"`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/services/content_collector.go internal/services/content_collector_test.go internal/services/stage_collect.go
git commit -m "feat: register socialFetcher and map social/twitter sources"
```

---

### Task 5: Frontend — 타입 시스템 변경 (twitter → social)

**Files:**
- Modify: `web/src/shared/types/index.ts:222-228`

**Step 1: Update PipelineSourceType**

In `web/src/shared/types/index.ts`, replace:

```typescript
export type PipelineSourceType =
  | 'rss'
  | 'hn'
  | 'reddit'
  | 'google_trends'
  | 'twitter'
  | 'http'
```

With:

```typescript
export type PipelineSourceType =
  | 'rss'
  | 'hn'
  | 'reddit'
  | 'google_trends'
  | 'social'
  | 'http'
```

**Step 2: Add `accounts` and `geo` fields to PipelineSource**

In `web/src/shared/types/index.ts`, in `PipelineSource`, add after `keywords`:

```typescript
accounts?: string[]     // social: follow account handles (e.g. "alice.bsky.social", "user@mastodon.social")
geo?: string            // google_trends: country code
```

**Step 3: Run type check**

Run: `cd web && npx tsc -b --noEmit 2>&1 | head -30`
Expected: Type errors wherever `'twitter'` was used — this confirms what needs updating in the next tasks.

**Step 4: Commit**

```bash
git add web/src/shared/types/index.ts
git commit -m "feat: rename twitter to social in PipelineSourceType, add geo field"
```

---

### Task 6: Frontend — AddSourceModal 업데이트

**Files:**
- Modify: `web/src/features/configure-pipeline-sources/AddSourceModal.tsx`

**Step 1: Update SIGNAL_SOURCES**

Replace the twitter entry in `SIGNAL_SOURCES`:

```typescript
{ type: 'twitter', source_type: 'signal', label: 'X / Twitter', description: 'Trending keywords', icon: <Hash className="h-4 w-4" />, accent: 'bg-foreground/8', accentText: 'text-foreground' },
```

With:

```typescript
{ type: 'social', source_type: 'signal', label: 'Social Trends', description: 'Bluesky & Mastodon trends', icon: <TrendingUp className="h-4 w-4" />, accent: 'bg-primary/12', accentText: 'text-primary' },
```

**Step 2: Update the keywords input condition**

Replace:

```typescript
{(draft.type === 'google_trends' || draft.type === 'twitter') && (
```

With:

```typescript
{(draft.type === 'google_trends' || draft.type === 'social') && (
```

**Step 3: Add accounts input for social type**

After the keywords block, add an accounts input for the social source type. Uses the same `KeywordTagInput` component (handles are just strings like keywords):

```typescript
{draft.type === 'social' && (
  <div>
    <label className="block text-xs font-medium text-muted-foreground mb-1.5">Follow accounts</label>
    <KeywordTagInput
      keywords={draft.accounts ?? []}
      onChange={(accts) => setDraft({ ...draft, accounts: accts })}
      placeholder="alice.bsky.social, user@mastodon.social"
    />
    <p className="text-[10px] text-muted-foreground mt-1">
      Bluesky: handle (e.g. alice.bsky.social) · Mastodon: user@instance
    </p>
  </div>
)}
```

Also update `handleSelectType` to initialize `accounts`:

```typescript
setDraft({
  id: generateId(),
  type: typeDef.type,
  source_type: typeDef.source_type,
  label: typeDef.label,
  limit: 20,
  keywords: [],
  accounts: [],
})
```

**Step 4: Add geo dropdown for google_trends**

After the keywords block for google_trends, add a geo selector. Insert this right after the keywords `KeywordTagInput` block, before the min_score block:

```typescript
{draft.type === 'google_trends' && (
  <div>
    <label className="block text-xs font-medium text-muted-foreground mb-1.5">Region</label>
    <select
      value={draft.geo ?? 'US'}
      onChange={(e) => setDraft({ ...draft, geo: e.target.value })}
      className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
    >
      <option value="US">United States</option>
      <option value="KR">South Korea</option>
      <option value="JP">Japan</option>
      <option value="GB">United Kingdom</option>
      <option value="DE">Germany</option>
      <option value="FR">France</option>
      <option value="IN">India</option>
      <option value="BR">Brazil</option>
      <option value="CA">Canada</option>
      <option value="AU">Australia</option>
    </select>
  </div>
)}
```

**Step 5: Run type check**

Run: `cd web && npx tsc -b --noEmit 2>&1 | head -20`
Expected: Fewer type errors (or clean if this was the last reference to 'twitter')

**Step 6: Commit**

```bash
git add web/src/features/configure-pipeline-sources/AddSourceModal.tsx
git commit -m "feat: update AddSourceModal with Social Trends and Google Trends geo"
```

---

### Task 7: Frontend — 나머지 twitter 참조 정리

**Files:** Any remaining files referencing `'twitter'` as a PipelineSourceType

**Step 1: Find all remaining 'twitter' references related to PipelineSourceType**

Run: `cd web && npx tsc -b --noEmit 2>&1` to find all type errors.

Also grep: search for `'twitter'` or `"twitter"` or `type === 'twitter'` in `web/src/`.

**Step 2: Fix each reference**

For each file with a `'twitter'` reference in the context of PipelineSourceType:
- Replace `'twitter'` → `'social'`
- If it's a display label, update to "Social Trends"

Common locations to check:
- `web/src/shared/ui/SourceTypeBadge.tsx` — if it references specific source types
- Any pipeline list/detail views that render source type labels
- Store files that filter by source type

**Step 3: Run type check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS — clean build

**Step 4: Commit**

```bash
git add -u web/src/
git commit -m "fix: update remaining twitter references to social"
```

---

### Task 8: Backend — PipelineSource.Type 필드 업데이트

**Files:**
- Modify: `internal/upal/pipeline.go:99`

**Step 1: Update the Type field comment**

In `internal/upal/pipeline.go`, update the comment on the Type field:

```go
Type      string   `json:"type,omitempty"`      // "rss" | "hn" | "reddit" | "google_trends" | "social" | "http"
```

**Step 2: Run all backend tests**

Run: `go test ./internal/... -v -race -count=1 2>&1 | tail -30`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/upal/pipeline.go
git commit -m "docs: update PipelineSource.Type comment to reflect social type"
```

---

### Task 9: End-to-end verification

**Step 1: Run full backend tests**

Run: `go test ./... -v -race -count=1 2>&1 | tail -40`
Expected: All PASS

**Step 2: Run frontend type check + build**

Run: `cd web && npx tsc -b --noEmit && npm run build`
Expected: Clean build

**Step 3: Manual smoke test**

1. Start dev servers: `make dev-backend` + `make dev-frontend`
2. Create a pipeline
3. Add a "Google Trends" source with geo=KR, keywords=["AI"]
4. Add a "Social Trends" source with keywords=["AI"], accounts=["bsky.app"] (a real Bluesky handle)
5. Click "Collect" and verify both sources return data
6. Check the session detail — source fetches should show items from:
   - Google Trends RSS (trending keywords)
   - Bluesky trending topics + keyword search + account feed
   - Mastodon trending tags + links

**Step 4: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix: address issues found during e2e verification"
```
