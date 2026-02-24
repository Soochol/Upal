# Workflows 페이지 3-Column Inbox-Style 레이아웃 설계

## 목적

Workflows 페이지(`/workflows`)를 Inbox/Pipelines와 동일한 3-column 레이아웃으로 재구성하여 워크플로 탐색 → 편집을 한 화면에서 수행할 수 있게 한다. 기존 `/editor` 라우트를 제거하고 `/workflows`로 통합.

## 레이아웃

```
┌──────────────┬──────────────────────┬──────────────┐
│  Workflow     │                      │  RightPanel   │
│  Sidebar      │   Canvas + Console   │ (Properties/  │
│  (340px)      │     (flex-1)         │  Console/     │
│               │                      │  Preview)     │
│  - Search     │  WorkflowHeader      │              │
│  - New btn    │  ReactFlow Canvas    │  MainLayout   │
│  - WF list    │  Bottom Console      │  rightPanel   │
│  - Templates  │                      │  prop 사용    │
│               │                      │              │
└──────────────┴──────────────────────┴──────────────┘
```

## URL 스킴

| URL | 동작 |
|-----|------|
| `/workflows` | 사이드바 표시, 캔버스는 empty state |
| `/workflows?w=MyWorkflow` | 해당 워크플로를 캔버스에 로드 |
| `/workflows?new=true` | 빈 워크플로 생성 모드 |
| `/workflows?generate=true` | AI 생성 프롬프트 활성화 |

## 주요 변경사항

### 1. 새 컴포넌트: `WorkflowSidebar`

`web/src/pages/workflows/WorkflowSidebar.tsx`

- 검색 입력, New/Generate 버튼
- 워크플로 리스트 (현재 Landing의 카드 그리드 → 리스트 아이템 형태로 변환)
- 하단 Quick Start Templates 섹션
- 선택 상태는 URL param `w`로 관리
- Pipelines의 `PipelineSidebar`와 동일한 구조/스타일

### 2. 페이지 통합: `WorkflowsPage`

`web/src/pages/workflows/index.tsx` (현재 Landing.tsx 대체)

- Editor.tsx의 모든 로직 흡수 (generate, run, auto-save, shortcuts)
- URL param `w` 변경 시 → `loadWorkflow()` → store 반영
- 워크플로 전환 시 auto-save 후 새 워크플로 로드
- MainLayout의 `rightPanel` prop에 RightPanel 전달

### 3. RightPanel 리팩토링

현재 RightPanel은 자체 `aside` 래퍼, resize handle, collapse/expand 상태를 포함. MainLayout의 `rightPanel` prop으로 이동하기 위해:

- 외부 `aside` 래퍼 제거 → MainLayout이 래핑
- 자체 resize handle 제거 → MainLayout의 resize 사용
- **collapse/expand 상태를 부모로 끌어올림**:
  - `isRightPanelOpen` state를 WorkflowsPage에서 관리
  - 노드 선택 시 자동 확장, Properties 탭에서 해제 시 자동 축소 로직 유지
  - `rightPanel={isRightPanelOpen ? <RightPanelContent ... /> : null}` 형태
- 축소 시 아이콘 스트립(w-12)은 → 토글 버튼으로 대체 (Pipelines의 `PanelRightOpen/Close` 패턴)

### 4. 삭제 대상

| 파일 | 이유 |
|------|------|
| `web/src/pages/Editor.tsx` | 로직이 WorkflowsPage로 이동 |
| `web/src/pages/Landing.tsx` | WorkflowsPage가 대체 |
| `web/src/pages/landing/WorkflowCard.tsx` | 사이드바 아이템으로 변환 (선택적 유지) |
| router.tsx의 `/editor` 라우트 | 제거 |

### 5. 라우터 변경

```tsx
// Before
<Route path="/workflows" element={<LandingPage />} />
<Route path="/editor" element={<EditorPage />} />

// After
<Route path="/workflows" element={<WorkflowsPage />} />
// /editor 제거
```

## 모바일 처리

Inbox/Pipelines와 동일 패턴:
- `list` 레벨: WorkflowSidebar만 표시 (full-width)
- `detail` 레벨: Canvas만 표시 (full-width)
- Back 버튼으로 레벨 간 이동
- RightPanel은 모바일에서 숨김 (기존 동작 유지)

## 선택/전환 흐름

1. 사이드바에서 워크플로 클릭
2. 현재 워크플로가 dirty면 auto-save
3. URL param `?w=name` 업데이트
4. `loadWorkflow(name)` → store 반영
5. Canvas 갱신, RightPanel 초기화

## Empty State

워크플로 미선택 시 캔버스 영역에 표시:
- 아이콘 + "Select a workflow" 메시지
- 또는 현재 Canvas의 빈 상태 (Add first node / Generate with AI 프롬프트)

## 기존 유지 컴포넌트

- `Canvas` 위젯 — 변경 없음
- `Console` 위젯 — 변경 없음
- `WorkflowHeader` — `headerContent`로 유지
- `useAutoSave`, `useKeyboardShortcuts`, `useReconnectRun` 훅들
- `NodeEditor`, `PanelPreview`, `PanelConsole` — RightPanel 내부 컴포넌트들
