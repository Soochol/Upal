# Auto-Save 통일 구현 계획

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 프론트엔드 5곳의 자동 저장을 `shared/hooks/useAutoSave` 훅 하나로 통일한다.

**Architecture:** 기존 공통 훅은 변경 없이 그대로 사용. 4곳의 개별 구현을 공통 훅 직접 호출로 교체. 디바운스 2초 통일.

**Tech Stack:** React 19, TypeScript, Zustand, TanStack Query

---

### Task 1: 워크플로우 캔버스 마이그레이션

캔버스 전용 auto-save를 공통 훅으로 교체한다. Zustand 스토어에서 데이터를 꺼내 공통 훅에 전달하는 방식으로 전환.

**Files:**
- Modify: `web/src/features/manage-canvas/model/useAutoSave.ts` — 전면 교체
- Modify: `web/src/features/manage-canvas/index.ts` — re-export 확인
- Check: `web/src/pages/workflows/index.tsx` — 소비하는 쪽 인터페이스 변경 여부
- Check: `web/src/widgets/workflow-header/ui/WorkflowHeader.tsx` — `SaveStatus` 타입 import 경로 확인

**Step 1: 기존 동작 확인**

Run: `cd web && npx tsc -b --noEmit`
Expected: 타입 에러 없이 통과

**Step 2: 캔버스 auto-save 교체**

`web/src/features/manage-canvas/model/useAutoSave.ts`를 수정:

- 기존 코드 전체를 공통 훅 기반으로 교체
- `useAutoSave` from `@/shared/hooks/useAutoSave` import
- `SaveStatus` 타입을 공통 훅에서 re-export
- Zustand 스토어에서 `nodes`, `edges`, `workflowName` 셀렉터로 읽어서 데이터 객체 구성
- `useExecutionStore`에서 `isRunning`, `useWorkflowStore`에서 `isTemplate` 읽어서 `enabled` 조건 구성
- `onSave` 콜백에 기존 `flushSave()` 로직 유지 (이름 자동 생성, 직렬화, API 호출)
- `isEqual`에 기존 스냅샷 비교 로직 사용 (노드의 id/data/position, edges, workflowName)
- 반환값: `{ saveStatus, saveNow, markClean }` — 기존과 동일

```typescript
import { useCallback, useMemo } from 'react'
import {
  useAutoSave as useGenericAutoSave,
  type SaveStatus,
} from '@/shared/hooks/useAutoSave'
import { useWorkflowStore } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import {
  serializeWorkflow,
  saveWorkflow,
  suggestWorkflowName,
} from '@/entities/workflow'

export type { SaveStatus }

type CanvasSnapshot = {
  nodes: Array<{ id: string; data: unknown; position: { x: number; y: number } }>
  edges: unknown[]
  workflowName: string
}

function snapshotEqual(a: CanvasSnapshot, b: CanvasSnapshot): boolean {
  return JSON.stringify(a) === JSON.stringify(b)
}

export function useAutoSave() {
  const nodes = useWorkflowStore((s) => s.nodes)
  const edges = useWorkflowStore((s) => s.edges)
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const isTemplate = useWorkflowStore((s) => s.isTemplate)
  const originalName = useWorkflowStore((s) => s.originalName)
  const setWorkflowName = useWorkflowStore((s) => s.setWorkflowName)
  const setOriginalName = useWorkflowStore((s) => s.setOriginalName)
  const isRunning = useExecutionStore((s) => s.isRunning)

  const data: CanvasSnapshot = useMemo(
    () => ({
      nodes: nodes.map((n) => ({ id: n.id, data: n.data, position: n.position })),
      edges,
      workflowName: workflowName ?? '',
    }),
    [nodes, edges, workflowName],
  )

  const onSave = useCallback(
    async (snapshot: CanvasSnapshot) => {
      let name = snapshot.workflowName
      if (!name) {
        const tempWf = serializeWorkflow('untitled', nodes, edges)
        try {
          name = await suggestWorkflowName(tempWf)
        } catch {
          name = 'untitled-workflow'
        }
        setWorkflowName(name)
      }

      const wf = serializeWorkflow(name, nodes, edges)
      await saveWorkflow(wf, originalName || undefined)

      if (originalName !== name) {
        setOriginalName(name)
      }
    },
    [nodes, edges, originalName, setWorkflowName, setOriginalName],
  )

  const enabled = !isTemplate && !isRunning && nodes.length > 0

  const { saveStatus, saveNow, markClean } = useGenericAutoSave<CanvasSnapshot>({
    data,
    onSave,
    delay: 2000,
    isEqual: snapshotEqual,
    enabled,
  })

  return { saveStatus, saveNow, markClean }
}
```

**Step 3: re-export 확인**

`web/src/features/manage-canvas/index.ts`에서 `SaveStatus`가 올바르게 re-export되는지 확인. 필요 시 수정.

**Step 4: 소비하는 쪽 확인**

- `web/src/pages/workflows/index.tsx` — `useAutoSave()` 반환값이 동일하므로 변경 불필요할 것으로 예상. 타입 체크로 확인.
- `web/src/widgets/workflow-header/ui/WorkflowHeader.tsx` — `SaveStatus` 타입이 동일한 경로에서 export되므로 변경 불필요 예상.

**Step 5: 타입 체크**

Run: `cd web && npx tsc -b --noEmit`
Expected: 타입 에러 없이 통과

**Step 6: 수동 테스트 항목**

- 캔버스에서 노드 추가/이동 → 2초 후 저장됨 표시 확인
- Ctrl+S → 즉시 저장 확인
- 워크플로우 실행 중 → 저장 안 됨 확인
- 빈 캔버스 → 저장 안 됨 확인
- 템플릿 모드 → 저장 안 됨 확인
- 이름 없는 워크플로우 → 저장 시 이름 자동 생성 확인

**Step 7: 커밋**

```bash
git add web/src/features/manage-canvas/model/useAutoSave.ts
git commit -m "refactor: migrate canvas auto-save to shared useAutoSave hook"
```

---

### Task 2: 파이프라인 설정 마이그레이션

`Pipelines.tsx`의 인라인 디바운스/dirty/unmount 저장 코드를 공통 훅으로 교체.

**Files:**
- Modify: `web/src/pages/Pipelines.tsx` — auto-save 관련 코드 교체

**Step 1: 기존 인라인 코드 제거 및 공통 훅 적용**

`Pipelines.tsx`에서 제거할 코드:
- `autoSaveStatus` 상태
- `isDirty` useMemo
- `isDirtyRef`, `doSave`, `doSaveRef`
- 디바운스 useEffect (800ms setTimeout)
- unmount 저장 useEffect
- 각 ref 동기화 useEffect (`templateRef`, `localSourcesRef` 등)

교체할 코드:
- 4개 로컬 상태(`localSources`, `localSchedule`, `localWorkflows`, `localModel`)는 유지
- 이들을 하나의 설정 객체로 묶어 공통 훅에 전달
- `onSave`에서 `updateSessionSettings` API 호출
- `saveOnUnmount: true`, `onBeforeUnloadSave` 설정
- `onError`에서 toast 표시
- `markClean()`을 서버 데이터 동기화 시 호출

```typescript
import { useAutoSave } from '@/shared/hooks/useAutoSave'

// 기존 4개 로컬 상태는 유지
const [localSources, setLocalSources] = useState<PipelineSource[]>([])
const [localSchedule, setLocalSchedule] = useState('')
const [localWorkflows, setLocalWorkflows] = useState<PipelineWorkflow[]>([])
const [localModel, setLocalModel] = useState('')

// 설정 데이터를 하나로 묶음
const settingsData = useMemo(() => ({
  sources: localSources,
  schedule: localSchedule,
  workflows: localWorkflows,
  model: localModel,
}), [localSources, localSchedule, localWorkflows, localModel])

const { saveStatus: autoSaveStatus, markClean } = useAutoSave({
  data: settingsData,
  onSave: async (data) => {
    if (!templateSession) return
    await updateSessionSettings(templateSession.id, data)
    queryClient.invalidateQueries({
      queryKey: ['content-sessions', { pipelineId: selectedPipelineId, templateOnly: true }],
    })
  },
  delay: 2000,
  enabled: !!templateSession,
  saveOnUnmount: true,
  onBeforeUnloadSave: (data) => {
    if (!templateSession) return
    fetch(`/api/content-sessions/${templateSession.id}/settings`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      keepalive: true,
      body: JSON.stringify(data),
    })
  },
  onError: (err) =>
    addToast(`Failed to save session settings: ${err instanceof Error ? err.message : 'unknown error'}`),
})

// 서버 → 로컬 동기화 시 markClean
useEffect(() => {
  if (templateSession) {
    setLocalSources(templateSession.session_sources ?? [])
    setLocalSchedule(templateSession.schedule ?? '')
    setLocalWorkflows(templateSession.session_workflows ?? [])
    setLocalModel(templateSession.model ?? '')
    markClean()
  }
}, [templateSession?.id])
```

**Step 2: `autoSaveStatus` 타입 호환 확인**

기존: `'idle' | 'saving' | 'saved'`
공통 훅: `'idle' | 'waiting' | 'saving' | 'saved' | 'error'`

`autoSaveStatus`를 사용하는 JSX에서 `'waiting'`과 `'error'` 상태도 표시하도록 업데이트. 또는 기존대로 `'saving'`과 `'saved'`만 표시해도 무방 — 동작에 문제 없음.

**Step 3: 타입 체크**

Run: `cd web && npx tsc -b --noEmit`
Expected: 타입 에러 없이 통과

**Step 4: 수동 테스트 항목**

- 파이프라인 소스/스케줄/워크플로우/모델 변경 → 2초 후 저장됨 확인
- 파이프라인 선택 변경 → 이전 설정이 저장되었는지 확인
- 브라우저 탭 닫기 → 변경사항 저장됨 확인

**Step 5: 커밋**

```bash
git add web/src/pages/Pipelines.tsx
git commit -m "refactor: migrate pipeline settings auto-save to shared useAutoSave hook"
```

---

### Task 3: 편집 방향 설정 폼 마이그레이션

`EditorialBriefForm.tsx`의 인라인 auto-save + 수동 저장을 공통 훅으로 교체.

**Files:**
- Modify: `web/src/features/define-editorial-brief/EditorialBriefForm.tsx`

**Step 1: 공통 훅 적용**

제거할 코드:
- `saveStatus` 로컬 상태
- `isFirstRender` ref
- auto-save useEffect (1초 디바운스)
- `handleSave` 함수 (수동 저장)

교체할 코드:

```typescript
import { useAutoSave } from '@/shared/hooks/useAutoSave'

const [draft, setDraft] = useState<PipelineContext>(initialContext ?? DEFAULT_CONTEXT)

const { saveStatus, saveNow } = useAutoSave({
  data: draft,
  onSave: async (data) => { await onSave(data) },
  delay: 2000,
  enabled: autoSave ?? false,
})

// 수동 저장 버튼 → saveNow() 호출
const handleSave = () => { void saveNow() }
```

**Step 2: JSX 상태 표시 업데이트**

- `'waiting'` 상태 추가 표시 (선택) 또는 기존처럼 `'saving'`/`'saved'`/`'error'`만 표시
- `autoSave=false`일 때 수동 저장 버튼은 `handleSave` → `saveNow()` 호출로 유지

**Step 3: 타입 체크**

Run: `cd web && npx tsc -b --noEmit`
Expected: 타입 에러 없이 통과

**Step 4: 수동 테스트 항목**

- 편집 방향 폼에서 내용 수정 → 2초 후 저장됨 확인
- 수동 저장 버튼 클릭 → 즉시 저장 확인
- `autoSave=false` 모드에서 버튼만 동작하는지 확인

**Step 5: 커밋**

```bash
git add web/src/features/define-editorial-brief/EditorialBriefForm.tsx
git commit -m "refactor: migrate editorial brief auto-save to shared useAutoSave hook"
```

---

### Task 4: 분석 결과 편집 마이그레이션

`AnalyzeStage.tsx`의 React Query mutation + 수동 디바운스를 공통 훅으로 교체.

**Files:**
- Modify: `web/src/pages/pipelines/session/stages/AnalyzeStage.tsx`

**Step 1: 공통 훅 적용**

제거할 코드:
- `saveStatus` 로컬 상태
- `debounceRef`, `savedTimerRef` refs
- `saveMutation` (useMutation)
- timer cleanup useEffect
- `debouncedSave` 함수

교체할 코드:

```typescript
import { useAutoSave } from '@/shared/hooks/useAutoSave'

const [editedSummary, setEditedSummary] = useState(analysis?.summary ?? '')
const [editedInsights, setEditedInsights] = useState<string[]>(analysis?.insights ?? [])

const analysisData = useMemo(() => ({
  summary: editedSummary,
  insights: editedInsights,
}), [editedSummary, editedInsights])

const { saveStatus } = useAutoSave({
  data: analysisData,
  onSave: async (data) => {
    await updateSessionAnalysis(session.id, data)
  },
  delay: 2000,
  enabled: isPendingReview,  // 리뷰 대기 상태에서만 편집 가능
})
```

**Step 2: 입력 핸들러 단순화**

`handleSummaryInput`, `handleInsightInput`에서 `debouncedSave()` 호출 제거. 상태만 업데이트하면 공통 훅이 자동 감지.

```typescript
const handleSummaryInput = useCallback(
  (e: React.FormEvent<HTMLDivElement>) => {
    setEditedSummary(e.currentTarget.textContent ?? '')
  },
  [],
)

const handleInsightInput = useCallback(
  (index: number, e: React.FormEvent<HTMLDivElement>) => {
    const text = e.currentTarget.textContent ?? ''
    setEditedInsights((prev) => {
      const next = [...prev]
      next[index] = text
      return next
    })
  },
  [],
)
```

**Step 3: `useMutation` import 제거**

`saveMutation` 관련 import (`useMutation` from `@tanstack/react-query`)가 이 컴포넌트에서 더 이상 사용되지 않으면 제거.

**Step 4: 타입 체크**

Run: `cd web && npx tsc -b --noEmit`
Expected: 타입 에러 없이 통과

**Step 5: 수동 테스트 항목**

- 분석 요약/인사이트 편집 → 2초 후 저장됨 확인
- 빠르게 여러 번 편집 → 마지막 편집 후 2초에 한 번만 저장 확인
- 리뷰 대기 상태가 아닌 세션에서는 편집 불가 확인

**Step 6: 커밋**

```bash
git add web/src/pages/pipelines/session/stages/AnalyzeStage.tsx
git commit -m "refactor: migrate analyze stage auto-save to shared useAutoSave hook"
```

---

### Task 5: 최종 정리 및 검증

**Step 1: 전체 타입 체크**

Run: `cd web && npx tsc -b --noEmit`
Expected: 타입 에러 없이 통과

**Step 2: ESLint**

Run: `cd web && npm run lint`
Expected: 에러 없이 통과

**Step 3: 미사용 import 정리**

각 수정된 파일에서 더 이상 사용되지 않는 import 제거:
- `Pipelines.tsx`: `useMemo` (isDirty용), `useRef` (각종 ref) — 다른 곳에서 사용 중이면 유지
- `EditorialBriefForm.tsx`: `useRef` (isFirstRender)
- `AnalyzeStage.tsx`: `useMutation`, `useCallback` (debouncedSave) — 다른 곳에서 사용 중이면 유지

**Step 4: 빌드 확인**

Run: `cd web && npm run build`
Expected: 빌드 성공

**Step 5: 최종 커밋**

```bash
git add -A
git commit -m "refactor: clean up unused imports after auto-save unification"
```
