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
	if !strings.Contains(text, "Bluesky") {
		t.Error("expected Bluesky section")
	}
	if !strings.Contains(text, "Mastodon") {
		t.Error("expected Mastodon section")
	}
	items, ok := data.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", data)
	}
	if len(items) == 0 {
		t.Error("expected items")
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
		t.Error("expected Bluesky account feed")
	}
	if !strings.Contains(text, "alice@mastodon.social") {
		t.Error("expected Mastodon account feed")
	}
	items, ok := data.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", data)
	}
	if len(items) == 0 {
		t.Error("expected items from account feeds")
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
	text, _, err := f.Fetch(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Trending1") {
		t.Error("expected trending topics")
	}
}
