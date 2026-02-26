# Session / Instance Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Pipeline 페이지에서 세션(템플릿)을 관리하고, Inbox에서 인스턴스(실행 결과)를 처리하도록 역할을 분리한다.

**Architecture:** 기존 `ContentSession` 테이블의 `IsTemplate`/`ParentSessionID` 구분을 프론트엔드에서 활용. 백엔드는 `session_name` 필드 추가 + 세션별 수동 실행 엔드포인트 추가만.

**Tech Stack:** Go 1.24 (Chi), React 19, TypeScript, TanStack Query

---

### Task 1: Backend — ContentSessionDetail에 session_name 추가

인스턴스의 부모 세션명을 Inbox에서 표시하기 위해 `session_name` 필드를 추가한다.

**Files:**
- Modify: `internal/upal/content.go:159-184`
- Modify: `internal/services/content_session_service.go:476-503`

**Step 1: ContentSessionDetail에 session_name 필드 추가**

`internal/upal/content.go` — `ContentSessionDetail` struct의 `PipelineName` 아래에 추가:

```go
SessionName     string               `json:"session_name,omitempty"`
```

**Step 2: sessionToDetail()에서 session_name 채우기**

`internal/services/content_session_service.go` — `sessionToDetail()` 함수에서 `ParentSessionID`가 있으면 부모 세션을 조회해서 이름을 채운다:

```go
func (s *ContentSessionService) sessionToDetail(
	ctx context.Context, sess *upal.ContentSession, names *pipelineNameCache,
) *upal.ContentSessionDetail {
	analysis, _ := s.analyses.GetBySession(ctx, sess.ID)
	wfResults := s.GetWorkflowResults(ctx, sess.ID)

	var sessionName string
	if sess.ParentSessionID != "" {
		if parent, err := s.sessions.Get(ctx, sess.ParentSessionID); err == nil {
			sessionName = parent.Name
		}
	}

	return &upal.ContentSessionDetail{
		// ... existing fields ...
		SessionName:     sessionName,
		// ...
	}
}
```

**Step 3: 테스트 — 백엔드 빌드 확인**

Run: `go build ./...`
Expected: 빌드 성공

**Step 4: Commit**

```bash
git add internal/upal/content.go internal/services/content_session_service.go
git commit -m "feat: add session_name to ContentSessionDetail for instance traceability"
```

---

### Task 2: Backend — 세션별 수동 실행 엔드포인트

현재 `POST /api/pipelines/{id}/collect`는 첫 번째 템플릿을 찾아 사용한다. N개 세션 지원을 위해 특정 템플릿 세션에서 인스턴스를 생성하는 엔드포인트를 추가한다.

**Files:**
- Modify: `internal/api/content.go` (새 핸들러 추가)
- Modify: `internal/api/server.go` (라우트 등록)

**Step 1: 라우트 등록 위치 확인**

`internal/api/server.go`에서 content-sessions 라우트 그룹을 찾아 새 엔드포인트를 추가한다.

**Step 2: 새 핸들러 작성**

`internal/api/content.go`에 `runSessionInstance` 핸들러 추가:

```go
// POST /api/content-sessions/{id}/run
// Creates a new instance from this template session and launches collection.
func (s *Server) runSessionInstance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	tmpl, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !tmpl.IsTemplate {
		http.Error(w, "only template sessions can spawn instances", http.StatusBadRequest)
		return
	}

	pipeline, err := s.pipelineSvc.Get(r.Context(), tmpl.PipelineID)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	var body struct {
		IsTest bool `json:"isTest"`
		Limit  int  `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	sess := &upal.ContentSession{
		PipelineID:      tmpl.PipelineID,
		TriggerType:     "manual",
		ParentSessionID: tmpl.ID,
		Sources:         tmpl.Sources,
		Schedule:        tmpl.Schedule,
		Model:           tmpl.Model,
		Workflows:       tmpl.Workflows,
		Context:         tmpl.Context,
	}
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if s.collector != nil {
		go s.collector.CollectAndAnalyze(context.Background(), pipeline, sess, body.IsTest, body.Limit)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"session_id": sess.ID})
}
```

**Step 3: 라우트 등록**

`internal/api/server.go`의 content-sessions 라우트 그룹에 추가:

```go
r.Post("/content-sessions/{id}/run", s.runSessionInstance)
```

**Step 4: 빌드 확인**

Run: `go build ./...`
Expected: 빌드 성공

**Step 5: Commit**

```bash
git add internal/api/content.go internal/api/server.go
git commit -m "feat: add POST /content-sessions/{id}/run endpoint for template-specific instance creation"
```

---

### Task 3: Frontend Types — session_name 추가

**Files:**
- Modify: `web/src/entities/content-session/types.ts:65-91`
- Modify: `web/src/entities/content-session/api.ts`

**Step 1: ContentSession 타입에 session_name 추가**

`web/src/entities/content-session/types.ts` — `pipeline_name` 아래에:

```typescript
session_name?: string
```

**Step 2: API 클라이언트에 runSessionInstance 함수 추가**

`web/src/entities/content-session/api.ts`에 추가:

```typescript
export async function runSessionInstance(id: string, options?: { isTest?: boolean; limit?: number }): Promise<{ session_id: string }> {
  return apiFetch<{ session_id: string }>(`${BASE}/${encodeURIComponent(id)}/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(options ?? {}),
  })
}
```

**Step 3: 타입 체크**

Run: `cd web && npx tsc -b`
Expected: 성공

**Step 4: Commit**

```bash
git add web/src/entities/content-session/types.ts web/src/entities/content-session/api.ts
git commit -m "feat: add session_name type and runSessionInstance API client"
```

---

### Task 4: Frontend — SessionListPanel을 템플릿 전용으로 변경

세션 목록이 인스턴스 대신 템플릿 세션만 보여주도록 변경한다.

**Files:**
- Modify: `web/src/pages/pipelines/SessionListPanel.tsx`

**Step 1: 데이터 소스를 templateOnly로 변경**

`fetchContentSessions({ pipelineId })` → `fetchContentSessions({ pipelineId, templateOnly: true })`

쿼리 키도 변경: `['content-sessions', { pipelineId, templateOnly: true }]`

**Step 2: 세션 아이템에 상태(draft/active) 표시 추가**

각 세션 아이템에 상태 뱃지 추가. active이면 초록색 dot, draft이면 회색 dot.

**Step 3: "New" 버튼이 템플릿 세션을 생성하도록 확인**

`onNewSession` 콜백이 `createDraftSession({ pipeline_id, is_template: true })`를 호출하는지 확인. 부모 페이지(`Pipelines.tsx`)의 `newSessionMutation`을 확인하고 필요시 수정.

**Step 4: 불필요한 UI 정리**

- `onStartSession` prop 제거 (이미 제거된 상태면 확인만)
- 검색은 유지 (세션명 검색)
- analysis summary 관련 표시 제거 (템플릿에는 analysis 없음)

**Step 5: 타입 체크**

Run: `cd web && npx tsc -b`
Expected: 성공

**Step 6: Commit**

```bash
git add web/src/pages/pipelines/SessionListPanel.tsx
git commit -m "feat: show template sessions only in pipeline session list"
```

---

### Task 5: Frontend — SessionSetupView에 Start/Stop + Run 버튼 추가

세션 설정 편집 화면에 활성화/비활성화 토글과 수동 실행 버튼을 추가한다.

**Files:**
- Modify: `web/src/pages/pipelines/session/SessionSetupView.tsx`

**Step 1: activate/deactivate import 추가**

```typescript
import { activateSession, deactivateSession, runSessionInstance } from '@/entities/content-session/api'
```

**Step 2: Start/Stop 토글 mutation 추가**

```typescript
const toggleMutation = useMutation({
  mutationFn: () => session?.status === 'active'
    ? deactivateSession(sessionId)
    : activateSession(sessionId),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
  },
})
```

**Step 3: Run (수동 실행) mutation 추가**

```typescript
const runMutation = useMutation({
  mutationFn: () => runSessionInstance(sessionId),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
  },
})
```

**Step 4: 헤더 영역에 버튼 배치**

세션 이름 옆에:
- **Start/Stop 토글** — active이면 "Active" (녹색) + Stop 버튼, draft이면 "Draft" + Start 버튼
- **Run 버튼** — "Run Now" 레이블, 즉시 인스턴스 1개 생성. 로딩 중에는 스피너 표시

**Step 5: 타입 체크**

Run: `cd web && npx tsc -b`
Expected: 성공

**Step 6: Commit**

```bash
git add web/src/pages/pipelines/session/SessionSetupView.tsx
git commit -m "feat: add start/stop toggle and manual run button to session settings"
```

---

### Task 6: Frontend — Inbox에 세션명 표시

Inbox 사이드바의 인스턴스 아이템에 파이프라인명 + 세션명을 모두 표시한다.

**Files:**
- Modify: `web/src/pages/inbox/InboxSidebar.tsx`

**Step 1: pipeline_name 표시 부분을 찾아서 session_name 추가**

기존 `{s.pipeline_name}` 표시 부분(약 Line 196)을 수정:

```tsx
{(s.pipeline_name || s.session_name) && (
  <p className="text-[10px] text-muted-foreground/60 uppercase tracking-wider font-bold truncate pl-6 mb-1">
    {[s.pipeline_name, s.session_name].filter(Boolean).join(' / ')}
  </p>
)}
```

결과: "Marketing Pipeline / Daily Tech News" 형태로 표시.

**Step 2: 타입 체크**

Run: `cd web && npx tsc -b`
Expected: 성공

**Step 3: Commit**

```bash
git add web/src/pages/inbox/InboxSidebar.tsx
git commit -m "feat: show pipeline name + session name in inbox instance list"
```

---

### Task 7: Frontend — Pipelines 페이지 newSessionMutation 수정

파이프라인 페이지에서 "New" 버튼으로 새 세션을 만들 때 템플릿 세션이 생성되도록 보장한다.

**Files:**
- Modify: `web/src/pages/pipelines/index.tsx` (또는 `Pipelines.tsx`)

**Step 1: newSessionMutation 확인 및 수정**

`newSessionMutation`이 `createDraftSession({ pipeline_id, is_template: true, name: 'New Session' })`를 호출하는지 확인. 아니면 수정.

**Step 2: invalidation 쿼리 키를 templateOnly에 맞게 업데이트**

세션 생성 후 `queryClient.invalidateQueries`의 키가 `['content-sessions', { pipelineId, templateOnly: true }]`와 매칭되는지 확인.

**Step 3: 타입 체크**

Run: `cd web && npx tsc -b`
Expected: 성공

**Step 4: Commit**

```bash
git add web/src/pages/pipelines/index.tsx
git commit -m "feat: ensure new session creates template with proper query invalidation"
```

---

### Task 8: 통합 검증

**Step 1: 백엔드 빌드 + 테스트**

Run: `make test`
Expected: 모든 테스트 통과

**Step 2: 프론트엔드 타입 체크**

Run: `make test-frontend`
Expected: 성공

**Step 3: 수동 E2E 확인 체크리스트**

1. 파이프라인 생성 → 자동으로 템플릿 세션 1개 생성됨
2. 세션 목록에 템플릿만 표시됨 (인스턴스 안 보임)
3. 세션 선택 → 설정 편집 화면 (sources, schedule, model, workflows, context)
4. Start/Stop 토글 동작 (draft ↔ active)
5. "Run Now" 클릭 → 인스턴스 생성 → Inbox에 나타남
6. Inbox에서 인스턴스에 "Pipeline X / Session Y" 표시됨
7. Inbox의 인스턴스 디테일은 기존 Collect→Analyze→Produce→Publish 뷰 그대로

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: session/instance redesign - templates in pipelines, instances in inbox"
```
