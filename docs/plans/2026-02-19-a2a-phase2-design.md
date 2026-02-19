# A2A Protocol Phase 2: Full Integration Design

**Goal:** Resolve the import cycle, refactor A2ARunner to use typed a2a.Client, wire everything into main.go, add error propagation with downstream cancellation, add external A2A agent node type, integrate ResolveTemplate, and add frontend A2A status display.

**Architecture:** Bottom-up refactoring — extract common types package first, then refactor runner, wire into server, add features (external node, template, frontend).

**Tech Stack:** Go 1.23, Chi router, React 19, TypeScript, Zustand, React Flow, Tailwind CSS v4

---

## Section 1: Common Types Package Extraction (Import Cycle Resolution)

### Problem

`a2a` imports `engine` (for `NodeExecutorInterface`, `NodeDefinition`, `GenerateID`), so `engine` cannot import `a2a` (for `Client`, `Task`, `Artifact` types). This forces `A2ARunner` in `engine` to duplicate `a2a.Client` logic using `map[string]any`.

### Solution

Extract protocol types (zero business logic) into `internal/a2atypes/`:

```
internal/a2atypes/        ← NEW (no external dependencies)
  ├── types.go            ← Part, Artifact, Task, Message, TaskState, AgentCard
  ├── jsonrpc.go          ← JSONRPCRequest, JSONRPCResponse, SendMessageParams
  └── id.go               ← GenerateID (moved from engine)

internal/a2a/             ← imports a2atypes + engine
  ├── server.go           ← NodeHandler (wraps engine.NodeExecutorInterface)
  ├── client.go           ← Client (uses a2atypes types)
  ├── template.go         ← ResolveTemplate (uses a2atypes.Artifact)
  └── router.go           ← MountA2ARoutes

internal/engine/          ← imports a2atypes (no cycle!)
  ├── a2a_runner.go       ← uses a2a.Client + a2atypes types
  ├── types.go            ← Session.Artifacts: map[string][]a2atypes.Artifact
  └── session.go          ← typed artifact methods
```

### Key Changes

- `Part`, `Artifact`, `Task`, `Message`, `AgentCard`, JSON-RPC types move to `a2atypes`
- `GenerateID` moves to `a2atypes` (or `internal/id/`)
- `Session.Artifacts` changes from `map[string][]any` to `map[string][]a2atypes.Artifact`
- `a2a` package keeps server, client, template, router (business logic that needs both packages)
- `engine` can now import `a2atypes` and `a2a` (no cycle because `a2a` → `engine` still valid)

Wait — `a2a` still imports `engine` and `engine` would import `a2a` for the Client. The actual fix:
- `a2a.Client` moves types to `a2atypes` so it no longer needs to import `engine`
- `engine/a2a_runner.go` imports `a2a` for `Client` + `a2atypes` for types
- `a2a/server.go` imports `engine` for `NodeExecutorInterface` — this is fine, one-directional

**Dependency graph after refactoring:**
```
a2atypes ← a2a ← engine
              ↗ (a2a/server.go imports engine)
```

This is NOT circular: `a2a` imports `engine` for server.go, and `engine` imports `a2a` for client.go... wait, that IS circular.

**Correct solution:** The `Client` must be in a package that does NOT import `engine`:
- `a2a/client.go` only uses `a2atypes` types → does NOT import `engine` ✓
- `a2a/server.go` imports `engine` for NodeExecutorInterface → `a2a` imports `engine`
- `engine/a2a_runner.go` imports `a2a` for Client → `engine` imports `a2a`
- CYCLE: `a2a` → `engine` → `a2a`

**Final correct solution:** Split `a2a` into two packages:
```
internal/a2atypes/        ← Protocol types (no dependencies)
internal/a2aclient/       ← Client (imports a2atypes only, NOT engine)
internal/a2a/             ← Server, Router, Template (imports engine + a2atypes)
internal/engine/          ← A2ARunner (imports a2aclient + a2atypes, NOT a2a)
```

Dependency graph:
```
a2atypes ← a2aclient ← engine
a2atypes ← a2a ← (reads engine types, NOT imported by engine)
engine   ← a2a/server.go (one-directional, engine does NOT import a2a)
```

No cycles. Clean separation.

## Section 2: A2ARunner Refactoring

### Current State

`engine/a2a_runner.go` contains:
- `sendMessage()` — 48 lines duplicating `a2a.Client` logic with `map[string]any`
- `buildMessageText()` — nested type assertions on `map[string][]any` artifacts
- No template resolution support

### Refactored Design

```go
type A2ARunner struct {
    eventBus *EventBus
    sessions *SessionManager
    client   *a2aclient.Client  // typed A2A client
}

func NewA2ARunner(eventBus *EventBus, sessions *SessionManager, client *a2aclient.Client) *A2ARunner

func (r *A2ARunner) Run(ctx context.Context, wf *WorkflowDefinition, nodeURLs map[string]string, userInputs map[string]any) (*Session, error)
```

Key changes:
- `sendMessage()` removed — uses `r.client.SendMessage()` directly
- `buildMessageText()` uses typed `a2atypes.Artifact` — `artifact.FirstText()` instead of nested type assertions
- `Session.Artifacts` stores `[]a2atypes.Artifact` (typed)
- Template resolution integrated via `a2a.ResolveTemplate()`

## Section 3: main.go / run.go Wiring

### main.go

```go
// Create A2A client and runner
a2aClient := a2aclient.NewClient(http.DefaultClient)
a2aRunner := engine.NewA2ARunner(eventBus, sessions, a2aClient)

// Pass to server
srv := api.NewServer(eventBus, sessions, runner, a2aRunner, executors)
```

### Server struct changes

```go
type Server struct {
    // ... existing fields ...
    a2aRunner *engine.A2ARunner  // NEW
}
```

### run.go workflow execution

On workflow run:
1. Mount A2A node routes for all nodes in the workflow
2. Build `nodeURLs` map from node IDs to local A2A endpoints
3. Run via A2ARunner (HTTP A2A communication between nodes)

```go
func (s *Server) runWorkflow(w http.ResponseWriter, r *http.Request) {
    // ... lookup workflow ...

    // Build node URL map for A2A routing
    baseURL := getBaseURL(r)
    nodeURLs := make(map[string]string)
    for _, n := range wf.Nodes {
        nodeURLs[n.ID] = fmt.Sprintf("%s/a2a/nodes/%s", baseURL, n.ID)
    }

    // Execute via A2ARunner
    sess, err := s.a2aRunner.Run(r.Context(), wf, nodeURLs, req.Inputs)
}
```

## Section 4: Error Propagation & Downstream Cancellation

### Current Problem

When a node fails in A2ARunner:
- Error stored via `errOnce.Do()`
- `done[nodeID]` channel closed (signals completion to children)
- Children see parent "completed" and proceed — executing on failed parent output

### Solution

Use `context.WithCancel`:

```go
func (r *A2ARunner) Run(ctx context.Context, ...) {
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    // In goroutine, on error:
    errOnce.Do(func() {
        execErr = fmt.Errorf("node %q: %w", nodeID, err)
        cancel()  // All pending goroutines see ctx.Done()
    })
}
```

Children waiting on `<-ctx.Done()` exit immediately without executing.

### SSE Error Events

New event type for better error visibility:
```go
EventWorkflowFailed EventType = "workflow.failed"
```

Published with error details when workflow fails, before the `done` SSE event.

## Section 5: External Node Type

### New Node Type

```go
// engine/types.go
NodeTypeExternal NodeType = "external"
```

### Config Schema

```json
{
  "id": "ext-agent",
  "type": "external",
  "config": {
    "endpoint_url": "https://external-agent.example.com/a2a",
    "timeout": 30,
    "description": "External translation agent"
  }
}
```

### Executor

`internal/nodes/external.go`:

```go
type ExternalNode struct {
    client *a2aclient.Client
}

func (n *ExternalNode) Execute(ctx context.Context, def *NodeDefinition, state map[string]any) (any, error) {
    endpointURL := def.Config["endpoint_url"].(string)
    inputText := state["__a2a_message__"].(string)

    msg := a2atypes.Message{Role: "user", Parts: []a2atypes.Part{a2atypes.TextPart(inputText)}}
    task, err := n.client.SendMessage(ctx, endpointURL, msg)
    if err != nil {
        return nil, fmt.Errorf("external agent call: %w", err)
    }
    return task.Artifacts[0].FirstText(), nil
}
```

### A2ARunner Integration

For external nodes, `nodeURLs[nodeID]` points to the external endpoint URL directly (from config), not the local A2A route.

### Frontend

- New node type in Canvas with external-link icon
- NodeEditor shows `endpoint_url` field + optional timeout
- Agent Card discovery: GET `{endpoint_url}/../agent-card` to show capabilities

## Section 6: ResolveTemplate Integration

### Current State

`a2a.ResolveTemplate()` exists but is never called.

### Integration Point

In A2ARunner, when building the message for a node:

```go
// If node has a prompt template, resolve it against artifacts
if prompt, ok := nodeDef.Config["prompt"].(string); ok {
    typedArtifacts := r.sessions.GetAllTypedArtifacts(sess.ID)
    resolvedPrompt := a2a.ResolveTemplate(prompt, typedArtifacts)
    messageText = resolvedPrompt
} else {
    messageText = buildMessageText(nodeID, typedArtifacts, dag)
}
```

This enables:
- `{{input1}}` → text from input1 node's artifact
- `{{input1.data}}` → JSON data from input1's data artifact
- Backward compatible with existing `{{node_id}}` template syntax

## Section 7: Frontend A2A Status Display

### New SSE Event Types

```go
EventA2ASending  EventType = "a2a.sending"   // A2A request being sent
EventA2AReceived EventType = "a2a.received"  // A2A response received
```

### Console Changes

Console.tsx shows A2A-specific events:
- "Sending A2A message to node `agent1` at `/a2a/nodes/agent1`"
- "Received A2A response from `agent1`: completed (1 artifact)"
- External nodes: "Sending A2A message to `https://external.com/a2a`"

### UpalNode Changes

- External nodes display with a globe/external-link icon
- Node status overlay shows A2A task state (working → completed)

### NodeEditor Changes

- External node type: shows endpoint_url, timeout fields
- Agent Card info section (fetched from endpoint)

### workflowStore Changes

- New node type `external` in store's node factory
- A2A event types added to event handling

---

## Migration Notes

- All existing tests must continue to pass after each task
- `map[string][]any` → `map[string][]a2atypes.Artifact` is a breaking change for Session serialization — ensure JSON compatibility
- The original `engine.Runner` (non-A2A) continues to work as before
- Frontend changes are additive (new node type, new event types)
