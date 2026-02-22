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
go test ./internal/services/... -v -race -run TestRunnerName

# Frontend
cd web && npm run lint        # ESLint
cd web && npm run build       # tsc -b && vite build
```

## Architecture

### Backend (Go 1.23, Chi router, Google ADK)

The backend follows a layered clean architecture with dependency injection throughout.

**`cmd/upal/main.go`** — Entry point. Wires all dependencies bottom-up: Config → LLMs → Tools → Repositories → Services → NodeRegistry → Skills → Generator → API Server.

**`internal/upal/`** — Core domain types (no external dependencies):
- `workflow.go` — `WorkflowDefinition`, `NodeDefinition`, `EdgeDefinition`, `NodeType` (input/agent/output/tool/asset)
- `pipeline.go` — `Pipeline`, `Stage`, `PipelineRun`, `StageConfig` (multi-stage orchestration types)
- `types.go` — `GenerateID()` utility
- `events.go` — event type definitions
- `ports/` — port interfaces: `WorkflowExecutor`, `RunHistoryPort`, `SchedulerPort`, `PipelineRunner`, `RetryExecutor`

**`internal/dag/`** — DAG construction and traversal (no LLM dependencies):
- `dag.go` — Builds DAG from workflow definition, topological sort, cycle detection, parent/child relationships

**`internal/agents/`** — Node builder implementations (ADK-based):
- `registry.go` — `NodeRegistry` maps node types to `NodeBuilder` implementations
- `input_builder.go` / `output_builder.go` / `llm_builder.go` / `tool_builder.go` / `asset_builder.go` — per-type ADK agent builders
- `builders.go` — shared template resolution (`{{key}}` placeholders), `namedLLM` wrapper
- `dag.go` — DAG-aware execution logic
- `context.go` — execution context helpers
- `eval.go` — result evaluation

**`internal/model/`** — LLM provider implementations (satisfy ADK `model.LLM` interface):
- `anthropic.go` / `openai.go` — provider implementations
- `gemini_image.go` / `zimage.go` — image generation models
- `claudecode.go` — Claude Code integration

**`internal/tools/`** — Built-in tool implementations:
- `registry.go` — `Registry` maps tool names to `Tool` instances
- `get_webpage.go` / `http_request.go` / `rss_feed.go` / `python_exec.go` / `content_store.go` / `publish.go`

**`internal/services/`** — Business logic layer:
- `workflow.go` — `WorkflowService` orchestrates workflow execution via ADK runner
- `pipeline_runner.go` — `PipelineRunner` executes pipeline stages sequentially via pluggable `StageExecutor`
- `pipeline_service.go` — pipeline CRUD and run management
- `runhistory.go` / `runmanager.go` — run lifecycle and history
- `execution.go` — `ExecutionRegistry` for pause/resume handle tracking
- `concurrency.go` — `ConcurrencyLimiter` (semaphore-based)
- `retry.go` — `RetryExecutor` with backoff
- `connection.go` — connection management with encrypted secrets
- `stage_*.go` — stage executor implementations (workflow, approval, notification, transform, collect, passthrough)
- `scheduler/` — cron-based scheduler with dispatch and sync
- `run/publisher.go` — SSE event publisher for background run streaming

**`internal/repository/`** — Data access layer (memory + optional PostgreSQL):
- Per-domain repositories: `WorkflowRepository`, `RunRepository`, `ScheduleRepository`, `TriggerRepository`, `PipelineRepository`, `ConnectionRepository`
- `memory/store.go` — generic thread-safe in-memory KV store used by memory adapters

**`internal/db/`** — PostgreSQL schema and query helpers (optional, graceful fallback to in-memory)

**`internal/api/`** — HTTP layer (Chi router):
- `server.go` — `Server` struct with all service dependencies, `Handler()` builds Chi router
- `run.go` / `workflow.go` / `pipelines.go` / `runs.go` / `triggers.go` / `generate.go` / `upload.go` / `models.go` / `connections.go` — handlers
- `a2a.go` — A2A protocol (agent card + JSON-RPC)
- `configure.go` — LLM-guided node configuration endpoint

**`internal/generate/`** — LLM-based generation:
- `generate.go` — `Generator` converts natural language → `WorkflowDefinition` JSON
- `pipeline.go` — generates `Pipeline` definitions
- `backfill.go` — backfills descriptions for existing nodes
- `thumbnail.go` — generates workflow thumbnails

**`internal/skills/`** — Embedded AI skill & prompt registry:
- `skills.go` — `Registry` loads embedded markdown files, resolves `{{include name}}` references
- `nodes/*.md` — per-node-type skill docs (agent, input, output, tool, asset)
- `stages/*.md` — pipeline stage docs (approval, collect, notification, schedule, transform, trigger)
- `prompts/*.md` — generation prompts (workflow-create, workflow-edit, pipeline-create, etc.)
- `_frameworks/*.md` — shared prompt framework fragments

**Supporting packages**: `internal/extract/` (PDF/image/office text extraction), `internal/storage/` (local file storage), `internal/notify/` (Slack/SMTP/Telegram), `internal/crypto/` (secret encryption), `internal/llmutil/` (response parsing), `internal/output/` (result formatting)

### Frontend (React 19, TypeScript, Vite — FSD Architecture)

Organized by Feature-Sliced Design layers: `app` → `pages` → `widgets` → `features` → `entities` → `shared`

**`web/src/app/`** — Router, providers, app-level config

**`web/src/pages/`** — Route-level components: `Editor`, `Pipelines`, `Runs`, `Connections`, `Landing`

**`web/src/widgets/`** — Composite UI regions:
- `workflow-canvas/` — React Flow canvas, `UpalNode`, `CanvasPromptBar`, `EmptyState`
- `right-panel/` — collapsible Properties/Logs/Data/Chat/Preview panel
- `bottom-console/` — execution event stream console
- `workflow-header/` — top bar with save/run controls
- `node-palette/` — drag-to-add node sidebar
- `pipeline-editor/` — pipeline stage editor
- `run-detail/` — run history detail with timeline

**`web/src/features/`** — Feature slices:
- `edit-node/` — per-type node editors (Agent, Asset, Input, Output, AI Chat)
- `execute-workflow/` — run execution, SSE event bus, reconnect
- `manage-canvas/` — auto-save, keyboard shortcuts, canvas actions
- `upload-asset/` — file upload hook

**`web/src/entities/`** — Domain models and API clients:
- `workflow/` — Zustand store, serializer, layout, API
- `run/` — Zustand store, API
- `pipeline/` — API
- `node/` — types, node type registry
- `ui/` — Zustand UI state store

**`web/src/shared/`** — Reusable utilities and components:
- `api/` — typed HTTP client, per-domain API modules
- `lib/` — event bus, prompt serialization, node configs, output formats, utility hooks
- `ui/` — Shadcn/ui components, `PromptEditor` (TipTap + mention), `ModelSelector`, `ContentViewer`

**React Flow**: `@xyflow/react` with custom `UpalNode` component. `proOptions={{ hideAttribution: true }}`.

**Styling**: Tailwind CSS v4 + oklch color tokens. Status colors: `text-success/warning/info/destructive`. Node-type colors: `--node-input/agent/output/tool/asset`.

### Data Flow

1. User builds workflow visually → serialized to `WorkflowDefinition` JSON
2. Save: POST `/api/workflows` → stored in memory + optional PostgreSQL
3. Run: POST `/api/workflows/{name}/run` with `{ "inputs": {...} }`
4. `WorkflowService` builds DAG via `internal/dag`, constructs ADK agents via `NodeRegistry`
5. ADK runner executes agents in topological order with parallel fan-out
6. Events published via `RunPublisher` → streamed to frontend via SSE (`GET /api/runs/{id}/events`)
7. Template references (`{{node_id}}`) in prompts resolve from ADK session state at runtime

### Model ID Format

All model references use `"provider/model"` format: `anthropic/claude-sonnet-4-20250514`, `gemini/gemini-2.0-flash`, `ollama/llama3.2`

## Configuration

`config.yaml` at project root defines server port, database URL, and provider API keys. Providers are registered at startup based on this config. Database is optional — omit the `database` section to use in-memory storage only.

## Testing Patterns

- Server tests use `httptest.NewRequest`/`NewRecorder` with a minimal `Server` struct
- Service/engine tests create workflow definitions and verify execution state + emitted events
- Node tests mock the ADK model interface
- Always use `-race` flag (concurrent DAG execution)

## API Endpoints

```
# Workflows
POST   /api/workflows                          Create workflow
GET    /api/workflows                          List workflows
POST   /api/workflows/suggest-name            Suggest workflow name
GET    /api/workflows/{name}                   Get workflow
PUT    /api/workflows/{name}                   Update workflow
DELETE /api/workflows/{name}                   Delete workflow
POST   /api/workflows/{name}/run               Execute workflow (SSE stream)
POST   /api/workflows/{name}/thumbnail         Generate thumbnail
GET    /api/workflows/{name}/runs              List runs for workflow
GET    /api/workflows/{name}/triggers          List triggers for workflow

# Runs
GET    /api/runs                               List all runs
GET    /api/runs/{id}                          Get run
GET    /api/runs/{id}/events                   Stream run events (SSE)
POST   /api/runs/{id}/nodes/{nodeId}/resume    Resume paused node

# Triggers & Webhooks
POST   /api/triggers                           Create trigger
DELETE /api/triggers/{id}                      Delete trigger
POST   /api/hooks/{id}                         Webhook handler

# Pipelines
POST   /api/pipelines                          Create pipeline
GET    /api/pipelines                          List pipelines
GET    /api/pipelines/{id}                     Get pipeline
PUT    /api/pipelines/{id}                     Update pipeline
DELETE /api/pipelines/{id}                     Delete pipeline
POST   /api/pipelines/{id}/start               Start pipeline run
GET    /api/pipelines/{id}/runs                List pipeline runs
POST   /api/pipelines/{id}/runs/{runId}/approve  Approve pipeline run
POST   /api/pipelines/{id}/runs/{runId}/reject   Reject pipeline run
GET    /api/pipelines/{id}/triggers            List pipeline triggers
POST   /api/pipelines/{id}/thumbnail           Generate thumbnail

# Generation
POST   /api/generate                           Generate workflow from description
POST   /api/generate-pipeline                  Generate pipeline from description
POST   /api/generate/backfill                  Backfill node descriptions
POST   /api/nodes/configure                    LLM-guided node configuration

# Files
POST   /api/upload                             Upload file
GET    /api/files                              List files
GET    /api/files/{id}/serve                   Serve file
DELETE /api/files/{id}                         Delete file

# Discovery
GET    /api/models                             List available models
GET    /api/tools                              List available tools

# Connections (optional)
POST   /api/connections                        Create connection
GET    /api/connections                        List connections
GET    /api/connections/{id}                   Get connection
PUT    /api/connections/{id}                   Update connection
DELETE /api/connections/{id}                   Delete connection
```

## Node Types

| Type | Purpose | Key Config Fields |
|------|---------|-------------------|
| `input` | Collect user inputs | `value` |
| `agent` | LLM call with optional tool loop | `model`, `system_prompt`, `prompt`, `tools` |
| `tool` | Direct tool execution | `tool`, `input` |
| `output` | Aggregate results | — |
| `asset` | Inject uploaded file content into session | `asset_id` |

## Solution Approach

When suggesting fixes or changes, always consider both perspectives:
- **Pragmatic angle** — quick workaround that solves the immediate problem
- **Structural angle** — architecturally correct approach from a big-picture perspective

Present one or more concrete options covering both angles (they may overlap or there may be intermediate options). For each, briefly note the trade-off: effort, maintainability, scope of impact, technical debt risk. Let the user decide which path to take.
