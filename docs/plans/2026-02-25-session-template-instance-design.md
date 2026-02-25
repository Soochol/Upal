# Session Template/Instance Architecture

## Problem

현재 ContentSession은 "설정 템플릿"과 "실행 인스턴스"를 구분하지 않는다. 파이프라인 페이지에서 템플릿과 실행 결과가 섞여 있고, 상태(pending, approved 등)가 설정 화면에 노출되어 혼란을 준다. 필터링과 상태 관리가 필요한 곳은 Inbox인데, 오히려 거기에는 필터가 없다.

## Design

### 데이터 모델

ContentSession에 두 필드 추가:

```go
type ContentSession struct {
    // ... 기존 필드 ...
    IsTemplate      bool   `json:"is_template"`
    ParentSessionID string `json:"parent_session_id,omitempty"`
}
```

| | Template | Instance |
|---|---|---|
| `is_template` | `true` | `false` |
| `parent_session_id` | `""` | 템플릿 ID |
| 상태 전이 | `draft` 고정 | full lifecycle |
| 설정 수정 | 가능 | 불가 (생성 시 복사) |
| 표시 위치 | Pipeline 페이지 | Inbox |

### 인스턴스 생성

템플릿에서 수집 트리거 시:
1. 템플릿의 설정(sources, model, workflows, context)을 복사
2. `is_template=false`, `parent_session_id=템플릿ID` 설정
3. 새 인스턴스로 CollectAndAnalyze() 실행
4. 템플릿은 항상 draft 유지

### API 변경

- `GET /api/content-sessions?pipeline_id=X` → `is_template=true`만 반환
- `GET /api/content-sessions?status=pending_review` → `is_template=false`만 반환
- `POST /api/content-sessions` → `is_template=true`로 생성
- `POST /api/content-sessions/{id}/collect` → 항상 새 인스턴스 생성

### 프론트엔드

**Pipeline 페이지** (현재 구조 유지):
- SessionListPanel: 템플릿만 조회, 상태 dot/배지 없음
- SessionSetupView: 설정 편집 + "수집 시작" 버튼 → 인스턴스 생성

**Review Inbox** (/inbox):
- 인스턴스만 조회 (pending_review)
- 상태별 필터 추가
- Approve → 워크플로우 실행 → Publish Inbox

**Publish Inbox** (/publish):
- 인스턴스만 조회 (approved/producing/error)
- 현재 구조 유지

### 백엔드 변경 범위

1. **데이터 모델** (`internal/upal/content.go`) — 두 필드 추가
2. **저장소** (`internal/repository/`) — persist + 필터링 파라미터
3. **API** (`internal/api/content.go`) — 조회 필터, collect 로직 변경
4. **수집기** (`internal/services/content_collector.go`) — 항상 인스턴스 생성
5. **스케줄러** — 파이프라인 템플릿 찾아 인스턴스 생성
6. **마이그레이션** — 기존 draft → template, non-draft → instance
