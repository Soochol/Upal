# Pipeline Session Setup Redesign

## Problem

현재 파이프라인 설정(소스, 스케줄, 브리프, 모델, 워크플로우)이 파이프라인 레벨에 존재하여 모든 세션이 동일한 설정을 공유한다. 스케줄에 의해 반복 실행되는 세션마다 inbox에서 워크플로우를 매번 수동 선택해야 하는 UX 문제가 있다.

## Solution

설정을 세션 레벨로 이동시키고, 워크플로우를 세션에서 미리 지정하여 inbox에서는 승인/거절만 하면 되도록 단순화한다.

## Design

### Concept Changes

| | Before | After |
|---|---|---|
| **Pipeline** | 소스/스케줄/브리프/모델/워크플로우 보유 | 순수 그룹핑 (이름, 설명만) |
| **Session** | 한 번의 수집+분석 실행 결과 | 설정(소스/스케줄/브리프/모델/워크플로우) + 반복 실행 단위 |
| **Main area** | 4단계 스테이지 뷰 | 세션 설정 뷰 (섹션 분리형) |
| **Right sidebar** | 파이프라인 공통 설정 | 제거 (AI 어시스턴트 → 하단 플로팅) |
| **Inbox** | 소스+분석+angle선택+워크플로우매칭+승인 | 소스+분석(요약/인사이트/점수)+승인/거절 |

### Layout

Pipeline List | Session List | Session Setup View (main)

Session Setup View is a single scrollable page with sections:
1. **Sources & Schedule** — 소스 목록 + 스케줄 선택
2. **Editorial Brief** — 주제, 타겟, 톤, 키워드, 언어
3. **Analysis Model** — 모델 선택
4. **Workflows** — 워크플로우 선택/생성 + 채널 설정
5. **Start Session** button (bottom, sticky) + header Start button

AI Assistant — floating bottom widget (like workflow canvas chat)

### User Flow

```
1. Select/create pipeline
2. + New Session → create session
3. Configure session (sources, brief, model, workflows, schedule)
4. Start Session → background collect+analyze
5. Results appear in Review Inbox as pending_review
6. (On schedule) auto-repeat → results keep flowing to inbox
7. Inbox: review analysis → Approve → auto-execute pre-configured workflows → Produce → Publish
```

### Data Model Changes

**ContentSession** — add fields:
- `sources: []PipelineSource`
- `schedule: string` (cron)
- `model: string`
- `workflows: []PipelineWorkflow`
- `context: *PipelineContext` (editorial brief)

**Pipeline** — simplify:
- Keep: `id`, `name`, `description`, `thumbnail_svg`, `created_at`, `updated_at`
- Remove from active use: `sources`, `schedule`, `model`, `workflows`, `context` (migrate to sessions)

### Frontend Changes

1. **`Pipelines.tsx`** — 3-panel → 2-panel layout (pipeline list + main area). Remove right sidebar.
2. **New `SessionSetupView.tsx`** — main area component with section-based layout for session settings
3. **`PipelineSettingsPanel.tsx`** — remove (settings moved to SessionSetupView)
4. **`SessionListPanel.tsx`** — keep, update to show session config status
5. **`PipelineNew.tsx`** — simplify to name + description only
6. **`AnalyzeStage.tsx`** — simplify inbox view: remove angle/workflow selection UI, keep summary/insights/score + approve/reject
7. **AI Assistant** — extract to floating bottom widget component
8. **Reuse**: `AddSourceModal`, `EditorialBriefForm`, `WorkflowPicker`, `ModelSelector`

### Backend Changes

1. **Domain types** (`internal/upal/content.go`) — add settings fields to `ContentSession`
2. **API** (`internal/api/content.go`) — session settings CRUD endpoints
3. **ContentSessionService** — manage session settings
4. **ContentCollector** — read settings from session instead of pipeline
5. **Scheduler** — schedule per session instead of per pipeline
6. **Migration** — move existing pipeline settings to sessions

### Inbox Simplification

Before: AnalyzeStage shows source data + AI analysis + angle checkboxes + workflow matching + approve
After: AnalyzeStage shows source data + AI analysis (summary, insights, score) + Approve/Reject

On Approve → system automatically runs session's pre-configured workflows (no manual selection needed).
