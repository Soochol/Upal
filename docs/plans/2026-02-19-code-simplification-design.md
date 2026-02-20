# Code Simplification Design — Parallel Tracks

**Date:** 2026-02-19
**Branch:** feat/adk-adoption
**Scope:** Full-stack (Go backend + React frontend)
**Execution:** Parallel agents, immediate implementation

## Context

The `feat/adk-adoption` branch added 14,667 lines across 206 files. While functional, the codebase has accumulated:
- Cross-file code duplication (JSON parsing, LLM resolution, model grouping)
- Oversized functions/components (builders.go 650L, NodeEditor 299L, PromptEditor 366L)
- Monolithic store (workflowStore.ts: 8 concerns, 25 methods, 292 lines)

## Strategy: Parallel Tracks

Two independent agents work simultaneously on non-overlapping file sets.

### Track 1: Go Backend

| # | Task | Files | Effect |
|---|------|-------|--------|
| 1 | Extract `stripAndParseJSON()` utility | New `internal/api/jsonutil.go`; modify `configure.go`, `generate/generate.go` | 3x identical code → 1 |
| 2 | Extract `resolveLLMFromID()` | New `internal/model/resolve.go`; modify `builders.go`, `configure.go` | 3x repeat → 1 fn |
| 3 | Consolidate `generateAutoLayout` + `generateManualLayout` | `builders.go` | 140L → ~80L |
| 4 | Extract tool execution loop from `buildLLMAgent()` | `builders.go` | 172L fn → ~100L + helper |
| 5 | Extract SSE error helpers from `run.go` | `run.go` | 4x copy-paste → `sendErrorEvent()` |
| 6 | Split `configureNode()` into smaller functions | `configure.go` | 122L → 4-5 focused fns |

**Expected:** ~200 lines of duplication removed, largest function reduced 40-50%.

### Track 2: React Frontend

| # | Task | Files | Effect |
|---|------|-------|--------|
| 1 | Extract `groupModels()` utility | New `lib/utils.ts`; modify `NodeEditor.tsx`, `AIChatEditor.tsx` | 3x dup → 1 fn |
| 2 | Extract `<ModelSelector />` component | New `components/editor/ModelSelector.tsx`; modify `NodeEditor.tsx` | 3x copy → 1 component |
| 3 | Split `NodeEditor` by node type | New 5 sub-components; modify `NodeEditor.tsx` | 299L → ~60L hub |
| 4 | Extract `useUpstreamNodes()` hook | New `hooks/useUpstreamNodes.ts`; modify `PromptEditor.tsx`, `AIChatEditor.tsx` | 2x dup → 1 hook |
| 5 | Split `workflowStore` into slices | `stores/workflowStore.ts` → canvas, execution, ui stores | 292L → 4-5 focused stores |
| 6 | Extract PromptEditor TipTap concerns | New `editor/extensions/`, `lib/promptSerialization.ts` | 366L → ~150L + utils |

**Expected:** ~400 lines of duplication removed, max component size ≤150 lines.

## Boundary Management

- `api.ts` — frontend track only (SSE parsing is frontend responsibility)
- `run.go` — backend track only (SSE event format unchanged)
- **API contract immutable** — neither track changes endpoint specs or event structures
- Each track maintains independent build/test pass

## Verification

- Go: `make test` passes after each task
- React: `make test-frontend` (tsc) passes after each task
- Final: `make build` full build passes

## Not In Scope

- New features or behavior changes
- API endpoint modifications
- Database schema changes
- UI/UX changes
