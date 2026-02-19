# Go ADK Full Adoption Design

**Date:** 2026-02-19
**Approach:** Full replacement (Approach A) — remove all existing engine/nodes/provider code, replace with Google ADK Go SDK

## Decision

Adopt Google ADK Go (`google.golang.org/adk`) as the core runtime. All existing execution, session, provider, and A2A code is deleted and replaced with ADK-native equivalents. No wrapping, no adapters, no compatibility layers.

## Architecture

### Layer Mapping

| Current | Action | ADK Replacement |
|---------|--------|-----------------|
| `internal/engine/runner.go` | Delete | `runner.Runner` |
| `internal/engine/a2a_runner.go` | Delete | `runner.Runner` (unified) |
| `internal/engine/session.go` | Delete | `session.Service` (in-memory) |
| `internal/engine/eventbus.go` | Delete | `runner.Runner` callbacks |
| `internal/engine/types.go` | Delete | `internal/upal/types.go` (WorkflowDef only) |
| `internal/engine/dag.go` | Move | `internal/dag/dag.go` |
| `internal/nodes/agent.go` | Delete | `llmagent.New()` direct call |
| `internal/nodes/input.go` | Delete | `agent.New()` Custom Agent |
| `internal/nodes/output.go` | Delete | `agent.New()` Custom Agent |
| `internal/nodes/tool.go` | Delete | `agent.New()` Custom Agent |
| `internal/nodes/external.go` | Delete | `remoteagent.NewA2A()` direct call |
| `internal/provider/provider.go` | Delete | `model.LLM` interface |
| `internal/provider/anthropic.go` | Delete | `internal/model/anthropic.go` (model.LLM impl) |
| `internal/provider/gemini.go` | Delete | ADK built-in `model/gemini` |
| `internal/provider/openai.go` | Delete | `internal/model/openai.go` (model.LLM impl) |
| `internal/a2atypes/` | Delete | ADK `genai.Content` + `a2a-go` |
| `internal/a2a/` | Delete | ADK `remoteagent` |
| `internal/a2aclient/` | Delete | ADK `remoteagent` |

### Retained (modified)

| File | Change |
|------|--------|
| `internal/api/server.go` | Server struct simplified, ADK Runner |
| `internal/api/run.go` | ADK Runner invocation, ADK Event direct SSE |
| `internal/api/workflow.go` | No change |
| `cmd/upal/main.go` | Rewritten for ADK dependency injection |
| `config.yaml` | Structure retained |
| `internal/db/` | No change |
| `internal/storage/` | No change |
| `internal/generate/` | No change |

## Data Flow (Single Path)

```
POST /api/workflows/{name}/run
  → WorkflowDef → DAG build (dag.go)
  → DAGAgent.Run(InvocationContext)           [ADK Custom Agent]
    → TopologicalOrder iteration:
      ├─ InputAgent:   UserContent → session.State
      ├─ LLMAgent:     State template → llmagent.Run() → State
      ├─ ToolAgent:    State template → tool.Execute() → State
      ├─ RemoteAgent:  remoteagent.Run() → A2A protocol → State
      └─ OutputAgent:  State collection → final result
  → iter.Seq2[*session.Event, error]
  → JSON serialize → SSE stream
  → Frontend consumes ADK Event directly
```

All node results flow through `session.State` as `*genai.Content`. Multimodal (text, image, audio, data) supported natively.

## Provider Implementation

- **Gemini:** ADK built-in `model/gemini` — zero custom code
- **Anthropic:** `internal/model/anthropic.go` — implements `model.LLM`, converts Anthropic API ↔ `genai.Content`
- **OpenAI:** `internal/model/openai.go` — implements `model.LLM`, converts OpenAI API ↔ `genai.Content`

These are not wrappers around existing code. They are direct `model.LLM` interface implementations that speak ADK's native `genai.Content` type system.

## Agent Implementations

```go
// agent node — ADK LLMAgent, no custom code
llmagent.New(llmagent.Config{
    Name:  nodeDef.ID,
    Model: llms[provider],
    // system prompt, tools from nodeDef.Config
})

// external node — ADK RemoteAgent, no custom code
remoteagent.NewA2A(remoteagent.A2AConfig{
    Name: nodeDef.ID,
    URL:  nodeDef.Config["endpoint_url"].(string),
})

// input/output/tool — ADK Custom Agent with RunFunc
agent.New(agent.Config{
    Name: nodeDef.ID,
    Run:  inputRunFunc,  // ~20 lines each
})
```

## DAGAgent (only significant custom code)

```go
// Custom Agent that executes workflow DAG
agent.New(agent.Config{
    Name: workflowDef.Name,
    Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
        dag := dag.Build(workflowDef)
        agents := buildAgentsFromDef(workflowDef, llms)
        // goroutine fan-out per topological level
        // each node: agents[nodeID].Run(ctx) → yield events
    },
})
```

## File Structure

```
cmd/upal/main.go                 Rewrite (~50 lines)
internal/
  api/
    server.go                    Modify (simplified Server struct)
    run.go                       Modify (ADK Runner invocation)
    workflow.go                  No change
    generate.go                  No change
    upload.go                    No change
    configure.go                 No change
  agents/
    dag.go                       New (~100 lines)
    input.go                     New (~20 lines)
    output.go                    New (~30 lines)
    tool.go                      New (~25 lines)
  model/
    anthropic.go                 New (~150 lines)
    openai.go                    New (~130 lines)
  dag/
    dag.go                       Move from engine/dag.go (113 lines)
  upal/
    types.go                     New (~30 lines, WorkflowDef + NodeDef only)
  db/                            No change
  storage/                       No change
  generate/                      No change
web/
  src/lib/api.ts                 Modify (ADK Event parsing)
  src/stores/workflowStore.ts    Modify (ADK Event handling)
```

## Code Impact

- **Deleted:** ~1,800 lines (engine/, nodes/, provider/, a2atypes/, a2a/, a2aclient/)
- **New:** ~500 lines (agents/, model/)
- **Modified:** ~200 lines (api/, main.go, frontend)
- **Net reduction:** ~1,100 lines

## Frontend Changes

Frontend consumes ADK `session.Event` directly via SSE. No adapter layer.

```typescript
interface ADKEvent {
  id: string
  author: string              // agent name = node ID
  invocation_id: string
  content?: { parts: Part[] } // genai.Content
  actions?: {
    state_delta: Record<string, any>
  }
}
```

Changes limited to `api.ts` (SSE parsing) and `workflowStore.ts` (event handling).

## Dependencies

```
google.golang.org/adk          ADK Go SDK
google.golang.org/genai         Google GenAI types (genai.Content, genai.Part)
github.com/a2aproject/a2a-go    A2A protocol (used by ADK remoteagent)
```

Removes direct HTTP client calls to provider APIs. ADK and model.LLM handle all LLM communication.
