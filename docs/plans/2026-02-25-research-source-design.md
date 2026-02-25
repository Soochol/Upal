# Research Source Type — Design Document

## Problem

Pipeline sources currently support only fixed-endpoint fetching (RSS, HTTP, scrape, social). Users who need topic-based exploration — "research the EV market trends" — have no way to express this as a source. They must manually research and paste findings, or build a separate workflow with web_search tools.

## Solution

Add a `research` source type to the collect stage. It reuses the existing `SourceFetcher` interface and runs an LLM-powered search loop within the collect phase, returning results in the same format as other sources.

Two depth modes:

- **Light** — single search pass + summarize (seconds)
- **Deep** — agent loop with iterative search + evaluation (minutes)

## Data Model

### CollectSource (backend internal)

New fields added to existing struct:

```go
Topic       string `json:"topic,omitempty"`        // Research: subject to investigate
Depth       string `json:"depth,omitempty"`         // Research: "light" | "deep"
MaxSearches int    `json:"max_searches,omitempty"`  // Research: max search iterations for deep mode (default 10)
```

### PipelineSource (frontend ↔ API)

New fields added to existing struct:

```go
Topic string `json:"topic,omitempty"`  // research: subject to investigate
Depth string `json:"depth,omitempty"`  // research: "light" | "deep"
```

### Source mapping (`mapPipelineSources`)

```go
case "research":
    cs.Type = "research"
    cs.Topic = ps.Topic
    cs.Depth = ps.Depth  // default "light"
```

### Output format

Same as all other sources: `(text string, data any, error)`.

- `text`: markdown research report
- `data`: structured list of searched URLs, titles, and summaries

## Fetcher Implementation

### researchFetcher

Implements `SourceFetcher` interface. Registered in `NewCollectStageExecutor()` alongside existing fetchers. Runs in parallel with other sources via errgroup.

### Light mode (timeout: 30s)

```
topic → LLM generates search query
      → web_search tool (1 call)
      → get_webpage on top 5-10 results
      → LLM summarizes findings as markdown
```

Single-pass. No loop. Fast.

### Deep mode (timeout: 5min)

```
topic → ADK agent with web_search + get_webpage tools
      → agent decomposes topic into 3-5 sub-questions
      → searches iteratively, evaluates sufficiency
      → stops when satisfied or max_searches reached
      → produces final markdown research report
```

Reuses existing ADK agent infrastructure (model initialization, tool registration, execution loop). No new agent framework needed.

### Agent loop control

- `max_searches` caps the number of web_search tool calls (default 10)
- System prompt instructs the agent to stop when findings are sufficient
- Hard timeout at 5 minutes regardless of search count

## LLM & Provider Compatibility

Uses the pipeline's configured model. No separate model setting.

| Provider | web_search support | Research available |
|----------|-------------------|-------------------|
| Anthropic | Yes (Google Search via API) | Yes |
| Gemini | Yes (native Google Search) | Yes |
| OpenAI | Yes (shared native tool registry) | Yes |
| Ollama | No (native tools unsupported) | **No** |

### Ollama blocking

- **Frontend**: hide research source type when pipeline model starts with `ollama/`
- **Backend**: researchFetcher returns clear error if model does not implement `NativeToolProvider`

Both layers enforce the restriction.

## Frontend UI

### Source category

Third category added to AddSourceModal alongside Static and Signal:

```
📡 Static    RSS · HN · HTTP · Scrape
📊 Signal    Reddit · Trends · Social
🔍 Research  AI Research
```

### Configuration (2-step modal, same as all sources)

Step 1: Select "AI Research" from Research category.

Step 2: Configure:
- **Topic** (text input, required): the subject to research
- **Depth** (radio: light / deep, default light): search depth

`max_searches` is hidden in initial version, using default value of 10.

## Skills & Prompts

### New skill file: `internal/skills/stages/stage-research.md`

Contains system prompts for both light and deep modes. English, consistent with all existing skills.

Light mode prompt: focused single-pass search and summarize.

Deep mode prompt: expert researcher persona with sub-question decomposition, iterative search, sufficiency evaluation, and structured report output.

### Update: `internal/skills/stages/stage-collect.md`

Add `research` to the supported source types list with field documentation (`topic`, `depth`).

### Prompt management

All prompts in skill files, not hardcoded in Go. Enables prompt tuning without code changes.

## Result Integration

Research output merges with other source results in the same format. The analysis stage sees research findings alongside RSS/HTTP/social data and processes them together. Users can curate which sources matter during the review step.

## Progress Feedback

### Mechanism: polling (existing infrastructure)

SessionSetupView already polls the session detail API every 3 seconds while status is `collecting`. No new SSE channel needed.

### Backend

researchFetcher updates `SourceFetch` record in DB after each search iteration:

```json
{
  "progress": {
    "current_step": 3,
    "max_steps": 10,
    "current_query": "EV battery technology trends 2026",
    "findings_count": 5
  }
}
```

### Frontend

During `collecting` status, the session detail response already includes sources. Display per-source progress inline:

```
✓ RSS: TechCrunch              3 items
✓ HTTP: competitor.com         done
● Research: EV market trends   searching 3/10
  └ "EV battery technology trends 2026"
```

Light mode completes fast enough that progress display is optional — just shows a spinner. Deep mode shows step count and current query.

## Non-Goals

- No research-specific model selection (uses pipeline model)
- No deduplication across sessions (same as existing sources)
- No `max_searches` UI exposure in initial version
- No new SSE channel for collect stage (existing 3s polling is sufficient)
