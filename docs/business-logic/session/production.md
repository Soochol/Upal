# 콘텐츠 제작과 발행

> 관련 코드: `internal/services/content_collector.go` (ProduceWorkflowsV2), `internal/api/session_runs.go` (produceNewRun, publishNewRun)

## 개요

제작(Produce)은 분석에서 승인된 앵글을 기반으로 **실제 워크플로우를 실행해 콘텐츠를 만드는** 단계이다. 발행(Publish)은 제작된 결과물을 **외부 채널에 기록**하는 최종 단계이다.

```
사용자가 앵글 선택 + 워크플로우 지정
  │
  ├─ produce 요청 (워크플로우 목록)
  │
  ├─ producing 상태로 전환
  ├─ 각 워크플로우 병렬 실행
  │     ├─ pending → running → success / failed
  │     └─ 실행 이력(RunHistory)에 기록
  │
  ├─ 하나라도 성공 → approved, 전부 실패 → error
  │
  └─ 사용자가 publish 요청 → published 상태로 전환
```

## 제작 단계

### 요청

사용자가 produce를 요청하면 **워크플로우 이름 + 채널 ID** 쌍의 목록을 전달한다:
```json
{
  "workflows": [
    { "name": "tech-blog-writer", "channel_id": "ch-youtube" },
    { "name": "shorts-creator", "channel_id": "ch-tiktok" }
  ]
}
```

### 실행 과정

1. 각 워크플로우에 대해 **WorkflowRun** 레코드를 `pending` 상태로 생성
2. Run 상태를 `producing`으로 전환
3. 모든 워크플로우를 **병렬로 실행** (errgroup 사용)

각 워크플로우 실행:
1. `pending` → `running`으로 전환
2. 워크플로우 정의 조회 (없으면 `failed`)
3. **입력 데이터 조합**: 분석 요약 + 인사이트 + 앵글 + 수집된 소스 원문을 하나의 텍스트로 합성
4. 워크플로우의 모든 `input` 노드에 이 텍스트를 주입
5. 워크플로우 실행 (DAG 기반, 이벤트 스트리밍)
6. 실행 결과에 따라 `success` 또는 `failed` 전환
7. 실행 이력(RunHistory)에 시작/완료/실패 기록

### 입력 데이터 구성

워크플로우에 주입되는 입력은 다음 순서로 조합된다:

```
## Summary
(분석 요약)

## Key Insights
- (인사이트 1)
- (인사이트 2)

## Suggested Angles
- [blog] 헤드라인 1
- [shorts] 헤드라인 2

## Collected Sources
### 소스 제목
URL: ...
(본문 500자까지)
---
```

### 최종 상태 결정

모든 워크플로우 실행이 끝나면:
- **하나라도 성공** → Run을 `approved` 상태로 전환
- **전부 실패** → Run을 `error` 상태로 전환

### 에러 추적

워크플로우 실행 실패 시 다음 정보가 기록된다:
- `error_message`: 에러 메시지
- `failed_node_id`: 실패한 노드 ID (이벤트에서 추출, 없으면 NodeRun 기록에서 검색)
- `run_id`: 실행 이력 ID (RunHistory에서 상세 추적 가능)

---

## 발행 단계

### 요청

사용자가 publish를 요청하면 **발행할 워크플로우 실행 ID 목록**을 전달한다:
```json
{
  "run_ids": ["run-abc", "run-def"]
}
```

### 처리

1. 지정된 워크플로우 실행들의 상태를 `published`로 변경
2. 각각에 대해 **PublishedContent** 레코드 생성 (채널, 제목, 발행 시각)
3. 채널이 미지정이면 `"default"` 사용
4. **전체 상태 확인**: Run의 모든 워크플로우 실행이 터미널 상태(published/rejected/failed)이면 Run 자체를 `published`로 전환

### 거부

사용자가 Run을 거부하면 `rejected` 상태로 전환되고 `reviewed_at` 타임스탬프가 기록된다.

---

## 워크플로우 자동 생성

분석에서 제안된 앵글에 매칭되는 워크플로우가 없을 때, 사용자는 **자동 생성**을 요청할 수 있다:

1. 앵글의 포맷과 헤드라인을 기반으로 워크플로우 설명 생성
2. Session 컨텍스트(프롬프트, 언어)를 추가 정보로 포함
3. LLM 기반 워크플로우 생성기로 워크플로우를 생성
4. 이름 충돌 시 랜덤 접미사를 붙여 재시도
5. 생성된 워크플로우를 저장하고 앵글에 연결 (`match_type: "generated"`)

---

## 스케줄 실행

Run은 `schedule` 필드에 cron 표현식을 가질 수 있다:
- `schedule_active`가 `true`이면 스케줄러에 의해 자동 실행
- `@once` 스케줄: 활성화하면 **즉시 1회 실행**하고 스케줄을 비활성 상태로 유지
- 스케줄 실행 시 `trigger_type`은 `"scheduled"`로 기록
