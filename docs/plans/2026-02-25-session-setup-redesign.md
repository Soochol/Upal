# Session Setup Page Redesign — Linear Property Rows

## Goal

SessionSetupView 전체를 Linear/Vercel 스타일의 property row 기반 UI로 리디자인.
카드 박스 대신 구분선, key-value 수평 행, 인라인 편집으로 세련되고 정보 밀도 높은 UI 달성.

## Current State

4개 섹션이 `rounded-xl border` 카드로 감싸진 세로 폼:
- Sources & Schedule (한 카드)
- Editorial Brief (긴 폼)
- Analysis Model
- Workflows

**문제점**: generic admin form, 시각적 위계 부재, 소스 리스트 밋밋함, Schedule이 Sources에 끼어있음.

## Design

### Layout Structure

```
┌──────────────────────────────────────────┐
│  Session #2           ✎        ● Saved   │  header
├──────────────────────────────────────────┤
│                                          │
│  SOURCES  5 signals · 3 static   [+ Add] │  section header + count + action
│  ──────────────────────────────────────── │
│  🟠 해커뉴스 기술 뉴스                 ×  │  source rows
│  🟠 Reddit Machine Learning            ×  │
│  ⚪ TechCrunch AI                      ×  │
│  ⚪ MIT News                           ×  │
│                                          │
│  SCHEDULE                                │
│  ──────────────────────────────────────── │
│  Frequency      Every 6 hours  ▼         │  property row
│  Cron           0 */6 * * *              │  readonly when preset
│                                          │
│  EDITORIAL BRIEF                         │
│  ──────────────────────────────────────── │
│  Purpose        AI 뉴스 수집...   ✎      │  inline edit on click
│  Audience       기술 관심층...     ✎      │
│  Tone           전문적이고...     ✎      │
│  Focus          AI  LLM  ML  +3          │  keyword tags inline
│  Exclude        스팸  가짜뉴스            │
│  Language       Korean  ▼                │
│                                          │
│  PROCESSING                              │
│  ──────────────────────────────────────── │
│  Model          haiku     [Reset]        │
│  Workflows      (none)        [+ Add]    │
│                                          │
├──────────────────────────────────────────┤
│  8 sources · haiku         [▶ Start]     │  sticky footer
└──────────────────────────────────────────┘
```

### Design Tokens

- **Section header**: `text-[11px] font-medium uppercase tracking-widest text-muted-foreground`
- **Section divider**: `border-b border-border/40` (카드 border 제거)
- **Property row**: `flex items-center py-2.5 hover:bg-muted/30 rounded-lg px-2 -mx-2 transition-colors`
- **Row label**: `w-28 shrink-0 text-xs text-muted-foreground`
- **Row value**: `text-sm text-foreground flex-1`
- **Source dot**: signal → `bg-primary` (8px), static → `bg-muted-foreground/40` (8px)

### Source Row Behavior

- 각 source는 `py-2 px-2` 행으로 표시
- 왼쪽: 8px 원형 dot (signal=primary, static=muted)
- 중앙: source label
- 오른쪽: hover 시 Trash2 아이콘 (opacity-0 → opacity-100)
- 섹션 헤더에 `5 signals · 3 static` 카운트 표시
- Empty state: 테두리 없는 미니멀 메시지 + Add 버튼

### Editorial Brief Inline Edit

- 각 필드가 property row로 표시 (label: value)
- 텍스트 값은 truncate로 한 줄 표시
- 행 클릭 시 해당 행만 확장 → textarea/input으로 전환
- blur 또는 Enter 시 닫히고 auto-save
- Keywords는 항상 태그 칩으로 표시, 클릭 시 KeywordTagInput

### Schedule

- Sources와 분리된 독립 섹션
- Frequency property row: select dropdown (preset 목록)
- Cron property row: preset 선택 시 readonly, "Custom cron" 선택 시 editable

### Processing Section

- Model + Workflows를 "Processing" 섹션으로 통합
- Model: ModelSelector를 property row 값으로 표시
- Workflows: 행 목록 또는 empty state

### Sticky Footer

- 현재와 동일 기능, 약간 더 컴팩트
- `8 sources · haiku` 같은 요약 정보

## Changes

### Files to Modify

1. `web/src/pages/pipelines/session/SessionSetupView.tsx` — 전면 재작성
2. `web/src/features/define-editorial-brief/EditorialBriefForm.tsx` — inline property row 모드 추가 또는 SessionSetupView에 인라인화

### Files Unchanged

- `AddSourceModal.tsx` — 모달은 그대로 유지
- `WorkflowPicker.tsx` — 모달은 그대로 유지
- `SourceTypeBadge.tsx` — 더 이상 사용하지 않을 수 있음 (dot으로 대체)
- Backend — 변경 없음

## Non-Goals

- AddSourceModal / WorkflowPicker 모달 리디자인
- 백엔드 API 변경
- 다른 페이지 디자인 변경
