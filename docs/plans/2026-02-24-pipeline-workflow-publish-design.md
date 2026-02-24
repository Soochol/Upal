# Pipeline Workflow + Publish Channel Design

## Problem

현재 파이프라인 UX는 소스 수집 → 분석이 완료되어야만 워크플로우를 선택할 수 있다. 이미 사용할 워크플로우를 아는 전문 사용자에게는 불필요한 대기. 또한 Produce 완료 후 결과물을 리뷰하고 개별 승인/발행하는 전용 페이지가 없다.

## Design

### 1. PublishChannel — 독립 엔티티

워크플로우 결과물의 배포 대상. Connection과 별개로 관리.

```go
type PublishChannel struct {
    ID   string             `json:"id"`   // "ch-" prefix
    Name string             `json:"name"` // "회사 블로그", "YouTube 메인"
    Type PublishChannelType `json:"type"` // wordpress, youtube, slack, telegram, substack, discord, medium, tiktok, http
}
```

API: CRUD `/api/publish-channels` (Connection 패턴 동일)
Repository: memory 구현 (DB optional)

### 2. PipelineWorkflow 확장

```go
type PipelineWorkflow struct {
    WorkflowName string `json:"workflow_name"`
    Label        string `json:"label,omitempty"`
    AutoSelect   bool   `json:"auto_select,omitempty"`
    ChannelID    string `json:"channel_id,omitempty"` // PublishChannel 참조
}
```

### 3. WorkflowResult 확장

```go
type WorkflowResult struct {
    WorkflowName string     `json:"workflow_name"`
    RunID        string     `json:"run_id"`
    Status       string     `json:"status"`
    OutputURL    string     `json:"output_url,omitempty"`
    CompletedAt  *time.Time `json:"completed_at,omitempty"`
    ChannelID    string     `json:"channel_id,omitempty"` // Pipeline에서 전파
}
```

### 4. 파이프라인 마법사 Step 4 변경

현재: 워크플로우만 선택
변경 후: 워크플로우 + 배포 채널 매핑

```
┌─ blog-producer ──────────────── WordPress ─┐
│  테크 블로그 제작        →  회사 블로그       │
└────────────────────────────────────────────┘
┌─ shorts-producer ────────────── YouTube ───┐
│  숏폼 영상 제작          →  YouTube 메인     │
└────────────────────────────────────────────┘
[+ Add Workflow]
```

- 워크플로우 선택: 기존 WorkflowPicker 재사용
- 배포처: PublishChannel 드롭다운 + "새 채널 만들기" 인라인 옵션
- 배포처는 선택사항 — 비워두면 Publish Inbox에서 수동 지정
- Step 4 전체 스킵 가능 → Analyze에서 확정

### 5. 세션 실행 흐름

```
파이프라인에 workflows 설정됨?
  ├─ YES → Collect → Analyze(사전 워크플로우 확인) → 승인 → Produce 병렬 → Publish Inbox → Published
  └─ NO  → Collect → Analyze(LLM추천 + 수동선택) → 승인 → Produce 병렬 → Publish Inbox → Published
```

- Analyze + 사용자 승인은 항상 필수 (워크플로우 사전 설정 여부 무관)
- YES 경로: Analyze에서 "이 콘텐츠로 이 워크플로우들 실행합니다" 확인
- NO 경로: 현재처럼 LLM 추천 보고 워크플로우+채널 직접 선택
- Produce는 errgroup 병렬 실행 (이미 구현됨)
- 중복 선택 방지: 같은 workflow_name deduplicate

### 6. 페이지 체계

```
/inbox           →  Analysis Inbox    (Produce 전 리뷰)     — 기존
/publish-inbox   →  Publish Inbox     (Publish 전 리뷰)     — 신규
/published       →  Published         (발행 완료 기록)       — 기존
```

#### Publish Inbox (신규)

Inbox와 동일한 패턴: 사이드바 리스트 + 상세 프리뷰

- 사이드바: `status === 'approved'` (produce 완료, publish 대기) 세션 목록
- 상세: 워크플로우별 결과물 카드
  - 각 카드: 콘텐츠 프리뷰 + 매핑된 채널 표시 + 개별 승인/거절 + Publish 버튼
  - 승인 → 해당 채널로 Publish (데이터 모델만, 실제 발행 연동은 다음 단계)
  - 거절 → publish 안 함
  - 모든 워크플로우 처리 완료 시 세션 → published

### 7. 스코프

이번 구현:
- PublishChannel CRUD (백엔드 + 프론트엔드)
- PipelineWorkflow.channel_id, WorkflowResult.channel_id 추가
- 마법사 Step 4 워크플로우+채널 매핑 UI
- Publish Inbox 페이지 (사이드바 + 프리뷰 + 개별 승인/Publish)
- channel_id 전파: Pipeline → ProduceWorkflows → WorkflowResult → PublishedContent

다음 단계 (이번 미포함):
- 실제 외부 플랫폼 발행 연동 (WordPress API, YouTube API 등)
- Analyze 단계에서 채널 선택 UI (NO 경로)
