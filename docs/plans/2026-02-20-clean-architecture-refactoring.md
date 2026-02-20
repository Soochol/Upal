# Clean Architecture 리팩토링 계획

> 작성: 2026-02-20
> 목표: 가독성, 확장성, 수정 용이성을 확보하는 Clean Architecture 기반 구조 확립

---

## 전체 진행 현황

| Task | 제목 | 상태 | 영향도 |
|------|------|------|--------|
| T1 | 백엔드 서비스 레이어 도입 | ✅ | CRITICAL |
| T2 | 백엔드 인터페이스 추출 | ✅ | HIGH |
| T3 | DAG 실행 로직 분리 (ADK 디커플링) | ⬜ | HIGH |
| T4 | Server God Object 분할 | ⬜ | MEDIUM |
| T5 | 프론트엔드 서비스 레이어 도입 | ⬜ | HIGH |
| T6 | 프론트엔드 스토어 디커플링 | ⬜ | MEDIUM |
| T7 | 프론트엔드 타입 안전성 강화 | ⬜ | MEDIUM |
| T8 | Editor.tsx 분할 | ⬜ | LOW-MEDIUM |

---

## T1: 백엔드 서비스 레이어 도입 (CRITICAL)

### 문제
HTTP 핸들러(`run.go` 281줄)가 DAG 생성, ADK 러너 초기화, 세션 관리, SSE 스트리밍을 직접 오케스트레이션한다.
비즈니스 로직이 HTTP 레이어에 산재하여 테스트/재사용 불가.

### 목표
`internal/services/workflow.go`에 `WorkflowService`를 만들어 비즈니스 오케스트레이션을 캡슐화.

### 체크리스트

- [x] **T1.1** `internal/services/` 패키지 생성
- [x] **T1.2** `WorkflowService` 구조체 정의 (Lookup, Validate, Run 메서드)
- [x] **T1.3** `WorkflowEvent`, `RunResult` 도메인 이벤트 타입 정의
- [x] **T1.4** `run.go`의 오케스트레이션 로직을 `WorkflowService.Run()`으로 이동
- [x] **T1.5** `run.go` 핸들러를 thin wrapper로 리팩토링 (281줄 → 96줄)
- [x] **T1.6** `classifyEvent`, `validateWorkflowForRun` 등을 서비스로 이동
- [x] **T1.7** `a2a.go`도 `WorkflowService` 사용하도록 리팩토링 (중복 제거)
- [x] **T1.8** 서비스 레이어 단위 테스트 작성 (7개 테스트)
- [x] **T1.9** 전체 테스트 통과 확인 (`go test ./... -race`)

### 변경 파일
- 신규: `internal/services/workflow.go`, `internal/services/workflow_test.go`
- 수정: `internal/api/run.go`, `internal/api/a2a.go`, `internal/api/server.go`, `cmd/upal/main.go`
- 수정: `internal/api/run_test.go`, `server_test.go`, `a2a_test.go` (시그니처 변경 반영)

---

## T2: 백엔드 인터페이스 추출 (HIGH)

### 문제
콘크리트 타입 직접 참조로 인해 교체/테스트 불가:
- `*db.DB` 직접 사용
- `map[string]adkmodel.LLM` 직접 전달
- `*skills.Registry` 콘크리트 타입

### 목표
각 어댑터에 인터페이스를 정의하여 의존성 역전 원칙(DIP) 적용.

### 체크리스트

- [x] **T2.1** `WorkflowRepository` 인터페이스 생성 (`internal/repository/workflow.go`)
- [x] **T2.2** 인메모리 구현체 (`internal/repository/memory.go`) — 기존 `WorkflowStore` 대체
- [x] **T2.3** `PersistentRepository` 구현체 (`internal/repository/persistent.go`) — dual-write 캡슐화
- [x] **T2.4** `SkillProvider` 인터페이스 생성 (`internal/skills/provider.go`)
- [x] **T2.5** `api/server.go`의 `*db.DB` + `*WorkflowStore` → `WorkflowRepository` 교체
- [x] **T2.6** `services/workflow.go`의 `WorkflowLookup` + `*db.DB` → `WorkflowRepository` 교체
- [x] **T2.7** `generate.Generator`의 `*skills.Registry` → 로컬 `skillProvider` 인터페이스 교체
- [x] **T2.8** `api/workflow.go`에서 `WorkflowStore` 타입 제거, CRUD 핸들러 단순화
- [x] **T2.9** 전체 테스트 통과 확인 (`go test ./... -race`)

### 변경 파일
- 신규: `internal/repository/workflow.go`, `memory.go`, `persistent.go`, `internal/skills/provider.go`
- 수정: `internal/api/server.go`, `internal/api/workflow.go`, `internal/api/a2a.go`, `internal/api/a2a_test.go`
- 수정: `internal/services/workflow.go`, `internal/services/workflow_test.go`
- 수정: `internal/generate/generate.go`, `cmd/upal/main.go`
- 제거: `api.WorkflowStore` 타입, `api.NewWorkflowStore()`, `services.WorkflowLookup` 인터페이스, `Server.SetDB()`

---

## T3: DAG 실행 로직 분리 — ADK 디커플링 (HIGH)

### 문제
`agents/dag.go`에서 순수 DAG 실행 로직(토폴로지 정렬, goroutine fan-out/fan-in, 채널 동기화)이
ADK `agent.Agent` 생성 코드 안에 묻혀 있다. ADK 없이 DAG 실행 테스트 불가.

### 목표
순수 `DAGExecutor`와 ADK 어댑터를 분리.

### 체크리스트

- [ ] **T3.1** `NodeAgent` 로컬 인터페이스 정의 (ADK 무관)
  ```go
  type NodeAgent interface {
      ID() string
      Run(ctx context.Context, state map[string]any) (any, error)
  }
  ```
- [ ] **T3.2** `DAGExecutor` 구현 — 순수 Go (ADK import 없음)
  - 토폴로지 순서 실행
  - goroutine fan-out + channel 동기화
  - `NodeAgent` 인터페이스로 노드 실행
- [ ] **T3.3** `agents/adk_adapter.go` — `DAGExecutor` 결과를 ADK `agent.Agent`로 래핑
- [ ] **T3.4** `BuildAgent()` → `NodeAgent` 인터페이스 반환하도록 수정
- [ ] **T3.5** 기존 `NewDAGAgent()`가 내부적으로 `DAGExecutor` + 어댑터 사용하도록 변경
- [ ] **T3.6** `DAGExecutor` 단위 테스트 (mock `NodeAgent` 사용)
- [ ] **T3.7** 통합 테스트 통과 확인

### 변경 파일
- 신규: `internal/agents/executor.go`, `internal/agents/adk_adapter.go`, `internal/agents/executor_test.go`
- 수정: `internal/agents/dag.go`, `internal/agents/builders.go`

---

## T4: Server God Object 분할 (MEDIUM)

### 문제
`Server` 구조체가 11개 필드를 가진 God Object.
모든 핸들러가 `Server`의 모든 의존성에 접근 가능.

### 목표
도메인별 핸들러로 분할하여 단일 책임 원칙(SRP) 적용.

### 체크리스트

- [ ] **T4.1** `internal/api/handlers/` 패키지 생성
- [ ] **T4.2** `WorkflowHandler` 추출 — CRUD 핸들러 (workflows만 의존)
- [ ] **T4.3** `RunHandler` 추출 — 실행 핸들러 (WorkflowService 의존)
- [ ] **T4.4** `ModelHandler` 추출 — 모델 조회 핸들러
- [ ] **T4.5** `GenerateHandler` 추출 — 워크플로우 생성 핸들러
- [ ] **T4.6** `server.go`를 thin 라우터 설정으로 축소
- [ ] **T4.7** 각 핸들러별 테스트 작성
- [ ] **T4.8** `make test` 통과 확인

### 변경 파일
- 신규: `internal/api/handlers/workflow.go`, `run.go`, `model.go`, `generate.go`
- 수정: `internal/api/server.go`
- 삭제/이동: `internal/api/run.go`, `internal/api/workflow.go`, `internal/api/generate.go` 등

---

## T5: 프론트엔드 서비스 레이어 도입 (HIGH)

### 문제
API 호출이 5개 파일에 산재 (`useAutoSave`, `useExecuteRun`, `AIChatEditor`, `Editor`, `Landing`).
비즈니스 오케스트레이션이 컴포넌트/훅에 분산.

### 목표
`lib/services/`에 도메인별 서비스 훅을 만들어 API + 스토어 조작 캡슐화.

### 체크리스트

- [ ] **T5.1** `lib/services/` 디렉토리 생성
- [ ] **T5.2** `useWorkflowService()` 훅 생성
  ```typescript
  export function useWorkflowService() {
    return {
      save: async (name, nodes, edges) => { ... },
      load: async (name) => { ... },
      list: async () => { ... },
      delete: async (name) => { ... },
      generate: async (description) => { ... },
    }
  }
  ```
- [ ] **T5.3** `useExecutionService()` 훅 생성 — `useExecuteRun` 로직 이동
- [ ] **T5.4** `useNodeConfigService()` 훅 생성 — `AIChatEditor`의 API + 스토어 로직 이동
- [ ] **T5.5** `Landing.tsx`가 `useWorkflowService()` 사용하도록 수정
- [ ] **T5.6** `Editor.tsx`가 서비스 훅 사용하도록 수정
- [ ] **T5.7** `AIChatEditor.tsx`가 `useNodeConfigService()` 사용하도록 수정
- [ ] **T5.8** `make test-frontend` 통과 확인

### 변경 파일
- 신규: `web/src/lib/services/workflowService.ts`, `executionService.ts`, `nodeConfigService.ts`
- 수정: `web/src/pages/Landing.tsx`, `Editor.tsx`, `components/panel/AIChatEditor.tsx`, `hooks/useExecuteRun.ts`

---

## T6: 프론트엔드 스토어 디커플링 (MEDIUM)

### 문제
`workflowStore`가 `useUIStore.getState().selectNode()`을 직접 호출.
스토어 간 결합으로 독립 테스트 불가, 순환 의존성 위험.

### 목표
스토어 간 직접 참조 제거. 이벤트 기반 or 콜백 기반 통신으로 교체.

### 체크리스트

- [ ] **T6.1** `lib/eventBus.ts` 생성 — 간단한 pub/sub
  ```typescript
  type Handler = (data: any) => void
  const bus = { subscribe, emit, unsubscribe }
  ```
- [ ] **T6.2** `workflowStore`에서 `useUIStore` import 제거
- [ ] **T6.3** 노드 삭제 시 `emit('nodes-removed', { ids })` 발행
- [ ] **T6.4** `uiStore` 초기화 시 `subscribe('nodes-removed', ...)` 등록
- [ ] **T6.5** 그룹 생성 시 직접 `selectNode` → 이벤트 발행으로 교체
- [ ] **T6.6** 스토어 간 import 없음 확인 (정적 분석)
- [ ] **T6.7** `make test-frontend` 통과 확인

### 변경 파일
- 신규: `web/src/lib/eventBus.ts`
- 수정: `web/src/stores/workflowStore.ts`, `web/src/stores/uiStore.ts`

---

## T7: 프론트엔드 타입 안전성 강화 (MEDIUM)

### 문제
`config: Record<string, unknown>`, `sessionState: Record<string, unknown>` 등 타입 없는 데이터.
백엔드 변경 시 런타임에서야 오류 발견.

### 목표
핵심 데이터에 구체적 타입 적용 + 런타임 검증 추가.

### 체크리스트

- [ ] **T7.1** `AgentNodeConfig`, `InputNodeConfig`, `OutputNodeConfig` 타입 정의
  ```typescript
  interface AgentNodeConfig {
    model?: string
    system_prompt?: string
    prompt?: string
    tools?: string[]
    description?: string
  }
  ```
- [ ] **T7.2** `NodeData.config`를 discriminated union으로 교체
- [ ] **T7.3** `SessionState` 타입 정의
  ```typescript
  type SessionState = Record<string, {
    output: string
    duration?: number
    error?: string
  }>
  ```
- [ ] **T7.4** `executionStore.sessionState` 타입 적용
- [ ] **T7.5** `RunEvent` 파싱에 zod 스키마 검증 추가 (optional)
- [ ] **T7.6** 정규식 기반 에러 파싱 → 구조화된 에러 코드로 교체
- [ ] **T7.7** `make test-frontend` 통과 확인

### 변경 파일
- 신규 or 수정: `web/src/lib/types.ts` (또는 기존 파일에 추가)
- 수정: `web/src/stores/workflowStore.ts`, `executionStore.ts`, `hooks/useExecuteRun.ts`, `lib/api.ts`

---

## T8: Editor.tsx 분할 (LOW-MEDIUM)

### 문제
`Editor.tsx`가 레이아웃 + 워크플로우 생성 + 오토세이브 + 키보드 단축키 + 상태 통합을 모두 담당.

### 목표
단일 책임 서브 컴포넌트로 분할.

### 체크리스트

- [ ] **T8.1** `EditorLayout.tsx` 추출 — 순수 레이아웃 (Header + Sidebar + Canvas + Panel + Console)
- [ ] **T8.2** 워크플로우 생성 로직을 서비스 훅으로 이동 (T5와 연계)
- [ ] **T8.3** `Editor.tsx`를 orchestrator로 축소 (훅 연결 + 레이아웃 렌더링만)
- [ ] **T8.4** `make test-frontend` 통과 확인

### 변경 파일
- 신규: `web/src/components/editor/EditorLayout.tsx`
- 수정: `web/src/pages/Editor.tsx`

---

## 의존성 그래프 (Task 간)

```
T1 (서비스 레이어) ← 독립 시작 가능
T2 (인터페이스)    ← 독립 시작 가능 (T1과 병렬 가능)
T3 (DAG 분리)     ← T1 완료 후 (서비스가 executor 사용)
T4 (Server 분할)  ← T1, T2 완료 후
T5 (FE 서비스)    ← 독립 시작 가능
T6 (스토어 분리)   ← 독립 시작 가능
T7 (타입 강화)    ← T5 완료 후 (서비스 계층에 타입 적용)
T8 (Editor 분할)  ← T5 완료 후
```

### 권장 실행 순서

```
Phase 1: T1 → T2 (백엔드 기반)
Phase 2: T3 → T4 (백엔드 심화)  ‖  T5 → T6 (프론트엔드 기반, 병렬 가능)
Phase 3: T7 → T8 (프론트엔드 마무리)
```
