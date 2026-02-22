---
name: tool-fetch_rss
description: Parse RSS/Atom/JSON feeds into structured items — no LLM token cost
---

## Overview

Fetches and parses RSS, Atom, or JSON Feed URLs into structured data. Pure code parsing — no LLM processing, no token cost. Use for news aggregation, blog monitoring, and content pipelines that need to check feeds on a schedule.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | URL of the RSS/Atom/JSON feed |
| `max_items` | number | No | Maximum items to return (default: all items in feed) |
| `since_date` | string | No | Only return items published after this date. ISO 8601 / RFC 3339 format: `"2026-02-20T00:00:00Z"`. Items with no parseable date are excluded when this is set. |

## Returns

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | Filtered and truncated item list |
| `items[].title` | string | Item headline |
| `items[].link` | string | URL of the full article |
| `items[].published` | string | ISO 8601 date string, or empty if unavailable |
| `items[].summary` | string | Feed-provided excerpt/description (often truncated) |
| `items[].author` | string | Author name, or empty if unavailable |
| `feed_title` | string | Name of the feed |
| `feed_url` | string | Feed's declared home URL |
| `item_count` | number | Number of items returned after filtering |

## Agent Prompt Patterns

**Basic fetch and summarize:**
```
Fetch the RSS feed at {{feed_url}} with max_items=20.
Summarize the 5 most relevant items for [topic], including title and link for each.
```

**Incremental collection with content_store:**
```
1. Get the last run timestamp: content_store action="get", key="{{pipeline_name}}:last-run"
2. If found=false, use 7 days ago (ISO 8601) as since_date.
3. Fetch the feed at {{feed_url}} with since_date = the stored value.
4. Process new items.
5. Update the timestamp: content_store action="set", key="{{pipeline_name}}:last-run", value=[current UTC time as ISO 8601]
```

**Deduplication with content_store:**
```
1. Fetch the feed at {{feed_url}} with max_items=50.
2. Get already-seen links: content_store action="get", key="{{pipeline_name}}:seen-links"
   Parse value as JSON array (use [] if not found).
3. Filter items: only those whose link is NOT in the seen list.
4. Process new items only.
5. Update seen list: content_store action="set", key="{{pipeline_name}}:seen-links",
   value=[JSON string of all links including newly processed ones]
```

**Multi-source aggregation:**
```
Fetch all three feeds:
- {{feed_url_1}}
- {{feed_url_2}}
- {{feed_url_3}}

Combine all items. Deduplicate by link. Sort by published date, newest first.
Summarize the top 10 most recent items with title, source feed, and one-sentence summary.
```

**Deep reading (fetch_rss + get_webpage):**
```
1. Fetch the feed at {{feed_url}} with max_items=5 and since_date={{since_date}}.
2. For each item, use get_webpage on item.link to read the full article.
3. Generate a structured digest with title, key points, and takeaway for each article.
```

## Pitfalls & Limitations

- **`summary` is usually truncated** — feed summaries are excerpts, not full articles. Use `get_webpage` on `item.link` for full content.
- **`since_date` must be RFC 3339** — use `"2026-02-20T00:00:00Z"` format. Human-readable dates like `"Feb 20"` will cause an error.
- **Items without dates are excluded** when `since_date` is set — some feeds omit `<pubDate>`. If missing items are a concern, omit `since_date` and handle deduplication via `content_store` instead.
- **Feed format quirks** — most RSS/Atom feeds work. Exotic or malformed feeds may fail parsing.
- **`summary` may contain HTML** — the tool does not strip HTML from feed summaries. Instruct the LLM to ignore any HTML tags in the summary field.
- **30-second timeout** — slow or unreachable feed servers will fail.
- **No authentication** — private/authenticated feeds are not supported.
