---
name: stage-collect
description: Guide for configuring collect stages — code-based data fetching from external sources
---

## "collect" stage — code-based data fetching (no LLM)

```json
"config": {
  "sources": [
    { "id": "src1", "type": "rss",    "url": "...", "limit": 20 },
    { "id": "src2", "type": "http",   "url": "...", "method": "GET", "headers": {}, "body": "" },
    { "id": "src3", "type": "scrape", "url": "...", "selector": "css", "attribute": "", "scrape_limit": 30 },
    { "id": "src4", "type": "social", "keywords": ["AI", "startup"], "accounts": ["alice.bsky.social", "user@mastodon.social"], "limit": 20 }
  ]
}
```

### Source type fields

- **rss**: `limit` (int, max items to fetch, default 20)
- **http**: `method` ("GET"|"POST", default "GET"), `headers` (object, optional), `body` (string, optional)
- **scrape**: `selector` (CSS selector, required), `attribute` (HTML attribute to extract; omit for text content), `scrape_limit` (int, max elements, default 30)
- **social**: `keywords` (string[], search terms for Bluesky), `accounts` (string[], Bluesky handles like `alice.bsky.social` or Mastodon handles like `user@mastodon.social`), `limit` (int, max items, default 20). Fetches trending topics + keyword search from Bluesky, trending tags + links from Mastodon, and recent posts from specified accounts.

All sources require: `id` (unique string), `type`. Most require `url`, except `social` (endpoints are built-in).

### Output fields available to downstream stages

| Field | Contents |
|-------|---------|
| `{{text}}` | All sources concatenated as plain text |
| `{{sources}}` | Structured data keyed by source id |

### Rules

- Each source `id` must be unique within the stage.
- Use `rss` for RSS/Atom feeds, `http` for REST APIs or webhooks, `scrape` for HTML pages, `social` for Bluesky & Mastodon trends.
- Prefer `rss` when a feed is available — more reliable than scraping.
- `scrape` requires a valid CSS selector; test it against the target page structure.
- `social` does not need a `url` — it calls Bluesky and Mastodon public APIs automatically. Keywords trigger Bluesky search; accounts fetch recent posts from specified handles.
