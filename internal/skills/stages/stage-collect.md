---
name: stage-collect
description: Guide for configuring collect stages — code-based data fetching from external sources
---

## "collect" stage — code-based data fetching (no LLM)

```json
"config": {
  "sources": [
    { "id": "src1", "type": "rss",            "url": "...", "limit": 20 },
    { "id": "src2", "type": "hn",             "min_score": 100, "limit": 20 },
    { "id": "src3", "type": "http",           "url": "...", "method": "GET", "headers": {}, "body": "" },
    { "id": "src4", "type": "reddit",         "subreddit": "MachineLearning", "limit": 20 },
    { "id": "src5", "type": "google_trends",  "keywords": ["AI"], "geo": "US", "limit": 20 },
    { "id": "src6", "type": "social",         "keywords": ["AI", "startup"], "accounts": ["alice.bsky.social", "user@mastodon.social"], "limit": 20 },
    { "id": "src7", "type": "scrape",         "url": "...", "selector": ".article-title", "scrape_limit": 30 }
  ]
}
```

### Source type fields

- **rss**: `url` (feed URL, required), `limit` (int, max items, default 20)
- **hn**: Hacker News top stories. `min_score` (int, optional — only fetch posts with ≥ this many points), `limit` (int, default 20). No `url` needed — built-in via hnrss.org.
- **http**: `url` (required), `method` ("GET"|"POST", default "GET"), `headers` (object, optional), `body` (string, optional)
- **reddit**: Subreddit hot posts. `subreddit` (string, e.g. "MachineLearning"; defaults to "all"), `limit` (int, default 20). No `url` needed — built-in via Reddit RSS.
- **google_trends**: Trending search topics. `keywords` (string[], terms to monitor), `geo` (string, ISO country code e.g. "US", "KR", "JP"; default "US"), `limit` (int, default 20). No `url` needed — built-in via Google Trends RSS.
- **social**: `keywords` (string[], search terms for Bluesky), `accounts` (string[], Bluesky handles like `alice.bsky.social` or Mastodon handles like `user@mastodon.social`), `limit` (int, max items, default 20). Fetches trending topics + keyword search from Bluesky, trending tags + links from Mastodon, and recent posts from specified accounts.
- **scrape**: `url` (required), `selector` (CSS selector, required), `attribute` (HTML attribute to extract; omit for text content), `scrape_limit` (int, max elements, default 30)
- **research**: `topic` (required), `depth` ("light" | "deep", default "light"). LLM-powered topic research using web search. Light mode does a single search pass; deep mode runs an iterative agent loop. Requires a model that supports native web_search tool (not available on Ollama).

All sources require: `id` (unique string), `type`. Sources that need a URL: `rss`, `http`, `scrape`. Built-in sources (no URL): `hn`, `reddit`, `google_trends`, `social`.

### Output fields available to downstream stages

| Field | Contents |
|-------|---------|
| `{{text}}` | All sources concatenated as plain text |
| `{{sources}}` | Structured data keyed by source id |

### Rules

- Each source `id` must be unique within the stage.
- Use `rss` for RSS/Atom feeds, `hn` for Hacker News, `reddit` for subreddit posts, `http` for REST APIs or webhooks, `google_trends` for search trend monitoring, `social` for Bluesky & Mastodon trends, `scrape` for HTML pages with CSS selector extraction.
- Prefer `rss` when a feed is available — more reliable than scraping.
- `hn`, `reddit`, `google_trends`, `social` are built-in — they do not need a `url` field.
