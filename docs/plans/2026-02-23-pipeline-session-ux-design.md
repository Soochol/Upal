# Pipeline Session UX Redesign

## Overview

파이프라인 세션의 전체 라이프사이클을 지원하는 UX 재설계.
현재 모달 기반 `SessionDetailModal`을 **전체 페이지 Linear Timeline**으로 전환하고,
Pipeline Settings에 **Workflows 섹션**을 추가하여 워크플로우 연결을 명확히 한다.

## User Flow

```
소스 수집 (트리거) → LLM 분석/요약 → [1차 승인] → 워크플로우 실행 → [2차 승인] → 게시
```

- **승인 게이트 2회**: Analyze 완료 후 (워크플로우 선택), Publish 전 (콘텐츠 리뷰)
- **트리거 타입**: manual, schedule(cron), surge(keyword spike)

---

## 1. Page Structure & Routing

### 변경 사항
- `SessionDetailModal` (모달 오버레이) → `SessionDetailPage` (전체 페이지)
- Route: `/pipelines/:id/sessions/:sessionId`
- Breadcrumb: `Pipelines / {pipeline.name} / Session #{n}`

### Layout

```
┌─── MainLayout ──────────────────────────────────────────────────┐
│ Header: Pipelines / My Pipeline / Session #3                    │
├─────────────────────────────────────────────────────────────────┤
│ ┌─ Sticky Progress Bar ──────────────────────────────────────┐  │
│ │ ● Collect ─── ● Analyze ─── ○ Produce ─── ○ Publish       │  │
│ │ Session #3 · pending_review · Score 82 · Feb 23, 10:30    │  │
│ └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  max-w-4xl mx-auto px-6                                         │
│                                                                  │
│  § 1. Collect   [completed — collapsed summary, expandable]     │
│  § 2. Analyze   [active — fully expanded, approval buttons]     │
│  § 3. Produce   [locked — "waiting for approval"]               │
│  § 4. Publish   [locked — "waiting for content"]                │
└─────────────────────────────────────────────────────────────────┘
```

### Stage States (3 types)
- **completed** (green): 축소된 1-2줄 요약 + 클릭으로 확장
- **active** (highlighted): 풀 확장, 인터랙션 가능
- **locked** (gray): 미리보기 플레이스홀더, 비활성

### Sticky Progress Bar
- 스크롤 시 상단 고정
- 각 단계 클릭 → 해당 섹션으로 smooth scroll
- 현재 세션 상태 + 메타 정보 (status badge, score, created_at)

---

## 2. Stage Details

### Stage 1: Collect (소스 수집)

소스별로 카드 그룹. 각 소스의 아이템 목록 (title, score, url).

**Active 뷰:**
```
┌─ § 1. Collect ───────────────────── ✓ 35 items collected ──┐
│  ┌─ HN Top Stories ──────────────────────── 12 items ─┐    │
│  │ 📊 142 │ Show HN: Building AI pipelines visually   → │  │
│  │ 📊  89 │ Rust vs Go for backend services            → │  │
│  │ ... +10 more                                        │  │
│  └─────────────────────────────────────────────────────┘  │
│  ┌─ r/artificial ────────────────────────── 8 items ──┐    │
│  │ ▲  234 │ OpenAI announces new reasoning model      → │  │
│  └─────────────────────────────────────────────────────┘  │
│  Collected Feb 23, 10:30 AM · Trigger: schedule           │
└───────────────────────────────────────────────────────────┘
```

**Collapsed 뷰:** `✓ 35 items from 3 sources (HN 12, Reddit 8, RSS 15) · Feb 23, 10:30`

**Collecting 뷰:** 각 소스마다 spinner + 실시간 카운트 (SSE 연동)

### Stage 2: Analyze (AI 분석 + 1차 승인)

2-column layout: 원본 소스(read-only) ↔ AI 생성 요약/인사이트(editable).

```
┌─ § 2. Analyze ──────────────────── ⏳ pending review ──────┐
│  ┌─ Score ────────────────────────────────────────────┐    │
│  │  [████████░░] 82/100  "고품질 소스가 풍부합니다"     │    │
│  └────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─ Raw Sources ─────────────┬─ AI Summary ──────────┐    │
│  │ <article>                 │ ## 핵심 요약            │    │
│  │   <title>Show HN:...     │ AI 에이전트 트렌드...    │    │
│  │ </article>                │                        │    │
│  │  [read-only]              │ ## 핵심 인사이트         │    │
│  │                           │ • 인사이트 1            │    │
│  │                           │ • 인사이트 2            │    │
│  │                           │  [editable, 저장됨]     │    │
│  └───────────────────────────┴────────────────────────┘    │
│                                                             │
│  ┌─ Recommended Workflows ────────────────────────────┐    │
│  │ ☑ blog     │ "AI 에이전트 트렌드 분석 블로그"        │    │
│  │ ☑ shorts   │ "30초 숏폼: LLM 실전 운영 팁"          │    │
│  │ ☐ debate   │ "찬반 토론: AI 에이전트는 필요한가?"     │    │
│  └────────────────────────────────────────────────────┘    │
│                                                             │
│        [Reject]  [✓ Approve & Run 2 Workflows]             │
└─────────────────────────────────────────────────────────────┘
```

**핵심:** Recommended Workflows는 Pipeline Settings에 등록된 워크플로우 풀에서 AI가 추천. 사용자가 체크/언체크로 최종 선택.

**변경점:**
- Summary/Insights contentEditable → 실제 PATCH 저장
- Approve 시 선택된 angle IDs + workflow 매핑이 백엔드에 전달

### Stage 3: Produce (워크플로우 실행)

선택된 워크플로우들이 병렬 실행. 각 워크플로우 카드에 실시간 로그.

```
┌─ § 3. Produce ──────────────────── ⚡ 2 workflows running ─┐
│  ┌─ 1. blog-post-generator ──────────────── Running ──┐    │
│  │ [████████░░░░] 53%                                  │    │
│  │ [System] Injecting context... ✓                     │    │
│  │ [Agent] Generating content body... ⏳               │    │
│  │ Tokens: 1,240 in / 450 out · 12s elapsed           │    │
│  └─────────────────────────────────────────────────────┘    │
│  ┌─ 2. shorts-script-generator ──────────── Pending ──┐    │
│  │ Waiting for agent to initialize...                  │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

**완료 시:** 각 카드에 결과 요약 + "Preview Output" 버튼
**SSE 연동:** RunPublisher 이벤트 스트림으로 실시간 로그 + 진행률

### Stage 4: Publish (리뷰 + 2차 승인 + 게시)

완료된 워크플로우 결과물을 미리보기. 개별/일괄 승인 후 게시.

```
┌─ § 4. Publish ──────────────────── 2 drafts ready ─────────┐
│  ┌─ Blog Post ────────────────────────────────────────┐    │
│  │ [rendered content preview]                          │    │
│  │ "AI 에이전트 트렌드 분석: 2026년 2월의 핵심 동향"    │    │
│  │ ...본문 미리보기...                                  │    │
│  │ [Edit]  [Schedule ▾]  [✓ Approve]                  │    │
│  └─────────────────────────────────────────────────────┘    │
│  ┌─ Shorts Script ────────────────────────────────────┐    │
│  │ [script preview]                                    │    │
│  │ [Edit]  [Schedule ▾]  [✓ Approve]                  │    │
│  └─────────────────────────────────────────────────────┘    │
│           [Reject All]  [✓ Publish All Approved]           │
└─────────────────────────────────────────────────────────────┘
```

**기능:**
- 실제 콘텐츠 프리뷰 렌더링 (markdown/HTML)
- 개별 Approve/Reject + 일괄 Publish
- "Schedule Post" 드롭다운: 즉시/예약 게시
- 게시 채널 선택 (Connection 엔티티 연동)

---

## 3. Pipeline Settings — Workflows Section (NEW)

기존 Settings 패널에 "Workflows" 섹션 추가.

```
Pipeline Settings (Right Panel)
├── § Sources & Schedule      ← 기존
│   ├── [RSS: TechCrunch]
│   ├── [HN Top Stories]
│   ├── Schedule: Daily 09:00
│   └── + Add source
├── § Workflows (NEW)
│   ├── blog-post-generator        [✕ remove]
│   ├── shorts-script-generator    [✕ remove]
│   ├── debate-generator           [✕ remove]
│   └── + Add workflow...          [opens workflow picker]
└── § Editorial Brief             ← 기존
    ├── Purpose, Audience, Tone
    └── Keywords
```

**Workflow Picker:**
- 기존 워크플로우 목록에서 선택 (GET /api/workflows)
- 선택 시 파이프라인의 workflows 배열에 추가
- AI Analyze 단계에서 이 풀에서 추천

**데이터 모델 변경:**
```typescript
// Pipeline type에 추가
type Pipeline = {
  // ... existing fields
  workflows?: PipelineWorkflow[]  // NEW
}

type PipelineWorkflow = {
  workflow_name: string    // references existing workflow
  label?: string           // display name override
  auto_select?: boolean    // AI가 기본으로 추천할지
}
```

---

## 4. Workflow Templates (3 Tiers)

파이프라인에 연결 가능한 워크플로우 템플릿들. 각각은 Upal의 DAG 워크플로우 정의.

### Tier 1: Core Content Production

| ID | Workflow | DAG Structure | Key Nodes |
|---|---|---|---|
| W1 | **Deep Dive Blog** | input → agent → output | agent: 2000자+ 분석글, 원문 인용 + 독자적 관점 |
| W2 | **Thread Builder** | input → agent → output | agent: Hook → 5-7 포인트 → CTA, 해시태그 최적화 |
| W3 | **Newsletter Digest** | input → agent → output | agent: 카테고리별 정리 + 에디터 코멘트 + 원문 링크 |
| W4 | **Short-form Script** | input → agent → output | agent: Hook(3초) → Problem → Insight → CTA 구조 |

### Tier 2: Differentiated (Multi-Agent)

| ID | Workflow | DAG Structure | Key Nodes |
|---|---|---|---|
| W5 | **Debate Generator** | input → agent(pro) + agent(con) → agent(synthesizer) → output | 찬성/반대 병렬 → 종합 |
| W6 | **Contrarian Angle** | input → agent(mainstream) → agent(contrarian) → output | 주류 분석 → 반대 시각 |
| W7 | **Trend Forecast** | input → tool(python_exec) → agent → output | 시계열 패턴 분석 → 예측 |
| W8 | **Cross-Domain Connector** | input(A) + input(B) → agent → output | 멀티 도메인 교차점 발견 |
| W9 | **Localization Adapter** | input → agent(ko) + agent(en) + agent(ja) → output | 문화적 재해석 병렬 |

### Tier 3: Interactive / Experimental

| ID | Workflow | DAG Structure | Key Nodes |
|---|---|---|---|
| W10 | **Quiz Builder** | input → agent(fact-extract) → agent(quiz-format) → output | 팩트 추출 → 퀴즈 포맷팅 |
| W11 | **Visual Data Story** | input → agent(data-extract) → agent(text) + image_model(visual) → output | 텍스트 + 이미지 병렬 생성 |
| W12 | **Case Study Deep Dive** | input → agent + tool(get_webpage) → output | 추가 리서치 + 심층 분석 |
| W13 | **Audio Briefing Script** | input → agent(script) → tts_model(audio) → output | 대본 생성 → 음성 변환 |

### Implementation Notes
- 모든 워크플로우는 `{{collected_data}}`, `{{analysis_summary}}`, `{{editorial_brief}}` 템플릿 변수로 파이프라인 컨텍스트 주입
- Tier 2의 병렬 패턴은 DAG 팬아웃으로 구현 (기존 아키텍처 지원)
- Tier 3의 image/TTS는 해당 모델 설정 시에만 활성화
- 각 워크플로우는 독립적인 WorkflowDefinition JSON으로 저장

---

## 5. Data Flow Summary

```
[Pipeline Settings]
     │
     ├── sources[] ─────────→ Stage 1: Collect
     ├── schedule (cron) ───→ Trigger
     ├── context (brief) ───→ All stages (injected)
     └── workflows[] ───────→ Stage 2: AI recommends from this pool
                                  │
                          [User approves + selects workflows]
                                  │
                              Stage 3: Produce (parallel workflow execution)
                                  │
                          [Workflow results ready]
                                  │
                              Stage 4: Publish
                                  │
                          [User reviews + approves]
                                  │
                              Published ✓
```

---

## 6. Backend Changes Required

### New/Modified Types
- `Pipeline.Workflows []PipelineWorkflow` — 파이프라인에 연결된 워크플로우 목록
- `ContentSession.SelectedWorkflows []string` — 승인 시 선택된 워크플로우 이름들

### New Endpoints
- `PATCH /api/content-sessions/{id}/analysis` — 편집된 summary/insights 저장
- `POST /api/content-sessions/{id}/produce` — 선택된 워크플로우 실행 (현재 stub → 실제 구현)

### Modified Endpoints
- `PATCH /api/content-sessions/{id}` — approve 시 `selected_workflows` 필드 포함
- `PUT /api/pipelines/{id}` — `workflows` 필드 추가

---

## 7. Frontend Changes Required

### New Components
- `SessionDetailPage` — 전체 페이지 (SessionDetailModal 대체)
- `StickyProgressBar` — 상단 고정 stepper
- `StageSection` — 각 단계의 wrapper (collapsed/active/locked 상태)
- `WorkflowPicker` — Settings 패널용 워크플로우 선택 모달

### Modified Components
- `PipelineSettingsPanel` — Workflows 섹션 추가
- `PipelineDetail` — 모달 대신 라우팅으로 세션 디테일 이동
- `AnalysisPanel` — editable 필드 저장 로직 + 워크플로우 풀 연동

### Removed Components
- `SessionDetailModal` — SessionDetailPage로 대체