# Project-Wide Code Simplification Design

## Goal

프로젝트 전체 코드를 순차적으로 심플화하여 유지보수성, 일관성을 확보하고 기술 부채를 청산한다.

## Approach

- **Bottom-Up**: 도메인 타입(안쪽) → UI 페이지(바깥쪽) 순서
- **Batch unit**: 도메인 레이어 단위 (관련 파일 묶음)
- **Executor**: 매 배치마다 `Task tool` + `subagent_type: "code-simplifier:code-simplifier"`로 fresh agent 실행
- **Context isolation**: 배치마다 새 Task를 생성하여 에이전트 컨텍스트 포화 방지

## Workflow per Batch

```
1. 대상 파일 목록 정의
2. Task tool → code-simplifier agent 실행 (파일 목록 + 심플화 지침 전달)
3. 에이전트가 심플화 수행
4. 빌드/타입체크 확인 (go build ./... / npx tsc -b)
5. 컴파일 오류 시 수정
6. 커밋
```

## Phase 1: Backend Core (도메인 → ports → repository)

### Batch 1-1: Domain Types (`internal/upal/`)

Target files:
- `internal/upal/content.go`
- `internal/upal/workflow.go`
- `internal/upal/pipeline.go`
- `internal/upal/node.go`
- `internal/upal/types.go`
- 기타 `internal/upal/*.go` (ports/ 제외)

Focus: 타입/상수 정리, 불필요한 필드 제거, 네이밍 일관성

### Batch 1-2: Port Interfaces (`internal/upal/ports/`)

Target files:
- `internal/upal/ports/*.go`

Focus: ContentSessionPort와 ContentSessionRepository 간 이중 추상화 검토, 단순 위임 메서드 정리, 네이밍 일관성

### Batch 1-3: Repository (`internal/repository/`)

Target files:
- `internal/repository/content*.go`
- `internal/repository/pipeline*.go`
- `internal/repository/workflow*.go`
- 기타 `internal/repository/*.go`

Focus: memory/persistent 쌍 패턴 일관성, 제네릭 store 활용도, 보일러플레이트 축소

## Phase 2: Backend Business (services → API)

### Batch 2-1: Content Services

Target files:
- `internal/services/content_session_service.go` (663 lines, 53 methods)
- `internal/services/content_collector.go` (977 lines — project largest)
- `internal/services/fetcher_research.go` (374 lines)
- `internal/services/fetcher_social.go` (341 lines)

Focus: content_collector 분할, 단순 위임 메서드 정리, fetcher 공통 패턴 추출

### Batch 2-2: Other Services

Target files:
- `internal/services/pipeline_*.go`
- `internal/services/stage_*.go`
- `internal/services/run/`
- `internal/services/scheduler/`
- 기타 `internal/services/*.go`

Focus: 서비스 간 일관성, 에러 처리 패턴 통일

### Batch 2-3: API Handlers

Target files:
- `internal/api/content.go` (809 lines, 25 handlers)
- `internal/api/pipeline.go`
- `internal/api/workflow.go`
- 기타 `internal/api/*.go`

Focus: content.go 하위 도메인별 분할 검토, HTTP 보일러플레이트 통일

## Phase 3: Frontend Foundation (shared → entities)

### Batch 3-1: Shared Layer

Target files:
- `web/src/shared/hooks/*.ts`
- `web/src/shared/lib/*.ts`
- `web/src/shared/ui/*.tsx`
- `web/src/shared/api/*.ts`

Focus: 미사용 훅/유틸 제거, API 클라이언트 일관성

### Batch 3-2: Entities Layer

Target files:
- `web/src/entities/content-session/*.ts`
- `web/src/entities/workflow/**/*.ts`
- `web/src/entities/pipeline/**/*.ts`
- 기타 `web/src/entities/*/`

Focus: 데이터 페칭 로직 엔티티 훅으로 통합, store 정리, barrel export 일관성

## Phase 4: Frontend UI (features → widgets → pages)

### Batch 4-1: Features + Widgets

Target files:
- `web/src/features/manage-canvas/model/useAutoSave.ts` (migration leftover)
- `web/src/widgets/pipeline-editor/ui/StageCard.tsx` (588 lines)
- `web/src/features/*/`
- `web/src/widgets/*/`

Focus: 마이그레이션 잔여물 제거, 큰 컴포넌트 분할 검토, features/widgets 책임 경계

### Batch 4-2: Pages — Pipelines/Inbox

Target files:
- `web/src/pages/pipelines/session/SessionSetupView.tsx` (680 lines)
- `web/src/pages/pipelines/SessionListPanel.tsx`
- `web/src/pages/pipelines/PipelineSidebar.tsx`
- `web/src/pages/pipelines/session/stages/AnalyzeStage.tsx`
- `web/src/pages/inbox/*.tsx`
- `web/src/pages/Pipelines.tsx`

Focus: useQuery를 엔티티 훅으로 내리기, 컴포넌트 분리, 페이지 간 일관성

### Batch 4-3: Pages — Others

Target files:
- `web/src/pages/pipelines/PipelineNew.tsx` (554 lines)
- `web/src/pages/runs/RunNodePanel.tsx` (506 lines)
- `web/src/pages/Connections.tsx` (329 lines)
- `web/src/pages/workflows/index.tsx` (432 lines)
- `web/src/pages/landing/ProductLanding.tsx` (re-export cleanup)
- 기타 `web/src/pages/*/`

Focus: 큰 페이지 컴포넌트 정리, 중복 re-export 제거

## Phase 5: Cleanup

### Batch 5-1: Final Pass

- 죽은 코드/미사용 export 탐지 및 제거
- re-export 파일 정리
- CLAUDE.md 컨벤션과 최종 일치 확인

## Execution Notes

- 각 배치는 독립된 `Task` 호출로 실행하여 에이전트 컨텍스트 포화 방지
- 배치 간 의존: Phase 순서를 지키되, 같은 Phase 내 배치는 순서대로 진행
- 배치 완료 후 반드시 `go build ./...` + `npx tsc -b` 통과 확인
- 컴파일 오류 발생 시 메인 컨텍스트에서 수정 후 다음 배치 진행
- 총 12 배치, 각 배치 커밋은 `refactor: simplify {target description}` 형식
