# Session Archive & Delete Design

**Date**: 2026-02-24
**Status**: Approved

## Problem

Sessions accumulate with no way to clean up. Users need to remove unwanted sessions (failed test runs, rejected sessions, etc.) without accidentally losing important data.

## Design Decision: Two-Step Soft Delete

**Archive first, then delete from archive.** This prevents accidental permanent data loss while keeping the active session list clean.

### Why `archived_at` timestamp, not `archived` status

- **Preserves original status**: An archived session retains its `pending_review`, `rejected`, `published` etc. status — users can see _why_ it was archived.
- **Orthogonal to state machine**: Archive is a separate dimension, not a state transition. No need to define transitions from every status → archived.
- **Enables restore**: Unarchive returns the session to its exact previous state.
- **Audit trail**: The timestamp records _when_ the session was archived.

## Data Model

### Schema Change

```sql
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
```

### Go Domain Type

```go
// internal/upal/content.go
type ContentSession struct {
    // ... existing fields ...
    ArchivedAt *time.Time `json:"archived_at,omitempty"`
}
```

### Frontend Type

```ts
// web/src/entities/content-session/types.ts
interface ContentSession {
    // ... existing fields ...
    archived_at?: string
}
```

## API Design

| Method   | Endpoint                                | Guard                      | Description               |
|----------|-----------------------------------------|----------------------------|---------------------------|
| `POST`   | `/api/content-sessions/{id}/archive`    | Any status, not archived   | Sets `archived_at = NOW()`|
| `POST`   | `/api/content-sessions/{id}/unarchive`  | Must be archived           | Sets `archived_at = NULL` |
| `DELETE`  | `/api/content-sessions/{id}`            | Must be archived           | Permanent delete          |

### List Filtering

Existing `GET /api/content-sessions` gains an `archived_only` query param:

- Default (no param): returns only `archived_at IS NULL` sessions
- `?archived_only=true`: returns only archived sessions (for the Archived tab)

## Deletion Cascade

| Child table        | FK CASCADE? | Action on session delete         |
|--------------------|-------------|----------------------------------|
| `source_fetches`   | Yes         | Automatic via DB cascade         |
| `llm_analyses`     | Yes         | Automatic via DB cascade         |
| `published_content`| No          | Service deletes explicitly first |
| `workflowResults`  | In-memory   | Service removes map entry        |

## Clean Architecture Layers

### 1. Repository Interface (`internal/repository/content.go`)

```go
type ContentSessionRepository interface {
    // ... existing methods ...
    Delete(ctx context.Context, id string) error
}
```

### 2. DB Layer (`internal/db/content.go`)

- `ArchiveContentSession(ctx, id)` — `UPDATE ... SET archived_at = NOW()`
- `UnarchiveContentSession(ctx, id)` — `UPDATE ... SET archived_at = NULL`
- `DeleteContentSession(ctx, id)` — `DELETE FROM content_sessions WHERE id = $1`
- `DeletePublishedContentBySession(ctx, sessionID)` — explicit cleanup
- Update all `List*` queries to respect `archived_at IS NULL` by default

### 3. Memory Repository (`internal/repository/content_memory.go`)

- Add `Delete` method via `store.Delete(ctx, id)`
- Add `archived_at` filtering to list methods

### 4. Persistent Repository (`internal/repository/content_persistent.go`)

- Wire `Delete` to both memory + DB
- Wire archive/unarchive to both layers

### 5. Service (`internal/services/content_session_service.go`)

```go
func (s *ContentSessionService) ArchiveSession(ctx, id) error
func (s *ContentSessionService) UnarchiveSession(ctx, id) error
func (s *ContentSessionService) DeleteSession(ctx, id) error  // guard: must be archived
```

`DeleteSession` responsibilities:
1. Verify session is archived
2. Delete `published_content` by session ID
3. Delete session (cascades `source_fetches`, `llm_analyses`)
4. Remove `workflowResults[sessionID]` from in-memory map

### 6. API Handlers (`internal/api/content.go`)

Three new handlers registered at:
```go
r.Post("/{id}/archive", s.archiveContentSession)
r.Post("/{id}/unarchive", s.unarchiveContentSession)
r.Delete("/{id}", s.deleteContentSession)
```

### 7. Frontend API Client (`web/src/entities/content-session/api.ts`)

```ts
export async function archiveSession(id: string): Promise<ContentSession>
export async function unarchiveSession(id: string): Promise<ContentSession>
export async function deleteSession(id: string): Promise<void>
```

### 8. UI (`web/src/pages/pipelines/PipelineDetail.tsx`)

- `SessionFilter` type gains `'archived'` value
- Filter tabs: `All | Pending | Producing | Published | Rejected | Archived`
- `All` tab excludes archived sessions (default API behavior)
- `Archived` tab fetches with `?archived_only=true`
- Session card: `···` context menu or swipe with "Archive" action
- Archived session card: shows original status badge + "Unarchive" / "Delete permanently" actions
- Permanent delete: confirmation dialog required
