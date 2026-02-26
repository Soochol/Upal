# Session / Instance Redesign

## Problem

Pipeline, Session, Instance의 3단계 계층이 필요하지만, 현재 `ContentSession` 테이블이 템플릿(설정)과 인스턴스(실행 결과)를 하나의 타입으로 혼재하고 있어 UI에서 역할 구분이 안 됨.

## Target Model

```
Pipeline
  └── Session (설정 + 스케줄, start/stop) ×N    ← IsTemplate=true
        └── Instance (실행 결과, Inbox로 감) ×N  ← IsTemplate=false, ParentSessionID 참조
```

## Design Decisions

- **새 테이블 없음** — 기존 `ContentSession`의 `IsTemplate` + `ParentSessionID`를 그대로 활용
- **백엔드 최소 변경** — `ContentSessionDetail`에 `session_name` 필드 추가만
- **프론트엔드 중심 변경** — 페이지별 역할을 명확히 분리

## Changes by Area

### 1. Backend

`ContentSessionDetail`에 `session_name` (string) 추가. 인스턴스의 `ParentSessionID`로 부모 세션을 조회해서 이름을 채움. 부모가 없거나 삭제된 경우 빈 문자열.

기존 API, 리포지토리, Collector, Scheduler — 변경 없음.

### 2. Pipelines Page — Session List

`SessionListPanel`이 `templateOnly: true`로 템플릿 세션만 조회.
- 각 항목: 세션명 + 상태(draft/active)
- "New" 버튼: 새 템플릿 세션 생성 (`createDraftSession` with `is_template: true`)

### 3. Pipelines Page — Session Detail (Settings Editor)

기존 `SessionDetailPreview`(Collect→Analyze→Produce 뷰) 대신 설정 편집 폼:
- Name (인라인 편집)
- Sources (기존 소스 편집 UI 재사용)
- Schedule (cron 입력)
- Model (모델 선택)
- Workflows (워크플로 선택)
- Context (에디토리얼 브리프: purpose, audience, tone, keywords, goals)
- Start/Stop 토글 — `activate`/`deactivate` API. active 상태면 스케줄대로 인스턴스 생성
- 수동 실행 버튼 — 즉시 인스턴스 1개 생성 (기존 `collectPipeline` API, 세션 ID 기준으로 변경 필요)

설정 변경: `updateSessionSettings` API + auto-save.

### 4. Inbox — Instance List

인스턴스 목록 아이템에 파이프라인명 + 세션명 둘 다 표시.
백엔드가 `pipeline_name` + `session_name`을 채워주므로 프론트에서 그대로 렌더.

### 5. Inbox — Instance Detail

기존 `SessionDetailPreview` 그대로 사용 (Collect→Analyze→Produce→Publish 단계 뷰). 변경 없음.

## Not Changing

- DB 스키마, 도메인 모델 (`ContentSession` struct)
- Repository layer
- Collector (`CollectAndAnalyze`)
- Scheduler
- `activate`/`deactivate` API (검증 후 그대로 사용)
