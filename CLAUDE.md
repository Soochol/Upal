# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Upal is a visual AI workflow platform — a Go backend with embedded React frontend that lets users build and execute DAG-based workflows connecting LLM providers. Multi-model support (Anthropic, Gemini, OpenAI, Ollama).

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

### Backend (Go 1.24, Chi router, Google ADK)

Layered clean architecture with dependency injection. Entry point: `cmd/upal/main.go` wires Config → LLMs → Tools → Repositories → Services → NodeRegistry → Skills → Generator → API Server.

| Package | Purpose |
|---------|---------|
| `internal/upal/` | Core domain types, port interfaces (`ports/`). No external deps |
| `internal/dag/` | DAG construction, topological sort, cycle detection |
| `internal/agents/` | ADK-based node builders (`NodeRegistry` → per-type builders). Template resolution (`{{key}}`) |
| `internal/model/` | LLM provider implementations (Anthropic, OpenAI, Gemini, TTS, image gen, Claude Code) |
| `internal/tools/` | Built-in tool registry (webpage, HTTP, RSS, Python, content store, publish, video) |
| `internal/services/` | Business logic: workflow execution, pipeline runner, stage executors, content sessions, scheduler, run history, connections |
| `internal/repository/` | Data access layer (memory + optional PostgreSQL). Per-domain repositories |
| `internal/db/` | PostgreSQL schema and queries (optional, graceful fallback to in-memory) |
| `internal/api/` | Chi HTTP handlers. See `server.go` `Handler()` for all routes |
| `internal/generate/` | LLM-based generation (workflow, pipeline, backfill, thumbnail) |
| `internal/skills/` | Embedded prompt registry with `{{include}}` resolution. Subdirs: `nodes/`, `stages/`, `prompts/`, `tools/`, `_frameworks/` |
| `internal/config/` | Configuration loading |
| `internal/extract/` | PDF/image/office text extraction |
| `internal/storage/` | Local file storage |
| `internal/notify/` | Slack/SMTP/Telegram notifications |
| `internal/crypto/` | Secret encryption |
| `internal/llmutil/` | LLM response parsing |
| `internal/output/` | Result formatting |

### Frontend (React 19, TypeScript, Vite — FSD Architecture)

Organized by Feature-Sliced Design: `app` → `pages` → `widgets` → `features` → `entities` → `shared`

| Layer | Contents |
|-------|----------|
| `app/` | Router, providers |
| `pages/` | Editor, Pipelines (+ session detail, stages), Runs, Connections, Landing, Inbox, Published |
| `widgets/` | workflow-canvas, right-panel, bottom-console, workflow-header, node-palette, pipeline-editor, run-detail |
| `features/` | edit-node, execute-workflow, manage-canvas, upload-asset, generate-workflow, generate-pipeline, configure-pipeline-sources, define-editorial-brief |
| `entities/` | workflow, run, pipeline, node, ui, content-session, published-content, surge |
| `shared/` | Typed API client, Shadcn/ui components, TipTap PromptEditor, event bus, utility hooks |

**React Flow**: `@xyflow/react` with custom `UpalNode`. `proOptions={{ hideAttribution: true }}`.

**Styling**: Tailwind CSS v4 + oklch color tokens. Status colors: `text-success/warning/info/destructive`. Node-type colors: `--node-input/agent/output/tool/asset`.

### Key Conventions

- **Model ID format**: `"provider/model"` — e.g. `anthropic/claude-sonnet-4-20250514`, `gemini/gemini-2.0-flash`, `ollama/llama3.2`
- **Template references**: `{{node_id}}` in prompts resolve from ADK session state at runtime
- **Data flow**: Visual workflow → `WorkflowDefinition` JSON → DAG → ADK agents (topological order, parallel fan-out) → SSE events to frontend
- **Auto-save**: 모든 자동 저장은 `shared/hooks/useAutoSave` 훅을 사용한다. 새 auto-save 추가 시 반드시 이 훅을 사용하고, `onError` 콜백을 포함해야 한다. `markClean()`은 React 상태 업데이트가 반영된 다음 렌더에서 호출해야 한다 (같은 이펙트에서 setState 직후 호출 금지).
- **Session UI constants**: 세션 상태 관련 상수(`SESSION_STATUS_DOT`, `SESSION_FILTER_TABS`, `SessionFilter`, `matchesSessionFilter`)는 `entities/content-session/constants.ts`에 정의되어 있다. 세션 목록을 표시하는 모든 컴포넌트(Inbox, Pipeline sessions 등)에서 이 공유 상수를 import해서 사용해야 한다. 로컬 복사본을 만들지 않는다. 새 세션 상태를 추가하면 `SESSION_STATUS_DOT`과 `matchesSessionFilter` 양쪽 모두 업데이트해야 한다.

## Configuration

`config.yaml` at project root defines server port, database URL, and provider API keys. Database is optional — omit the `database` section to use in-memory storage only.

## Testing Patterns

- Server tests use `httptest.NewRequest`/`NewRecorder` with a minimal `Server` struct
- Service/engine tests create workflow definitions and verify execution state + emitted events
- Node tests mock the ADK model interface
- Always use `-race` flag (concurrent DAG execution)

## Skills System (`internal/skills/`)

Skills are the **single source of truth** for all LLM behavior in Upal. Every LLM invocation (workflow generation, node configuration, pipeline creation, backfill, thumbnail) must be guided by the appropriate skill. Never hardcode LLM instructions in Go code — always use the skills registry.

### Registry Two-Tier Architecture

```go
type Registry struct {
    skills  map[string]string  // on-demand docs (nodes/, stages/, tools/)
    prompts map[string]string  // pre-loaded base instructions (prompts/)
}
// Get(name) → skill doc | ""     — on-demand, for tool calls during generation
// GetPrompt(name) → prompt | ""  — pre-loaded, for system prompt base
```

### Skill Categories & Naming

| Category | Location | Naming | Access | Purpose |
|----------|----------|--------|--------|---------|
| **Node skills** | `nodes/` | `{type}-node` (agent-node, input-node, ...) | `Get()` | Node configuration guidance for LLM |
| **Stage skills** | `stages/` | `stage-{type}` (stage-approval, stage-collect, ...) | `Get()` | Pipeline stage configuration guidance |
| **Tool skills** | `tools/` | `tool-{name}` (tool-web_search, tool-python_exec, ...) | `Get()` | Tool usage documentation for LLM |
| **Prompts** | `prompts/` | `{action}` (workflow-create, pipeline-edit, ...) | `GetPrompt()` | Base system prompts for generation tasks |
| **Frameworks** | `_frameworks/` | — | Internal only | Shared fragments resolved via `{{include name}}` |

### How Skills Are Consumed

1. **Workflow/Pipeline generation** (`internal/generate/`): Base prompt from `GetPrompt("workflow-create")`. During generation, LLM calls `get_skill(name)` tool → `executeSkillCalls()` resolves via `Get(name)`.
2. **Node configuration** (`internal/api/configure.go`): Appends `Get(nodeType + "-node")` to system prompt. e.g. `Get("agent-node")`.
3. **Backfill** (`internal/generate/backfill.go`): Uses `GetPrompt("node-describe")` for description generation.
4. **Thumbnail** (`internal/generate/thumbnail.go`): Uses `GetPrompt("thumbnail")`.

### Rules for Writing/Modifying Skills

- **`{{include name}}`** in skill files resolves to `_frameworks/{name}.md` content at load time
- **`{{node_id}}`** template syntax in user prompts — resolved at workflow runtime from ADK session state. Never use placeholder text like `[여기에 입력]`
- **System prompts** must follow `_frameworks/system-prompt.md` quality: rich expert persona with role, expertise (3-5 competencies), style, constraints. Never generic ("You are a helpful assistant")
- **User prompts** must follow `_frameworks/prompt-framework.md`: reference upstream data via `{{node_id}}`, one clear task per prompt
- New LLM-facing features **must** have a corresponding skill file. Add to the appropriate subdirectory and the LLM will discover it via `get_skill()` tool calls or direct `Get()`/`GetPrompt()` lookups

## Node Types

| Type | Purpose | Key Config Fields |
|------|---------|-------------------|
| `input` | Collect user inputs | `value` |
| `agent` | LLM call with optional tool loop | `model`, `system_prompt`, `prompt`, `tools` |
| `tool` | Direct tool execution | `tool`, `input` |
| `output` | Aggregate results | — |
| `asset` | Inject uploaded file content into session | `asset_id` |

## Communication Style

코드 작업을 설명할 때 코드 수준의 디테일(함수명, 변수, 구현 방식)이 아니라 **비즈니스 로직과 동작 변화**에 초점을 맞춘다.

- "X 함수에 Y 파라미터를 추가하고 Z 타입으로 변경합니다" ❌
- "파이프라인 발행 시 채널을 선택할 수 있게 됩니다" ✅

코드 diff는 사용자가 직접 확인하므로, **왜 하는지**(목적)와 **뭐가 달라지는지**(사용자·시스템 관점의 동작 변화)를 중심으로 설명한다.

설계 제안·리뷰에서도 동일 원칙을 적용한다. 코드 표현이나 변수명을 쓰지 않고 "실행 중이면 저장 안 함", "변경 여부 추적"처럼 **동작 중심**으로 설명한다.

## Solution Approach

When suggesting fixes or changes, always consider both perspectives:
- **Pragmatic angle** — quick workaround that solves the immediate problem
- **Structural angle** — architecturally correct approach from a big-picture perspective

Present one or more concrete options covering both angles (they may overlap or there may be intermediate options). For each, briefly note the trade-off: effort, maintainability, scope of impact, technical debt risk. Let the user decide which path to take.
