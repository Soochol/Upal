# Session/Run Restructuring Design

Date: 2026-02-26

## Summary

Pipeline 개념을 제거하고 Session/Run 2계층 모델로 전환한다. Session은 설정(무엇을 수집하고 어떤 워크플로우를 돌릴지), Run은 실행(이번 수집/분석/워크플로우 결과)을 담당한다.

## Motivation

현재 구조의 문제:
1. **Pipeline이 얇은 래퍼**: 실제 설정 대부분(sources, schedule, model, workflows, context)이 ContentSession에 있음
2. **이중 오케스트레이션**: PipelineRun(stage 실행)과 ContentSession(collect→analyze→produce 플로우)이 병행
3. **template/instance가 Pipeline과 중복**: Pipeline이 이미 "템플릿" 역할인데 Session에도 template/instance가 있어 계층이 불필요하게 깊음

## Domain Model

### Entity Hierarchy

```
Session (설정)
  ├── sources[]        — 수집 소스 (RSS, Reddit, research 등)
  ├── schedule         — cron 표현식
  ├── model            — LLM 모델 ID
  ├── workflows[]      — 실행할 워크플로우 목록
  ├── context          — 프롬프트, 언어, 리서치 설정
  └── stages[]         — 오케스트레이션 단계 선언

Run (실행)
  ├── session_id       — FK → Session
  ├── status           — 상태 머신
  ├── trigger_type     — schedule | manual | surge
  ├── source_fetches[] — 수집 결과
  ├── analysis         — LLM 분석 결과
  └── workflow_runs[]  — 워크플로우 실행 결과
       ├── workflow_name
       ├── run_id      — 워크플로우 이벤트 스트림 연결
       ├── status      — pending | running | success | failed | published | rejected
       ├── channel_id
       └── output_url
```

### Removed Entities

- **Pipeline** → Session으로 흡수
- **PipelineRun** → Run 상태 머신이 대체
- **ContentSession** → Session + Run으로 분리

### State Machines

**Session**: `draft | active | archived`
- draft: 설정 편집 중
- active: 스케줄 활성화됨, Run 생성 가능
- archived: 비활성

**Run**: `collecting → analyzing → pending_review → approved → producing → published`
- `pending_review → rejected` (리젝트)
- `producing → error` (실패)
- ContentSession의 상태 머신과 동일하되 `active` 제거 (Session 레벨로 이동)

### Naming Mapping

| Current | New |
|---------|-----|
| Pipeline | Session |
| ContentSession (template) | Session |
| ContentSession (instance) | Run |
| WorkflowResult | WorkflowRun |
| PipelineRun | Removed (Run state machine replaces) |
| SourceFetch | SourceFetch (session_id → run_id) |
| LLMAnalysis | Analysis (session_id → run_id) |

### ID Prefixes

| Entity | Prefix |
|--------|--------|
| Session | `sess_` |
| Run | `run_` |
| WorkflowRun | `wrun_` |

## API Design

### Session Routes (설정 관리)

| Method | Route | Purpose |
|--------|-------|---------|
| `POST /api/sessions` | 세션 생성 |
| `GET /api/sessions` | 세션 목록 |
| `GET /api/sessions/{id}` | 세션 상세 |
| `PUT /api/sessions/{id}` | 세션 수정 |
| `DELETE /api/sessions/{id}` | 세션 삭제 |
| `POST /api/sessions/{id}/activate` | 스케줄 활성화 |
| `POST /api/sessions/{id}/deactivate` | 스케줄 비활성화 |
| `POST /api/sessions/{id}/configure` | AI 설정 생성 |
| `POST /api/sessions/{id}/thumbnail` | 썸네일 생성 |

### Run Routes (실행 관리)

| Method | Route | Purpose |
|--------|-------|---------|
| `POST /api/sessions/{id}/runs` | 실행 시작 |
| `GET /api/sessions/{id}/runs` | 세션의 실행 목록 |
| `GET /api/runs` | 전체 실행 목록 (Inbox) |
| `GET /api/runs/{id}` | 실행 상세 |
| `POST /api/runs/{id}/produce` | 워크플로우 실행 |
| `POST /api/runs/{id}/publish` | 발행 |
| `POST /api/runs/{id}/reject` | 리젝트 |
| `GET /api/runs/{id}/sources` | 수집 결과 |
| `GET /api/runs/{id}/analysis` | 분석 결과 |
| `PATCH /api/runs/{id}/analysis` | 분석 수정 |

### Removed Routes

- `/api/pipelines/*` 전체
- `/api/content-sessions/*` 전체

## Frontend Design

### Page Structure

| Page | Role | Change |
|------|------|--------|
| **Sessions** (← Pipelines) | 세션 목록 + 설정 편집 | 3단→2단 구조 단순화 |
| **Inbox** | 전체 Run 목록, 상태별 필터 | 데이터소스 변경 |
| **Runs** | 워크플로우 실행 이벤트 뷰어 | 현재와 동일 |
| **Published** | 발행된 콘텐트 | 현재와 동일 |

Sessions 페이지: 사이드바에 세션 목록, 선택 시 "설정 탭 / Run 히스토리 탭" 전환.

## Stage Orchestration

Run 상태 머신이 stage 역할을 흡수:
- collect stage → Run 상태 `collecting`
- analyze → `analyzing`
- approval → `pending_review`
- workflow → `producing`
- notification → 상태 전환 시 훅

Session에 `stages[]` 선언을 두어 어떤 단계를 거치는지 설정. Run 실행 시 참조.

삭제: PipelineRunner, PipelineRun 타입 & 리포지토리. StageExecutor들은 유지하되 RunService에서 직접 호출.

## Migration Strategy

인메모리 기본이므로 DB 마이그레이션 부담 낮음. PostgreSQL 사용 시:

1. sessions 테이블: pipelines + content_sessions(is_template=true) 병합
2. runs 테이블: content_sessions(is_template=false) 이동
3. source_fetches/llm_analyses: session_id → run_id rename
4. 구 테이블 삭제

## Backend Package Changes

| File/Package | Change |
|-------------|--------|
| `internal/upal/pipeline.go` | 삭제 → Session 타입 |
| `internal/upal/content.go` | ContentSession → Run. Session 타입 신규 |
| `internal/repository/pipeline_*.go` | 삭제 |
| `internal/repository/content_session_*.go` | session_repo + run_repo 분리 |
| `internal/services/pipeline_service.go` | 삭제 → SessionService |
| `internal/services/pipeline_runner.go` | 삭제 → RunService에 흡수 |
| `internal/services/content_session_service.go` | SessionService + RunService 분리 |
| `internal/services/content_collector.go` | RunService 호출로 변경 |
| `internal/api/server.go` | route 전면 교체 |
| `internal/api/` handlers | pipeline/content-session → session/run |
| `cmd/upal/main.go` | DI 와이어링 업데이트 |

## Frontend Package Changes

| Area | Change |
|------|--------|
| `entities/pipeline/` | 삭제 → `entities/session/` |
| `entities/content-session/` | Run 타입으로 리네임 |
| `pages/pipelines/` | → `pages/sessions/` (2단 구조) |
| `pages/inbox/` | API 엔드포인트 변경 |
| `shared/api/` | pipeline/content-session → session/run API |
| `features/configure-pipeline-sources/` | → `features/configure-session-sources/` |
| `features/generate-pipeline/` | → `features/generate-session/` |
| `widgets/pipeline-editor/` | → 세션 설정 위젯 |
