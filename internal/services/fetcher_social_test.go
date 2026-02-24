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
	bskyTrending := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"topics": []map[string]any{
				{"topic": "AI", "link": "https://bsky.app/search?q=AI"},
			},
		})
	}))
	defer bskyTrending.Close()

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

	mastoTags := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "ai", "url": "https://mastodon.social/tags/ai"},
		})
	}))
	defer mastoTags.Close()

	mastoLinks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"title": "AI News", "url": "https://example.com/ai", "description": "Latest AI"},
		})
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

	src := upal.CollectSource{ID: "social-1", Type: "social", Keywords: []string{"AI"}, Limit: 10}
	text, data, err := f.Fetch(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Bluesky Trending") {
		t.Error("expected Bluesky Trending section in text")
	}
	if !strings.Contains(text, "Mastodon Trending") {
		t.Error("expected Mastodon Trending section in text")
	}

	items, ok := data.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", data)
	}
	if len(items) < 3 {
		t.Fatalf("expected at least 3 items (trending + search + tags), got %d", len(items))
	}

	// Verify Bluesky trending item fields
	found := false
	for _, item := range items {
		if item["fetched_from"] == "bluesky_trending" && item["title"] == "AI" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected item with fetched_from=bluesky_trending and title=AI")
	}

	// Verify Bluesky search item fields
	found = false
	for _, item := range items {
		if item["fetched_from"] == "bluesky_search" {
			if item["url"] != "at://did:plc:abc/app.bsky.feed.post/123" {
				t.Errorf("expected search item url, got %q", item["url"])
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected item with fetched_from=bluesky_search")
	}

	// Verify Mastodon trending tag item
	found = false
	for _, item := range items {
		if item["fetched_from"] == "mastodon_trending_tag" && item["title"] == "#ai" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected item with fetched_from=mastodon_trending_tag and title=#ai")
	}
}

func TestSocialFetcher_AccountFeeds(t *testing.T) {
	bskyAuthorFeed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"feed": []map[string]any{
				{"post": map[string]any{
					"uri":    "at://did:plc:xyz/app.bsky.feed.post/456",
					"author": map[string]any{"handle": "bob.bsky.social"},
					"record": map[string]any{"text": "Check out my new project"},
				}},
			},
		})
	}))
	defer bskyAuthorFeed.Close()

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

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"topics": []any{}})
	}))
	defer empty.Close()
	emptyArr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{})
	}))
	defer emptyArr.Close()

	f := services.NewSocialFetcherWithEndpoints(http.DefaultClient, services.SocialEndpoints{
		BlueskyTrending: empty.URL, BlueskySearch: empty.URL,
		BlueskyAuthorFeed: bskyAuthorFeed.URL,
		MastodonTags: emptyArr.URL, MastodonLinks: emptyArr.URL,
		MastodonBase: mastoBase.URL,
	})

	src := upal.CollectSource{ID: "social-feed", Type: "social", Accounts: []string{"bob.bsky.social", "alice@mastodon.social"}, Limit: 10}
	text, data, err := f.Fetch(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "bob.bsky.social") {
		t.Error("expected Bluesky account feed in text")
	}
	if !strings.Contains(text, "alice@mastodon.social") {
		t.Error("expected Mastodon account feed in text")
	}

	items, ok := data.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", data)
	}
	if len(items) < 2 {
		t.Fatalf("expected at least 2 items (1 bsky + 1 mastodon), got %d", len(items))
	}

	// Verify Bluesky account feed item
	found := false
	for _, item := range items {
		if item["fetched_from"] == "bluesky_account" {
			if item["content"] != "Check out my new project" {
				t.Errorf("expected bsky content, got %q", item["content"])
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected item with fetched_from=bluesky_account")
	}

	// Verify Mastodon account feed item — HTML should be stripped
	found = false
	for _, item := range items {
		if item["fetched_from"] == "mastodon_account" {
			content, _ := item["content"].(string)
			if strings.Contains(content, "<p>") {
				t.Error("expected HTML tags to be stripped from Mastodon content")
			}
			if !strings.Contains(content, "Hello from Mastodon") {
				t.Errorf("expected stripped text content, got %q", content)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected item with fetched_from=mastodon_account")
	}
}

func TestSocialFetcher_NoKeywords(t *testing.T) {
	bskyTrending := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"topics": []map[string]any{{"topic": "Trending1", "link": "https://bsky.app/search?q=Trending1"}},
		})
	}))
	defer bskyTrending.Close()
	bskySearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"posts": []any{}})
	}))
	defer bskySearch.Close()
	mastoTags := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{{"name": "trending", "url": "https://mastodon.social/tags/trending"}})
	}))
	defer mastoTags.Close()
	mastoLinks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{})
	}))
	defer mastoLinks.Close()

	f := services.NewSocialFetcherWithEndpoints(http.DefaultClient, services.SocialEndpoints{
		BlueskyTrending: bskyTrending.URL, BlueskySearch: bskySearch.URL,
		BlueskyAuthorFeed: bskySearch.URL,
		MastodonTags: mastoTags.URL, MastodonLinks: mastoLinks.URL,
		MastodonBase: mastoTags.URL,
	})

	src := upal.CollectSource{ID: "social-2", Type: "social", Limit: 10}
	text, data, err := f.Fetch(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Trending1") {
		t.Error("expected trending topics in text")
	}

	items, ok := data.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", data)
	}
	// Should have trending + mastodon tags = at least 2 items
	if len(items) < 2 {
		t.Errorf("expected at least 2 items (trending + tags), got %d", len(items))
	}
}

func TestSocialFetcher_APIError(t *testing.T) {
	// All endpoints return HTTP 500 — fetcher should still return text with error info, not crash
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer errServer.Close()

	f := services.NewSocialFetcherWithEndpoints(http.DefaultClient, services.SocialEndpoints{
		BlueskyTrending:   errServer.URL,
		BlueskySearch:     errServer.URL,
		BlueskyAuthorFeed: errServer.URL,
		MastodonTags:      errServer.URL,
		MastodonLinks:     errServer.URL,
		MastodonBase:      errServer.URL,
	})

	src := upal.CollectSource{ID: "err-test", Type: "social", Keywords: []string{"test"}, Limit: 10}
	text, data, err := f.Fetch(context.Background(), src)
	if err != nil {
		t.Fatalf("Fetch should not return error on API failures (partial failure design): %v", err)
	}
	if !strings.Contains(text, "[error]") {
		t.Error("expected error messages in text output")
	}
	items, ok := data.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", data)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items when all APIs fail, got %d", len(items))
	}
}

func TestSocialFetcher_LimitEnforced(t *testing.T) {
	// Return many items, verify limit caps the result
	bskyTrending := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		topics := make([]map[string]any, 20)
		for i := range topics {
			topics[i] = map[string]any{"topic": "topic", "link": "https://example.com"}
		}
		json.NewEncoder(w).Encode(map[string]any{"topics": topics})
	}))
	defer bskyTrending.Close()
	emptyJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{})
	}))
	defer emptyJSON.Close()
	emptyObj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"topics": []any{}, "posts": []any{}})
	}))
	defer emptyObj.Close()

	f := services.NewSocialFetcherWithEndpoints(http.DefaultClient, services.SocialEndpoints{
		BlueskyTrending: bskyTrending.URL, BlueskySearch: emptyObj.URL,
		BlueskyAuthorFeed: emptyObj.URL,
		MastodonTags: emptyJSON.URL, MastodonLinks: emptyJSON.URL,
		MastodonBase: emptyJSON.URL,
	})

	src := upal.CollectSource{ID: "limit-test", Type: "social", Limit: 5}
	_, data, err := f.Fetch(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	items, ok := data.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", data)
	}
	if len(items) > 5 {
		t.Errorf("expected at most 5 items (limit), got %d", len(items))
	}
}
