# Session + Run 라이프사이클

> 관련 코드: `internal/upal/session.go`, `internal/upal/content.go`, `internal/services/session_service.go`, `internal/services/run_service.go`, `internal/services/content_collector.go`, `internal/api/sessions.go`, `internal/api/session_runs.go`

## 개요

Session은 **콘텐츠 수집-분석-제작 파이프라인의 설정 컨테이너**이다. 소스, 스케줄, 워크플로우, 컨텍스트 등 "어떻게 콘텐츠를 만들 것인가"를 정의한다.

Run은 Session의 **한 번의 실행 사이클**이다. Session 설정을 기반으로 실제 수집→분석→리뷰→제작→발행을 수행한다. 하나의 Session에서 여러 Run을 생성할 수 있다.

```
Session (설정 컨테이너)
├── Run 1 (1차 실행) → collecting → analyzing → pending_review → producing → published
├── Run 2 (2차 실행) → collecting → analyzing → pending_review → rejected
└── Run 3 (예약 실행) → collecting → error
```

## Session 라이프사이클

### 상태

| 상태 | 의미 |
|------|------|
| `draft` | 초기 생성 상태. 설정을 자유롭게 변경할 수 있다 |
| `active` | 활성화됨. 스케줄 실행이 가능하고, 설정 변경도 가능하다 |
| `archived` | 보관됨 (현재 코드에서 직접 전환하는 로직은 없음) |

### 전환 규칙

- **생성 시**: 상태 미지정이면 `draft`로 시작
- **활성화**: `draft` → `active` (draft에서만 활성화 가능)
- **비활성화**: `active` → `draft` (active에서만 비활성화 가능)
- **설정 변경**: `draft` 또는 `active` 상태에서만 가능. 그 외 상태에서 설정 변경을 시도하면 거부된다

### Session이 담는 설정

| 설정 | 용도 |
|------|------|
| **Sources** | RSS, HN, Reddit, HTTP, Google Trends, Social, Research 등 수집 소스 |
| **Schedule** | cron 표현식 또는 `@once` (1회 즉시 실행) |
| **Model** | LLM 프로바이더/모델 (예: `anthropic/claude-sonnet-4-20250514`) |
| **Workflows** | 콘텐츠 제작에 사용할 워크플로우 목록 |
| **Context** | 전체에 적용되는 프롬프트, 언어, 리서치 깊이 설정 |

### 삭제

Session을 삭제하면 해당 Session의 **모든 Run과 그 하위 데이터가 먼저 제거**된다 (cascade).

---

## Run 라이프사이클

### 상태 흐름

```
                          ┌─────────┐
                          │  draft  │ ← 생성 시 초기 상태
                          └────┬────┘
                               │ collect 시작
                          ┌────▼────────┐
                     ┌────│ collecting  │────┐
                     │    └────┬────────┘    │ 모든 소스 실패
                     │         │             │
                  취소(→draft) │         ┌───▼───┐
                     │    ┌────▼─────┐   │ error │
                     │    │ analyzing│   └───────┘
                     │    └────┬─────┘
                     │         │
                     │    ┌────▼───────────┐
                     └────│ pending_review │
                          └──┬─────────┬──┘
                    승인     │         │  거부
                   ┌─────────▼┐   ┌────▼────┐
                   │ approved │   │rejected │
                   └─────┬────┘   └─────────┘
                         │ produce 시작
                   ┌─────▼─────┐
                   │ producing │
                   └─────┬─────┘
                         │
                  ┌──────▼─────┐
                  │ published  │ (또는 error)
                  └────────────┘
```

### 상태별 의미

| 상태 | 의미 |
|------|------|
| `draft` | 생성 직후. 소스/워크플로우/컨텍스트 설정 가능. 취소 시에도 이 상태로 복귀 |
| `collecting` | 소스에서 콘텐츠를 수집 중 |
| `analyzing` | 수집된 콘텐츠를 LLM이 분석 중 |
| `pending_review` | 분석 완료, 사용자 리뷰 대기 |
| `approved` | 사용자가 분석 결과를 승인 |
| `rejected` | 사용자가 분석 결과를 거부 |
| `producing` | 승인된 앵글로 워크플로우 실행 중 |
| `published` | 제작된 콘텐츠가 외부 채널에 발행 완료 |
| `error` | 수집 또는 제작 중 오류 발생 |

### 전환 규칙

- **생성**: Session에 속한 Run을 생성. 기본 트리거 타입은 `manual`
- **수집 시작**: `draft` 상태에서만 가능. `draft`가 아니면 거부
- **취소**: `collecting` 또는 `analyzing` 상태에서만 가능. `draft`로 복귀
- **승인/거부**: 리뷰 시 `reviewed_at` 타임스탬프가 기록됨
- **발행 전환**: 모든 워크플로우 실행이 터미널 상태(published/rejected/failed)이면 자동으로 `published`로 전환

### Run 설정 오버라이드

Run은 생성 시 자체 설정을 가질 수 있다. 실행 시 **Run 설정이 있으면 우선 사용**하고, 없으면 Session 설정을 따른다:
- Sources: Run에 소스가 있으면 Run 소스 사용, 없으면 Session 소스
- Context: Run에 컨텍스트가 있으면 Run 컨텍스트 사용, 없으면 Session 컨텍스트

### 삭제

Run 삭제 시 **하위 데이터가 cascade로 제거**된다:
1. 발행된 콘텐츠 (실패 시 삭제 중단 — orphan 방지)
2. 워크플로우 실행 결과
3. 소스 fetch 기록
4. LLM 분석 결과

---

## 참고: 레거시 경로

현재 코드에는 **Pipeline/ContentSession** (V1)과 **Session/Run** (V2) 두 경로가 공존한다:
- V1: `Pipeline` → `ContentSession` → fetches/analysis/workflow results
- V2: `Session` → `Run` → fetches/analysis/workflow runs

V2가 새 경로이며, ContentCollector에 V1 메서드와 V2 메서드(`CollectAndAnalyzeV2`, `ProduceWorkflowsV2` 등)가 함께 존재한다. 향후 V1은 제거될 예정이다.
