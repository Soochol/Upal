---
name: tool-get_webpage
description: Fetch readable text from a URL — HTML stripped, up to 100KB
---

## Overview

Fetches a URL and returns clean, human-readable text with all HTML stripped. Use this when you have a specific URL and need to read its full content — article text, documentation, product pages, etc.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | Full URL including scheme (e.g. `"https://example.com/article"`) |

## Returns

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | HTML `<title>` of the page |
| `text` | string | Extracted readable text, max 100KB. Appends `... [truncated at 100KB]` if cut. |
| `url` | string | The URL that was fetched |

## Agent Prompt Patterns

**Read a known URL:**
```
Fetch the content from {{url_input}} using get_webpage.
Extract the main argument and list the key supporting points.
```

**Follow up on search results:**
```
1. Search for [topic] and identify the 3 most relevant result URLs.
2. Use get_webpage to fetch each URL.
3. Synthesize the full content into a structured report.
```

**Documentation extraction:**
```
Fetch the documentation at [url].
Extract: (1) authentication method, (2) rate limits, (3) available endpoints.
Format as a JSON object.
```

**Comparison across pages:**
```
Fetch both pages:
- {{url_a}}
- {{url_b}}
Compare their approaches to [specific topic] and highlight the key differences.
```

**Read and summarize with length awareness:**
```
Fetch {{article_url}} using get_webpage.
If the text ends with "[truncated at 100KB]", note that the article was cut off.
Summarize what was retrieved, covering the main argument and key points.
```

## Pitfalls & Limitations

- **JavaScript-rendered content** — SPAs and dynamic pages may return incomplete text. The tool fetches raw HTML before JS execution. Use `http_request` against an API endpoint if the site offers one.
- **100KB text cap** — very long pages are truncated. Check for the `[truncated at 100KB]` marker.
- **Login-gated pages** — returns the login page, not the protected content. No authentication support.
- **PDFs and binary files** — URLs pointing to PDFs, downloads, or non-HTML resources will fail or return garbled output. Use an asset node for known files.
- **30-second timeout** — slow sites will fail. If a site consistently times out, use `web_search` for its cached snippets instead.
- **No JavaScript execution** — pages that require user interaction to reveal content will not work.
