# Quick Start Templates — Design Doc

**Date**: 2026-02-23
**Status**: Approved

## Goal

Transform the 3 placeholder template cards on `/workflows` into 6 fully functional quick-start templates that load real workflow definitions in the editor with a read-only view, run capability, and a Remix button to create editable copies.

## User Flow

1. User clicks a template card on Landing page
2. Template `WorkflowDefinition` is loaded into the workflow store with `isTemplate: true`
3. Editor opens in **read-only mode** — canvas shows nodes/edges but all editing is disabled
4. **Top banner** displays: "Template: [name]" + **Remix** button + **Run** button
5. **Run**: executes the template as-is (saves temporarily, runs via existing flow)
6. **Remix**: copies the template data to a new `Untitled-xxxx` workflow, sets `isTemplate: false`, switches to full edit mode

## Architecture

### Template Data (`web/src/shared/lib/templates.ts`)

New file exporting an array of template definitions:

```typescript
type TemplateDefinition = {
  id: string                        // e.g. "basic-rag-agent"
  title: string                     // Display name
  description: string               // One-liner
  icon: LucideIcon                  // Card icon
  color: string                     // Tailwind bg/text classes
  tags: string[]                    // e.g. ["RAG", "Web"]
  difficulty: 'Beginner' | 'Intermediate'
  workflow: WorkflowDefinition      // Real nodes + edges
}
```

### 6 Templates

| # | ID | Title | Nodes | Difficulty |
|---|---|---|---|---|
| 1 | `basic-rag-agent` | Basic RAG Agent | input → agent(get_webpage tool) → output | Beginner |
| 2 | `content-summarizer` | Content Summarizer | input → agent(summarize prompt) → output | Beginner |
| 3 | `sentiment-analyzer` | Sentiment Analyzer | input → agent(JSON extract) → output | Beginner |
| 4 | `data-pipeline` | Data Pipeline | input → agent(classify prompt) → output | Intermediate |
| 5 | `web-research-agent` | Web Research Agent | input → agent(search tools) → agent(synthesize) → output | Intermediate |
| 6 | `multi-step-writer` | Multi-Step Writer | input → agent(outline) → agent(write) → agent(edit) → output | Intermediate |

Each agent node has: `model`, `system_prompt`, `prompt`, `tools[]`, `description` fully populated.

### Store Changes (`entities/workflow/model/store.ts`)

Add to `WorkflowState`:
- `isTemplate: boolean` (default: `false`)
- `setIsTemplate: (v: boolean) => void`

### Landing Page (`pages/Landing.tsx`)

- Replace 3 hardcoded template buttons with 6 template cards
- Each card shows `WorkflowMiniGraph` preview (reusing existing component)
- Cards display: title, description, node count badge, difficulty badge
- Click handler `openTemplate(template)`:
  1. `deserializeWorkflow(template.workflow)` → get nodes/edges
  2. Set workflow store: nodes, edges, workflowName = template title
  3. Set `isTemplate: true`, `originalName: ''`
  4. Navigate to `/editor`

### Editor Read-Only (`pages/Editor.tsx`)

When `isTemplate === true`:
- Skip `useAutoSave()` (or disable saving)
- Pass `readOnly={true}` to `<Canvas>`
- Pass `isTemplate` + `onRemix` to `<WorkflowHeader>`
- `handleRemix()`: sets `isTemplate: false`, generates new name `Untitled-xxxx`, enables full editing

### Canvas Read-Only (`widgets/workflow-canvas/ui/Canvas.tsx`)

Add `readOnly?: boolean` prop. When true:
- `nodesDraggable={false}`
- `nodesConnectable={false}`
- `elementsSelectable={false}`
- `deleteKeyCode={null}`
- Hide `<NodePalette>`, `<CanvasPromptBar>`, `<EmptyState>`
- Pan/zoom still enabled for exploration

### WorkflowHeader (`widgets/workflow-header/ui/WorkflowHeader.tsx`)

Add `isTemplate?: boolean` + `onRemix?: () => void` + `templateName?: string` props.

When `isTemplate`:
- Replace name Input with static "Template: [name]" text + template badge
- Replace SaveStatus with **Remix** button (primary CTA)
- Keep Run button

## Files Changed

| File | Action |
|------|--------|
| `web/src/shared/lib/templates.ts` | NEW — 6 template definitions |
| `web/src/entities/workflow/model/store.ts` | MODIFY — add `isTemplate` state |
| `web/src/pages/Landing.tsx` | MODIFY — 6 cards with MiniGraph preview, `openTemplate` handler |
| `web/src/pages/Editor.tsx` | MODIFY — read-only mode when `isTemplate`, remix handler |
| `web/src/widgets/workflow-header/ui/WorkflowHeader.tsx` | MODIFY — template banner + Remix button |
| `web/src/widgets/workflow-canvas/ui/Canvas.tsx` | MODIFY — `readOnly` prop |

## Non-Goals

- No backend template API (templates are frontend-only presets)
- No template categories/filtering UI (6 templates don't need it)
- No template editing/creation by users
