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
    ├ [코드] 정적 소스 fetch  ─┐
    │   HN RSS, Reuters 등    │
    ├ [코드] 트래픽 신호 fetch ─┼─ 병렬/순차 → 세션에 통합 저장
    │   Google Trends, Reddit │
    └ [코드] 서지 감지 모니터  ─┘
    │
    └ [LLM] 종합 분석 (수집된 원시 데이터 기반)
        필터링 + 요약 + 각도 제안
        → 사용자 알림 (Discord DM + Upal UI 업데이트)

[서지 감지 — 별도 실시간 레이어]

  키워드 언급 급증 감지 (코드)
    → 알림만 전송: "🔥 서지 감지: DeepSeek 급증 중"
    → 세션 생성은 사용자 결정 (수동 트리거)

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

## 파이프라인 사전 정의 (Editorial Brief)

파이프라인은 생성 시 한 번 사전 정의된다. 이 컨텍스트는 하위 모든 레이어(소스 수집 → LLM 분석 → 워크플로우 제작)에 시스템 프롬프트로 주입된다.

```go
type PipelineContext struct {
    Purpose         string   // 주제/목적 설명
    TargetAudience  string   // 타겟 독자
    ToneStyle       string   // 톤/스타일 지침
    FocusKeywords   []string // 포커스 키워드 (수집 필터링 기준)
    ExcludeKeywords []string // 제외 키워드
    ContentGoals    string   // 콘텐츠 목표 (빈도, 포맷)
    Language        string   // "ko" | "en"
}
```

### 컨텍스트 전파 흐름

```
Pipeline.Context (사전 정의)
  │
  ├──→ 소스 수집 툴 (코드)
  │       포커스 키워드로 쿼리 파라미터 구성
  │       제외 키워드로 클라이언트 사이드 필터링
  │
  ├──→ LLM 분석 단계
  │       "타겟 독자 관점에서 가치 있는 각도 제안"
  │       "목적/주제에 맞는 인사이트 추출"
  │
  └──→ 제작 워크플로우 (블로그/쇼츠/뉴스레터)
          "정의된 톤/스타일로 작성"
          "정의된 언어로 작성"
          "정의된 독자를 위해 최적화"
```

동일한 소스(HN, Reuters)에서도 파이프라인 컨텍스트가 다르면 완전히 다른 성격의 콘텐츠가 생성된다.

---

## 소스 수집 — 코드 레이어

**소스 수집은 전적으로 코드(Go Tool 구현)이다. LLM 없음.**
수집된 원시 데이터가 세션에 저장되면, 그 다음 단계에서 LLM이 분석한다.

### 소스 타입 1: 정적 소스 (발행 기반)

무엇이 발행됐는지를 가져온다.

| 툴 | 방식 | 비용 |
|---|---|---|
| `rss_feed` (기존) | RSS/Atom 피드 파싱 | 무료 |
| `get_webpage` (기존) | 뉴스 사이트 스크래핑 | 무료 |
| `http_request` (기존) | 뉴스 API 호출 | 일부 유료 |

### 소스 타입 2: 트래픽 신호 소스 (인게이지먼트 기반)

무엇이 실제로 주목받고 있는지를 가져온다.

| 신규 툴 | 신호 | API | 비용 |
|---|---|---|---|
| `hn_fetch` | 포인트+댓글 기반 인기글 | HN Firebase API | 무료 |
| `reddit_fetch` | upvote+댓글 기반 hot/rising | Reddit OAuth API | 무료 |
| `google_trends` | 검색량 급증 키워드 | Trends API (비공식) / SerpAPI | 무료~유료 |
| `youtube_trending` | 조회수 급증 영상 | YouTube Data API v3 | 무료 쿼터 내 |
| `github_trending` | star 급증 리포 | HTML 스크래핑 | 무료 |
| `news_volume` | 특정 키워드 기사 수 추적 | NewsAPI.org | 무료 티어 |
| `product_hunt` | upvote 급증 신제품 | Product Hunt API | 무료 |
| `financial_data` | 주가·코인 시세·거래량 | Alpha Vantage / Polygon.io | 무료~유료 |

### 정적 vs 트래픽 신호 조합 효과

```
정적 소스만 사용:     "무엇이 발행됐나"
트래픽 신호만 사용:   "무엇이 주목받나"
두 가지 조합:         "발행됐고 + 실제로 반응 있는" → 콘텐츠 가치 높음
```

### 파이프라인 소스 설정 UI

```
파이프라인 설정: IT AI

수집 소스                          [+ 소스 추가]
  ─────────────────────────────────────────────
  [정적]  HN RSS         https://...        [삭제]
  [정적]  Reuters AI     https://...        [삭제]
  [신호]  Reddit         r/MachineLearning  min: 100 upvotes  [삭제]
  [신호]  Google Trends  키워드: AI, LLM    [삭제]
  [신호]  HN Fetch       min: 50 points     [삭제]

스케줄: [cron]  0 */6 * * *   (6시간마다)
알림:   [Discord] webhook URL: ...

                              [지금 수집하기 ▶]  ← 수동 트리거
```

### 세션 생성 트리거 (두 가지 모두 동일한 플로우)

```
트리거 방식
  ├── 스케줄러 (cron)     → 자동, 주기적으로 새 Session 생성
  └── 사용자 수동 트리거   → "지금 수집하기" 버튼 → 동일하게 새 Session 생성

트리거 방식과 무관하게:
  새 ContentSession 생성 → [코드] 소스 수집 → [LLM] 분석 → 사용자 알림
```

---

## 서지 감지 (알림 전용)

서지 감지는 독립 실시간 모니터링 레이어다. **자동으로 세션을 생성하거나 콘텐츠를 만들지 않는다.**

```go
// 서지 감지 조건 예시
type SurgeConfig struct {
    Keyword        string
    WindowMinutes  int     // 감지 윈도우
    Multiplier     float64 // 평소 대비 N배 급증
}

// 감지 시 동작: 알림만
→ Discord: "🔥 서지 감지: [키워드] 언급 10배 급증 중"
→ UI 인박스: 서지 배지 표시 + [세션 생성] 버튼

// 세션 생성은 항상 사용자 결정
사용자가 [세션 생성] 클릭 → 일반 수동 트리거와 동일한 플로우
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
    TriggerType string    // "schedule" | "manual" | "surge"
    CreatedAt   time.Time
    ReviewedAt  *time.Time
}
```

### SourceFetch

```go
type SourceFetch struct {
    ID         string
    SessionID  string
    SourceType string    // "static" | "signal"
    SourceName string    // "hn_rss" | "reddit" | "google_trends" | ...
    RawItems   []SourceItem
    FetchedAt  time.Time
}

type SourceItem struct {
    Title       string
    URL         string
    Content     string
    Score       int     // 원본 소스 점수 (HN points, Reddit upvotes, 검색량 등)
    SignalType  string  // 트래픽 신호 소스의 경우 신호 종류 명시
}
```

### LLMAnalysis

```go
type LLMAnalysis struct {
    ID              string
    SessionID       string
    FilteredCount   int            // 필터링 후 선별된 아이템 수
    Summary         string         // 종합 요약
    Insights        []Insight
    SuggestedAngles []ContentAngle
    CreatedAt       time.Time
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
    ExternalURL   string
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

# Source Fetches
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
│  🔥 서지 감지: DeepSeek 10배 급증 중  [세션 생성]        │
│  ─────────────────────────────────────────────────────  │
│  IT AI  · Session 12 · Jan 2 08:00          Score: 94  │
│  "GPT-5 출시로 AI 경쟁 재편, 투자 관점 주목"             │
│  정적(HN:15, Reuters:8) 신호(Trends:급증, Reddit:234)   │
│  [▶ 세션 보기]  [✅ 승인]  [❌ 거절]                    │
│  ─────────────────────────────────────────────────────  │
│  Finance · Session 8  · Jan 2 06:00         Score: 87  │
│  "BTC ETF 유입 급증, 기관 수요 사상 최고"                │
│  정적(Reuters:12) 신호(Trends:급증, 거래량:+340%)       │
│  [▶ 세션 보기]  [✅ 승인]  [❌ 거절]                    │
└─────────────────────────────────────────────────────────┘
```

### 2. 세션 상세 (드릴다운)

```
IT AI Pipeline > Session 12

[소스]  [LLM 분석]  [결과물]

── 소스 탭 ─────────────────────────────────
[정적] HN RSS (15개)
  • "OpenAI announces GPT-5..."  pts:847  🔗
  • "Anthropic raises $..."      pts:623  🔗

[신호] Reddit r/MachineLearning (top 5)
  • "GPT-5 is here..."  upvotes:4,200  🔗

[신호] Google Trends
  • "GPT-5": 검색량 전주 대비 +840%
  • "OpenAI": 전주 대비 +320%

── LLM 분석 탭 ─────────────────────────────
정적+신호 종합 55개 → 7개 선별
요약: "GPT-5 출시 + 검색량 급증 + 커뮤니티 폭발적 반응..."

추천 각도:
  ● 쇼츠: "GPT-5 출시 30초 요약"
  ● 블로그: "GPT-5가 바꾸는 AI 생태계 지형도"

[✅ 승인하고 제작] → [✓] 블로그  [✓] 쇼츠  [실행]

── 결과물 탭 ────────────────────────────────
Blog Workflow Run #42   [완료] → [리뷰] [게시]
Shorts Workflow Run #43 [실행 중...]
```

---

## 툴 목록

### 소스 수집 툴 (코드 전용, LLM 없음)

| 툴 | 타입 | 기능 | API |
|---|---|---|---|
| `rss_feed` (기존) | 정적 | RSS/Atom 파싱 | — |
| `get_webpage` (기존) | 정적 | 웹 스크래핑 | — |
| `http_request` (기존) | 정적/신호 | HTTP API 호출 | — |
| `hn_fetch` (신규) | 신호 | HN 포인트 기반 인기글 | HN Firebase API |
| `reddit_fetch` (신규) | 신호 | hot/rising 포스트 | Reddit OAuth API |
| `google_trends` (신규) | 신호 | 검색량 급증 키워드 | SerpAPI / Trends |
| `youtube_trending` (신규) | 신호 | 카테고리별 트렌딩 영상 | YouTube Data API v3 |
| `github_trending` (신규) | 신호 | star 급증 리포 | HTML 스크래핑 |
| `news_volume` (신규) | 신호 | 키워드 기사 수 추적 | NewsAPI.org |
| `product_hunt` (신규) | 신호 | upvote 급증 신제품 | Product Hunt API |
| `financial_data` (신규) | 신호 | 주가·코인·거래량 | Alpha Vantage / Polygon.io |

### 제작·발행 툴 (LLM 워크플로우에서 사용)

| 툴 | 기능 | API |
|---|---|---|
| `tts` (신규) | 텍스트 → 음성 나레이션 | OpenAI TTS / ElevenLabs |
| `image_gen` (신규) | 썸네일·시각자료 생성 | DALL-E 3 / Stable Diffusion |
| `youtube_upload` (신규) | 영상 업로드 + 메타데이터 | YouTube Data API v3 |
| `substack_publish` (신규) | 뉴스레터 발행 | Substack API / SMTP |
| `discord_notify` (신규) | 대화형 알림 (버튼 포함) | Discord Interactions API |

---

## 워크플로우 템플릿 (사전 제공)

| 템플릿 | 노드 구성 |
|--------|---------|
| Blog Writer | Input → Agent(초안) → Agent(편집) → Output |
| Shorts Script | Input → Agent(스크립트) → tts → image_gen → Output |
| Newsletter | Input → Agent(큐레이션) → Agent(포맷) → substack_publish |
| Long-form Video | Input → Agent(구성) → Agent(스크립트) → tts → Output |

---

## Discord 봇 알림 포맷

### 일반 세션 완료 알림
```
🤖 Upal Content Bot

[IT AI] 새 세션 분석 완료 (Score: 94)
소스: 정적 23건 + 신호 4종 → 선별 7개
요약: GPT-5 출시로 AI 경쟁 재편...
추천: 블로그 + 쇼츠

✅ 승인    ❌ 거절    🔗 세션 보기
```

### 서지 감지 알림 (알림만, 승인 버튼 없음)
```
🔥 서지 감지: DeepSeek

언급량 1시간 내 10배 급증
출처: HN, Twitter, Google Trends 동시 급증

🔗 세션 생성하기   (직접 Upal에서 결정)
```

---

## 구현 순서 (Phase)

### Phase 1 — 데이터 기반 (2–3주)
- `ContentSession`, `SourceFetch`, `LLMAnalysis`, `PublishedContent` 모델 + DB
- 신규 API 엔드포인트
- `WorkflowRun.session_id` 필드 추가
- 통합 인박스 UI `/inbox` (서지 배지 포함)
- 세션 상세 드릴다운 UI (정적/신호 소스 구분 표시)

### Phase 2 — 소스 수집 툴 (2–3주)
- Go Tool 구현: `hn_fetch`, `reddit_fetch`, `google_trends`
- Go Tool 구현: `youtube_trending`, `github_trending`, `news_volume`
- Go Tool 구현: `financial_data`, `product_hunt`
- 파이프라인 소스 설정 UI (타입별 파라미터)
- 스케줄러 → ContentSession 자동 생성 연결
- 서지 감지 모니터링 레이어 (알림 전용)

### Phase 3 — 제작 파이프라인 (3–4주)
- Go Tool 구현: `tts`, `image_gen`
- 블로그·쇼츠·뉴스레터 워크플로우 템플릿
- 세션 승인 → N개 워크플로우 동시 트리거
- 결과물 리뷰 → 게시 흐름 UI

### Phase 4 — 발행 & 알림 (2주)
- Go Tool 구현: `youtube_upload`, `substack_publish`
- Go Tool 구현: `discord_notify` (대화형 버튼)
- Discord 봇 연동 (세션 알림 + 서지 알림 분리)
- `PublishedContent` 이력 + 통계 UI

---

## 성공 지표

- 파이프라인 세션 → 게시까지 사람 개입: 리뷰 2회 외 0
- 소스 수집 → 사용자 알림: 10분 이내
- 멀티 파이프라인 동시 실행: 지연 없음
- 전체 계보: Pipeline → Session → Source → LLM → Workflow → Published 드릴다운 가능
- 서지 감지 → 알림: 1시간 이내
