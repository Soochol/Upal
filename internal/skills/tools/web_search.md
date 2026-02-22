---
name: tool-web_search
description: Native web search — provider-handled, real-time results
---

## Overview

`web_search` is a **native tool** executed by the LLM provider server-side (e.g. Anthropic, Google). Upal declares it as an available tool; the provider handles the search and injects results into the conversation. No Upal backend call is made.

**Use when**: Current events, recent data not in training, factual lookups requiring up-to-date information.

**Do not use when**: You already have a specific URL to read (use `get_webpage`), you need structured API data (use `http_request`), or the model is Ollama/self-hosted (native tools unsupported).

## Provider Compatibility

| Provider | Notes |
|----------|-------|
| Anthropic | Google Search via Anthropic API |
| Google Gemini | Native Google Search grounding |
| Ollama / self-hosted | Not supported — do not enable |

## Agent Prompt Patterns

**Focused lookup:**
```
Search for [specific query]. Report the key facts: [fact 1], [fact 2], [fact 3].
```

**Multi-angle research:**
```
Research [topic] using web search. Run at least 3 searches covering:
1. [angle 1]
2. [angle 2]
3. [angle 3]
Synthesize findings into a structured report.
```

**Recency-aware:**
```
Search for news about [topic] from the past 7 days.
Identify the 3 most significant developments and explain their impact.
```

**Deep reading (combine with get_webpage):**
```
1. Search for recent articles about [topic].
2. For the 2–3 most relevant results, use get_webpage to read the full content.
3. Synthesize all sources into a comprehensive analysis.
```

**Fact verification:**
```
Search to verify the following claim: "[claim]".
Cite at least 2 independent sources. If sources conflict, note the discrepancy.
```

## Pitfalls & Limitations

- **Snippets only by default** — results are short excerpts. Use `get_webpage` to read full articles.
- **Provider-specific** — not available on models that don't support native tools.
- **Avoid redundant queries** — if the answer is clear from the first result, stop searching.
- **Query precision matters** — vague queries return poor results. Use specific, targeted search terms.
- **No URL control** — you cannot specify which sites to search. Use `get_webpage` for known URLs.
