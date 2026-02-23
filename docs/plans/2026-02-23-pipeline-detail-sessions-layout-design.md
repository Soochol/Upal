# Pipeline Detail — Sessions-First Layout Redesign

**Date:** 2026-02-23
**Status:** Approved

## Problem

The current `PipelineDetail` page buries sessions in a narrow left column. Users must click "Collect Now" (which creates a new session and navigates away) or a tiny "View" link to interact with sessions. There is no way to see multiple sessions at a glance, nor to approve/reject them without leaving the page.

## Data Model Clarification

| Setting | Scope | Notes |
|---------|-------|-------|
| Data Sources + Schedule | Pipeline-level | Same for all sessions |
| Editorial Brief & Context | Pipeline-level | Same for all sessions |
| Workflows | Session-level | Each session tracks which workflows ran and their results |

## Design

### Overall Page Structure

```
Header: ← Pipelines / {name}                [▶ Collect Now]
Meta strip: {n} sources · {cron} · Last: {date}
───────────────────────────────────────────────────────────
Sessions ({count})         [All] [Pending ●n] [Published] …
  <session cards>
───────────────────────────────────────────────────────────
▶ Pipeline Settings  (accordion, collapsed by default)
   ├ Data Sources & Schedule
   └ Editorial Brief & Context
```

Sessions occupy the full page width as the primary content. Pipeline settings are demoted to a collapsible accordion below — they rarely change after initial setup.

### Session Cards (State-Aware)

Cards render different content depending on `session.status`.

**`pending_review`** — full action card:
```
┌──────────────────────────────────────────────────────────┐
│ ● Session 5   ⏳ Pending Review   Score: 87              │
│ Feb 23, 14:30 · 12 articles · manual                    │
│ ──────────────────────────────────────────────────────── │
│ "AI model releases dominate this week's tech cycle,      │
│  with 3 distinct angles across enterprise adoption..."   │
│ ──────────────────────────────────────────────────────── │
│ Workflows:  [ Blog Post ]  [ Newsletter ]  [ Shorts ]    │
│ [✓ Approve & Run All]                    [✗ Reject]      │
└──────────────────────────────────────────────────────────┘
```

**`collecting`** — progress state:
```
┌──────────────────────────────────────────────────────────┐
│ ⚙ Session 6   ⟳ Collecting…                             │
│ Feb 23, 16:00 · triggered manually                       │
│ ░░░░░░░░░░░░░░░░░░░░░░░░░░░ (animated progress bar)      │
└──────────────────────────────────────────────────────────┘
```

**`producing`** — workflow execution in progress:
```
┌──────────────────────────────────────────────────────────┐
│ ⚙ Session 5   ⟳ Producing…   Score: 87                  │
│ Feb 23, 14:35                                            │
│ Blog Post ⟳  ·  Newsletter ⟳  ·  Shorts ⟳              │
│                                                [View →]  │
└──────────────────────────────────────────────────────────┘
```

**`published`** — completed, compact:
```
┌──────────────────────────────────────────────────────────┐
│ ✓ Session 4   ✅ Published   Score: 91                   │
│ Feb 22, 09:00 · 8 articles · schedule                    │
│ Blog Post ✓  ·  Newsletter ✓  ·  Shorts ✗               │
│                                                [View →]  │
└──────────────────────────────────────────────────────────┘
```

**`rejected`** — muted:
```
┌──────────────────────────────────────────────────────────┐
│ ✗ Session 3   🚫 Rejected                  (muted)       │
│ Feb 21, 14:00 · "Low quality sources"        [View →]    │
└──────────────────────────────────────────────────────────┘
```

### Filter Tabs

```
[All (5)]  [Pending ●2]  [Producing]  [Published]  [Rejected]
```

- "Pending" tab shows a live badge count driven by `pipeline.pending_session_count`
- Filter is client-side (sessions already fetched)

### Pipeline Settings Accordion

- Collapsed by default — users rarely edit sources or brief after setup
- Expand to reveal existing `SourceConfigTab` and `EditorialBriefForm` components unchanged
- `WorkflowTemplatesTab` is removed from the accordion since workflow info moves into session cards

## Key UX Improvements

| Aspect | Before | After |
|--------|--------|-------|
| Sessions visibility | Narrow left column | Full-width, primary content |
| Approve/Reject | Navigate to /inbox/{id} first | Inline on card |
| Workflow info | Not shown on pipeline page | Per-session execution status |
| Settings access | Always-visible right column | Collapsible, on-demand |
| Pending filter | Badge → navigate to /inbox | Filter tab inline |

## Components to Change

- `web/src/pages/pipelines/PipelineDetail.tsx` — full layout restructure
  - New `SessionCard` sub-component (state-aware rendering)
  - New `SessionFilterTabs` sub-component
  - New `PipelineSettingsAccordion` wrapper around existing config tabs
  - `SessionHistoryTab` replaced by the new card list
  - `WorkflowTemplatesTab` removed (workflow info moves into `SessionCard`)
- Approve/Reject mutations added directly in `PipelineDetail` (calls existing `approveSession` / `rejectSession` API)

## Out of Scope

- Session detail page (`/inbox/{id}`) — no changes
- Content Inbox page (`/inbox`) — no changes
- Backend API — no changes needed
