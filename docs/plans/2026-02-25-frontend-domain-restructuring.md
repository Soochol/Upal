# Frontend Domain Restructuring Plan

## Overview

프론트엔드 FSD(Feature-Sliced Design) 구조를 도메인별로 검증하고 개선한다. 각 도메인의 types/api/store/exports를 완결시키고, 중복을 제거하고, FSD 위반을 수정한다.

## Current State

| Entity | types | api | store | index | 문제 |
|--------|-------|-----|-------|-------|------|
| pipeline | shared에 산재 | 이중 정의 | 없음 | API만 | 가장 불완전 |
| run | shared에 산재 | 이중 정의 | 있음 | 있음 | types 부재 |
| connection | shared에 산재 | shared에만 | 없음 | entity 자체 없음 | 신규 필요 |
| workflow | 부분적 | 있음 | 있음 | 있음 | FSD 위반 1건 |
| content-session | 완전 | 완전 | 있음 | 있음 | 검증만 필요 |
| settings | 없음 | 없음 | 있음 | 없음 | barrel export 누락 |

## Approach

도메인별 수직 슬라이스: 한 도메인을 types → API → store → imports → 검증까지 완결하고 다음으로 넘어간다. shared/types에서 타입을 빼낼 때는 re-export로 하위호환을 유지한다.

## Phases

### Phase 1: Pipeline Domain

Pipeline은 session, run 등 다른 도메인이 참조하는 기반 도메인이므로 가장 먼저 정리한다.

**현재 문제:**
- Pipeline, PipelineSource, PipelineContext, PipelineWorkflow, Stage, StageConfig, CollectSource, PipelineRun, StageResult 타입이 모두 `shared/types/index.ts`에 위치
- `shared/api/pipelines.ts`와 `entities/pipeline/api/index.ts`에 동일 API 함수 이중 정의
- Zustand store 없이 React Query만 사용 — 선택된 파이프라인, 검색 필터 등 로컬 상태 관리 불가

**작업:**
1. `entities/pipeline/types.ts` 생성 — shared/types에서 Pipeline 관련 타입 이동
2. `shared/api/pipelines.ts` 삭제 — entity API가 canonical
3. `entities/pipeline/model/store.ts` 생성 — usePipelineStore (selectedPipelineId, search 등)
4. `shared/types/index.ts`에 re-export 추가 (하위호환)
5. 소비자 import 경로 수정 (pages, features, widgets)
6. `entities/pipeline/index.ts` 업데이트 — types + api + store barrel export
7. `npm run build` 타입체크 통과 확인

### Phase 2: Run Domain

**현재 문제:**
- RunRecord, NodeRunRecord, RunListResponse, RunEvent 관련 타입이 `shared/types/`에 위치
- `shared/api/runs.ts`와 `entities/run/api/index.ts` 이중 정의 (entity 버전이 에러 핸들링 더 나음)

**작업:**
1. `entities/run/types.ts` 생성 — RunRecord, NodeRunRecord, RunListResponse, NodeRunStatus, RunEvent 관련 타입 이동
2. `shared/api/runs.ts` 삭제
3. `shared/types/index.ts`에 re-export 추가
4. 소비자 import 수정
5. useExecutionStore가 entity 내부 타입을 import하도록 수정
6. 타입체크 통과 확인

### Phase 3: Connection Domain (신규 Entity)

**현재 문제:**
- entity 자체가 존재하지 않음
- types: shared/types의 Connection, ConnectionCreate, ConnectionType
- API: shared/api/connections.ts에만 존재
- pages/connections/index.tsx가 shared/api 직접 참조

**작업:**
1. `entities/connection/` 디렉토리 생성
2. `entities/connection/types.ts` — Connection, ConnectionCreate, ConnectionType 이동
3. `entities/connection/api.ts` — shared/api/connections.ts 내용 이동
4. `entities/connection/index.ts` — barrel export
5. shared에 re-export 추가
6. Connections.tsx import 수정
7. 타입체크 확인

### Phase 4: Workflow Domain (마무리)

**현재 문제:**
- WorkflowPicker.tsx가 apiFetch 직접 호출 (FSD 위반)

**작업:**
1. WorkflowPicker.tsx 수정 — `apiFetch<WorkflowListItem[]>('/api/workflows')` → `listWorkflows()` from `@/entities/workflow`
2. workflow types 감사 — NodeData는 entity에 있고, Schedule/Trigger는 workflow 고유가 아니므로 shared 유지 확인
3. import 경로 일관성 검증

### Phase 5: Content Session (검증)

**현재 상태:** 가장 완성도 높은 entity

**작업:**
1. ContentSessionStatus, SourceType 위치 확인 — 여러 도메인에서 참조하므로 shared 유지 또는 entity 이동 결정
2. PipelineSession, SessionStage, SessionStatus — content-session entity로 이동 여부 결정
3. store/API 완결성 최종 검증

### Phase 6: Shared Layer 최종 정리

**작업:**
1. `shared/types/index.ts` 정리 — 인프라 타입만 남았는지 확인 (ModelInfo, ToolInfo, OptionSchema, ConfigureRequest/Response, UploadResult)
2. `shared/api/` 정리 — 삭제된 파일 반영, barrel export 업데이트
3. `entities/settings/index.ts` 생성 — 누락된 barrel export 추가
4. re-export 전략 최종 확정 — 불필요한 re-export 제거
5. `npm run build` + `npm run lint` 전체 통과 확인

## Backward Compatibility Strategy

각 Phase에서 shared/types에서 타입을 빼낼 때:
```typescript
// shared/types/index.ts — re-export for backward compatibility
export type { Pipeline, PipelineSource, ... } from '@/entities/pipeline'
```

이렇게 하면 기존 `import { Pipeline } from '@/shared/types'` 코드가 깨지지 않는다. Phase 6에서 모든 import가 entity 경로로 전환된 후 re-export를 제거한다.

## Success Criteria

- 모든 도메인 entity가 types.ts + api.ts(필요시) + store.ts(필요시) + index.ts 구조를 갖춤
- shared/types에 인프라/cross-domain 타입만 남음
- API 이중 정의 제거
- FSD 위반 0건
- `npm run build` + `npm run lint` 통과
