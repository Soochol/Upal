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

// SocialEndpoints holds API endpoints for Bluesky and Mastodon.
// Exported so tests can inject httptest servers.
type SocialEndpoints struct {
	BlueskyTrending   string
	BlueskySearch     string
	BlueskyAuthorFeed string
	MastodonTags      string
	MastodonLinks     string
	MastodonBase      string
}

// socialFetcher fetches trending topics, keyword searches, and account feeds
// from Bluesky and Mastodon public APIs.
type socialFetcher struct {
	client    *http.Client
	endpoints SocialEndpoints
}

// NewSocialFetcher creates a socialFetcher with production API endpoints.
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
func NewSocialFetcherWithEndpoints(client *http.Client, endpoints SocialEndpoints) *socialFetcher {
	return &socialFetcher{
		client:    client,
		endpoints: endpoints,
	}
}

func (f *socialFetcher) Type() string { return "social" }

func (f *socialFetcher) Fetch(ctx context.Context, src upal.CollectSource) (string, any, error) {
	limit := src.Limit
	if limit <= 0 {
		limit = 20
	}
	// Per-category limit to avoid early sources starving later ones.
	perCategoryLimit := max(limit/3, 5)

	var sb strings.Builder
	var allItems []map[string]any

	// 1. Bluesky trending topics
	f.fetchBlueskyTrending(ctx, &sb, &allItems)

	// 2. Bluesky keyword search
	for _, kw := range src.Keywords {
		f.fetchBlueskySearch(ctx, kw, &sb, &allItems)
	}

	// 3. Account feeds
	// Routing: handles containing "@" (e.g. user@mastodon.social) → Mastodon,
	// handles without "@" (e.g. alice.bsky.social) → Bluesky.
	for _, acct := range src.Accounts {
		if strings.Contains(acct, "@") {
			f.fetchMastodonAccount(ctx, acct, perCategoryLimit, &sb, &allItems)
		} else {
			f.fetchBlueskyAuthorFeed(ctx, acct, perCategoryLimit, &sb, &allItems)
		}
	}

	// 4. Mastodon trending tags
	f.fetchMastodonTags(ctx, &sb, &allItems)

	// 5. Mastodon trending links
	f.fetchMastodonLinks(ctx, &sb, &allItems)

	// Apply limit to total items
	if len(allItems) > limit {
		allItems = allItems[:limit]
	}

	return sb.String(), allItems, nil
}

// fetchBlueskyTrending fetches trending topics from Bluesky.
func (f *socialFetcher) fetchBlueskyTrending(ctx context.Context, sb *strings.Builder, items *[]map[string]any) {
	var result struct {
		Topics []struct {
			Topic string `json:"topic"`
			Link  string `json:"link"`
		} `json:"topics"`
	}
	if err := f.fetchJSON(ctx, f.endpoints.BlueskyTrending, &result); err != nil {
		fmt.Fprintf(sb, "=== Bluesky Trending ===\n[error] %v\n\n", err)
		return
	}

	fmt.Fprintf(sb, "=== Bluesky Trending ===\n")
	for _, t := range result.Topics {
		fmt.Fprintf(sb, "- %s (%s)\n", t.Topic, t.Link)
		*items = append(*items, map[string]any{
			"title":        t.Topic,
			"url":          t.Link,
			"fetched_from": "bluesky_trending",
		})
	}
	sb.WriteString("\n")
}

// fetchBlueskySearch searches Bluesky posts by keyword.
func (f *socialFetcher) fetchBlueskySearch(ctx context.Context, keyword string, sb *strings.Builder, items *[]map[string]any) {
	u := f.endpoints.BlueskySearch + "?q=" + url.QueryEscape(keyword)
	var result struct {
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
	if err := f.fetchJSON(ctx, u, &result); err != nil {
		fmt.Fprintf(sb, "=== Bluesky Search: %s ===\n[error] %v\n\n", keyword, err)
		return
	}

	fmt.Fprintf(sb, "=== Bluesky Search: %s ===\n", keyword)
	for _, p := range result.Posts {
		fmt.Fprintf(sb, "- @%s: %s\n", p.Author.Handle, truncate(p.Record.Text, 200))
		*items = append(*items, map[string]any{
			"title":        fmt.Sprintf("@%s on Bluesky", p.Author.Handle),
			"url":          p.URI,
			"content":      p.Record.Text,
			"fetched_from": "bluesky_search",
		})
	}
	sb.WriteString("\n")
}

// fetchBlueskyAuthorFeed fetches recent posts from a Bluesky account.
func (f *socialFetcher) fetchBlueskyAuthorFeed(ctx context.Context, handle string, limit int, sb *strings.Builder, items *[]map[string]any) {
	u := fmt.Sprintf("%s?actor=%s&limit=%d", f.endpoints.BlueskyAuthorFeed, url.QueryEscape(handle), limit)
	var result struct {
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
	if err := f.fetchJSON(ctx, u, &result); err != nil {
		fmt.Fprintf(sb, "=== Bluesky Feed: %s ===\n[error] %v\n\n", handle, err)
		return
	}

	fmt.Fprintf(sb, "=== Bluesky Feed: %s ===\n", handle)
	for _, entry := range result.Feed {
		p := entry.Post
		fmt.Fprintf(sb, "- @%s: %s\n", p.Author.Handle, truncate(p.Record.Text, 200))
		*items = append(*items, map[string]any{
			"title":        fmt.Sprintf("@%s on Bluesky", p.Author.Handle),
			"url":          p.URI,
			"content":      p.Record.Text,
			"fetched_from": "bluesky_account",
		})
	}
	sb.WriteString("\n")
}

// fetchMastodonAccount looks up a Mastodon account and fetches recent statuses.
func (f *socialFetcher) fetchMastodonAccount(ctx context.Context, acct string, limit int, sb *strings.Builder, items *[]map[string]any) {
	// Step 1: Look up account ID
	lookupURL := fmt.Sprintf("%s/api/v1/accounts/lookup?acct=%s", f.endpoints.MastodonBase, url.QueryEscape(acct))
	var account struct {
		ID string `json:"id"`
	}
	if err := f.fetchJSON(ctx, lookupURL, &account); err != nil {
		fmt.Fprintf(sb, "=== Mastodon Feed: %s ===\n[error] lookup: %v\n\n", acct, err)
		return
	}

	// Step 2: Fetch statuses
	statusesURL := fmt.Sprintf("%s/api/v1/accounts/%s/statuses?limit=%d", f.endpoints.MastodonBase, account.ID, limit)
	var statuses []struct {
		Content   string `json:"content"`
		URL       string `json:"url"`
		CreatedAt string `json:"created_at"`
	}
	if err := f.fetchJSON(ctx, statusesURL, &statuses); err != nil {
		fmt.Fprintf(sb, "=== Mastodon Feed: %s ===\n[error] statuses: %v\n\n", acct, err)
		return
	}

	fmt.Fprintf(sb, "=== Mastodon Feed: %s ===\n", acct)
	for _, s := range statuses {
		text := stripHTMLTags(s.Content)
		fmt.Fprintf(sb, "- [%s] %s\n  %s\n", s.CreatedAt, truncate(text, 200), s.URL)
		*items = append(*items, map[string]any{
			"title":        fmt.Sprintf("%s on Mastodon", acct),
			"url":          s.URL,
			"content":      text,
			"fetched_from": "mastodon_account",
		})
	}
	sb.WriteString("\n")
}

// fetchMastodonTags fetches trending tags from Mastodon.
func (f *socialFetcher) fetchMastodonTags(ctx context.Context, sb *strings.Builder, items *[]map[string]any) {
	var tags []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := f.fetchJSON(ctx, f.endpoints.MastodonTags, &tags); err != nil {
		fmt.Fprintf(sb, "=== Mastodon Trending Tags ===\n[error] %v\n\n", err)
		return
	}

	fmt.Fprintf(sb, "=== Mastodon Trending Tags ===\n")
	for _, tag := range tags {
		fmt.Fprintf(sb, "- #%s (%s)\n", tag.Name, tag.URL)
		*items = append(*items, map[string]any{
			"title":        "#" + tag.Name,
			"url":          tag.URL,
			"fetched_from": "mastodon_trending_tag",
		})
	}
	sb.WriteString("\n")
}

// fetchMastodonLinks fetches trending links from Mastodon.
func (f *socialFetcher) fetchMastodonLinks(ctx context.Context, sb *strings.Builder, items *[]map[string]any) {
	var links []struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Description string `json:"description"`
	}
	if err := f.fetchJSON(ctx, f.endpoints.MastodonLinks, &links); err != nil {
		fmt.Fprintf(sb, "=== Mastodon Trending Links ===\n[error] %v\n\n", err)
		return
	}

	fmt.Fprintf(sb, "=== Mastodon Trending Links ===\n")
	for _, link := range links {
		fmt.Fprintf(sb, "- %s\n  %s\n  %s\n", link.Title, link.URL, truncate(link.Description, 200))
		*items = append(*items, map[string]any{
			"title":        link.Title,
			"url":          link.URL,
			"content":      link.Description,
			"fetched_from": "mastodon_trending_link",
		})
	}
	sb.WriteString("\n")
}

// fetchJSON performs a GET request and unmarshals the JSON response into out.
func (f *socialFetcher) fetchJSON(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}

// truncate cuts a string to n characters, appending "…" if truncated.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// stripHTMLTags removes HTML tags from a string (simple approach for Mastodon content).
func stripHTMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}
