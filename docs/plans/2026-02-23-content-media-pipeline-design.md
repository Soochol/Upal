# Content Media Pipeline — System Design
*2026-02-23*

> **관련 문서**: 프론트엔드 설계 → `2026-02-23-content-media-pipeline-frontend.md`

---

## 1. 개요

Upal을 자동화된 콘텐츠 제작 플랫폼으로 확장한다. 다수의 소스에서 주기적으로 소재를 발굴하고, 사용자 리뷰 게이트를 거쳐 블로그·YouTube Shorts·롱폼 영상·뉴스레터를 자동 제작·발행하는 풀 미디어 오퍼레이션을 구축한다.

### 목표 수익 모델
- YouTube 광고 수익 (YPP)
- Substack 뉴스레터 구독
- Discord/Patreon 월정액 멤버십

---

## 2. 전체 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│  자동 레이어 (스케줄러/트리거, 계속 반복)                          │
│                                                                 │
│  Pipeline A (IT AI)    Pipeline B (Finance)    Pipeline C ...   │
│  cron: 6h              cron: 1h                trigger: manual  │
│       │                     │                       │           │
│       ▼  새 Session 생성     ▼                       ▼           │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Session N                                              │   │
│  │   [코드] 정적 소스 fetch ─┐                             │   │
│  │   [코드] 트래픽 신호 fetch ┼─ 병렬 실행 → 세션에 통합    │   │
│  │   [코드] 서지 감지 모니터 ─┘  저장                       │   │
│  │        ↓                                               │   │
│  │   [LLM] 종합 분석 (필터링 + 요약 + 각도 제안)            │   │
│  │        ↓                                               │   │
│  │   사용자 알림 (Discord + UI)                            │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  서지 감지 레이어 (독립 실시간 모니터링)                           │
│                                                                 │
│  키워드 언급 급증 감지 → 알림만 전송 (세션 생성 X)                │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  리뷰 게이트 1 (사용자 주도, 비동기)                              │
│                                                                 │
│  통합 인박스 → 세션 리뷰                                         │
│    ✅ 승인 → 워크플로우 선택 [블로그] [쇼츠] [뉴스레터]           │
│    ❌ 거절 → 아카이브                                            │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  제작 레이어 (자동)                                               │
│                                                                 │
│  승인된 Session → 병렬 워크플로우 실행                            │
│    Blog Workflow    → LLM 초안 생성                             │
│    Shorts Workflow  → 스크립트 + TTS + 썸네일                   │
│    Newsletter Wf   → 포맷 생성                                  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  리뷰 게이트 2 (사용자 주도)                                      │
│                                                                 │
│  워크플로우 결과물 리뷰                                           │
│    ✅ 승인 → 자동 발행                                           │
│    ✏️  수정 → 재실행                                              │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  발행 레이어 (자동)                                               │
│                                                                 │
│  youtube_upload → YouTube                                      │
│  substack_publish → Substack                                   │
│  discord_notify → Discord 멤버십 채널                           │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 파이프라인 사전 정의 (Editorial Brief)

파이프라인 생성 시 한 번 정의. 이 컨텍스트는 하위 모든 레이어에 주입된다.

```go
type PipelineContext struct {
    Purpose         string   // 주제/목적 설명
    TargetAudience  string   // 타겟 독자
    ToneStyle       string   // 톤/스타일 지침
    FocusKeywords   []string // 포커스 키워드 (수집 필터링)
    ExcludeKeywords []string // 제외 키워드
    ContentGoals    string   // 콘텐츠 목표 (빈도, 포맷)
    Language        string   // "ko" | "en"
}
```

### 컨텍스트 전파

| 레이어 | 적용 방식 |
|--------|---------|
| 소스 수집 툴 (코드) | FocusKeywords를 API 쿼리 파라미터로, ExcludeKeywords로 클라이언트 필터링 |
| LLM 분석 | Purpose + TargetAudience + ToneStyle을 시스템 프롬프트에 주입 |
| 제작 워크플로우 | ToneStyle + Language + TargetAudience를 워크플로우 입력으로 전달 |

동일한 소스(HN, Reuters)에서도 파이프라인 컨텍스트가 다르면 완전히 다른 콘텐츠가 생성된다.

---

## 4. 소스 수집 — 코드 레이어

**소스 수집은 전적으로 Go Tool 구현이다. LLM 없음.**
수집된 원시 데이터를 세션에 저장한 뒤, LLM이 분석한다.

### 4.1 정적 소스 (발행 기반) — 무엇이 발행됐나

| 툴 | 상태 | 기능 |
|---|---|---|
| `rss_feed` | 기존 | RSS/Atom 피드 파싱 |
| `get_webpage` | 기존 | 웹 스크래핑 |
| `http_request` | 기존 | 뉴스 REST API 호출 |

### 4.2 트래픽 신호 소스 (인게이지먼트 기반) — 무엇이 주목받나

| 툴 | 신호 | 외부 API | 비용 |
|---|---|---|---|
| `hn_fetch` | HN 포인트+댓글 기반 인기글 | HN Firebase API | 무료 |
| `reddit_fetch` | upvote+댓글 기반 hot/rising | Reddit OAuth API | 무료 |
| `google_trends` | 검색량 급증 키워드 | SerpAPI / pytrends | 무료~$50/월 |
| `youtube_trending` | 카테고리별 트렌딩 영상 | YouTube Data API v3 | 무료 쿼터 내 |
| `github_trending` | star 급증 리포 | HTML 스크래핑 | 무료 |
| `news_volume` | 키워드 기사 수 추적 | NewsAPI.org | 무료 100req/일 |
| `product_hunt` | upvote 급증 신제품 | Product Hunt API | 무료 |
| `financial_data` | 주가·코인·거래량·지표 | Alpha Vantage / Polygon.io | 무료~유료 |

### 4.3 조합 효과

```
정적 소스만:   "무엇이 발행됐나"
신호만:        "무엇이 주목받나"
두 가지 조합:  "발행됐고 + 실제로 반응 있는" → 콘텐츠 가치 극대화
```

### 4.4 툴 파라미터 구조

```go
// 예: reddit_fetch 툴 입력
type RedditFetchInput struct {
    Subreddits  []string // ["MachineLearning", "artificial"]
    Sort        string   // "hot" | "rising" | "top"
    Limit       int      // 최대 가져올 수
    MinUpvotes  int      // 최소 upvote 필터
    Keywords    []string // 파이프라인 FocusKeywords에서 주입
}
```

---

## 5. 서지 감지 (알림 전용)

독립 실시간 모니터링 레이어. **자동으로 세션을 생성하거나 콘텐츠를 만들지 않는다.**

```go
type SurgeDetector struct {
    // 백그라운드 goroutine으로 실행
    // 각 파이프라인의 FocusKeywords를 모니터링
}

type SurgeConfig struct {
    Keyword       string
    WindowMinutes int     // 감지 윈도우 (기본 60분)
    Multiplier    float64 // 평소 대비 N배 급증 시 알림 (기본 5.0)
}

// 감지 시 동작
→ Discord: "🔥 서지 감지: [키워드] 언급 10배 급증 중"
→ UI: 인박스 상단 서지 배너 표시 + [세션 생성] 버튼
→ 세션 생성: 사용자가 버튼을 눌러야만 실행
```

### 서지 감지 소스
- NewsAPI 키워드 볼륨 (15분 폴링)
- Reddit 언급 급증
- HN 포인트 급증
- Google Trends 실시간 급등

---

## 6. 데이터 모델

### 6.1 전체 연결 체인 (계보)

```
Pipeline (기존 확장)
  └─ PipelineContext (신규 embedded)
  └─ PipelineSources[] (신규)
  └─ id ──→ ContentSession.pipeline_id
                └─ id ──→ SourceFetch.session_id        (소스별 원본 결과)
                └─ id ──→ LLMAnalysis.session_id        (종합 분석 결과)
                └─ id ──→ WorkflowRun.session_id        (제작 실행)
                              └─ id ──→ PublishedContent.workflow_run_id
```

### 6.2 Pipeline 확장

```go
// 기존 Pipeline 구조체에 추가
type Pipeline struct {
    // 기존 필드 유지 ...

    // 신규
    Context PipelineContext
    Sources []PipelineSource
}

type PipelineSource struct {
    ID         string
    PipelineID string
    ToolName   string            // "hn_fetch" | "reddit_fetch" | "rss_feed" | ...
    SourceType string            // "static" | "signal"
    Config     map[string]any    // 툴별 파라미터 (subreddits, min_upvotes 등)
    Enabled    bool
}
```

### 6.3 ContentSession

```go
type ContentSession struct {
    ID          string
    PipelineID  string
    Status      ContentSessionStatus
    TriggerType string    // "schedule" | "manual" | "surge"
    SourceCount int       // 수집된 원시 아이템 총 수
    CreatedAt   time.Time
    ReviewedAt  *time.Time
}

type ContentSessionStatus string
const (
    SessionCollecting    ContentSessionStatus = "collecting"
    SessionAnalyzing     ContentSessionStatus = "analyzing"
    SessionPendingReview ContentSessionStatus = "pending_review"
    SessionApproved      ContentSessionStatus = "approved"
    SessionRejected      ContentSessionStatus = "rejected"
    SessionProducing     ContentSessionStatus = "producing"
    SessionPublished     ContentSessionStatus = "published"
)
```

### 6.4 SourceFetch

```go
type SourceFetch struct {
    ID         string
    SessionID  string
    ToolName   string       // "hn_fetch" | "reddit_fetch" | ...
    SourceType string       // "static" | "signal"
    RawItems   []SourceItem
    Error      *string      // 수집 실패 시 에러 메시지 (부분 실패 허용)
    FetchedAt  time.Time
}

type SourceItem struct {
    Title      string
    URL        string
    Content    string   // 본문 또는 요약
    Score      int      // 원본 점수 (HN points, Reddit upvotes, 검색량 등)
    SignalType string   // "upvotes" | "search_volume" | "article_count" | ...
    FetchedFrom string  // 출처 식별자
}
```

### 6.5 LLMAnalysis

```go
type LLMAnalysis struct {
    ID              string
    SessionID       string
    RawItemCount    int            // 입력된 원시 아이템 수
    FilteredCount   int            // LLM 선별 후 수
    Summary         string         // 종합 요약
    Insights        []string       // 핵심 인사이트 목록
    SuggestedAngles []ContentAngle
    OverallScore    int            // LLM이 평가한 콘텐츠 가치 점수 (0-100)
    CreatedAt       time.Time
}

type ContentAngle struct {
    Format    string   // "blog" | "shorts" | "newsletter" | "longform"
    Headline  string
    Rationale string
}
```

### 6.6 PublishedContent

```go
type PublishedContent struct {
    ID            string
    WorkflowRunID string
    SessionID     string    // 역참조용 (session_id → workflow_run_id → published)
    Channel       string    // "youtube" | "substack" | "discord"
    ExternalURL   string
    Title         string
    PublishedAt   time.Time
}
```

### 6.7 WorkflowRun 확장

```go
// internal/upal/workflow.go 또는 run 타입에 추가
SessionID  *string  // 어떤 ContentSession에서 생성됐는지 (nullable — 일반 실행은 nil)
```

---

## 7. API 엔드포인트

### 7.1 Pipeline 확장 (기존 엔드포인트 확장)

```
PUT  /api/pipelines/{id}           기존 + context, sources 필드 포함
GET  /api/pipelines/{id}           기존 + context, sources 반환

# 소스 설정
GET    /api/pipelines/{id}/sources         소스 목록
POST   /api/pipelines/{id}/sources         소스 추가
DELETE /api/pipelines/{id}/sources/{sid}   소스 삭제

# 수동 트리거
POST   /api/pipelines/{id}/collect         즉시 소스 수집 → 새 Session 생성
```

### 7.2 ContentSession (신규)

```
GET    /api/content-sessions                       통합 인박스 (전 파이프라인)
GET    /api/content-sessions?pipeline_id=X         파이프라인별 필터
GET    /api/content-sessions?status=pending_review 리뷰 대기 목록
GET    /api/content-sessions/{id}                  세션 상세
PATCH  /api/content-sessions/{id}                  상태 변경 (approve/reject)
POST   /api/content-sessions/{id}/produce          워크플로우 트리거
GET    /api/content-sessions/{id}/sources          소스별 원본 목록
GET    /api/content-sessions/{id}/analysis         LLM 분석 결과
GET    /api/content-sessions/{id}/workflows        생성된 워크플로우 목록
```

### 7.3 PublishedContent (신규)

```
GET    /api/published                              전체 발행 이력
GET    /api/published?pipeline_id=X                파이프라인별
GET    /api/published?session_id=X                 세션에서 발행된 것
GET    /api/published?channel=youtube              채널별
```

### 7.4 Surge (신규)

```
GET    /api/surges                                 서지 감지 이력
POST   /api/surges/{id}/dismiss                    서지 알림 닫기
POST   /api/surges/{id}/create-session             서지에서 세션 생성 (수동)
```

---

## 8. 소스 수집 → 세션 생성 플로우

```
POST /api/pipelines/{id}/collect  (또는 스케줄러 트리거)
  │
  ├─ 1. ContentSession 생성 (status: "collecting")
  │
  ├─ 2. Pipeline.Sources 병렬 실행
  │       각 소스 툴 호출 → SourceFetch 레코드 저장
  │       실패한 소스는 error 필드 기록 후 계속 진행 (부분 실패 허용)
  │
  ├─ 3. ContentSession.status → "analyzing"
  │
  ├─ 4. LLM 분석 워크플로우 실행
  │       입력: 모든 SourceFetch.RawItems + Pipeline.Context
  │       출력: LLMAnalysis 저장
  │
  ├─ 5. ContentSession.status → "pending_review"
  │
  └─ 6. 알림 발송
          Discord: 세션 완료 알림 + 승인/거절 버튼
          UI: 인박스 업데이트 (SSE 또는 폴링)
```

---

## 9. 세션 승인 → 워크플로우 트리거 플로우

```
POST /api/content-sessions/{id}/produce
  Body: { "workflows": ["blog", "shorts"] }
  │
  ├─ ContentSession.status → "producing"
  │
  └─ 선택된 워크플로우 템플릿별로 WorkflowRun 생성 (병렬)
       WorkflowRun.session_id = ContentSession.id
       입력: LLMAnalysis.SuggestedAngles + Pipeline.Context
       각 Run은 독립 실행 (기존 workflow 실행 메커니즘 재사용)
```

---

## 10. 툴 목록

### 10.1 소스 수집 툴 (코드 전용)

| 툴 | 상태 | 타입 | 외부 API | 비용 |
|---|---|---|---|---|
| `rss_feed` | 기존 | 정적 | — | 무료 |
| `get_webpage` | 기존 | 정적 | — | 무료 |
| `http_request` | 기존 | 정적/신호 | — | 무료 |
| `hn_fetch` | 신규 | 신호 | HN Firebase API | 무료 |
| `reddit_fetch` | 신규 | 신호 | Reddit OAuth | 무료 |
| `google_trends` | 신규 | 신호 | SerpAPI | $50/월~ |
| `youtube_trending` | 신규 | 신호 | YouTube Data v3 | 무료 쿼터 |
| `github_trending` | 신규 | 신호 | 스크래핑 | 무료 |
| `news_volume` | 신규 | 신호 | NewsAPI.org | 무료 100req/일 |
| `product_hunt` | 신규 | 신호 | PH API | 무료 |
| `financial_data` | 신규 | 신호 | Alpha Vantage | 무료~유료 |

### 10.2 제작 툴 (LLM 워크플로우에서 사용)

| 툴 | 상태 | 기능 | 외부 API | 비용 |
|---|---|---|---|---|
| `tts` | 신규 | 텍스트 → 음성 | OpenAI TTS / ElevenLabs | $0.015/1K chars~ |
| `image_gen` | 신규 | 이미지·썸네일 생성 | DALL-E 3 / SD | $0.04/image~ |
| `youtube_upload` | 신규 | 영상 업로드 | YouTube Data API v3 | 무료 |
| `substack_publish` | 신규 | 뉴스레터 발행 | Substack API / SMTP | 무료 |
| `discord_notify` | 신규 | 대화형 알림 (버튼) | Discord Interactions | 무료 |

---

## 11. 워크플로우 템플릿

| 템플릿 ID | 이름 | 노드 구성 |
|---|---|---|
| `blog_writer` | 블로그 작성 | Input → Agent(초안) → Agent(편집·SEO) → Output |
| `shorts_script` | 쇼츠 스크립트 | Input → Agent(스크립트) → tts → image_gen → Output |
| `newsletter` | 뉴스레터 | Input → Agent(큐레이션) → Agent(포맷) → substack_publish |
| `longform_video` | 롱폼 영상 스크립트 | Input → Agent(구성) → Agent(스크립트) → tts → Output |

모든 템플릿은 입력으로 `session_summary`(LLM 분석 결과)와 `pipeline_context`를 받는다.

---

## 12. Discord 알림 포맷

### 세션 완료 알림
```
🤖 Upal Content Bot

[IT AI] 새 세션 분석 완료  Score: 94/100
───────────────────────────────
정적 23건 + 신호 4종 수집 → 7개 선별
요약: GPT-5 출시로 AI 경쟁 구도 재편...
추천: 블로그 + 쇼츠

✅ 승인    ❌ 거절    🔗 세션 보기
```

### 서지 감지 알림 (버튼 없음, 링크만)
```
🔥 서지 감지: DeepSeek

1시간 내 언급 10배 급증
출처: HN, Twitter, Google Trends 동시 급등

🔗 Upal에서 세션 생성하기
```

버튼 클릭 → Discord Interactions API → `PATCH /api/content-sessions/{id}` → 제작 파이프라인 트리거

---

## 13. 에러 처리

| 시나리오 | 처리 방식 |
|---|---|
| 소스 수집 부분 실패 | SourceFetch.Error 기록 후 계속 진행 (나머지 소스로 분석) |
| LLM 분석 실패 | ContentSession.status → "error", 재시도 버튼 제공 |
| 워크플로우 실행 실패 | 기존 WorkflowRun 에러 처리 재사용, 세션은 영향 없음 |
| 발행 실패 | PublishedContent 미생성, 수동 재시도 UI 제공 |
| Discord 알림 실패 | 로그만, Upal UI는 정상 업데이트 |
| 서지 감지 소스 다운 | 해당 소스 스킵, 다른 소스로 계속 모니터링 |

---

## 14. 보안 고려사항

| 항목 | 처리 방식 |
|---|---|
| 외부 API 키 (Reddit, YouTube 등) | 기존 Connection 암호화 시스템 재사용 |
| Discord Interactions 검증 | Ed25519 서명 검증 (Discord 요구사항) |
| YouTube OAuth | OAuth2 토큰, 기존 Connection 저장소에 저장 |
| Substack/SMTP 자격증명 | 기존 Connection 암호화 저장 |
| Webhook 엔드포인트 | Discord 서명 검증 미들웨어 추가 |

---

## 15. 성능 고려사항

| 항목 | 고려사항 |
|---|---|
| 멀티 파이프라인 동시 실행 | 기존 ConcurrencyLimiter 재사용 |
| 소스 수집 병렬 처리 | goroutine per source, WaitGroup으로 집계 |
| 서지 감지 폴링 | 15분 간격, 파이프라인 수에 비례하지 않게 키워드 합산 쿼리 |
| LLM 분석 대용량 입력 | 소스 아이템 최대 100개로 제한, 초과 시 점수 상위 100개만 전달 |
| SSE 알림 | 기존 SSE 인프라 재사용 (run events와 동일한 패턴) |

---

## 16. 구현 순서 (Phase)

### Phase 1 — 데이터 기반 (2–3주)
- [ ] `Pipeline` 구조체 확장 (`Context`, `Sources` 필드)
- [ ] `ContentSession`, `SourceFetch`, `LLMAnalysis`, `PublishedContent` 모델 + DB 스키마
- [ ] `WorkflowRun.session_id` 필드 추가
- [ ] 신규 API 엔드포인트 (`/content-sessions`, `/published`, `/surges`)
- [ ] Connection 시스템에 신규 API 키 타입 추가 (Reddit, YouTube OAuth 등)

### Phase 2 — 소스 수집 툴 (2–3주)
- [ ] `hn_fetch` Go 툴 구현
- [ ] `reddit_fetch` Go 툴 구현
- [ ] `google_trends` Go 툴 구현
- [ ] `youtube_trending` Go 툴 구현
- [ ] `github_trending` Go 툴 구현
- [ ] `news_volume` Go 툴 구현
- [ ] `financial_data` Go 툴 구현
- [ ] `product_hunt` Go 툴 구현
- [ ] 스케줄러 → ContentSession 자동 생성 연결
- [ ] 서지 감지 모니터링 goroutine 구현

### Phase 3 — 제작 파이프라인 (3–4주)
- [ ] `tts` Go 툴 구현 (OpenAI TTS)
- [ ] `image_gen` Go 툴 구현 (DALL-E 3)
- [ ] 워크플로우 템플릿 4종 정의 및 등록
- [ ] 세션 승인 → N개 워크플로우 동시 트리거 메커니즘
- [ ] LLM 분석 워크플로우 (소스 데이터 → LLMAnalysis 저장)

### Phase 4 — 발행 & 알림 (2주)
- [ ] `youtube_upload` Go 툴 구현
- [ ] `substack_publish` Go 툴 구현
- [ ] `discord_notify` Go 툴 구현 (대화형 버튼)
- [ ] Discord Interactions 웹훅 핸들러 + Ed25519 검증
- [ ] `PublishedContent` 저장 로직

---

## 17. 성공 지표

- 소스 수집 → 사용자 알림: 10분 이내
- 파이프라인 세션 → 게시까지 사람 개입: 리뷰 2회 외 0
- 멀티 파이프라인 동시 실행: 파이프라인 수에 무관하게 지연 없음
- 서지 감지 → 알림: 1시간 이내
- 전체 계보 추적: Pipeline → Session → Source → LLM → Workflow → Published 드릴다운 가능
- 소스 부분 실패 시: 나머지 소스로 세션 정상 완료
