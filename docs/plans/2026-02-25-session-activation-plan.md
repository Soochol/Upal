# Session Template Activation & Schedule Integration — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** "Start Session" 버튼이 세션 템플릿의 cron 스케줄을 스케줄러에 등록하여 주기적으로 인스턴스를 Inbox로 생성하고, 활성 세션 카드에 pulse 애니메이션을 표시한다.

**Architecture:** Schedule 모델에 `SessionID` 필드 추가, `ContentSession`에 `ScheduleID`+`active` 상태 추가. 스케줄러 디스패치에 세션 기반 분기 추가. 프론트엔드는 activate/deactivate API로 토글.

**Tech Stack:** Go (Chi router, robfig/cron), React 19, TypeScript, Tailwind CSS v4, PostgreSQL

---

### Task 1: Data Model — Schedule에 SessionID 추가

**Files:**
- Modify: `internal/upal/scheduler.go:106-119`
- Modify: `internal/db/db.go:110-123`
- Modify: `internal/db/schedule.go`

**Step 1: Schedule 구조체에 SessionID 필드 추가**

`internal/upal/scheduler.go` — Schedule struct에 추가:

```go
type Schedule struct {
	ID           string         `json:"id"`
	WorkflowName string         `json:"workflow_name,omitempty"`
	PipelineID   string         `json:"pipeline_id,omitempty"`
	SessionID    string         `json:"session_id,omitempty"` // NEW: content session template reference
	CronExpr     string         `json:"cron_expr"`
	// ... rest unchanged
}
```

**Step 2: DB 스키마에 session_id 컬럼 추가**

`internal/db/db.go` — schedules CREATE TABLE 뒤에 ALTER 추가:

```sql
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS session_id TEXT NOT NULL DEFAULT '';
```

**Step 3: DB CRUD에 session_id 반영**

`internal/db/schedule.go` — 모든 INSERT/SELECT/UPDATE 쿼리의 컬럼 목록에 `session_id` 추가. `scanSchedules`의 Scan에도 `&s.SessionID` 추가.

**Step 4: 확인**

Run: `go build ./...`
Expected: 컴파일 성공

**Step 5: Commit**

```
feat: add SessionID field to Schedule model and DB schema
```

---

### Task 2: Data Model — ContentSession에 ScheduleID + active 상태 추가

**Files:**
- Modify: `internal/upal/content.go:10-41`
- Modify: `internal/db/db.go:176-186`
- Modify: `internal/db/content.go:14-94`
- Modify: `web/src/shared/types/index.ts:215-224`
- Modify: `web/src/entities/content-session/types.ts`

**Step 1: Go 도메인 모델 변경**

`internal/upal/content.go`:
- 상태 상수 추가: `SessionActive ContentSessionStatus = "active"`
- ContentSession struct에 추가: `ScheduleID string \`json:"schedule_id,omitempty"\``

**Step 2: DB 스키마 마이그레이션**

`internal/db/db.go` — content_sessions 테이블에:

```sql
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS schedule_id TEXT NOT NULL DEFAULT '';
```

**Step 3: DB CRUD 반영**

`internal/db/content.go`:
- `sessionCols` 상수에 `schedule_id` 추가
- `scanSession`의 Scan에 `&s.ScheduleID` 추가
- `CreateContentSession`의 INSERT에 `schedule_id` 추가
- UPDATE 쿼리들에도 반영

**Step 4: 프론트엔드 타입 추가**

`web/src/shared/types/index.ts` — ContentSessionStatus에 `'active'` 추가
`web/src/entities/content-session/types.ts` — ContentSession에 `schedule_id?: string` 추가

**Step 5: 확인**

Run: `go build ./...`
Run: `cd web && npx tsc -b`
Expected: 양쪽 컴파일 성공

**Step 6: Commit**

```
feat: add ScheduleID and active status to ContentSession model
```

---

### Task 3: Scheduler — 세션 기반 디스패치 추가

**Files:**
- Modify: `internal/services/scheduler/scheduler.go:37-57`
- Modify: `internal/services/scheduler/dispatch.go:18-54`
- Modify: `internal/services/scheduler/cron.go:45-51`

**Step 1: ContentSessionCollector 인터페이스 추가**

`internal/services/scheduler/scheduler.go` — 기존 `ContentCollector` 인터페이스 옆에:

```go
// ContentSessionCollector creates instances from a session template.
type ContentSessionCollector interface {
	CollectFromTemplate(ctx context.Context, templateID string) error
}
```

SchedulerService struct에 필드 추가:
```go
sessionCollector ContentSessionCollector
```

Setter 추가:
```go
func (s *SchedulerService) SetContentSessionCollector(c ContentSessionCollector) {
	s.sessionCollector = c
}
```

**Step 2: dispatch.go에 세션 분기 추가**

`internal/services/scheduler/dispatch.go` — `executeScheduledRun()` 상단, PipelineID 분기 앞에:

```go
// Session-template-triggered collection.
if schedule.SessionID != "" && s.sessionCollector != nil {
	slog.Info("scheduler: executing scheduled session collection",
		"schedule", schedule.ID, "session", schedule.SessionID)

	if err := s.sessionCollector.CollectFromTemplate(ctx, schedule.SessionID); err != nil {
		slog.Error("scheduler: session collection failed",
			"schedule", schedule.ID, "session", schedule.SessionID, "err", err)
	}

	now := time.Now()
	schedule.LastRunAt = &now
	if cronSched, parseErr := parseCronExpr(schedule.CronExpr, schedule.Timezone); parseErr == nil {
		schedule.NextRunAt = cronSched.Next(now)
	}
	schedule.UpdatedAt = now
	if updateErr := s.scheduleRepo.Update(ctx, schedule); updateErr != nil {
		slog.Warn("scheduler: failed to update schedule after session run", "err", updateErr)
	}
	return
}
```

**Step 3: cron.go 로깅에 세션 케이스 추가**

`internal/services/scheduler/cron.go:45-51` — registerCronJob의 로깅에 SessionID 분기:

```go
if schedule.SessionID != "" {
	slog.Info("scheduler: registered cron job",
		"id", schedule.ID, "session", schedule.SessionID, "cron", schedule.CronExpr)
} else if schedule.PipelineID != "" {
	// ... existing
}
```

**Step 4: 확인**

Run: `go build ./...`
Expected: 컴파일 성공

**Step 5: Commit**

```
feat: add session-based dispatch to scheduler
```

---

### Task 4: ContentCollector — CollectFromTemplate 구현

**Files:**
- Modify: `internal/services/content_collector.go:67-89`

**Step 1: CollectFromTemplate 메서드 추가**

`internal/services/content_collector.go` — `CollectPipeline` 메서드 뒤에:

```go
// CollectFromTemplate creates a child instance from a session template and
// launches background collection + analysis.
func (c *ContentCollector) CollectFromTemplate(ctx context.Context, templateID string) error {
	template, err := c.contentSvc.GetSession(ctx, templateID)
	if err != nil {
		return fmt.Errorf("template %s: %w", templateID, err)
	}
	if !template.IsTemplate {
		return fmt.Errorf("session %s is not a template", templateID)
	}

	instanceName := template.Name
	if instanceName != "" {
		instanceName += " — " + time.Now().Format("01/02 15:04")
	}
	sess := &upal.ContentSession{
		PipelineID:      template.PipelineID,
		Name:            instanceName,
		TriggerType:     "schedule",
		IsTemplate:      false,
		ParentSessionID: template.ID,
		Sources:         template.Sources,
		Model:           template.Model,
		Workflows:       template.Workflows,
		Context:         template.Context,
	}
	if err := c.contentSvc.CreateSession(ctx, sess); err != nil {
		return fmt.Errorf("create instance: %w", err)
	}
	go c.CollectAndAnalyze(context.Background(), sess, false, 0)
	return nil
}
```

**Step 2: 확인**

Run: `go build ./...`
Expected: 컴파일 성공

**Step 3: Commit**

```
feat: implement CollectFromTemplate for scheduled session collection
```

---

### Task 5: API — activate/deactivate 엔드포인트

**Files:**
- Modify: `internal/api/server.go:117-136`
- Modify: `internal/api/content.go`
- Modify: `internal/upal/ports/scheduler.go:39-42`

**Step 1: SchedulerPort 인터페이스 확장**

`internal/upal/ports/scheduler.go` — SchedulerPort에 추가:

```go
type SchedulerPort interface {
	SyncPipelineSchedules(ctx context.Context, pipeline *upal.Pipeline) error
	RemovePipelineSchedules(ctx context.Context, pipelineID string) error
	AddSchedule(ctx context.Context, schedule *upal.Schedule) error    // NEW
	RemoveSchedule(ctx context.Context, id string) error               // NEW
}
```

**Step 2: activate 핸들러 구현**

`internal/api/content.go` — 새 핸들러:

```go
// POST /api/content-sessions/{id}/activate
func (s *Server) activateSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	session, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !session.IsTemplate {
		http.Error(w, "only templates can be activated", http.StatusBadRequest)
		return
	}
	if session.Schedule == "" {
		http.Error(w, "schedule is required to activate", http.StatusBadRequest)
		return
	}
	if session.Status == upal.SessionActive {
		http.Error(w, "session is already active", http.StatusConflict)
		return
	}

	// Register cron schedule.
	sched := &upal.Schedule{
		SessionID: session.ID,
		CronExpr:  session.Schedule,
		Enabled:   true,
		Timezone:  "UTC",
	}
	if err := s.schedulerSvc.AddSchedule(r.Context(), sched); err != nil {
		http.Error(w, "failed to register schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update session: store schedule_id and transition to active.
	session.ScheduleID = sched.ID
	session.Status = upal.SessionActive
	if err := s.contentSvc.UpdateSession(r.Context(), session); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Trigger immediate first collection.
	if s.collector != nil {
		go s.collector.CollectFromTemplate(context.Background(), session.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "active", "schedule_id": sched.ID})
}
```

**Step 3: deactivate 핸들러 구현**

```go
// POST /api/content-sessions/{id}/deactivate
func (s *Server) deactivateSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	session, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if session.Status != upal.SessionActive {
		http.Error(w, "session is not active", http.StatusBadRequest)
		return
	}

	// Remove cron schedule.
	if session.ScheduleID != "" {
		if err := s.schedulerSvc.RemoveSchedule(r.Context(), session.ScheduleID); err != nil {
			log.Printf("warn: failed to remove schedule %s: %v", session.ScheduleID, err)
		}
	}

	session.ScheduleID = ""
	session.Status = upal.SessionDraft
	if err := s.contentSvc.UpdateSession(r.Context(), session); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "draft"})
}
```

**Step 4: 라우트 등록**

`internal/api/server.go:127` — collectSession 라우트 옆에:

```go
r.Post("/{id}/activate", s.activateSession)
r.Post("/{id}/deactivate", s.deactivateSession)
```

**Step 5: ContentSessionService에 UpdateSession 메서드 추가 (없다면)**

`internal/services/content_session_service.go` — 상태와 schedule_id를 함께 저장하는 범용 업데이트:

```go
func (s *ContentSessionService) UpdateSession(ctx context.Context, sess *upal.ContentSession) error {
	return s.sessions.Update(ctx, sess)
}
```

**Step 6: UpdateSessionSettings에서 active 상태도 수정 허용**

`internal/services/content_session_service.go:215` — 현재 draft만 허용하는 조건을 active도 포함:

```go
if sess.Status != upal.SessionDraft && sess.Status != upal.SessionActive {
	return fmt.Errorf("session %q: settings can only be changed in draft or active status", id)
}
```

**Step 7: 확인**

Run: `go build ./...`
Expected: 컴파일 성공

**Step 8: Commit**

```
feat: add activate/deactivate API endpoints for session templates
```

---

### Task 6: Wiring — 스케줄러에 SessionCollector 연결

**Files:**
- Modify: `cmd/upal/main.go` (또는 서버 빌드 지점)

**Step 1: main.go에서 SetContentSessionCollector 호출**

스케줄러 서비스에 collector 연결하는 기존 코드(`SetContentCollector`) 옆에:

```go
schedulerSvc.SetContentSessionCollector(collector)
```

**Step 2: 확인**

Run: `go build ./...`
Expected: 컴파일 성공

**Step 3: Commit**

```
feat: wire session collector into scheduler service
```

---

### Task 7: Frontend — activate/deactivate API 함수

**Files:**
- Modify: `web/src/entities/content-session/api.ts`

**Step 1: API 함수 추가**

```typescript
export async function activateSession(id: string): Promise<{ status: string; schedule_id: string }> {
  return apiFetch(`${BASE}/${encodeURIComponent(id)}/activate`, { method: 'POST' })
}

export async function deactivateSession(id: string): Promise<{ status: string }> {
  return apiFetch(`${BASE}/${encodeURIComponent(id)}/deactivate`, { method: 'POST' })
}
```

**Step 2: 타입체크 확인**

Run: `cd web && npx tsc -b`
Expected: 성공

**Step 3: Commit**

```
feat: add activate/deactivate API functions
```

---

### Task 8: Frontend — Start/Stop 토글 버튼

**Files:**
- Modify: `web/src/pages/pipelines/session/SessionSetupView.tsx:166-253`

**Step 1: import 변경**

기존 `collectSession` import를 `activateSession`, `deactivateSession`으로 교체:

```typescript
import {
  fetchContentSession,
  updateSessionSettings,
  activateSession,
  deactivateSession,
} from '@/entities/content-session/api'
```

`lucide-react` import에 `Square` 추가.

**Step 2: mutation 교체**

기존 `collectMutation`을 두 개의 mutation으로 교체:

```typescript
const activateMutation = useMutation({
  mutationFn: () => activateSession(sessionId),
  onSuccess: () => {
    addToast('세션이 활성화되었습니다. 스케줄에 따라 수집이 진행됩니다.')
    queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
  },
})

const deactivateMutation = useMutation({
  mutationFn: () => deactivateSession(sessionId),
  onSuccess: () => {
    addToast('세션이 비활성화되었습니다.')
    queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
  },
})
```

**Step 3: 버튼 토글 UI**

기존 `isDraft && (...)` 버튼을 교체:

```tsx
const isActive = session?.status === 'active'
const isDraft = session?.status === 'draft'
```

```tsx
{(isDraft || isActive) && (
  <button
    onClick={() => isActive ? deactivateMutation.mutate() : activateMutation.mutate()}
    disabled={
      (isDraft && (localSources.length === 0 || !localSchedule)) ||
      activateMutation.isPending || deactivateMutation.isPending
    }
    className={cn(
      'flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-opacity disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed shrink-0',
      isActive
        ? 'bg-destructive/10 text-destructive hover:bg-destructive/20'
        : 'bg-foreground text-background hover:opacity-90',
    )}
  >
    {activateMutation.isPending || deactivateMutation.isPending ? (
      <><Loader2 className="h-3 w-3 animate-spin" />{isActive ? 'Stopping...' : 'Starting...'}</>
    ) : isActive ? (
      <><Square className="h-3 w-3" />Stop Session</>
    ) : (
      <><Play className="h-3 w-3" />Start Session</>
    )}
  </button>
)}
```

**Step 4: active 상태에서도 설정 수정 허용**

`UpdateSessionSettings`의 조건을 이미 Task 5에서 변경했으므로 프론트엔드 수정 불필요.
단, 설정이 변경되면 스케줄 재등록이 필요한데 — 이건 active 상태에서 schedule을 변경하면 자동으로 deactivate→reactivate하거나, 다음 iteration에서 처리. 현재는 active 상태에서 schedule 변경 시 toast로 "스케줄을 변경하려면 세션을 먼저 정지하세요" 안내하는 게 가장 간단.

**Step 5: 타입체크 확인**

Run: `cd web && npx tsc -b`
Expected: 성공

**Step 6: Commit**

```
feat: replace Start Session with activate/deactivate toggle
```

---

### Task 9: Frontend — 세션 카드 활성 애니메이션

**Files:**
- Modify: `web/src/pages/pipelines/SessionListPanel.tsx:14-24, 159-178`
- Modify: `web/src/styles/react-flow-custom.css` (또는 `index.css`)

**Step 1: STATUS_DOT_COLOR에 active 추가**

`SessionListPanel.tsx:14-24`:

```typescript
const STATUS_DOT_COLOR: Record<ContentSessionStatus, string> = {
  draft:          'bg-muted-foreground/50',
  active:         'bg-success animate-active-pulse',  // NEW
  collecting:     'bg-info',
  // ... rest unchanged
}
```

**Step 2: CSS 애니메이션 추가**

`web/src/index.css` — Module Imports 섹션 위에:

```css
/* Active session pulse animation */
@keyframes activePulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(1.4); }
}
.animate-active-pulse {
  animation: activePulse 2s ease-in-out infinite;
}
```

**Step 3: active 카드에 glow 테두리**

`SessionListPanel.tsx:162-166` — 카드 className에 active 분기 추가:

```typescript
const isActive = s.status === 'active'
```

```typescript
className={cn(
  'group w-full text-left p-3 rounded-xl transition-all duration-200 cursor-pointer border min-h-[84px]',
  isSelected
    ? 'bg-primary/5 border-primary/40 shadow-sm ring-1 ring-primary/20'
    : isActive
      ? 'bg-card border-success/30 shadow-[0_0_8px_var(--color-success)/0.15] hover:border-success/50'
      : 'bg-card border-border/60 hover:border-primary/40 hover:bg-muted/50',
)}
```

**Step 4: 타입체크 확인**

Run: `cd web && npx tsc -b`
Expected: 성공

**Step 5: 시각 확인**

Run: `make dev-frontend`
파이프라인 페이지에서 active 세션 카드의 pulse dot + glow border 확인

**Step 6: Commit**

```
feat: add pulse animation to active session cards
```

---

### Task 10: 통합 테스트 + 마무리

**Step 1: 백엔드 빌드 확인**

Run: `make build`
Expected: 성공

**Step 2: 프론트엔드 빌드 확인**

Run: `cd web && npm run build`
Expected: 성공

**Step 3: Go 테스트**

Run: `make test`
Expected: 기존 테스트 통과

**Step 4: E2E 수동 테스트**

1. 파이프라인 페이지에서 세션 생성
2. Sources + Schedule 설정
3. "Start Session" 클릭 → 토스트 확인 + 카드 애니메이션 확인
4. "Stop Session" 클릭 → 토스트 확인 + 애니메이션 제거 확인
5. Inbox에서 첫 인스턴스 생성 확인

**Step 5: Commit (필요 시)**

```
fix: address integration issues from session activation feature
```
