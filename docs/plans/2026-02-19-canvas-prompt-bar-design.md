# Canvas Prompt Bar — "Edit these steps" Feature

**Date**: 2026-02-19
**Status**: Approved

## Overview

Add a Google Opal-style floating input bar at the bottom of the editor canvas that allows users to generate or edit workflows using natural language. The bar is always visible and context-aware: when nodes exist, it sends the current workflow as context (edit mode); when the canvas is empty, it creates a new workflow (create mode).

## Design

### Frontend: `CanvasPromptBar` Component

- **Position**: Absolute, bottom-center of Canvas, z-indexed above ReactFlow
- **Layout**: Single-line text input + send button, with disclaimer text below
- **States**: idle, loading (with spinner + disabled input)
- **Behavior**:
  - Enter key or button click submits
  - When nodes exist on canvas → sends `existing_workflow` alongside description
  - When canvas is empty → sends description only (create mode)

### Backend: `/api/generate` Extension

Extend the existing `GenerateRequest` with an optional `existing_workflow` field:

```go
type GenerateRequest struct {
    Description      string                         `json:"description"`
    Model            string                         `json:"model"`
    ExistingWorkflow *engine.WorkflowDefinition     `json:"existing_workflow,omitempty"`
}
```

When `ExistingWorkflow` is present, the generator switches to an "edit mode" system prompt that includes the current workflow JSON and instructs the LLM to modify it according to the user's description.

### Files Changed

| File | Change |
|------|--------|
| `web/src/components/editor/CanvasPromptBar.tsx` | New component |
| `web/src/components/editor/Canvas.tsx` | Add CanvasPromptBar |
| `web/src/pages/Editor.tsx` | Extend handleGenerate for edit mode |
| `web/src/lib/api.ts` | Add existingWorkflow param to generateWorkflow |
| `internal/api/generate.go` | Add ExistingWorkflow field |
| `internal/generate/generate.go` | Add edit mode system prompt + Generate method branching |
