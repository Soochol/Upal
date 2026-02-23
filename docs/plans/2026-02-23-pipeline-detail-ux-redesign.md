# Pipeline Detail UX Redesign

**Date:** 2026-02-23
**Status:** Approved

## Problem

Accessing a session requires 3 navigation levels:
1. `/pipelines` — pipeline list
2. `/pipelines/:id` — pipeline detail (sessions list + settings accordion)
3. `/inbox/:sessionId` — session detail

This is too many clicks. The goal is to reduce to **2 levels max**.

## Design

### `/pipelines/:id` — Redesigned Layout

Split into two columns:

**Left/Main (flexible width):**
- Header row: "Sessions" title + search input + "Collect Now" button
- Filter tabs: All | Pending(N) | Producing | Published | Rejected
- Session table (inspired by Google AI Studio projects table):

| 세션 | 상태 | 점수 | 생성일 | 결과 | |
|------|------|------|--------|------|--|
| Session 3 | ⏳ Pending | ★8.2 | Feb 23 22:07 | [✓ Approve] [✗ Reject] | ⋯ |
| Session 2 | ✓ Published | ★7.8 | Feb 22 09:00 | linkedin · twitter | ⋯ |
| Session 1 | ✗ Rejected | — | Feb 21 09:00 | — | ⋯ |

- Pending rows: inline Approve & Reject buttons
- Published/Producing rows: workflow result badges
- Row click: navigate to `/pipelines/:id/sessions/:sessionId` (modal opens)

**Right Panel (fixed ~280px):**
- Pipeline Settings (previously hidden in accordion):
  - Sources & Schedule (with Add/Remove source, schedule picker)
  - Editorial Brief (keyword tags, tone, context)
- Always visible — no accordion toggle

### Session Detail Modal

**Trigger:** Clicking any session row
**URL:** `/pipelines/:id/sessions/:sessionId` (shareable link)
**Close:** navigates back to `/pipelines/:id`

**Layout:** Full-screen overlay
```
┌─ backdrop ─────────────────────────────────────────────────────────┐
│ ┌────────────────────────────────────────────────────────────────┐ │
│ │ Session 3 · Tech Digest              ⏳ Pending Review   [✕]  │ │
│ │ Feb 23, 22:07 · ★ 8.2                                        │ │
│ │ ─────────────────────────────────────────────────────────     │ │
│ │  [1. Collect ✓]──[2. Analyze ✓]──[3. Workflow ●]──[4. Pub]  │ │
│ │                                                               │ │
│ │  (현재 SessionDetail 4-stage 콘텐츠 그대로)                    │ │
│ │                                                               │ │
│ │  ────────────────────────────────────────────────────────     │ │
│ │              [✗ Reject]  [✓ Approve & Run Selected]          │ │
│ └────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

### Route Changes

| Before | After | Notes |
|--------|-------|-------|
| `/pipelines/:id` | `/pipelines/:id` | Redesigned (table + right panel) |
| `/inbox/:sessionId` | `/pipelines/:id/sessions/:sessionId` | Session detail as modal |
| `/inbox` | — | Removed from router + nav |

Legacy `/inbox/:sessionId` redirect: fetch session → get `pipeline_id` → redirect to `/pipelines/:pipelineId/sessions/:sessionId`.

## Implementation Scope

### Files to change
1. `web/src/app/router.tsx` — add `/pipelines/:id/sessions/:sessionId`, remove `/inbox` routes
2. `web/src/pages/pipelines/PipelineDetail.tsx` — full redesign
3. `web/src/pages/inbox/SessionDetail.tsx` — convert to modal component
4. `web/src/pages/inbox/index.tsx` — delete
5. `web/src/shared/ui/Header.tsx` — remove Inbox nav item

### New components (inside PipelineDetail or co-located)
- `SessionTable` — table rows with search/filter
- `SessionDetailModal` — full-screen overlay wrapping existing SessionDetail panels
- `PipelineSettingsPanel` — right column (sources, schedule, editorial brief extracted from accordion)

## Decisions
- **Q1 URL:** Yes — session popup updates URL to `/pipelines/:id/sessions/:sessionId`
- **Q2 /inbox:** Delete — sessions accessible only via pipeline
