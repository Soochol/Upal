# Pipeline Workflow + Publish Channel Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add PublishChannel entity, workflow+channel mapping in pipeline wizard, channel_id propagation through produce, and a Publish Inbox page for per-workflow review/publish.

**Architecture:** New PublishChannel CRUD following Connection pattern (domain → repository → service → API → frontend). Extend PipelineWorkflow and WorkflowResult with channel_id. New /publish-inbox page replicating Inbox dual-pane pattern.

**Tech Stack:** Go (Chi router, memstore), React 19 (React Query, Zustand), TypeScript, Tailwind CSS v4

**Design doc:** `docs/plans/2026-02-24-pipeline-workflow-publish-design.md`

---

### Task 1: PublishChannel Domain Type

**Files:**
- Create: `internal/upal/publish_channel.go`

**Step 1: Create domain type**

```go
package upal

import "fmt"

type PublishChannelType string

const (
	ChannelWordPress PublishChannelType = "wordpress"
	ChannelYouTube   PublishChannelType = "youtube"
	ChannelSlack     PublishChannelType = "slack"
	ChannelTelegram  PublishChannelType = "telegram"
	ChannelSubstack  PublishChannelType = "substack"
	ChannelDiscord   PublishChannelType = "discord"
	ChannelMedium    PublishChannelType = "medium"
	ChannelTikTok    PublishChannelType = "tiktok"
	ChannelHTTP      PublishChannelType = "http"
)

type PublishChannel struct {
	ID   string             `json:"id"`
	Name string             `json:"name"`
	Type PublishChannelType `json:"type"`
}

func NewPublishChannelID() string {
	return fmt.Sprintf("ch-%s", GenerateShortID())
}
```

**Step 2: Verify build**

Run: `go build ./...`

**Step 3: Commit**

```
feat: add PublishChannel domain type
```

---

### Task 2: Add channel_id to PipelineWorkflow and WorkflowResult

**Files:**
- Modify: `internal/upal/pipeline.go:108-113` — add ChannelID field
- Modify: `internal/upal/content.go:123-130` — add ChannelID field

**Step 1: Add ChannelID to PipelineWorkflow**

In `internal/upal/pipeline.go`, add to PipelineWorkflow struct:
```go
ChannelID    string `json:"channel_id,omitempty"`
```

**Step 2: Add ChannelID to WorkflowResult**

In `internal/upal/content.go`, add to WorkflowResult struct:
```go
ChannelID    string               `json:"channel_id,omitempty"`
```

**Step 3: Verify build**

Run: `go build ./...`

**Step 4: Commit**

```
feat: add channel_id to PipelineWorkflow and WorkflowResult
```

---

### Task 3: PublishChannel Repository

**Files:**
- Create: `internal/repository/publish_channel_memory.go`

Follow `internal/repository/connection_memory.go` pattern exactly.

**Step 1: Create memory repository**

```go
package repository

import (
	"context"
	"errors"
	"fmt"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

type PublishChannelRepository interface {
	Create(ctx context.Context, ch *upal.PublishChannel) error
	Get(ctx context.Context, id string) (*upal.PublishChannel, error)
	List(ctx context.Context) ([]*upal.PublishChannel, error)
	Update(ctx context.Context, ch *upal.PublishChannel) error
	Delete(ctx context.Context, id string) error
}

type MemoryPublishChannelRepository struct {
	store *memstore.Store[*upal.PublishChannel]
}

func NewMemoryPublishChannelRepository() *MemoryPublishChannelRepository {
	return &MemoryPublishChannelRepository{
		store: memstore.New(func(ch *upal.PublishChannel) string { return ch.ID }),
	}
}

func (r *MemoryPublishChannelRepository) Create(ctx context.Context, ch *upal.PublishChannel) error {
	if r.store.Has(ctx, ch.ID) {
		return fmt.Errorf("publish channel %q already exists", ch.ID)
	}
	return r.store.Set(ctx, ch)
}

func (r *MemoryPublishChannelRepository) Get(ctx context.Context, id string) (*upal.PublishChannel, error) {
	ch, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("publish channel %q: %w", id, ErrNotFound)
	}
	return ch, err
}

func (r *MemoryPublishChannelRepository) List(ctx context.Context) ([]*upal.PublishChannel, error) {
	return r.store.All(ctx)
}

func (r *MemoryPublishChannelRepository) Update(ctx context.Context, ch *upal.PublishChannel) error {
	if !r.store.Has(ctx, ch.ID) {
		return fmt.Errorf("publish channel %q: %w", ch.ID, ErrNotFound)
	}
	return r.store.Set(ctx, ch)
}

func (r *MemoryPublishChannelRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return fmt.Errorf("publish channel %q: %w", id, ErrNotFound)
	}
	return err
}
```

**Step 2: Verify build**

Run: `go build ./...`

**Step 3: Commit**

```
feat: add PublishChannel memory repository
```

---

### Task 4: PublishChannel API Handlers

**Files:**
- Create: `internal/api/publish_channels.go`
- Modify: `internal/api/server.go:22-47` — add `publishChannelRepo` field
- Modify: `internal/api/server.go:148-156` — add routes
- Modify: `cmd/upal/main.go:210-218` — wire repository

**Step 1: Create API handlers**

Create `internal/api/publish_channels.go` following `internal/api/connections.go` pattern:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) createPublishChannel(w http.ResponseWriter, r *http.Request) {
	var ch upal.PublishChannel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if ch.Name == "" || ch.Type == "" {
		http.Error(w, "name and type are required", http.StatusBadRequest)
		return
	}
	ch.ID = upal.NewPublishChannelID()
	if err := s.publishChannelRepo.Create(r.Context(), &ch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ch)
}

func (s *Server) listPublishChannels(w http.ResponseWriter, r *http.Request) {
	chs, err := s.publishChannelRepo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chs)
}

func (s *Server) getPublishChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.publishChannelRepo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "publish channel not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ch)
}

func (s *Server) updatePublishChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var ch upal.PublishChannel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	ch.ID = id
	if err := s.publishChannelRepo.Update(r.Context(), &ch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ch)
}

func (s *Server) deletePublishChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.publishChannelRepo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

**Step 2: Add field to Server struct** (`server.go:22-47`)

Add `publishChannelRepo repository.PublishChannelRepository` field.

**Step 3: Add setter** (after SetConnectionService in server.go)

```go
func (s *Server) SetPublishChannelRepo(repo repository.PublishChannelRepository) {
	s.publishChannelRepo = repo
}
```

**Step 4: Register routes** (server.go, after connections route block ~line 156)

```go
if s.publishChannelRepo != nil {
	r.Route("/publish-channels", func(r chi.Router) {
		r.Post("/", s.createPublishChannel)
		r.Get("/", s.listPublishChannels)
		r.Get("/{id}", s.getPublishChannel)
		r.Put("/{id}", s.updatePublishChannel)
		r.Delete("/{id}", s.deletePublishChannel)
	})
}
```

**Step 5: Wire in main.go** (after connection setup ~line 218)

```go
publishChannelRepo := repository.NewMemoryPublishChannelRepository()
srv.SetPublishChannelRepo(publishChannelRepo)
```

**Step 6: Verify build**

Run: `go build ./...`

**Step 7: Commit**

```
feat: add PublishChannel CRUD API endpoints
```

---

### Task 5: Channel ID Propagation in ProduceWorkflows

**Files:**
- Modify: `internal/services/content_collector.go:444-462` — pass channel_id from pipeline
- Modify: `internal/api/content.go:129-167` — accept channel mapping in produce request
- Modify: `internal/api/content.go:220-281` — use channel_id from WorkflowResult in publish

**Step 1: Extend produce request body**

In `internal/api/content.go` produce handler, change body struct:
```go
var body struct {
	Workflows []struct {
		Name      string `json:"name"`
		ChannelID string `json:"channel_id,omitempty"`
	} `json:"workflows"`
}
```

Pass workflow names and channel_ids to ProduceWorkflows.

**Step 2: Update ProduceWorkflows signature**

In `internal/services/content_collector.go`, change to accept channel mapping:
```go
type WorkflowRequest struct {
	Name      string
	ChannelID string
}

func (c *ContentCollector) ProduceWorkflows(ctx context.Context, sessionID string, requests []WorkflowRequest)
```

Initialize results with ChannelID:
```go
results[i] = upal.WorkflowResult{
	WorkflowName: req.Name,
	Status:       upal.WFResultPending,
	ChannelID:    req.ChannelID,
}
```

**Step 3: Use channel_id in publish handler**

In `internal/api/content.go` publish handler, read ChannelID from WorkflowResult:
```go
runToChannel := make(map[string]string, len(wfResults))
for _, wr := range wfResults {
	if wr.RunID != "" {
		runToChannel[wr.RunID] = wr.ChannelID
	}
}
// ...
pc := &upal.PublishedContent{
	Channel: runToChannel[runID],  // instead of hardcoded "default"
}
```

**Step 4: Verify build and tests**

Run: `go build ./... && go test ./internal/services/... -v -race`

**Step 5: Commit**

```
feat: propagate channel_id through produce to publish
```

---

### Task 6: Frontend Types + PublishChannel API Client

**Files:**
- Modify: `web/src/shared/types/index.ts:243-247` — add channel_id to PipelineWorkflow
- Modify: `web/src/entities/content-session/types.ts:42-48` — add channel_id to WorkflowResult
- Create: `web/src/entities/publish-channel/types.ts`
- Create: `web/src/entities/publish-channel/api.ts`

**Step 1: Add PublishChannel types**

```typescript
// web/src/entities/publish-channel/types.ts
export type PublishChannelType =
  | 'wordpress' | 'youtube' | 'slack' | 'telegram'
  | 'substack' | 'discord' | 'medium' | 'tiktok' | 'http'

export type PublishChannel = {
  id: string
  name: string
  type: PublishChannelType
}
```

**Step 2: Add API client**

```typescript
// web/src/entities/publish-channel/api.ts
import { apiFetch } from '@/shared/api'
import type { PublishChannel } from './types'

const BASE = '/api/publish-channels'

export async function fetchPublishChannels(): Promise<PublishChannel[]> {
  return apiFetch(BASE)
}

export async function createPublishChannel(ch: Omit<PublishChannel, 'id'>): Promise<PublishChannel> {
  return apiFetch(BASE, { method: 'POST', body: JSON.stringify(ch) })
}

export async function updatePublishChannel(id: string, ch: Partial<PublishChannel>): Promise<PublishChannel> {
  return apiFetch(`${BASE}/${id}`, { method: 'PUT', body: JSON.stringify(ch) })
}

export async function deletePublishChannel(id: string): Promise<void> {
  await apiFetch(`${BASE}/${id}`, { method: 'DELETE' })
}
```

**Step 3: Update existing types**

Add `channel_id?: string` to `PipelineWorkflow` in `web/src/shared/types/index.ts`.
Add `channel_id?: string` to `WorkflowResult` in `web/src/entities/content-session/types.ts`.

**Step 4: Verify frontend types**

Run: `cd web && npx tsc -b`

**Step 5: Commit**

```
feat: add PublishChannel frontend types and API client
```

---

### Task 7: Pipeline Wizard Step 4 — Workflow + Channel Mapping

**Files:**
- Modify: `web/src/pages/pipelines/PipelineNew.tsx` — Step 4 workflow+channel UI
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx` — settings panel workflow section

**Step 1: Update Step 4 in PipelineNew.tsx**

Add channel selection dropdown next to each workflow in the list. Fetch publish channels via `useQuery({ queryKey: ['publish-channels'], queryFn: fetchPublishChannels })`.

Each workflow row:
```
[Workflow Name]  →  [Channel Dropdown ▾]  [✕ Remove]
```

Channel dropdown options from `fetchPublishChannels()` + "None" option + "Create new..." option.

When "Create new..." selected, show inline name+type inputs, call `createPublishChannel`, add to list.

**Step 2: Update PipelineDetail settings panel**

Same pattern in the Workflows section of the right settings panel — add channel dropdown next to each workflow.

**Step 3: Verify frontend types and build**

Run: `cd web && npx tsc -b`

**Step 4: Commit**

```
feat: add workflow+channel mapping in pipeline wizard and settings
```

---

### Task 8: Update Produce API and Store for Channel Mapping

**Files:**
- Modify: `web/src/entities/content-session/api.ts` — update produceSession signature
- Modify: `web/src/entities/content-session/store.ts` — pass channel mapping on approval

**Step 1: Update produceSession API**

Change from `produceSession(id, workflows: string[])` to:
```typescript
export async function produceSession(
  id: string,
  workflows: { name: string; channel_id?: string }[],
): Promise<{ session_id: string; status: string }>
```

**Step 2: Update store approval flow**

In `approveSession`, pass workflow objects with channel_id from pipeline config.

**Step 3: Verify frontend types**

Run: `cd web && npx tsc -b`

**Step 4: Commit**

```
feat: pass channel mapping through produce API
```

---

### Task 9: Publish Inbox Page — Layout + Sidebar

**Files:**
- Create: `web/src/pages/publish-inbox/index.tsx`
- Create: `web/src/pages/publish-inbox/PublishInboxSidebar.tsx`
- Create: `web/src/pages/publish-inbox/PublishInboxPreview.tsx`
- Modify: `web/src/app/router.tsx` — add /publish-inbox route
- Modify: `web/src/app/layout.tsx` — add nav item

Replicate `web/src/pages/inbox/` pattern exactly.

**Step 1: Create page layout**

`web/src/pages/publish-inbox/index.tsx` — same structure as inbox/index.tsx but:
- Query: `fetchContentSessions({ status: 'approved' })`
- Header: "Publish Inbox"
- Empty state: "No content awaiting publish"

**Step 2: Create sidebar**

`web/src/pages/publish-inbox/PublishInboxSidebar.tsx` — same as InboxSidebar but:
- Show workflow count and completion stats instead of score
- Badge showing how many workflows succeeded

**Step 3: Add route and nav**

In `router.tsx`, add lazy import and route for `/publish-inbox`.
In `layout.tsx`, add nav item with `Send` icon from lucide-react.

**Step 4: Verify frontend build**

Run: `cd web && npx tsc -b`

**Step 5: Commit**

```
feat: add Publish Inbox page layout with sidebar
```

---

### Task 10: Publish Inbox Preview — Per-Workflow Review Cards

**Files:**
- Create: `web/src/pages/publish-inbox/PublishInboxPreview.tsx`

**Step 1: Create preview component**

Fetch session detail + pipeline data (same queries as InboxPreview).

Display each `workflow_result` with `status === 'success'` as a card:

```
┌─ blog-producer ──────────────────────────┐
│  Channel: WordPress (회사 블로그)          │
│  ──────────────────────────────────────── │
│  [Content preview area]                   │
│  Run ID: run-abc123                       │
│  Completed: 2 min ago                     │
│  ──────────────────────────────────────── │
│  [Reject]                   [Publish →]   │
└──────────────────────────────────────────┘
```

Each card has independent Publish/Reject buttons.

**Step 2: Individual publish action**

Publish button calls `POST /api/content-sessions/{id}/publish` with single `run_ids: [runId]`.

After all workflows are either published or rejected, transition session to published.

**Step 3: Verify frontend build**

Run: `cd web && npx tsc -b`

**Step 4: Commit**

```
feat: add Publish Inbox preview with per-workflow review cards
```

---

### Task 11: Backend Publish Endpoint — Individual Publish Support

**Files:**
- Modify: `internal/api/content.go:220-281` — support individual publish (don't transition to published until all done)

**Step 1: Change publish logic**

Current: publishes all run_ids and transitions session to `published`.
New: publish single run_id, mark that WorkflowResult as published, only transition to `published` when all successful results are either published or rejected.

Add new status to WorkflowResult: `"published"` and `"rejected"`.

**Step 2: Add reject endpoint**

`POST /api/content-sessions/{id}/reject-result` with `{ run_id: "..." }` — marks the WorkflowResult as rejected.

**Step 3: Verify build and tests**

Run: `go build ./... && go test ./internal/... -v -race`

**Step 4: Commit**

```
feat: support individual publish and reject per workflow result
```

---

### Task 12: End-to-End Verification

**Step 1: Start backend**

Run: `make dev-backend`

**Step 2: Start frontend**

Run: `make dev-frontend`

**Step 3: Test flow**

1. Create a PublishChannel via API: `POST /api/publish-channels {"name": "Test Blog", "type": "wordpress"}`
2. Create pipeline with workflow + channel mapping in Step 4
3. Start session → Collect → Analyze → Approve
4. Verify Produce runs workflows in parallel (cards show running simultaneously)
5. Navigate to /publish-inbox
6. Verify sidebar shows session
7. Preview shows per-workflow cards with channel info
8. Publish one, reject another
9. Verify Published page shows the published item

**Step 4: Run all tests**

Run: `make test && make test-frontend`

**Step 5: Commit**

```
test: verify pipeline workflow+publish channel end-to-end
```
