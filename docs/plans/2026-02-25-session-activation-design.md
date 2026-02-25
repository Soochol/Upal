# Session Template Activation & Schedule Integration

## Problem

"Start Session" 버튼이 1회성 collect를 실행하고 "수집을 시작했습니다" 토스트를 보여준다. 실제 의도는 세션 템플릿을 활성화하여 설정된 cron 스케줄에 따라 주기적으로 인스턴스를 생성하고 Inbox로 보내는 것이다. 현재 세션의 `schedule` 필드(cron)는 스케줄러에 연결되지 않는다.

## Design

### 상태 모델 변경

템플릿에 `active` 상태 추가:

| 상태 | 의미 | 설정 수정 | 스케줄 |
|------|------|-----------|--------|
| `draft` | 설정 중 | 가능 | 미등록 |
| `active` | 운영 중 | 가능 | 등록됨 |

### 백엔드

#### 새 API 엔드포인트

- `POST /api/content-sessions/{id}/activate` — 템플릿의 schedule(cron)을 스케줄러에 등록, 상태를 `active`로 전환
- `POST /api/content-sessions/{id}/deactivate` — 스케줄러에서 제거, 상태를 `draft`로 전환

#### Schedule 모델 확장

기존 `Schedule` 구조체에 `SessionID` 필드 추가:

```go
type Schedule struct {
    // ... 기존 필드 ...
    PipelineID   string  // 기존: 파이프라인 스케줄용
    SessionID    string  // 신규: 세션 템플릿 스케줄용
}
```

#### 스케줄러 디스패치 확장

`executeScheduledRun()`에서 `schedule.SessionID`가 있으면:
1. 해당 세션 템플릿 조회
2. 템플릿의 설정(sources, model, workflows, context) 복사하여 인스턴스 생성
3. `CollectAndAnalyze()` 실행

#### activate 핸들러 로직

1. 세션 조회 + `is_template=true` 검증
2. `schedule` 필드(cron) 존재 검증
3. `Schedule` 레코드 생성 (SessionID = 세션ID, CronExpr = 세션.schedule)
4. 세션에 `schedule_id` 저장 (스케줄 추적용)
5. 상태를 `active`로 전환
6. 즉시 1회 수집 실행 (첫 인스턴스 바로 생성)

#### deactivate 핸들러 로직

1. 세션의 `schedule_id`로 스케줄 삭제
2. `schedule_id` 클리어
3. 상태를 `draft`로 전환

### 프론트엔드

#### Start/Stop 토글 버튼 (SessionSetupView)

- `draft` 상태: **Start Session** (Play 아이콘) → activate API 호출
- `active` 상태: **Stop Session** (Square 아이콘) → deactivate API 호출
- 토스트: "세션이 활성화되었습니다. 스케줄에 따라 수집이 진행됩니다." / "세션이 비활성화되었습니다."

#### 세션 카드 활성 애니메이션 (SessionListPanel)

`active` 상태 템플릿:
- Status dot: pulse 애니메이션 (확대/축소 + 투명도 변화)
- 카드 테두리: 은은한 glow 효과 (primary 색상 기반 shadow)
- 과하지 않게 "살아있다"는 느낌

### 데이터 모델 변경 요약

**ContentSession** 추가 필드:
- `ScheduleID string` — 연결된 스케줄러 레코드 ID

**Schedule** 추가 필드:
- `SessionID string` — 세션 템플릿 참조

**ContentSessionStatus** 추가 값:
- `"active"` — 스케줄 등록됨
