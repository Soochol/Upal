# DDD 계층 구조 점진적 개선 계획

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 도메인별로 순차 진행하며 DDD 계층 위반을 교정한다. 포트 인터페이스 누락 추가, API 핸들러의 비즈니스 로직을 서비스 레이어로 이동, 구체 타입 의존을 포트로 전환.

**Architecture:** Clean Architecture / Hexagonal Architecture. 의존 방향: Domain ← Ports ← Services ← API. 각 도메인을 잎 노드(독립)부터 루트(의존 많은 쪽) 순서로 개선한다.

**Tech Stack:** Go 1.24, Chi router, Google ADK, PostgreSQL (optional)

**진행 순서:** Connection → Workflow → Pipeline → Content Session → Shared Infrastructure

---

## Phase 1: Connection 도메인 (워밍업)

**현재 문제:**
- `server.go:39` — `connectionSvc *services.ConnectionService` (구체 타입 의존)
- 포트 인터페이스 없음

**개선 목표:** ConnectionService에 포트 인터페이스 도입, API 레이어가 포트에만 의존하도록 전환

---

### Task 1.1: ConnectionService 포트 인터페이스 정의

**Files:**
- Create: `internal/upal/ports/connection.go`

**Step 1: 포트 인터페이스 작성**

`internal/services/connection.go`의 public 메서드 시그니처를 기반으로 포트 정의:

```go
package ports

import (
	"context"

	"github.com/your-org/upal/internal/upal"
)

// ConnectionPort defines the connection management boundary.
type ConnectionPort interface {
	Create(ctx context.Context, conn *upal.Connection) error
	Get(ctx context.Context, id string) (*upal.Connection, error)
	Resolve(ctx context.Context, id string) (*upal.Connection, error)
	List(ctx context.Context) ([]upal.ConnectionSafe, error)
	Update(ctx context.Context, conn *upal.Connection) error
	Delete(ctx context.Context, id string) error
}
```

**Step 2: 테스트 — 컴파일 확인**

Run: `go build ./internal/upal/ports/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/upal/ports/connection.go
git commit -m "feat: add ConnectionPort interface"
```

---

### Task 1.2: API Server를 포트 의존으로 전환

**Files:**
- Modify: `internal/api/server.go` (line 39, setter line 237-239)

**Step 1: server.go 필드 타입 변경**

```go
// Before
connectionSvc *services.ConnectionService

// After
connectionSvc ports.ConnectionPort
```

**Step 2: setter 시그니처 변경**

```go
// Before
func (s *Server) SetConnectionService(svc *services.ConnectionService) {

// After
func (s *Server) SetConnectionService(svc ports.ConnectionPort) {
```

**Step 3: import 정리**

`services` 패키지 import가 connectionSvc 때문만이었다면 제거. `ports` import 추가.

**Step 4: 테스트 — 전체 빌드**

Run: `go build ./...`
Expected: PASS — `ConnectionService`는 이미 `ConnectionPort`의 모든 메서드를 구현하므로 main.go 변경 불필요.

**Step 5: 기존 테스트 실행**

Run: `make test`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/api/server.go
git commit -m "refactor: use ConnectionPort interface in API server"
```

---

### Task 1.3: 외부 소비자도 포트 의존으로 전환 확인

**Step 1: ConnectionService 구체 타입 참조 검색**

Run: `grep -rn "*services.ConnectionService\|*ConnectionService" internal/ --include="*.go" | grep -v "_test.go" | grep -v "services/connection.go"`

서비스 내부(notification, approval executor 등)에서 구체 타입을 사용하는 곳이 있으면 포트로 전환.

**Step 2: 발견된 구체 참조 각각을 포트로 전환**

해당 파일의 필드 타입과 생성자 파라미터를 `ports.ConnectionPort`로 변경.

**Step 3: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: propagate ConnectionPort to all consumers"
```

---

## Phase 2: Workflow 도메인

**현재 문제:**
- `api/workflow.go:15-34` — `validateWorkflowTools()` 비즈니스 로직이 API 핸들러에 위치
- 도구 검증과 모델 검증이 분리되어 있음 (도구 → API, 모델 → Service)

**개선 목표:** 워크플로우 검증 로직을 WorkflowService.Validate()로 통합

---

### Task 2.1: WorkflowService에 도구 검증 통합

**Files:**
- Modify: `internal/services/workflow.go` (lines 29-37 struct, lines 67-85 Validate)
- Modify: `internal/api/workflow.go` (lines 15-34 validateWorkflowTools, lines 45, 95 호출부)

**Step 1: WorkflowService에 toolReg 의존성 확인**

`WorkflowService`는 이미 `toolReg *tools.Registry` 필드를 가지고 있음 (line 36). 추가 주입 불필요.

**Step 2: Validate() 메서드에 도구 검증 로직 추가**

`internal/services/workflow.go`의 `Validate()` 메서드 끝에 도구 검증을 추가:

```go
func (s *WorkflowService) Validate(wf *upal.WorkflowDefinition) error {
	// 기존 모델 검증 로직 유지...
	for _, n := range wf.Nodes {
		if n.Type == upal.NodeTypeAgent {
			// ... 기존 모델 검증 ...
		}
	}

	// 도구 검증 추가
	if s.toolReg != nil {
		for _, n := range wf.Nodes {
			if n.Type != upal.NodeTypeTool {
				continue
			}
			toolName, _ := n.Config["tool"].(string)
			if toolName == "" {
				return fmt.Errorf("tool node %q: missing required config field \"tool\"", n.ID)
			}
			_, isCustom := s.toolReg.Get(toolName)
			isNative := s.toolReg.IsNative(toolName)
			if !isCustom && !isNative {
				return fmt.Errorf("tool node %q: unknown tool %q", n.ID, toolName)
			}
		}
	}
	return nil
}
```

**Step 3: API 핸들러에서 validateWorkflowTools 호출 제거**

`internal/api/workflow.go`:
- `createWorkflow` (line 45): `s.validateWorkflowTools(&wf)` 호출 제거
- `updateWorkflow` (line 95): `s.validateWorkflowTools(&wf)` 호출 제거

이미 `runWorkflow` 핸들러(`api/run.go:52`)에서 `s.workflowSvc.Validate(wf)`를 호출하고 있음. create/update 핸들러에서도 서비스의 `Validate()`를 호출하도록 변경:

```go
// createWorkflow에서
if err := s.workflowSvc.Validate(&wf); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
```

**Step 4: validateWorkflowTools 함수 삭제**

`internal/api/workflow.go`에서 `validateWorkflowTools` 함수 전체(lines 15-34) 삭제.

**Step 5: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/services/workflow.go internal/api/workflow.go
git commit -m "refactor: move tool validation from API handler to WorkflowService.Validate()"
```

---

### Task 2.2: create/update 핸들러의 Validate 호출 경로 정비

**Files:**
- Modify: `internal/api/workflow.go`

**Step 1: createWorkflow 핸들러가 서비스 Validate를 호출하는지 확인**

현재 `createWorkflow`는 `s.repo.Create()`를 직접 호출함. 서비스의 `Validate()`를 거쳐야 함.

다만 create/update는 `s.repo`에 직접 접근하는 패턴(CRUD)이므로, Validate만 서비스에 위임하면 충분함:

```go
func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
    // ... decode ...
    if err := s.workflowSvc.Validate(&wf); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if err := s.repo.Create(r.Context(), &wf); err != nil {
        // ...
    }
    // ...
}
```

update도 동일 패턴 적용.

**Step 2: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/api/workflow.go
git commit -m "refactor: ensure create/update handlers call WorkflowService.Validate()"
```

---

## Phase 3: Pipeline 도메인

**현재 문제:**
- `api/pipelines.go:31-36, 78-83` — 스케줄 동기화 오케스트레이션이 API 핸들러에 위치
- `server.go:43` — `pipelineSvc *services.PipelineService` (구체 타입 의존)
- PipelineService에 포트 인터페이스 없음 (PipelineRegistry만 존재)

**개선 목표:** 스케줄 동기화를 PipelineService로 이동, PipelineServicePort 도입

---

### Task 3.1: PipelineService에 스케줄 동기화 통합

**Files:**
- Modify: `internal/services/pipeline_service.go`
- Modify: `internal/api/pipelines.go`

**Step 1: PipelineService에 SchedulerPort 의존성 추가**

```go
type PipelineService struct {
	repo         repository.PipelineRepository
	runRepo      repository.PipelineRunRepository
	schedulerSvc ports.SchedulerPort // 추가
}

func NewPipelineService(
	repo repository.PipelineRepository,
	runRepo repository.PipelineRunRepository,
) *PipelineService {
	return &PipelineService{repo: repo, runRepo: runRepo}
}

// setter로 주입 (순환 의존 방지 — scheduler도 pipeline에 의존하므로)
func (s *PipelineService) SetSchedulerService(svc ports.SchedulerPort) {
	s.schedulerSvc = svc
}
```

**Step 2: Create/Update 메서드에 스케줄 동기화 내장**

```go
func (s *PipelineService) Create(ctx context.Context, p *upal.Pipeline) error {
	// 기존 검증 + ID 할당 + repo.Create ...

	// 스케줄 동기화 (scheduler가 설정된 경우에만)
	if s.schedulerSvc != nil {
		if err := s.schedulerSvc.SyncPipelineSchedules(ctx, p); err == nil {
			_ = s.repo.Update(ctx, p) // schedule_id 반영
		}
	}
	return nil
}

func (s *PipelineService) Update(ctx context.Context, p *upal.Pipeline) error {
	// 기존 repo.Update ...

	// 스케줄 동기화
	if s.schedulerSvc != nil {
		if err := s.schedulerSvc.SyncPipelineSchedules(ctx, p); err == nil {
			_ = s.repo.Update(ctx, p)
		}
	}
	return nil
}
```

**Step 3: API 핸들러에서 스케줄 동기화 코드 제거**

`internal/api/pipelines.go`:
- `createPipeline` (lines 31-36): 스케줄 동기화 블록 삭제
- `updatePipeline` (lines 78-83): 스케줄 동기화 블록 삭제

**Step 4: main.go에서 setter 호출 추가**

```go
pipelineSvc.SetSchedulerService(schedulerSvc)
```

**Step 5: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/services/pipeline_service.go internal/api/pipelines.go cmd/upal/main.go
git commit -m "refactor: move schedule sync orchestration from API handler to PipelineService"
```

---

### Task 3.2: PipelineServicePort 인터페이스 정의

**Files:**
- Create: `internal/upal/ports/pipeline.go`
- Modify: `internal/api/server.go`

**Step 1: 포트 인터페이스 작성**

```go
package ports

import (
	"context"

	"github.com/your-org/upal/internal/upal"
)

// PipelineServicePort defines the pipeline management boundary.
type PipelineServicePort interface {
	Create(ctx context.Context, p *upal.Pipeline) error
	Get(ctx context.Context, id string) (*upal.Pipeline, error)
	List(ctx context.Context) ([]*upal.Pipeline, error)
	Update(ctx context.Context, p *upal.Pipeline) error
	Delete(ctx context.Context, id string) error
	GetRun(ctx context.Context, id string) (*upal.PipelineRun, error)
	ListRuns(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error)
	CreateRun(ctx context.Context, run *upal.PipelineRun) error
	UpdateRun(ctx context.Context, run *upal.PipelineRun) error
	RejectRun(ctx context.Context, pipelineID, runID string) (*upal.PipelineRun, error)
}
```

**주의:** 기존 `ports.PipelineRegistry`(Get만 있음)는 SchedulerService가 사용 중. 제거하지 않고 `PipelineServicePort`가 superset으로 존재. PipelineRegistry는 그대로 유지하되, 향후 PipelineServicePort로 통합 가능.

**Step 2: server.go 필드 + setter 전환**

```go
// Before
pipelineSvc *services.PipelineService

// After
pipelineSvc ports.PipelineServicePort
```

setter 시그니처도 동일하게 변경.

**Step 3: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/upal/ports/pipeline.go internal/api/server.go
git commit -m "refactor: add PipelineServicePort and use in API server"
```

---

### Task 3.3: deletePipeline 핸들러의 스케줄 정리 로직 이동

**Files:**
- Modify: `internal/services/pipeline_service.go`
- Modify: `internal/api/pipelines.go`

**Step 1: PipelineService.Delete()에 스케줄 정리 내장**

```go
func (s *PipelineService) Delete(ctx context.Context, id string) error {
	if s.schedulerSvc != nil {
		_ = s.schedulerSvc.RemovePipelineSchedules(ctx, id)
	}
	return s.repo.Delete(ctx, id)
}
```

**Step 2: API 핸들러에서 스케줄 정리 코드 제거**

`deletePipeline` (lines 88-99)에서 `schedulerSvc.RemovePipelineSchedules` 호출 삭제. 단순히 `s.pipelineSvc.Delete()` 호출만 남김.

**Step 3: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/services/pipeline_service.go internal/api/pipelines.go
git commit -m "refactor: move schedule cleanup into PipelineService.Delete()"
```

---

## Phase 4: Content Session 도메인

**현재 문제:**
- `server.go:45,46` — `contentSvc`, `collector` 모두 구체 타입 의존
- `api/content.go:77-86` — 상태 필터링 로직이 API 핸들러에 위치
- `api/content.go:156-171` — approve 후 자동 produce 오케스트레이션이 API에 위치
- `content_collector.go:27-37` — 3개 구체 서비스 의존

**개선 목표:** ContentSessionServicePort 도입, 비즈니스 로직 서비스로 이동, ContentCollector 의존 정리

---

### Task 4.1: ContentSessionServicePort 인터페이스 정의

**Files:**
- Create: `internal/upal/ports/content.go`

**Step 1: 포트 인터페이스 작성**

ContentSessionService의 public 메서드 중 API에서 사용하는 것들을 포트로 추출:

```go
package ports

import (
	"context"

	"github.com/your-org/upal/internal/upal"
)

// ContentSessionPort defines the content session management boundary.
type ContentSessionPort interface {
	CreateSession(ctx context.Context, sess *upal.ContentSession) error
	GetSession(ctx context.Context, id string) (*upal.ContentSession, error)
	GetSessionDetail(ctx context.Context, id string) (*upal.ContentSessionDetail, error)
	ListSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error)
	ListSessionDetailsByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSessionDetail, error)
	ListArchivedSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error)
	ListTemplateDetailsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error)
	UpdateSession(ctx context.Context, sess *upal.ContentSession) error
	UpdateSessionStatus(ctx context.Context, id string, status upal.ContentSessionStatus) error
	UpdateSessionSourceCount(ctx context.Context, id string, count int) error
	ApproveSession(ctx context.Context, id string) error
	RejectSession(ctx context.Context, id string) error
	ArchiveSession(ctx context.Context, id string) error
	UnarchiveSession(ctx context.Context, id string) error
	DeleteSession(ctx context.Context, id string) error
	RecordSourceFetch(ctx context.Context, sf *upal.SourceFetch) error
	UpdateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error
	ListSourceFetches(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error)
	RecordAnalysis(ctx context.Context, a *upal.LLMAnalysis) error
	GetAnalysis(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error)
	UpdateAnalysis(ctx context.Context, sessionID string, summary string, insights []string) error
	UpdateAnalysisAngles(ctx context.Context, sessionID string, angles []upal.ContentAngle) error
	UpdateAngleWorkflow(ctx context.Context, sessionID, angleID, workflowName string) error
	RecordPublished(ctx context.Context, pc *upal.PublishedContent) error
	ListPublished(ctx context.Context) ([]*upal.PublishedContent, error)
	ListPublishedBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error)
	CreateSurge(ctx context.Context, se *upal.SurgeEvent) error
	ListSurges(ctx context.Context) ([]*upal.SurgeEvent, error)
	ListActiveSurges(ctx context.Context) ([]*upal.SurgeEvent, error)
	DismissSurge(ctx context.Context, id string) error
	SetWorkflowResults(ctx context.Context, sessionID string, results []upal.WorkflowResult)
	GetWorkflowResults(ctx context.Context, sessionID string) []upal.WorkflowResult
}
```

**참고:** 인터페이스가 크다면 추후 역할별로 분리 가능 (SessionWriter, SessionReader, AnalysisManager 등). 우선은 단일 포트로 시작.

**Step 2: 빌드 확인**

Run: `go build ./internal/upal/ports/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/upal/ports/content.go
git commit -m "feat: add ContentSessionPort interface"
```

---

### Task 4.2: API Server를 ContentSessionPort 의존으로 전환

**Files:**
- Modify: `internal/api/server.go` (line 45, setter line 272-274)

**Step 1: 필드 + setter 타입 변경**

```go
// Before
contentSvc *services.ContentSessionService

// After
contentSvc ports.ContentSessionPort
```

**Step 2: content.go 핸들러에서 포트에 없는 메서드 사용 확인**

`UpdateSessionSettings`, `ListSessionsByPipelineAndStatus` 등 포트에 포함되지 않은 메서드가 있으면 포트에 추가하거나, 핸들러가 해당 기능을 포트의 다른 메서드 조합으로 대체할 수 있는지 확인.

빠진 메서드가 있으면 포트에 추가.

**Step 3: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/api/server.go internal/upal/ports/content.go
git commit -m "refactor: use ContentSessionPort in API server"
```

---

### Task 4.3: 상태 필터링 로직을 서비스로 이동

**Files:**
- Modify: `internal/services/content_session_service.go`
- Modify: `internal/api/content.go`

**Step 1: 서비스에 필터링 통합 메서드 추가**

```go
func (s *ContentSessionService) ListSessionDetailsByPipelineAndStatus(
	ctx context.Context, pipelineID string, status upal.ContentSessionStatus,
) ([]*upal.ContentSessionDetail, error) {
	details, err := s.ListSessionDetails(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	if status == "" {
		return details, nil
	}
	filtered := make([]*upal.ContentSessionDetail, 0, len(details))
	for _, d := range details {
		if d.Status == status {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}
```

**Step 2: API 핸들러에서 필터링 코드를 서비스 호출로 대체**

`listContentSessions`의 lines 77-86 필터링 블록을 단일 서비스 호출로 대체.

**Step 3: 포트에 새 메서드 추가**

**Step 4: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/content_session_service.go internal/api/content.go internal/upal/ports/content.go
git commit -m "refactor: move status filtering from API handler to ContentSessionService"
```

---

### Task 4.4: ContentCollector 구체 의존을 포트로 전환

**Files:**
- Modify: `internal/services/content_collector.go` (lines 27-37 struct, lines 40-63 constructor)

**Step 1: contentSvc 필드를 포트로 변경**

```go
// Before
contentSvc *ContentSessionService

// After
contentSvc ports.ContentSessionPort
```

**Step 2: workflowSvc 필드를 기존 포트로 변경**

```go
// Before
workflowSvc *WorkflowService

// After
workflowSvc ports.WorkflowExecutor
```

**Step 3: collectExec은 내부 컴포넌트이므로 유지**

`CollectStageExecutor`는 ContentCollector와 같은 패키지 내부의 구현 디테일이므로 포트 불필요.

**Step 4: 생성자 시그니처 업데이트 + main.go 조정**

```go
func NewContentCollector(
	contentSvc ports.ContentSessionPort,       // 변경
	collectExec *CollectStageExecutor,          // 유지
	workflowSvc ports.WorkflowExecutor,        // 변경
	// ... 나머지 동일
)
```

**Step 5: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/services/content_collector.go cmd/upal/main.go
git commit -m "refactor: use port interfaces in ContentCollector dependencies"
```

---

## Phase 5: 공유 인프라 정리

**현재 문제:**
- `server.go:27` — `limiter *services.ConcurrencyLimiter` (포트 `ConcurrencyControl` 존재하나 미사용)
- `server.go:40,41,42` — `executionReg`, `runManager`, `runPublisher` 모두 구체 타입
- `RunPublisher`가 `*services.RunManager`, `*services.ExecutionRegistry` 구체 의존

**개선 목표:** 이미 존재하는 포트 활용, 필요한 곳에 최소한의 포트 추가

---

### Task 5.1: ConcurrencyLimiter를 기존 포트로 전환

**Files:**
- Modify: `internal/api/server.go` (line 27, setter line 222-224)

**Step 1: 필드 타입을 기존 포트로 변경**

```go
// Before
limiter *services.ConcurrencyLimiter

// After
limiter ports.ConcurrencyControl
```

**Step 2: setter 시그니처 변경**

```go
func (s *Server) SetConcurrencyLimiter(limiter ports.ConcurrencyControl) {
```

**Step 3: API 핸들러에서 ConcurrencyLimiter 전용 메서드(Stats 등) 사용 여부 확인**

포트에 없는 메서드를 사용한다면 포트에 추가하거나 별도 처리.

**Step 4: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/api/server.go
git commit -m "refactor: use ConcurrencyControl port in API server"
```

---

### Task 5.2: RunManager, ExecutionRegistry 포트 정의

**Files:**
- Create: `internal/upal/ports/run.go`
- Modify: `internal/api/server.go`

**Step 1: RunManager 포트 — API가 사용하는 메서드만 추출**

```go
package ports

// RunManagerPort defines the run event streaming boundary.
type RunManagerPort interface {
	Register(runID string)
	Append(runID string, ev EventRecord)
	Complete(runID string, payload map[string]any)
	Fail(runID string, errMsg string)
	Subscribe(runID string, startSeq int) ([]EventRecord, <-chan struct{})
}
```

`EventRecord` 타입도 domain으로 이동하거나 포트 파일에 정의 필요. `RunManager`의 `EventRecord`가 현재 어디에 정의되어 있는지 확인 후 결정.

**Step 2: ExecutionRegistry 포트**

```go
// ExecutionRegistryPort defines the execution pause/resume boundary.
type ExecutionRegistryPort interface {
	Register(runID string) *upal.ExecutionHandle
	Get(runID string) *upal.ExecutionHandle
	Unregister(runID string)
}
```

**Step 3: server.go 필드 + setter 전환**

**Step 4: RunPublisher 의존도 포트로 전환**

`internal/services/run/publisher.go`의 `runManager`, `executionReg` 필드를 포트로 변경.

**Step 5: 빌드 + 테스트**

Run: `go build ./... && make test`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/upal/ports/run.go internal/api/server.go internal/services/run/publisher.go
git commit -m "refactor: add RunManagerPort and ExecutionRegistryPort, use in server and publisher"
```

---

### Task 5.3: Generator, toolReg 등 나머지 구체 의존 평가

**Step 1: 평가 — 포트 도입 여부 결정**

| 필드 | 현재 타입 | 포트 필요? | 이유 |
|------|----------|-----------|------|
| `generator` | `*generate.Generator` | △ 보류 | API에서 직접 호출하는 메서드가 적고, 이미 `ports.WorkflowGenerator`가 일부 존재 |
| `toolReg` | `*tools.Registry` | × 불필요 | 인프라 레지스트리, 읽기 전용 lookup만 수행 |
| `runPublisher` | `*runpub.RunPublisher` | △ 보류 | API에서 Launch만 호출, 단일 구현만 존재할 가능성 높음 |

포트를 만들어도 구현체가 하나뿐이고 테스트에서 mock이 불필요한 경우는 보류. 향후 필요 시 추가.

**Step 2: 결정 사항 기록 후 Commit**

```bash
git commit --allow-empty -m "docs: record DDD Phase 5 evaluation - generator/toolReg ports deferred"
```

---

## 완료 기준

각 Phase 완료 시 검증 체크리스트:

- [ ] `go build ./...` 성공
- [ ] `make test` 전체 통과
- [ ] `server.go`의 구체 서비스 의존 수가 줄었는지 확인
- [ ] API 핸들러 파일에 비즈니스 로직(검증, 오케스트레이션, 필터링)이 남아있지 않은지 확인
- [ ] 새로 추가한 포트 인터페이스가 `internal/upal/ports/`에 위치하는지 확인

## 최종 목표 상태

| 필드 | Before | After |
|------|--------|-------|
| `connectionSvc` | `*services.ConnectionService` | `ports.ConnectionPort` |
| `pipelineSvc` | `*services.PipelineService` | `ports.PipelineServicePort` |
| `contentSvc` | `*services.ContentSessionService` | `ports.ContentSessionPort` |
| `limiter` | `*services.ConcurrencyLimiter` | `ports.ConcurrencyControl` |
| `executionReg` | `*services.ExecutionRegistry` | `ports.ExecutionRegistryPort` |
| `runManager` | `*services.RunManager` | `ports.RunManagerPort` |

비즈니스 로직 이동:
| 로직 | Before (API) | After (Service) |
|------|-------------|-----------------|
| 도구 검증 | `api/workflow.go:validateWorkflowTools` | `WorkflowService.Validate()` |
| 스케줄 동기화 | `api/pipelines.go:31-36, 78-83` | `PipelineService.Create/Update()` |
| 스케줄 정리 | `api/pipelines.go:93-94` | `PipelineService.Delete()` |
| 상태 필터링 | `api/content.go:77-86` | `ContentSessionService.ListSessionDetailsByPipelineAndStatus()` |
