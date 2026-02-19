# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Upal is a visual AI workflow platform — a Go backend with embedded React frontend that lets users build and execute DAG-based workflows connecting LLM providers. Think Google Opal-style visual workflow editor with multi-model support (Anthropic, Gemini, OpenAI, Ollama).

## Build & Development Commands

```bash
# Full build (frontend + backend)
make build                    # → bin/upal

# Development (run in separate terminals)
make dev-backend              # go run ./cmd/upal serve (port 8080)
make dev-frontend             # vite dev server (port 5173, proxies /api → 8080)

# Tests
make test                     # go test ./... -v -race
make test-frontend            # cd web && npx tsc -b (type-check only)

# Single Go test
go test ./internal/engine/... -v -race -run TestRunnerName

# Frontend
cd web && npm run lint        # ESLint
cd web && npm run build       # tsc -b && vite build
```

## Architecture

### Backend (Go 1.23, Chi router)

The backend follows a layered architecture with clear interfaces between layers:

**`cmd/upal/main.go`** — Entry point. Wires up all dependencies: EventBus → SessionManager → Providers → Tools → NodeExecutors → Server. Loads `config.yaml`.

**`internal/engine/`** — Core execution engine (no LLM dependencies):
- `dag.go` — Builds DAG from workflow definition, topological sort
- `runner.go` — Executes DAG with goroutine fan-out, WaitGroup fan-in, channel-based sync per node
- `session.go` — Session state management (thread-safe via RWMutex)
- `eventbus.go` — Pub/sub event bus for execution lifecycle events
- `types.go` — WorkflowDefinition, NodeDefinition, EdgeDefinition, Event types

**`internal/nodes/`** — Node type implementations (all satisfy `NodeExecutor` interface):
- `agent.go` — LLM call with optional agentic tool-use loop (max_turns)
- `input.go` / `output.go` / `tool.go` — Input collection, result aggregation, direct tool execution

**`internal/provider/`** — LLM provider abstraction:
- `provider.go` — `Provider` interface (ChatCompletion, ChatCompletionStream), shared types
- `registry.go` — Maps provider names to instances, resolves `"provider/model"` format model IDs
- `anthropic.go` / `gemini.go` — Provider implementations (OpenAI via config)

**`internal/api/`** — HTTP layer:
- `server.go` — Chi router setup, CORS, static file serving
- `workflow.go` — CRUD handlers for workflows (in-memory store + optional PostgreSQL)
- `run.go` — Workflow execution endpoint with SSE streaming
- `generate.go` — LLM-based workflow generation from natural language
- `upload.go` — File upload handler

**`internal/db/`** — PostgreSQL persistence (optional, graceful fallback to in-memory)

**`internal/generate/`** — Workflow generation: sends description to LLM with system prompt defining valid workflow JSON structure

### Frontend (React 19, TypeScript, Vite)

**State**: Zustand store in `stores/workflowStore.ts` — nodes, edges, workflow name, run state, node statuses, execution events

**Visual editor**: React Flow (`@xyflow/react`) with custom node component (`UpalNode.tsx`)

**Key components**:
- `editor/Canvas.tsx` — React Flow canvas with node types, minimap
- `editor/nodes/NodeEditor.tsx` — Right sidebar panel for editing selected node config
- `console/Console.tsx` — Bottom panel showing real-time execution events
- `dialogs/RunDialog.tsx` — Input collection modal before workflow execution
- `dialogs/GenerateDialog.tsx` — Natural language workflow generation modal

**API client**: `lib/api.ts` — REST calls + SSE stream parsing for run events

**Styling**: Tailwind CSS v4 + Shadcn/ui components in `components/ui/`

**Path alias**: `@/` → `./src/` (configured in tsconfig and vite)

### Data Flow

1. User builds workflow visually → serialized to `WorkflowDefinition` JSON
2. Save: POST `/api/workflows` → stored in memory + optional PostgreSQL
3. Run: POST `/api/workflows/{name}/run` with `{ "inputs": {...} }`
4. Backend builds DAG → topological sort → parallel goroutine execution
5. Each node executes via its `NodeExecutor`, state stored in `session.State[nodeID]`
6. Events published to EventBus → streamed to frontend via SSE
7. Template references (`{{node_id}}`) in prompts resolve from session state at runtime

### Model ID Format

All model references use `"provider/model"` format: `anthropic/claude-sonnet-4-20250514`, `gemini/gemini-2.0-flash`, `ollama/llama3.2`

## Configuration

`config.yaml` at project root defines server port, database URL, and provider API keys. Providers are registered at startup based on this config. Database is optional — omit the `database` section to use in-memory storage only.

## Testing Patterns

- Server tests use `httptest.NewRequest`/`NewRecorder` with a minimal `Server` struct
- Engine tests create workflow definitions and verify execution state + emitted events
- Node tests mock the provider interface
- Always use `-race` flag (concurrent DAG execution)

## API Endpoints

```
POST   /api/workflows              Create workflow
GET    /api/workflows              List workflows
GET    /api/workflows/{name}       Get workflow
PUT    /api/workflows/{name}       Update workflow
DELETE /api/workflows/{name}       Delete workflow
POST   /api/workflows/{name}/run   Execute workflow (SSE stream response)
POST   /api/generate               Generate workflow from description
POST   /api/upload                 Upload file
```

## Node Types

| Type | Purpose | Key Config Fields |
|------|---------|-------------------|
| `input` | Collect user inputs | `value` |
| `agent` | LLM call with optional tool loop | `model`, `system_prompt`, `prompt`, `tools`, `max_turns` |
| `tool` | Direct tool execution | `tool`, `input` |
| `output` | Aggregate results | — |

## Solution Approach

When suggesting fixes or changes, always consider both perspectives:
- **Pragmatic angle** — quick workaround that solves the immediate problem
- **Structural angle** — architecturally correct approach from a big-picture perspective

Present one or more concrete options covering both angles (they may overlap or there may be intermediate options). For each, briefly note the trade-off: effort, maintainability, scope of impact, technical debt risk. Let the user decide which path to take.
