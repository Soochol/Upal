# Content Media Pipeline — Frontend Design
*2026-02-23*

> **관련 문서**: 시스템 설계 → `2026-02-23-content-media-pipeline-design.md`

---

## 1. 개요

콘텐츠 미디어 파이프라인 기능 추가에 따른 프론트엔드 확장 설계.
기존 Workflows / Editor / Pipelines / Runs / Connections 구조에 콘텐츠 운영 레이어를 추가한다.

**FSD 레이어 원칙 유지**: `app` → `pages` → `widgets` → `features` → `entities` → `shared`

---

## 2. 현재 구조 vs 변경 후 구조

### 현재 라우트
```
/              → ProductLanding
/workflows     → Landing (워크플로우 목록)
/editor        → Editor
/runs          → Runs
/runs/:id      → RunDetail
/pipelines     → Pipelines
/pipelines/:id → Pipelines (상세)
/connections   → Connections
```

### 변경 후 라우트
```
/              → ProductLanding
/workflows     → Landing (워크플로우 목록) — 변경 없음
/editor        → Editor — 변경 없음
/runs          → Runs — 변경 없음
/runs/:id      → RunDetail — 변경 없음
/connections   → Connections — 변경 없음

/pipelines           → Pipelines 목록 (콘텐츠 파이프라인 섹션 추가)
/pipelines/new       → 파이프라인 생성 (신규 — Editorial Brief 포함)
/pipelines/:id       → 파이프라인 상세 (소스 설정 + 세션 목록)  [대폭 변경]

/inbox               → 통합 Content Inbox  [신규]
/inbox/:sessionId    → 세션 상세 (소스 + 분석 + 결과물)  [신규]

/published           → 발행 이력  [신규]
```

### 현재 헤더 네비게이션 변경
```
현재: Workflows | Editor | Runs | Pipelines | Connections

변경:
  Workflows | Editor | Runs
  ─── 콘텐츠 운영 ───
  Pipelines | Inbox | Published
  ─── 설정 ───
  Connections
```

또는 단순하게 배지를 활용:
```
Workflows | Editor | Runs | Pipelines | Inbox [3] | Connections
                                                    ↑ pending review 수
```

---

## 3. 신규 페이지별 설계

---

### 3.1 Content Inbox (`/inbox`)

**역할**: 모든 파이프라인의 ContentSession을 통합해서 리뷰하는 에디토리얼 허브.

#### 레이아웃

```
┌─────────────────────────────────────────────────────────────────┐
│ Header (기존)                                         [Inbox 3] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌── 서지 배너 (서지 감지 시만 표시) ──────────────────────────┐ │
│  │ 🔥  DeepSeek 언급 10배 급증 중  [세션 생성]  [닫기]        │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
│  Content Inbox                                                  │
│  [전체] [대기 중 3] [승인됨] [제작 중] [완료]                   │
│  ─────────────────────────────────  파이프라인: [전체 ▼]       │
│                                                                 │
│  ┌── SessionCard ──────────────────────────────────────────┐  │
│  │  IT AI  ·  Session 12  ·  Jan 2, 08:00      Score: 94  │  │
│  │  ─────────────────────────────────────────────────────  │  │
│  │  "GPT-5 출시로 AI 경쟁 재편, 투자 관점 주목"             │  │
│  │                                                         │  │
│  │  정적  HN(15)  Reuters(8)                               │  │
│  │  신호  Trends(급등) · Reddit(4.2K↑) · HN(pts:847)      │  │
│  │  → 선별: 7개 아이템                                     │  │
│  │                                                         │  │
│  │  [▶ 세션 보기]          [✅ 승인]  [❌ 거절]            │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌── SessionCard ──────────────────────────────────────────┐  │
│  │  Finance  ·  Session 8  ·  Jan 2, 06:00    Score: 87   │  │
│  │  "BTC ETF 유입 급증, 기관 수요 사상 최고"                │  │
│  │  정적 Reuters(12) · 신호 Trends(급등) 거래량(+340%)     │  │
│  │  [▶ 세션 보기]          [✅ 승인]  [❌ 거절]            │  │
│  └─────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

#### 컴포넌트 구성
```
pages/Inbox.tsx
  widgets/content-inbox/
    InboxPageHeader.tsx       (필터 탭 + 파이프라인 셀렉터)
    SurgeBanner.tsx           (서지 감지 배너, 조건부 표시)
    SessionCard.tsx           (세션 카드)
      SourceSummaryChips.tsx  (정적/신호 소스 요약 뱃지)
    SessionList.tsx           (무한 스크롤 또는 페이지네이션)
```

#### 상태 관리
```typescript
// entities/content-session/store.ts (Zustand)
interface ContentSessionStore {
  sessions: ContentSession[]
  surges: SurgeAlert[]
  filters: { status?: string; pipelineId?: string }
  fetchSessions: () => Promise<void>
  approveSession: (id: string) => Promise<void>
  rejectSession: (id: string) => Promise<void>
  dismissSurge: (id: string) => void
  createSessionFromSurge: (surgeId: string) => Promise<void>
}
```

---

### 3.2 세션 상세 (`/inbox/:sessionId`)

**역할**: 특정 세션의 소스 원본 → LLM 분석 → 결과물을 드릴다운으로 확인. 워크플로우 실행 승인.

#### 레이아웃

```
┌─────────────────────────────────────────────────────────────────┐
│ Header                                                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ◀ Content Inbox  /  IT AI  /  Session 12  (Breadcrumb)        │
│                                                                 │
│  Session 12  ·  Jan 2, 08:00  ·  trigger: schedule  Score: 94  │
│  Status: [대기 중 리뷰]                                         │
│                                                                 │
│  ┌── 탭 ───────────────────────────────────────────────────┐  │
│  │  [소스]  [LLM 분석]  [결과물]                            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ── 소스 탭 ────────────────────────────────────────────────── │
│                                                                 │
│  [정적] HN RSS  ·  15개 수집                                   │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ pts:847  OpenAI announces GPT-5...          🔗 원문 열기 │  │
│  │ pts:623  Anthropic raises $2B...            🔗 원문 열기 │  │
│  │ pts:411  ...                                🔗 원문 열기 │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [신호] Google Trends  ·  3개 키워드                           │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ "GPT-5"      전주 대비 +840%  📈                        │  │
│  │ "OpenAI"     전주 대비 +320%  📈                        │  │
│  │ "AI 모델"    전주 대비 +210%  📈                        │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [신호] Reddit r/MachineLearning  ·  top 5                     │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ ↑4,200  GPT-5 is here, and it's wild...    🔗 원문 열기 │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ── LLM 분석 탭 ────────────────────────────────────────────── │
│                                                                 │
│  총 55개 수집 → 7개 선별                                        │
│                                                                 │
│  요약                                                           │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ GPT-5 출시와 함께 AI 경쟁 구도가 급격히 재편됩니다.      │  │
│  │ HN과 Reddit에서 폭발적 반응이 나타나고 있으며...         │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  핵심 인사이트                                                   │
│  • GPT-5 성능이 기존 대비 3배 개선, 코딩 벤치마크 1위          │
│  • Anthropic과의 격차 다시 벌어지는 신호                        │
│  • 국내 AI 스타트업 주가 동반 상승 중                          │
│                                                                 │
│  추천 콘텐츠 각도                                               │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ ● 쇼츠    "GPT-5 출시 30초 요약"         [선택 ✓]       │ │
│  │ ● 블로그  "GPT-5가 바꾸는 AI 생태계"     [선택 ✓]       │ │
│  │ ● 뉴스레터 "이번 주 AI 핵심 뉴스"        [선택  ]       │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
│  [✅ 승인하고 선택된 워크플로우 실행]    [❌ 거절]              │
│                                                                 │
│  ── 결과물 탭 ──────────────────────────────────────────────── │
│                                                                 │
│  Blog Workflow Run #42   [완료]    [결과 보기]  [게시]          │
│  Shorts Workflow Run #43 [실행 중...]                           │
└─────────────────────────────────────────────────────────────────┘
```

#### 컴포넌트 구성
```
pages/SessionDetail.tsx
  widgets/session-detail/
    SessionDetailHeader.tsx     (브레드크럼 + 메타정보 + 상태)
    SessionTabs.tsx             (소스 / LLM 분석 / 결과물)
    SourcePanel.tsx             (소스별 원본 목록)
      SourceGroup.tsx           (툴 이름 + 아이템 목록)
      SourceItemRow.tsx         (제목 + 점수 + 링크)
    AnalysisPanel.tsx           (LLM 분석 결과)
      InsightList.tsx
      ContentAngleSelector.tsx  (포맷 선택 체크박스)
    WorkflowResultsPanel.tsx    (생성된 워크플로우 실행 목록)
    SessionActionBar.tsx        (승인/거절 버튼, 고정 하단 바)
```

---

### 3.3 파이프라인 상세 — 대폭 변경 (`/pipelines/:id`)

**역할**: 파이프라인 설정(소스 + Editorial Brief) + 세션 이력 + 수동 트리거.

#### 레이아웃

```
┌─────────────────────────────────────────────────────────────────┐
│ Header                                                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ◀ Pipelines  /  IT AI Pipeline                                │
│                                                                 │
│  IT AI Pipeline                    [지금 수집하기 ▶]  [설정 ✏️] │
│  목적: IT 업계 종사자 + AI 관심 일반인 대상...                   │
│  스케줄: 매 6시간  ·  소스: 5개  ·  최근 세션: 2시간 전         │
│                                                                 │
│  ┌── 탭 ─────────────────────────────────────────────────────┐ │
│  │  [세션 이력]  [소스 설정]  [Editorial Brief]  [워크플로우] │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ── 세션 이력 탭 ───────────────────────────────────────────── │
│                                                                 │
│  ┌── MiniSessionCard ──────────────────────────────────────┐  │
│  │  Session 12  Jan 2, 08:00  Score:94  [대기 중]  [보기▶] │  │
│  └─────────────────────────────────────────────────────────┘  │
│  ┌── MiniSessionCard ──────────────────────────────────────┐  │
│  │  Session 11  Jan 1, 14:00  Score:78  [게시완료]  [보기▶] │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ── 소스 설정 탭 ───────────────────────────────────────────── │
│                                                                 │
│  수집 소스                                       [+ 소스 추가]  │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ [정적] HN RSS       https://...                 [삭제]   │  │
│  │ [정적] Reuters AI   https://...                 [삭제]   │  │
│  │ [신호] Reddit       r/MachineLearning  min:100   [설정]  │  │
│  │ [신호] Google Trends  키워드: AI, LLM             [설정]  │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  스케줄:  [cron]  0 */6 * * *   (6시간마다)                    │
│  알림:    Discord  webhook: https://...                         │
│                                                                 │
│  [저장]                                                         │
│                                                                 │
│  ── Editorial Brief 탭 ─────────────────────────────────────── │
│                                                                 │
│  주제/목적                                                      │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ IT 업계 종사자와 AI에 관심 있는 일반인을 대상으로...      │  │
│  └─────────────────────────────────────────────────────────┘  │
│  타겟 독자: IT 업계 종사자, AI 관심 일반인                      │
│  톤/스타일: 기술적이지만 접근 쉬운 한국어                       │
│  포커스 키워드: [LLM ×] [AI 모델 ×] [빅테크 ×] [+추가]        │
│  제외 키워드:   [게임 ×] [스포츠 ×] [+추가]                    │
│  언어: [한국어]                                                 │
│  [저장]                                                         │
└─────────────────────────────────────────────────────────────────┘
```

#### 컴포넌트 구성
```
pages/pipelines/PipelineDetail.tsx  (기존 대폭 확장)
  widgets/pipeline-detail/           (신규 위젯)
    PipelineDetailHeader.tsx         (파이프라인 제목 + 메타 + 수집 버튼)
    PipelineDetailTabs.tsx
    SessionHistoryTab.tsx            (세션 목록)
      MiniSessionCard.tsx
    SourceConfigTab.tsx              (소스 설정)
      SourceList.tsx
      SourceConfigModal.tsx          (소스별 파라미터 설정)
      AddSourceModal.tsx             (소스 타입 선택 + 설정)
    EditorialBriefTab.tsx            (파이프라인 컨텍스트 폼)
      KeywordTagInput.tsx            (태그 형식 키워드 입력)
    WorkflowTemplatesTab.tsx         (이 파이프라인에 연결된 템플릿)
```

---

### 3.4 파이프라인 생성 — 확장 (`/pipelines/new`)

**현재**: 파이프라인 이름 + 스테이지만 입력.
**변경**: 3단계 생성 플로우 추가.

```
Step 1: 기본 정보
  이름, 설명

Step 2: Editorial Brief (신규)
  주제/목적, 타겟 독자, 톤, 포커스/제외 키워드, 언어

Step 3: 소스 설정 (신규)
  정적/신호 소스 추가, 스케줄 설정

[생성 완료] → /pipelines/:id 로 이동
```

---

### 3.5 Pipelines 목록 (`/pipelines`) — 소폭 변경

기존 파이프라인 목록 카드에 콘텐츠 운영 상태 요약 추가.

```
┌─────────────────────────────────────────────────────────────────┐
│  Pipelines                                   [+ 새 파이프라인]  │
├─────────────────────────────────────────────────────────────────┤
│  [일반 파이프라인] [콘텐츠 파이프라인]  ← 탭 추가              │
│                                                                 │
│  ┌── PipelineCard ────────────────────────────────────────┐   │
│  │  IT AI Pipeline                  스케줄: 6h            │   │
│  │  소스: 5개  ·  최근 세션: 2시간 전  ·  ⏳ 리뷰 대기: 2  │   │
│  │  [세션 보기]                      [지금 수집하기 ▶]    │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌── PipelineCard ────────────────────────────────────────┐   │
│  │  Finance Pipeline                스케줄: 1h            │   │
│  │  소스: 3개  ·  최근 세션: 45분 전  ·  ✅ 정상 운영 중  │   │
│  └────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

### 3.6 Published (`/published`) — 신규

**역할**: 게시된 콘텐츠 이력. 채널별·파이프라인별 필터.

```
┌─────────────────────────────────────────────────────────────────┐
│  Published Content                                              │
│  [전체] [YouTube] [Substack] [Discord]   파이프라인: [전체 ▼]  │
├─────────────────────────────────────────────────────────────────┤
│  ┌── PublishedCard ───────────────────────────────────────┐   │
│  │  📺 YouTube  ·  IT AI  ·  Jan 2, 10:30                 │   │
│  │  "GPT-5 출시 30초 요약"                                │   │
│  │  🔗 youtube.com/...    세션: Session 12  [계보 보기]   │   │
│  └────────────────────────────────────────────────────────┘   │
│  ┌── PublishedCard ───────────────────────────────────────┐   │
│  │  📧 Substack  ·  IT AI  ·  Jan 2, 11:00               │   │
│  │  "GPT-5가 바꾸는 AI 생태계 지형도"                     │   │
│  │  🔗 substack.com/...   세션: Session 12  [계보 보기]   │   │
│  └────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. 신규 FSD 레이어 구조

### entities

```
entities/
  content-session/        (신규)
    types.ts              ContentSession, SourceFetch, LLMAnalysis, ContentAngle
    api.ts                /api/content-sessions CRUD
    store.ts              Zustand 스토어
    index.ts

  published-content/      (신규)
    types.ts              PublishedContent
    api.ts                /api/published
    index.ts

  surge/                  (신규)
    types.ts              SurgeAlert
    api.ts                /api/surges
    store.ts
    index.ts

  pipeline/               (기존 확장)
    types.ts              + PipelineContext, PipelineSource 타입 추가
    api.ts                + sources, collect, brief 엔드포인트 추가
```

### features

```
features/
  review-session/         (신규)
    ApproveSessionForm.tsx   (포맷 선택 + 승인)
    RejectSessionDialog.tsx

  configure-pipeline-sources/  (신규)
    SourceTypeSelector.tsx
    SourceConfigForm.tsx     (툴별 파라미터 폼)

  define-editorial-brief/  (신규)
    EditorialBriefForm.tsx
    KeywordTagInput.tsx
```

### widgets

```
widgets/
  content-inbox/          (신규)
  session-detail/         (신규)
  pipeline-detail/        (신규, 기존 pipeline-editor 대체 또는 병존)
```

### pages

```
pages/
  Inbox.tsx               (신규)
  SessionDetail.tsx       (신규)
  Published.tsx           (신규)

  pipelines/
    index.tsx             (기존, 소폭 수정)
    PipelineDetail.tsx    (기존, 대폭 확장)
    PipelineNew.tsx       (신규 또는 기존 플로우에 스텝 추가)
```

---

## 5. 라우터 변경

```typescript
// web/src/app/router.tsx 변경
<Routes>
  <Route path="/" element={<ProductLandingPage />} />
  <Route path="/workflows" element={<LandingPage />} />
  <Route path="/editor" element={<EditorPage />} />
  <Route path="/runs" element={<RunsPage />} />
  <Route path="/runs/:id" element={<RunDetail />} />
  <Route path="/connections" element={<ConnectionsPage />} />

  {/* 기존 */}
  <Route path="/pipelines" element={<PipelinesPage />} />
  <Route path="/pipelines/new" element={<PipelineNewPage />} />   {/* 신규 */}
  <Route path="/pipelines/:id" element={<PipelineDetailPage />} /> {/* 분리 */}

  {/* 신규 */}
  <Route path="/inbox" element={<InboxPage />} />
  <Route path="/inbox/:sessionId" element={<SessionDetailPage />} />
  <Route path="/published" element={<PublishedPage />} />
</Routes>
```

---

## 6. 헤더 네비게이션 변경

```typescript
// web/src/shared/ui/Header.tsx
const navLinks = [
  { to: "/workflows", label: "Workflows" },
  { to: "/editor", label: "Editor" },
  { to: "/runs", label: "Runs" },
  { to: "/pipelines", label: "Pipelines" },
  { to: "/inbox", label: "Inbox", badge: pendingCount },  // 신규
  { to: "/published", label: "Published" },               // 신규
  { to: "/connections", label: "Connections" },
]
```

`pendingCount`는 `useContentSessionStore`에서 `status === "pending_review"` 수를 읽어온다.

---

## 7. 공통 컴포넌트

```
shared/ui/
  StatusBadge.tsx         (pending_review | approved | producing | published 뱃지)
  ScoreIndicator.tsx      (LLM 점수 0-100 시각화)
  SourceTypeBadge.tsx     ([정적] [신호] 뱃지)
  BreadcrumbNav.tsx       (드릴다운 네비게이션 — 재사용 가능)
  SurgeBanner.tsx         (서지 감지 배너)
```

---

## 8. SSE 연동

세션 수집 중 상태 업데이트를 실시간으로 표시.

```typescript
// 기존 execute-workflow SSE 패턴 재사용
// entities/content-session/api.ts

export function subscribeToSession(sessionId: string, onEvent: (e: SessionEvent) => void) {
  const es = new EventSource(`/api/content-sessions/${sessionId}/events`)
  es.onmessage = (e) => onEvent(JSON.parse(e.data))
  return () => es.close()
}

type SessionEvent =
  | { type: 'source_fetched'; tool: string; count: number }
  | { type: 'analysis_complete'; score: number; summary: string }
  | { type: 'status_changed'; status: ContentSessionStatus }
```

세션 상세 페이지에서 `collecting` 상태일 때 소스 탭에 실시간으로 수집 진행 표시.

---

## 9. 구현 순서 (Phase)

### Phase 1 — 데이터/라우트 기반 (백엔드 Phase 1과 병렬)
- [ ] FSD entities 추가: `content-session`, `published-content`, `surge`
- [ ] 라우터에 신규 라우트 추가
- [ ] Header 네비게이션에 Inbox, Published 추가
- [ ] Inbox 페이지 (목업 데이터로 UI 먼저)
- [ ] SessionDetail 페이지 (목업 데이터)
- [ ] Published 페이지 (목업 데이터)

### Phase 2 — 파이프라인 설정 UI (백엔드 Phase 2와 병렬)
- [ ] PipelineDetail 탭 구조 (소스 설정 / Editorial Brief / 세션 이력)
- [ ] 소스 설정 폼 (타입별 파라미터)
- [ ] Editorial Brief 폼 (KeywordTagInput 포함)
- [ ] 파이프라인 생성 3단계 플로우
- [ ] "지금 수집하기" 수동 트리거 버튼

### Phase 3 — 세션 리뷰 플로우 (백엔드 Phase 3와 병렬)
- [ ] Inbox → SessionDetail 드릴다운 연결
- [ ] ContentAngleSelector (포맷 선택 체크박스)
- [ ] 승인 → 워크플로우 선택 → 실행 플로우
- [ ] SSE로 세션 수집 진행 실시간 표시
- [ ] SurgeBanner + 서지에서 세션 생성 플로우

### Phase 4 — 발행 이력 (백엔드 Phase 4와 병렬)
- [ ] Published 페이지 실데이터 연결
- [ ] "계보 보기" — Published → WorkflowRun → Session → Pipeline 역추적
- [ ] Inbox 배지 실시간 업데이트 (SSE 또는 폴링)

---

## 10. 리팩토링 영향 범위

| 파일 | 변경 수준 | 내용 |
|---|---|---|
| `app/router.tsx` | 소폭 | 신규 라우트 3개 추가 |
| `shared/ui/Header.tsx` | 소폭 | Inbox, Published 링크 + 배지 추가 |
| `pages/Pipelines.tsx` | 중간 | 콘텐츠 파이프라인 탭, 카드 정보 추가 |
| `pages/pipelines/PipelineDetail.tsx` | 대폭 | 탭 구조 전면 재설계 |
| `pages/Inbox.tsx` | 신규 | — |
| `pages/SessionDetail.tsx` | 신규 | — |
| `pages/Published.tsx` | 신규 | — |
| `entities/pipeline/` | 소폭 | Context, Sources 타입/API 추가 |
| `widgets/pipeline-editor/` | 검토 필요 | PipelineDetail 위젯과 분리/통합 결정 필요 |
