# Backend Architecture Refactor Design

## Goal

Go 백엔드의 아키텍처 문제 15개를 체계적으로 해결한다. Bottom-up 순서로 4개 Phase에 걸쳐 진행하며, 각 Phase는 다음 Phase의 기반이 된다.

## Identified Issues

### Category A: 중복 정의 (7건)

| # | 이슈 | 위치 | 심각도 |
|---|------|------|--------|
| A1 | 모델 해석 로직 3중 구현 | `agents/builders.go:41-61`, `api/server.go:280-297`, `llmutil/resolver.go:12-38`, `services/workflow.go:77` | High |
| A2 | `SchedulerConfig` = `ConcurrencyLimits` | `config/config.go:22-25`, `upal/scheduler.go:80-83` | Medium |
| A3 | `ModelOption`/`ToolEntry` vs `ModelInfo`/`ToolInfo` | `generate/generate.go:19-31`, `api/models.go:49-58`, `tools/registry.go:86-90` | Medium |
| A4 | `EventRecord` wraps `WorkflowEvent` | `upal/events.go:5-9`, `services/runmanager.go:9-14` | Medium |
| A5 | 상태 문자열 리터럴 vs 타입 상수 | `upal/scheduler.go`, `upal/workflow.go` vs services/api 전반 | High |
| A6 | `ErrNotFound` 사용 불일치 | `repository/workflow.go:12` vs 각 memory repo | Medium |
| A7 | `ConnectionResolver` 미사용 | `agents/registry.go:22-24` | Low |

### Category B: 관심사 분리 위반 (5건)

| # | 이슈 | 위치 | 심각도 |
|---|------|------|--------|
| B1 | LLM 프롬프트 Go 코드 인라인 | `generate/thumbnail.go:95-101,159-164`, `agents/output_extract.go:45-57`, `services/content_collector.go:574-577` | High |
| B2 | HTTP 핸들러에 비즈니스 로직 (7개) | `api/configure.go`, `api/name.go`, `api/content.go`, `api/generate.go`, `api/pipelines.go`, `api/triggers.go` | High |
| B3 | 모델 카탈로그 API 레이어 정의 | `api/models.go:1-292` | Medium |
| B4 | 하드코딩된 설정값 (15+) | `main.go`, `services/`, `api/`, `model/` 전반 | Medium |
| B5 | 문자열 기반 에러 체크 | `api/content.go` (10회+), `model/gemini_text.go:126-128` | Medium |

### Category C: 데이터 흐름 단절 (10건)

| # | 이슈 | 위치 | 심각도 |
|---|------|------|--------|
| C1 | `ExecutionRegistry` 미등록 (resume dead code) | `services/execution.go`, `api/run.go`, `services/run/publisher.go` | High |
| C2 | 토큰 사용량 미기록 | `services/workflow.go:246-296` → `services/run/publisher.go:103-125` | Medium |
| C3 | `GenerateRequest.Model` 무시됨 | `api/generate.go:21,101-103` | Low |
| C4 | 웹훅 입력 파이프라인 미전달 | `api/webhooks.go:60,75` | High |
| C5 | 파이프라인 워크플로우 run history 미기록 | `services/stage_workflow.go:54,59-61` | High |
| C6 | Pipeline 컨텍스트/모델 → 스테이지 미전달 | `services/pipeline_runner.go:20,107` | Medium |
| C7 | `ContentSessionDetail.PipelineName` 미채움 | `services/content_session_service.go:393-406` | Low |
| C8 | `WorkflowResults` 인메모리 전용 | `services/content_session_service.go:22-23` | High |
| C9 | `userID := "default"` 하드코딩 | `services/workflow.go:110` | High |
| C10 | 스케줄러 콘텐츠 수집 라이프사이클 우회 | `services/scheduler.go` → `services/pipeline_runner.go` vs `services/content_collector.go` | High |

## Approach

Bottom-up by layer. 기반 타입을 먼저 정비하고 상위 레이어를 순차적으로 수정한다.

## Phase 1: 도메인 타입 정비

영향받는 이슈: A2, A3, A4, A5, A6, A7

### 1a. Typed Status Enums

`internal/upal/pipeline.go`에 추가:

```go
type PipelineRunStatus string
const (
    PipelineRunWaiting   PipelineRunStatus = "waiting"
    PipelineRunRunning   PipelineRunStatus = "running"
    PipelineRunCompleted PipelineRunStatus = "completed"
    PipelineRunFailed    PipelineRunStatus = "failed"
    PipelineRunRejected  PipelineRunStatus = "rejected"
)

type StageStatus string
const (
    StageStatusPending   StageStatus = "pending"
    StageStatusRunning   StageStatus = "running"
    StageStatusCompleted StageStatus = "completed"
    StageStatusFailed    StageStatus = "failed"
    StageStatusSkipped   StageStatus = "skipped"
)
```

`internal/upal/content.go`에 추가:

```go
type ContentSessionStatus string
const (
    SessionPending       ContentSessionStatus = "pending"
    SessionCollecting    ContentSessionStatus = "collecting"
    SessionAnalyzing     ContentSessionStatus = "analyzing"
    SessionPendingReview ContentSessionStatus = "pending_review"
    SessionApproved      ContentSessionStatus = "approved"
    SessionProducing     ContentSessionStatus = "producing"
    SessionCompleted     ContentSessionStatus = "completed"
    SessionFailed        ContentSessionStatus = "failed"
    SessionArchived      ContentSessionStatus = "archived"
)

type WorkflowResultStatus string
const (
    WFResultPending WorkflowResultStatus = "pending"
    WFResultRunning WorkflowResultStatus = "running"
    WFResultSuccess WorkflowResultStatus = "success"
    WFResultFailed  WorkflowResultStatus = "failed"
)
```

`PipelineRun.Status` → `PipelineRunStatus`, `StageResult.Status` → `StageStatus`, `ContentSession.Status` → `ContentSessionStatus`, `WorkflowResult.Status` → `WorkflowResultStatus`로 필드 타입 변경.

서비스/API 전반의 문자열 리터럴을 상수 참조로 교체.

### 1b. ErrNotFound 통일

`internal/repository/errors.go` 생성:

```go
package repository

import "errors"

var ErrNotFound = errors.New("not found")
```

기존 `workflow.go`의 `ErrNotFound` 삭제. 모든 memory/persistent repo에서 wrapping 사용:

```go
return nil, fmt.Errorf("pipeline %q: %w", id, ErrNotFound)
```

API 레이어: `strings.Contains(err.Error(), "not found")` → `errors.Is(err, repository.ErrNotFound)`.

### 1c. 공유 DTO (`internal/upal/`)

```go
// internal/upal/model_summary.go
type ModelSummary struct {
    ID       string `json:"id"`
    Category string `json:"category"`
    Tier     string `json:"tier,omitempty"`
    Hint     string `json:"hint,omitempty"`
}

type ToolSummary struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}
```

`generate.ModelOption` → `upal.ModelSummary`, `generate.ToolEntry` → `upal.ToolSummary`로 교체. `main.go`의 변환 코드 단순화.

### 1d. `SchedulerConfig` → `ConcurrencyLimits` 통합

`config.Config`에서 `Scheduler config.SchedulerConfig` → `Scheduler upal.ConcurrencyLimits` 변경.

`main.go`의 필드별 복사 제거, `cfg.Scheduler`를 직접 전달.

### 1e. Dead Code 정리

- `agents/registry.go:22-24` — `ConnectionResolver` 삭제
- `api/server.go:263-266` — 로컬 `toolInfo` struct 삭제, `tools.ToolInfo`에 JSON tag 추가
- `services/runmanager.go:9-14` — `EventRecord`가 `WorkflowEvent` embed:

```go
type EventRecord struct {
    upal.WorkflowEvent
    Seq int `json:"seq"`
}
```

`run/publisher.go`의 필드별 변환 → 직접 할당으로 단순화.

---

## Phase 2: 모델 해석 통합

영향받는 이슈: A1, B3

### 2a. `LLMResolver` 단일화

`ports.LLMResolver` + `llmutil.MapResolver`는 이미 존재. 중복 구현 제거:

**삭제:**
- `agents/builders.go` — `resolveLLM()` 함수
- `api/server.go` — `resolveModel()` 메서드 + `resolvedModel` struct
- `services/workflow.go:77` — 인라인 `SplitN` 검증 (이미 LLMResolver 사용 시)

**주입 추가:**
- `BuildDeps` struct에 `LLMResolver ports.LLMResolver` 필드 추가
- `api.Server` struct에 `llmResolver ports.LLMResolver` 필드 추가
- `main.go`에서 동일한 `MapResolver` 인스턴스를 양쪽에 주입

노드 빌더: `deps.LLMs[provider]` 직접 접근 → `deps.LLMResolver.Resolve(modelID)` 사용.
API 핸들러: `s.resolveModel(modelID)` → `s.llmResolver.Resolve(modelID)` 사용.

### 2b. 모델 카탈로그 도메인 이동

**도메인 타입** (`internal/upal/models.go`) — Phase 1의 `ModelSummary` 확장:

```go
type ModelInfo struct {
    ID            string         `json:"id"`
    Provider      string         `json:"provider"`
    Name          string         `json:"name"`
    Category      ModelCategory  `json:"category"`
    Tier          ModelTier      `json:"tier,omitempty"`
    Hint          string         `json:"hint,omitempty"`
    Options       []OptionSchema `json:"options,omitempty"`
    SupportsTools bool           `json:"supportsTools"`
}

type ModelCategory string
const (
    ModelCategoryText  ModelCategory = "text"
    ModelCategoryImage ModelCategory = "image"
    ModelCategoryTTS   ModelCategory = "tts"
)

type ModelTier string
const (
    ModelTierHigh ModelTier = "high"
    ModelTierMid  ModelTier = "mid"
    ModelTierLow  ModelTier = "low"
)
```

**카탈로그 데이터** (`internal/model/catalog.go`):

```go
func AnthropicModels() []upal.ModelInfo { ... }
func OpenAIModels() []upal.ModelInfo { ... }
func GeminiModels() []upal.ModelInfo { ... }
func AllStaticModels() []upal.ModelInfo { ... }
```

기존 `api/models.go`의 모델 목록 데이터를 `model/catalog.go`로 이동. `api/models.go`는 HTTP 핸들러만 남김.

Ollama 동적 탐색 로직은 `model/ollama_discovery.go`로 분리, 반환 타입을 `[]upal.ModelInfo`로 통일.

`generate` 패키지는 `upal.ModelSummary` 사용 (Phase 1에서 정의). `upal.ModelInfo` → `upal.ModelSummary` 변환은 `main.go`에서 한 줄 매핑.

---

## Phase 3: 관심사 분리

영향받는 이슈: B1, B2, B4, B5

### 3a. 핸들러 → 서비스 추출

| 핸들러 | 현재 위치 | 추출 위치 | 메서드 시그니처 |
|--------|----------|-----------|----------------|
| `configureNode` | `api/configure.go:50-190` | `generate/configure.go` | `Generator.ConfigureNode(ctx, wf, nodeID, model string) (*ConfigureResult, error)` |
| `suggestWorkflowName` | `api/name.go:28-101` | `generate/name.go` | `Generator.SuggestName(ctx, wf) (string, error)` |
| `publishContentSession` | `api/content.go:222-280` | `services/content_session_service.go` | `ContentSessionService.PublishSession(ctx, sessionID, selectedItems) error` |
| `backfillDescriptions` | `api/generate.go:168-216` | `generate/backfill.go` | 기존 `Backfill` struct에 `BackfillWorkflow(ctx, wf) error` 추가 |
| `rejectPipelineRun` | `api/pipelines.go:202-235` | `services/pipeline_service.go` | `PipelineService.RejectRun(ctx, pipelineID, runID, reason) error` |
| `createTrigger` | `api/triggers.go:16-56` | `services/trigger_service.go` | `TriggerService.CreateTrigger(ctx, req) (*Trigger, error)` |
| `listContentSessions` | `api/content.go:18-75` | `services/content_session_service.go` | 기존 `ListSessionDetails`에 `ListFilter` 파라미터 추가 |

핸들러 패턴 (변경 후):
```go
func (s *Server) configureNode(w http.ResponseWriter, r *http.Request) {
    var req ConfigureNodeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    result, err := s.generator.ConfigureNode(r.Context(), req.Workflow, req.NodeID, req.Model)
    if err != nil {
        handleError(w, err)
        return
    }
    json.NewEncoder(w).Encode(result)
}
```

### 3b. 인라인 프롬프트 → 스킬

`skill-prompt-extraction` 커밋에서 시스템 프롬프트 4개 추출 완료. 남은 user prompt / format 지시:

| 코드 위치 | 스킬 파일 | 플레이스홀더 |
|-----------|----------|-------------|
| `thumbnail.go:95-101` | `prompts/thumbnail-user.md` | `{{workflow_name}}`, `{{description}}`, `{{nodes}}`, `{{agent_context}}` |
| `thumbnail.go:159-164` | 동일 파일 내 pipeline 섹션 | `{{pipeline_name}}`, `{{stages}}`, `{{description}}` |
| `output_extract.go:45-57` | `_frameworks/output-extract.md` | `{{key}}` or `{{tag}}` |
| `content_collector.go:574-577` | `prompts/angle-workflow.md` | `{{format}}`, `{{headline}}` |

Go 코드에서 `skills.GetPrompt("thumbnail-user")` 호출 후 template 치환.

### 3c. 설정값 → Config

`internal/config/config.go`에 섹션 추가:

```go
type Config struct {
    Server    ServerConfig             `yaml:"server"`
    Database  DatabaseConfig           `yaml:"database"`
    Providers map[string]ProviderConfig `yaml:"providers"`
    Scheduler upal.ConcurrencyLimits   `yaml:"scheduler"` // Phase 1에서 변경
    Runs      RunsConfig               `yaml:"runs"`
    Generator GeneratorConfig          `yaml:"generator"`
}

type ServerConfig struct {
    Port           int    `yaml:"port"`
    UploadMaxSize  int64  `yaml:"upload_max_size"`  // default: 50MB
    PaginationLimit int   `yaml:"pagination_limit"` // default: 50
}

type RunsConfig struct {
    TTL        time.Duration `yaml:"ttl"`         // default: 5m
    GCInterval time.Duration `yaml:"gc_interval"` // default: 1m
}

type GeneratorConfig struct {
    ThumbnailTimeout time.Duration `yaml:"thumbnail_timeout"` // default: 60s
    AnalysisTimeout  time.Duration `yaml:"analysis_timeout"`  // default: 5m
}

type ProviderConfig struct {
    APIKey        string `yaml:"api_key"`
    DefaultMaxTokens int `yaml:"default_max_tokens"` // anthropic: 4096
}
```

각 사용처에서 주입받아 사용. 기본값은 config 로딩 시 설정.

### 3d. 문자열 에러 체크 → `errors.Is`

Phase 1의 `ErrNotFound` 통일 이후, `api/content.go`의 10개+ `strings.Contains` 패턴을 `errors.Is(err, repository.ErrNotFound)` 로 전수 교체.

추가 sentinel errors:
```go
// internal/upal/errors.go
var (
    ErrSessionNotReviewable = errors.New("session is not in reviewable state")
    ErrRunNotApprovalStage  = errors.New("run is not in approval stage")
    ErrInvalidTransition    = errors.New("invalid status transition")
)
```

`services` 레이어에서 sentinel error 반환, API 레이어에서 `errors.Is` 로 HTTP status 매핑.

---

## Phase 4: 데이터 흐름 복원

영향받는 이슈: C1, C2, C3, C4, C5, C6, C7, C8, C9, C10

### 4a. ExecutionRegistry 활성화

`RunPublisher`에 `ExecutionRegistry` 주입:

```go
type RunPublisher struct {
    workflowSvc  ports.WorkflowExecutor
    runHistory   *RunHistoryService
    runManager   *RunManager
    executionReg *ExecutionRegistry  // 추가
}
```

`Launch()` 수정:
```go
func (p *RunPublisher) Launch(ctx context.Context, runID string, wf *upal.WorkflowDefinition, inputs map[string]string) {
    handle := p.executionReg.Register(runID)
    defer p.executionReg.Unregister(runID)
    // ... 기존 실행 로직
}
```

### 4b. 토큰 사용량 기록

`RunPublisher.trackNodeRun()` 수정:

```go
case services.EventNodeCompleted:
    rec.Status = "completed"
    if tokens, ok := ev.Payload["tokens"].(map[string]any); ok {
        rec.Usage = &upal.TokenUsage{
            PromptTokens:     toInt(tokens["prompt"]),
            CompletionTokens: toInt(tokens["completion"]),
            TotalTokens:      toInt(tokens["total"]),
        }
    }
```

`Launch()` 완료 시 합산:
```go
totalUsage := aggregateNodeUsage(nodeRecords)
p.runHistory.CompleteRun(ctx, runID, state, totalUsage)
```

### 4c. 웹훅 입력 → 파이프라인

`PipelineRunner.Start()` 시그니처 변경:

```go
func (r *PipelineRunner) Start(ctx context.Context, pipeline *upal.Pipeline, inputs map[string]any) error
```

`webhooks.go` 수정:
```go
s.pipelineRunner.Start(context.Background(), pipeline, inputs)
```

`executeFrom()`:
```go
func (r *PipelineRunner) executeFrom(ctx context.Context, pipeline *upal.Pipeline, startIdx int, run *upal.PipelineRun, inputs map[string]any) {
    var prevResult *upal.StageResult
    if inputs != nil {
        prevResult = &upal.StageResult{Output: inputs}
    }
    // ...
}
```

기존 호출자(스케줄러 등)는 `nil` 전달.

### 4d. 파이프라인 워크플로우 → RunPublisher

`WorkflowStageExecutor`에 `RunPublisher` 주입:

```go
type WorkflowStageExecutor struct {
    workflowSvc  ports.WorkflowExecutor
    workflowRepo ports.WorkflowRepository
    publisher    *run.RunPublisher  // 추가
}
```

`Execute()` 내 직접 `workflowSvc.Run()` 호출 → `publisher.Launch()` 사용:

```go
func (e *WorkflowStageExecutor) Execute(ctx context.Context, pipeline *upal.Pipeline, stage upal.Stage, prev *upal.StageResult) (*upal.StageResult, error) {
    // ...
    runID := uuid.New().String()
    e.publisher.Launch(ctx, runID, wf, inputs)
    // publisher가 이벤트 스트리밍, 토큰 추적, 에러 히스토리 처리
}
```

### 4e. Pipeline 컨텍스트 → 스테이지

`StageExecutor` 인터페이스 확장:

```go
type StageExecutor interface {
    Execute(ctx context.Context, pipeline *upal.Pipeline, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error)
}
```

모든 executor 구현체 시그니처 수정:
- `CollectStageExecutor`
- `WorkflowStageExecutor`
- `ApprovalStageExecutor`
- `TransformStageExecutor`
- `NotifyStageExecutor`

`PipelineRunner.executeFrom()`에서 `executor.Execute(ctx, pipeline, stage, prevResult)` 호출.

### 4f. WorkflowResults 영속화

`internal/upal/ports/content.go`에 인터페이스 추가:

```go
type WorkflowResultRepository interface {
    Save(ctx context.Context, sessionID string, results []upal.WorkflowResult) error
    GetBySession(ctx context.Context, sessionID string) ([]upal.WorkflowResult, error)
    DeleteBySession(ctx context.Context, sessionID string) error
}
```

구현:
- `repository/workflow_result_memory.go` — 인메모리 (기존 map 로직 이동)
- `repository/workflow_result_persistent.go` — PostgreSQL (workflow_results 테이블)

`ContentSessionService`의 인메모리 `workflowResults` map 제거 → repository 사용.

DB 스키마:
```sql
CREATE TABLE IF NOT EXISTS workflow_results (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES content_sessions(id) ON DELETE CASCADE,
    workflow_id TEXT NOT NULL,
    run_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    output JSONB,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_workflow_results_session ON workflow_results(session_id);
```

### 4g. 필드 채우기

**`ContentSessionDetail.PipelineName`:**
`ContentSessionService`에 `PipelineRepository` 주입. `GetSessionDetail()`에서:
```go
if sess.PipelineID != "" {
    if p, err := s.pipelineRepo.Get(ctx, sess.PipelineID); err == nil {
        detail.PipelineName = p.Name
    }
}
```

**`Pipeline.PendingSessionCount`:**
API list/get 핸들러에서 세션 카운트 쿼리 후 설정:
```go
count, _ := s.contentSessionSvc.CountByStatus(ctx, pipeline.ID, upal.SessionPendingReview)
pipeline.PendingSessionCount = count
```

**`RunRecord.SessionID`:**
`ContentCollector.ProduceWorkflows()`에서 `RunPublisher.Launch()` 사용 시 sessionID 전달 (RunPublisher에 sessionID 파라미터 추가 또는 context로 전달).

### 4h. userID 전파 구조

당장 인증 시스템 전체 구현은 범위 밖. 전파 구조만 준비:

```go
// internal/upal/context.go
type contextKey string
const userIDKey contextKey = "userID"

func WithUserID(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) string {
    if v, ok := ctx.Value(userIDKey).(string); ok {
        return v
    }
    return "default"
}
```

`WorkflowService.Run()`에서 `userID := upal.UserIDFromContext(ctx)` 사용. API 미들웨어에서 인증 헤더 파싱 후 `upal.WithUserID(ctx, userID)` 설정 (stub으로 "default" 유지).

### 4i. 스케줄러 → 콘텐츠 수집 통합

`SchedulerService`에 `ContentCollector` 주입:

```go
type SchedulerService struct {
    // ...
    pipelineRunner   *PipelineRunner
    contentCollector *ContentCollector  // 추가
}
```

스케줄 디스패치 시 분기:
```go
func (s *SchedulerService) dispatchPipeline(ctx context.Context, pipeline *upal.Pipeline) {
    if len(pipeline.Sources) > 0 && s.contentCollector != nil {
        // 콘텐츠 파이프라인: 수집 → 분석 → 리뷰 큐
        go s.contentCollector.CollectAndAnalyze(ctx, pipeline, sess, false, 0)
    } else {
        // 일반 파이프라인: 스테이지 순차 실행
        go s.pipelineRunner.Start(ctx, pipeline, nil)
    }
}
```

`main.go`에서 `scheduler.SetContentCollector(collector)` 추가.

---

## Phase Dependencies

```
Phase 1 (도메인 타입) ──┬──→ Phase 2 (모델 통합)
                        │
                        └──→ Phase 3 (관심사 분리) ──→ Phase 4 (데이터 흐름)
```

Phase 2와 Phase 3는 Phase 1 완료 후 병렬 가능. Phase 4는 Phase 3의 서비스 추출 완료 후 진행.

## Not In Scope

- 프런트엔드 변경 (API 응답 형태가 바뀌는 경우 최소한의 타입 업데이트만)
- 인증/인가 시스템 구현 (userID 전파 구조만 준비)
- 프롬프트 내용 개선 (구조만 분리, 텍스트는 그대로 이동)
- 데이터베이스 마이그레이션 도구 도입 (SQL 파일만 추가)
- `context.Background()` → detached context 유틸리티 (별도 작업)

## Testing Strategy

- 각 Phase 완료 후 `make test` 통과 확인
- Phase 1: 타입 변경으로 인한 컴파일 에러 전수 수정 → 기존 테스트 통과
- Phase 2: `resolver_test.go` 확장, 기존 빌더 테스트 수정
- Phase 3: 추출된 서비스 메서드에 대한 unit test 추가
- Phase 4: integration test — 웹훅 → 파이프라인 → 워크플로우 → run history 확인