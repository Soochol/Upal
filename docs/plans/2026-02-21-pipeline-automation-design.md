# Pipeline Automation Platform Design

**Date:** 2026-02-21
**Status:** Approved
**Scope:** 4 categories — Workflow, Schedule, Trigger, Pipeline

## Context

Upal is a visual AI workflow platform. This design adds a **Pipeline** orchestration layer that chains Workflows, Schedules, Approvals, and Triggers into end-to-end automation flows — targeting individual content creators (blog, SNS, YouTube, Naver Clip).

### Design Principles

- **Cost minimization:** Data collection via code tools (no LLM). LLM only for summary/generation/judgment using cheap models (Haiku/Flash).
- **Extensibility:** All new components use interfaces — new tools, stage types, notification channels, and publish targets can be added as plugins.
- **User involvement:** 2 approval gates — topic selection + final content review before publishing.

---

## 1. Architecture — 4 Category Relationships

```
┌─────────────────────────────────────────────────────┐
│                    Pipeline                          │
│  "Orchestrator that executes Stages in sequence"     │
│                                                     │
│  Stage 1          Stage 2         Stage 3           │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐      │
│  │ Schedule │───→│ Approval │───→│ Workflow │      │
│  │ +Workflow│    │ (wait)   │    │ (execute)│      │
│  └──────────┘    └──────────┘    └──────────┘      │
│                                                     │
│  Trigger can start a Pipeline or individual Workflow │
└─────────────────────────────────────────────────────┘
```

| Category | Role | Standalone | Inside Pipeline |
|----------|------|------------|-----------------|
| **Workflow** | DAG execution unit | User runs directly | Stage execution body |
| **Schedule** | Cron-based repetition | Single Workflow repeat | Pipeline first-stage trigger |
| **Trigger** | External event reaction | Webhook → Workflow | Pipeline start or stage progression |
| **Pipeline** | Stage chain orchestration | — | Combines Workflow + Approval + Schedule |

---

## 2. Pipeline Data Model

```go
type Pipeline struct {
    ID          string    `json:"id"`          // "pipe-xxxx"
    Name        string    `json:"name"`        // "Content Automation"
    Description string    `json:"description"`
    Stages      []Stage   `json:"stages"`      // Ordered stages
    Status      string    `json:"status"`      // idle | running | waiting | completed | failed
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Stage struct {
    ID        string      `json:"id"`         // "stage-1"
    Name      string      `json:"name"`       // "RSS Collection"
    Type      string      `json:"type"`       // "workflow" | "approval" | "schedule" | "trigger" | "transform"
    Config    StageConfig `json:"config"`
    DependsOn []string    `json:"depends_on"` // Previous stage IDs
    Status    string      `json:"status"`     // pending | running | waiting | completed | skipped | failed
}

type StageConfig struct {
    // workflow type
    WorkflowName string            `json:"workflow_name,omitempty"`
    InputMapping map[string]string  `json:"input_mapping,omitempty"` // "articles": "{{stage-1.output.summary}}"

    // approval type
    Message      string `json:"message,omitempty"`
    ConnectionID string `json:"connection_id,omitempty"`
    Timeout      int    `json:"timeout,omitempty"`

    // schedule type
    Cron         string `json:"cron,omitempty"`
    Timezone     string `json:"timezone,omitempty"`

    // trigger type
    TriggerID    string `json:"trigger_id,omitempty"`
}

type StageResult struct {
    StageID   string                 `json:"stage_id"`
    Status    string                 `json:"status"`
    Output    map[string]interface{} `json:"output"`
    Timestamp time.Time              `json:"timestamp"`
}
```

### Pipeline State Machine

```
idle → (Schedule trigger or manual start) → running
running → Stage completed → advance to next Stage
running → Approval Stage → waiting (user pending)
waiting → (user approves) → running (next Stage)
waiting → (user rejects) → failed or idle
running → last Stage completed → completed → idle (if Schedule, wait for next cron)
running → error → failed
```

---

## 3. New Tools

### 3.1 `fetch_rss` — RSS/Atom Feed Parser

```
Input:  { url: string, max_items?: number, since_date?: string }
Output: { items: [{ title, link, published, summary, author }], feed_title, feed_url }
```

- Go `gofeed` library — supports RSS 2.0, Atom, JSON Feed
- `since_date` filters to new items only (cost savings)
- Zero LLM cost — structured data via code
- Extensibility: same pattern for future `fetch_*` tools (fetch_api, fetch_sitemap, etc.)

### 3.2 `content_store` — Key-Value Persistent Store

```
Input:  { action: "get"|"set"|"list"|"delete", key: string, value?: string, prefix?: string }
Output: { value: string } or { keys: string[] }
```

- Persists data across pipeline runs (seen URLs, last collection time, etc.)
- Deduplication: "Have we already collected this URL?"
- File-based storage (JSON file) — no DB dependency
- Extensibility: storage backend interface (file → SQLite → PostgreSQL)

### 3.3 `publish` — Multi-Channel Publisher

```
Input:  { channel: string, content: string, title?: string, metadata?: object }
Output: { status: string, url?: string }
```

- Phase 1 channels: `markdown_file` (local save), `webhook` (external HTTP call)
- Extensibility: `Publisher` interface — add new channels without modifying core
- Future channels: WordPress API, Tistory, Twitter, etc.
- Content formatting is done by Agent nodes (LLM), publish tool only handles delivery

### Cost Allocation by Stage

| Stage | Tool | LLM Usage |
|-------|------|-----------|
| RSS collection | `fetch_rss` | None |
| Web crawling | `get_webpage` (existing) | None |
| API calls | `http_request` (existing) | None |
| Deduplication | `content_store` | None |
| Data processing | `python_exec` (existing) | None |
| Summary/analysis/generation | Agent node | **LLM** (cheap model) |
| Publishing | `publish` | None |

---

## 4. Pipeline Backend — API & Execution Engine

### API Endpoints

```
POST   /api/pipelines                                    Create pipeline
GET    /api/pipelines                                    List pipelines
GET    /api/pipelines/{id}                               Get pipeline (with current status)
PUT    /api/pipelines/{id}                               Update pipeline
DELETE /api/pipelines/{id}                               Delete pipeline

POST   /api/pipelines/{id}/start                         Start pipeline manually
POST   /api/pipelines/{id}/stop                          Stop pipeline

POST   /api/pipelines/{id}/stages/{stageId}/approve      Approve stage
POST   /api/pipelines/{id}/stages/{stageId}/reject       Reject stage

GET    /api/pipelines/{id}/runs                          Pipeline run history
GET    /api/pipelines/{id}/runs/{runId}                  Specific run detail (SSE stream)
```

### Execution Engine

```go
// Extensible StageExecutor interface — implement to add new stage types
type StageExecutor interface {
    Execute(ctx context.Context, stage Stage, prevResult StageResult) (StageResult, error)
    Type() string
}

type PipelineRunner struct {
    executors map[string]StageExecutor // "workflow", "approval", "schedule", ...
    store     PipelineStore
    eventBus  EventBus
}

func (r *PipelineRunner) Run(ctx context.Context, pipeline Pipeline) {
    for _, stage := range pipeline.Stages {
        executor := r.executors[stage.Type]
        result, err := executor.Execute(ctx, stage, prevResult)
        // Store result → pass to next Stage
    }
}
```

### Stage Type Execution Logic

| Stage Type | Execution | Blocking |
|------------|-----------|----------|
| `workflow` | Run workflow, store result in StageResult | Wait until complete |
| `approval` | Send notification → wait for user response | **Indefinite** (configurable timeout) |
| `schedule` | Register cron, advance on trigger | **Wait until next cron** |
| `trigger` | Wait for webhook, advance on receipt | **Wait for event** |
| `transform` | Process previous result (JSONPath, mapping) — no LLM | Immediate |

### Pipeline Run Persistence

```
Pipeline Run Record:
  run_id, pipeline_id, status, started_at, completed_at
  current_stage_id, stage_results: map[stageId]StageResult
```

- State survives server restarts (approval waiting, schedule sleeping)
- Schedule stages register cron then put Pipeline Run in "sleeping" state

---

## 5. Pipeline Frontend UI

### New Page: `/pipelines`

Add **Pipelines** tab to existing navigation (Editor, Runs, Schedules).

#### Pipeline List View

```
┌──────────────────────────────────────────────────────────┐
│  Pipelines                               [+ New Pipeline]│
├──────────────────────────────────────────────────────────┤
│  ┌─ Content Automation ────────── ● Running (Stage 2) ──┐│
│  │ Daily 9am collect → Topic approval → Generate → Pub  ││
│  │ ○───●───◉───○───○                                    ││
│  │ Collect Summary Approval Generate Publish  [View][Stop]│
│  └──────────────────────────────────────────────────────┘│
│  ┌─ Competitor Monitor ────────── ● Idle (next: tmrw 9am)┐│
│  │ Daily crawl → Detect changes → Report → Notify       ││
│  │ ○───○───○───○                                        ││
│  │ Crawl  Analyze Report Notify          [View] [Start] ││
│  └──────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────┘
```

#### Pipeline Editor (Detail View)

```
┌──────────────────────────────────────────────────────────┐
│  ← Content Automation                [Save] [Run] [Delete]│
├──────────────────────────────────────────────────────────┤
│  Stage Timeline (horizontal):                             │
│  ┌────────┐    ┌────────┐    ┌────────┐    ┌────────┐   │
│  │Schedule│───→│Workflow│───→│Approval│───→│Workflow│   │
│  │Daily 9 │    │RSS     │    │Topic   │    │Content │   │
│  │        │    │Collect │    │Select  │    │Generate│   │
│  └────────┘    └────────┘    └────────┘    └────────┘   │
│      ↕              ↕              ↕            ↕        │
│  [cron config] [workflow    [message     [workflow       │
│  [timezone]     select]      config]      select]        │
│                [input map]  [channel]    [input map]     │
│                             [timeout]                    │
├──────────────────────────────────────────────────────────┤
│  Run History                                              │
│  ┌─ Run #3 ── 2/21 09:00 ── ● Approval Pending ────────┐│
│  │  Stage 1 ✓ 12s │ Stage 2 ✓ 3s │ Stage 3 ◉ waiting  ││
│  │  [Approve] [Reject]                                  ││
│  └──────────────────────────────────────────────────────┘│
│  ┌─ Run #2 ── 2/20 09:00 ── ✓ Completed ───────────────┐│
│  │  Stage 1 ✓ 11s │ Stage 2 ✓ 2s │ Stage 3 ✓ │ ...    ││
│  └──────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────┘
```

### UI Components

| Component | Role |
|-----------|------|
| `PipelineList` | Pipeline card list with current status |
| `PipelineEditor` | Stage timeline editor, add/delete/reorder stages |
| `StageCard` | Individual stage config card (type-specific form) |
| `PipelineRunHistory` | Run history timeline |
| `PipelineRunDetail` | Per-run stage status + approval buttons |
| `ApprovalBanner` | Top banner for pipelines awaiting approval |

### Approval UX Flow

1. Schedule triggers collection workflow → completes
2. Approval Stage reached → Notification sent (Telegram/Slack/Email)
3. User visits **Upal UI** or **clicks notification link**
4. Pipeline detail page shows **previous stage results** + **Approve/Reject buttons**
5. On approve → next Stage (content generation) starts automatically

---

## 6. Use Case Templates

### Template 1: Daily Content Ideas (Core Use Case)

```
Pipeline: "Daily Content Ideas"

Stage 1 — Schedule (daily 9am)
  └─ Workflow: "RSS Collect & Summarize"
      [Tool: fetch_rss × N feeds]
      → [Tool: content_store dedup]
      → [Iterator: new items]
        → [Agent: keyword extraction (Flash/Haiku)]
      → [Agent: trend summary + 3 topic suggestions (Flash/Haiku)]

Stage 2 — Approval (topic selection)
  └─ Notification: Telegram "Today's trend summary + 3 suggestions"
  └─ User response: select topic number or free input

Stage 3 — Workflow: "Content Generation"
  └─ [Agent: draft content for selected topic (Flash/Haiku)]
      → [Branch: channel split]
        → [Agent: blog format]
        → [Agent: SNS short-form]
        → [Agent: YouTube script]

Stage 4 — Approval (final review)
  └─ Notification: "Generated content preview"
  └─ User: approve / request edits / reject

Stage 5 — Workflow: "Publish"
  └─ [Tool: publish (markdown save)]
     [Tool: publish (webhook → external services)]
     [Notification: "Published" alert]
```

**Daily cost estimate:**
- RSS collect/parse/dedup: **$0** (code tools)
- Summary + topic suggestion (Haiku): ~2K input tokens × 10 articles → **~$0.005**
- Content generation (Haiku, 3 channels): ~1K input + ~2K output × 3 → **~$0.01**
- **Daily total: ~$0.02 or less**

### Template 2: Industry Monitoring

```
Pipeline: "Weekly Industry Report"

Stage 1 — Schedule (weekly Monday 8am)
  └─ Workflow: "Data Collection"
      [Tool: fetch_rss (industry news)]
      [Tool: get_webpage (competitor sites)]
      [Tool: content_store (change detection)]

Stage 2 — Workflow: "Analysis Report"
  └─ [Agent: summarize key changes + implications]
     → [Notification: send report via Telegram/Email]
```

### Template 3: Research Briefing

```
Pipeline: "Weekly Research Briefing"

Stage 1 — Schedule (weekly Friday)
  └─ Workflow: "Paper Collection"
      [Tool: http_request (arXiv/PapersWithCode API)]
      [Tool: content_store (exclude previously seen)]
      [Iterator: new papers]
        → [Agent: abstract summary + relevance score (Flash)]
      [Branch: filter high-relevance]
      [Agent: compile weekly briefing]

Stage 2 — Workflow: "Publish"
  └─ [Tool: publish (markdown save)]
     [Notification: "Briefing ready"]
```

### Template 4: Event-Driven (Trigger)

```
Pipeline: "Customer Review Response"

Stage 1 — Trigger (Webhook from review platform)
  └─ Workflow: "Review Analysis"
      [Agent: sentiment analysis + issue extraction]
      [Branch: negative?]
        → Yes: [Agent: draft response]
        → No: [Tool: content_store (log only)]

Stage 2 — Approval (negative reviews only)
  └─ Notification: "Negative review detected — review draft response"

Stage 3 — Workflow: "Publish Response"
  └─ [Tool: publish (webhook → review platform reply)]
```

---

## 7. Implementation Phases

### Phase 1: Infrastructure (Tools)

| Item | Location | Description |
|------|----------|-------------|
| `fetch_rss` tool | `internal/tools/rss_feed.go` | gofeed-based RSS/Atom parser |
| `content_store` tool | `internal/tools/content_store.go` | Key-value persistent store |
| `publish` tool | `internal/tools/publish.go` | markdown_file + webhook channels |
| Tool registry | `cmd/upal/main.go` | Register 3 new tools |

### Phase 2: Pipeline Backend

| Item | Location | Description |
|------|----------|-------------|
| Pipeline types | `internal/upal/types.go` | Pipeline, Stage, StageConfig, StageResult |
| Pipeline Repository | `internal/repository/pipeline.go` | CRUD + in-memory implementation |
| Pipeline Service | `internal/services/pipeline.go` | Business logic |
| PipelineRunner | `internal/services/pipeline_runner.go` | Stage execution engine + StageExecutor interface |
| Stage Executors | `internal/services/stage_*.go` | workflow, approval, schedule, trigger, transform |
| Pipeline API | `internal/api/pipelines.go` | REST endpoints |
| Server routing | `internal/api/server.go` | Route registration |

### Phase 3: Pipeline Frontend

| Item | Location | Description |
|------|----------|-------------|
| Pipeline page | `web/src/pages/Pipelines.tsx` | List + editor |
| Pipeline components | `web/src/components/pipelines/` | List, Editor, StageCard, RunHistory |
| Pipeline API client | `web/src/lib/api/pipelines.ts` | REST call functions |
| Pipeline store | `web/src/stores/pipelineStore.ts` | Zustand state management |
| Navigation update | `web/src/App.tsx` (router) | Add /pipelines route |
| ApprovalBanner | `web/src/components/pipelines/` | Approval pending notification banner |

### Phase 4: Templates & Extensions

| Item | Description |
|------|-------------|
| 4 template workflows | Pre-configured pipeline examples |
| Connection management UI | Telegram/Slack/SMTP connection setup screen |
| publish tool extensions | Additional channels (platform API integrations) |
| Pipeline stats dashboard | Run count, success rate, cost estimates |

### Dependency Order

```
Phase 1 (Tools) ────→ Phase 2 (Backend) ────→ Phase 3 (Frontend) → Phase 4
                          ↓
                   Leverages existing Schedule/
                   Trigger/Notification systems
```

---

## 8. Extensibility Points Summary

| Extension Type | Interface | How to Add |
|----------------|-----------|------------|
| New tool | `Tool` interface | Implement + register in `cmd/upal/main.go` |
| New stage type | `StageExecutor` interface | Implement + register in PipelineRunner |
| New notification channel | `Sender` interface | Implement + register in SenderRegistry |
| New publish target | `Publisher` interface (inside publish tool) | Implement + register |
| New data source | Follow `fetch_rss` pattern | Create `fetch_*` tool |
| New storage backend | `ContentStore` interface | Implement file/SQLite/PostgreSQL adapter |
