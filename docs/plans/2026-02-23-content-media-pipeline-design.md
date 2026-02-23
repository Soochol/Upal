# Content Media Pipeline — Design Document
*2026-02-23*

## Overview

Upal을 자동화된 콘텐츠 제작 플랫폼으로 확장한다. 다수의 소스에서 주기적으로 소재를 발굴하고, 사용자 리뷰를 거쳐 블로그·YouTube Shorts·롱폼 영상 등을 자동 제작·발행하는 풀 미디어 오퍼레이션 파이프라인을 구축한다.

## 목표 수익 모델

- YouTube 광고 수익 (YPP)
- Substack 뉴스레터 구독
- Discord/Patreon 월정액 멤버십

---

## 전체 아키텍처

```
[자동 레이어 — 스케줄러/트리거, 계속 반복]

Pipeline (IT AI)          Pipeline (Finance)        Pipeline (Global Tech)
  cron: 6h                  cron: 1h                  trigger: manual
  │                         │                         │
  ▼ 새 Session 생성          ▼                         ▼
  Session N
    ├ Source: HN RSS     ─┐
    ├ Source: Reuters    ─┼─ 병렬/순차 수집 → 세션 안에 통합 저장
    └ Source: Twitter/X  ─┘
    │
    └ LLM 종합 분석 (필터링 + 요약 + 각도 제안)
        → 사용자 알림 (Discord DM + Upal UI 업데이트)

[리뷰 게이트 1 — 사용자 주도, 비동기]

사용자: 통합 인박스에서 세션 리뷰
  ├ Session N → ✅ 승인 → 워크플로우 선택 [블로그] [쇼츠] [뉴스레터]
  └ Session M → ❌ 거절

[제작 레이어 — 자동]

승인된 Session → 병렬 워크플로우 실행
  ├ Blog Workflow     → 초안 생성 (LLM)
  ├ Shorts Workflow   → 스크립트 + TTS + 썸네일
  └ Newsletter Wf    → 뉴스레터 포맷 생성

[리뷰 게이트 2 — 사용자 주도]

사용자: 워크플로우 결과물 리뷰
  ├ 승인 → 자동 발행 (YouTube / Substack / Discord)
  └ 수정 요청 → 재실행

[발행 레이어 — 자동]

  ├ YouTube Data API → 영상 업로드
  ├ Substack API     → 뉴스레터 발행
  └ Discord Webhook  → 멤버십 채널 공유
```

---

## 데이터 모델

### 연결 체인 (전체 계보)

```
Pipeline
  └─ id ──→ ContentSession.pipeline_id
                └─ id ──→ SourceFetch.session_id        (소스별 원본 결과)
                └─ id ──→ LLMAnalysis.session_id        (종합 분석 결과)
                └─ id ──→ WorkflowRun.session_id        (제작 실행, 신규 필드)
                              └─ id ──→ PublishedContent.workflow_run_id
```

### ContentSession

```go
type ContentSession struct {
    ID          string
    PipelineID  string
    Status      string    // "collecting" | "analyzing" | "pending_review"
                          // | "approved" | "rejected" | "producing" | "published"
    SourceCount int       // 수집된 소스 총 개수
    CreatedAt   time.Time
    ReviewedAt  *time.Time
}
```

### SourceFetch

```go
type SourceFetch struct {
    ID        string
    SessionID string
    Source    string    // "hn_rss" | "reuters" | "twitter" | "reddit" | "financial_api"
    RawItems  []SourceItem
    FetchedAt time.Time
}

type SourceItem struct {
    Title   string
    URL     string
    Content string
    Score   int    // 원본 소스 점수 (HN upvotes 등)
}
```

### LLMAnalysis

```go
type LLMAnalysis struct {
    ID           string
    SessionID    string
    FilteredCount int        // 필터링 후 선별된 아이템 수
    Summary      string      // 종합 요약
    Insights     []Insight   // LLM이 발견한 핵심 인사이트
    SuggestedAngles []ContentAngle
    CreatedAt    time.Time
}

type ContentAngle struct {
    Format    string   // "blog" | "shorts" | "newsletter" | "longform"
    Headline  string
    Rationale string
}
```

### PublishedContent

```go
type PublishedContent struct {
    ID            string
    WorkflowRunID string
    Channel       string    // "youtube" | "substack" | "discord"
    ExternalURL   string    // 실제 게시된 URL
    PublishedAt   time.Time
}
```

### WorkflowRun 확장 (기존 + 신규 필드)

```go
// 기존 WorkflowRun에 추가
SessionID  *string  // 어떤 ContentSession에서 생성됐는지
```

---

## API 엔드포인트 (신규)

```
# Content Sessions
GET    /api/content-sessions                    # 전체 파이프라인 통합 인박스
GET    /api/content-sessions?pipeline_id=X      # 파이프라인별 필터
GET    /api/content-sessions/{id}               # 세션 상세 (소스 + 분석 + 결과물)
PATCH  /api/content-sessions/{id}               # 승인/거절/재검토 요청
POST   /api/content-sessions/{id}/produce       # 워크플로우 트리거 (포맷 선택)

# Source Fetches (세션 내 소스 상세)
GET    /api/content-sessions/{id}/sources       # 소스별 원본 목록

# Published Content
GET    /api/published                           # 게시된 콘텐츠 이력
GET    /api/published?session_id=X              # 세션에서 게시된 것
```

---

## UI 설계

### 1. 통합 인박스 (새 페이지: /inbox)

```
┌─────────────────────────────────────────────────────────┐
│  Content Inbox              [All] [Pending] [Approved]  │
│                                                         │
│  IT AI  · Session 12 · Jan 2 08:00          Score: 94  │
│  "GPT-5 출시로 AI 경쟁 재편, 투자 관점 주목"             │
│  Sources: HN(15) Reuters(8) Twitter(32) → 선별: 7개     │
│  [▶ 세션 보기]  [✅ 승인]  [❌ 거절]                    │
│  ─────────────────────────────────────────────────────  │
│  Finance · Session 8  · Jan 2 06:00         Score: 87  │
│  "BTC ETF 유입 급증, 기관 수요 사상 최고"                │
│  Sources: Reuters(12) Bloomberg(6) → 선별: 5개          │
│  [▶ 세션 보기]  [✅ 승인]  [❌ 거절]                    │
└─────────────────────────────────────────────────────────┘
```

### 2. 세션 상세 (드릴다운)

```
IT AI Pipeline > Session 12

[소스]  [LLM 분석]  [결과물]

── 소스 탭 ─────────────────────────────────
HN RSS (15개 수집)
  • "OpenAI announces GPT-5..." 🔗
  • "Anthropic raises $..."    🔗
  ...

Reuters (8개 수집)
  • "AI chip export rules..."  🔗
  ...

── LLM 분석 탭 ─────────────────────────────
필터링: 55개 → 7개 선별
요약: "이번 사이클 핵심은 GPT-5 출시와 AI 반도체 규제..."

추천 각도:
  ● 블로그: "GPT-5가 바꾸는 AI 생태계 지형도"
  ● 쇼츠: "GPT-5 출시 30초 요약"

[✅ 승인하고 제작] → [✓] 블로그  [✓] 쇼츠  [ ] 뉴스레터  [실행]

── 결과물 탭 ────────────────────────────────
Blog Workflow Run #42   [완료] → [리뷰] [게시]
Shorts Workflow Run #43 [실행 중...]
```

### 3. 파이프라인 소스 설정 (기존 파이프라인 설정에 추가)

```
파이프라인 설정: IT AI

수집 소스
  ┌─ [+ 소스 추가] ──────────────────────────┐
  │  ● HN RSS         URL: ...  [삭제]       │
  │  ● Reuters AI     URL: ...  [삭제]       │
  │  ● Twitter/X      키워드: "AI LLM"  [삭제]│
  └──────────────────────────────────────────┘

스케줄: [cron] 0 */6 * * *   (6시간마다)
알림: [Discord] webhook URL: ...
```

---

## 새로 추가할 툴 (Tools)

| 툴 이름 | 기능 | 외부 API |
|---------|------|---------|
| `tts` | 텍스트 → 음성 나레이션 생성 | OpenAI TTS / ElevenLabs |
| `image_gen` | 썸네일·시각자료 생성 | DALL-E 3 / Stable Diffusion |
| `youtube_upload` | 영상 업로드 + 메타데이터 설정 | YouTube Data API v3 |
| `substack_publish` | 뉴스레터 초안·발행 | Substack API / SMTP |
| `discord_notify` | 대화형 알림 (승인 버튼 포함) | Discord Interactions API |
| `financial_data` | 주가·코인 시세·지표 수집 | Alpha Vantage / Polygon.io |

---

## 워크플로우 템플릿 (사전 제공)

| 템플릿 | 노드 구성 |
|--------|---------|
| Blog Writer | Input → Agent(초안) → Agent(편집) → Output |
| Shorts Script | Input → Agent(스크립트) → TTS → image_gen → Output |
| Newsletter | Input → Agent(큐레이션) → Agent(포맷) → substack_publish |
| Long-form Video | Input → Agent(구성) → Agent(스크립트) → TTS → Output |

---

## Discord 봇 알림 포맷

```
🤖 Upal Content Bot

[IT AI] 새 세션 분석 완료 (Score: 94)
─────────────────────────────
요약: GPT-5 출시로 AI 경쟁 재편...
소스: HN 15건 · Reuters 8건 · Twitter 32건
선별: 7개 아이템

추천: 블로그 + 쇼츠

✅ 승인    ❌ 거절    🔗 세션 보기
```

버튼 클릭 → Discord Interactions API → `PATCH /api/content-sessions/{id}` → 프로덕션 파이프라인 자동 트리거

---

## 구현 순서 (Phase)

### Phase 1 — 데이터 기반 (2–3주)
- `ContentSession`, `SourceFetch`, `LLMAnalysis`, `PublishedContent` 모델 + DB
- 신규 API 엔드포인트
- `WorkflowRun.session_id` 필드 추가
- 통합 인박스 UI `/inbox`
- 세션 상세 드릴다운 UI

### Phase 2 — 발굴 자동화 (2–3주)
- 소스 수집 워크플로우 템플릿 (HN, RSS, Twitter, 금융 API)
- `financial_data` 툴 추가
- 파이프라인 소스 설정 UI
- 스케줄러 → ContentSession 자동 생성 연결

### Phase 3 — 제작 파이프라인 (3–4주)
- `tts`, `image_gen` 툴 추가
- 블로그·쇼츠·뉴스레터 워크플로우 템플릿
- 세션 승인 → N개 워크플로우 동시 트리거 메커니즘
- 결과물 리뷰 → 게시 흐름 UI

### Phase 4 — 발행 & 알림 (2주)
- `youtube_upload`, `substack_publish` 툴 추가
- `discord_notify` 툴 (대화형 버튼 포함)
- Discord 봇 알림 연동
- `PublishedContent` 이력 + 통계 UI

---

## 성공 지표

- 파이프라인 세션 → 게시까지 사람 개입 시간: 리뷰 2회 (게이트 1, 2) 외 0
- 소스 수집 → 사용자 알림: 10분 이내
- 멀티 파이프라인 동시 실행: 지연 없음
- 전체 계보 추적: Pipeline → Session → Source → LLM → Workflow → Published 드릴다운 가능
