# Unified GlobalChatBar — Implementation Plan (Phase 1)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the current single-purpose GlobalChatBar and CanvasPromptBar with a unified AI chat that works across all pages, powered by a central `/api/chat` SSE endpoint where the backend LLM decides which tools to call.

**Architecture:** Backend exposes `POST /api/chat` SSE stream. Frontend sends `page` + `context` + `history`. Backend resolves available tools per page/context, constructs system prompt from skills, runs multi-turn LLM with tool call loop, streams events back. Frontend applies tool results to UI via page-registered callbacks.

**Tech Stack:** Go (Chi router, genai SDK), React 19 (Zustand, SSE via EventSource), Tailwind CSS v4

---

### Task 1: Backend — Chat Tool Interface & Registry

**Files:**
- Create: `internal/chat/tools.go`
- Create: `internal/chat/registry.go`

**Step 1: Create the ChatTool interface and ChatRegistry**

```go
// internal/chat/tools.go
package chat

import "context"

// ChatTool defines an action the chat LLM can invoke via tool calls.
type ChatTool struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema for LLM function declaration
	Execute     func(ctx context.Context, args map[string]any) (any, error)
}
```

```go
// internal/chat/registry.go
package chat

import (
	"fmt"

	"google.golang.org/genai"
)

// ChatRegistry holds all chat tools and resolves which are available per page+context.
type ChatRegistry struct {
	tools map[string]*ChatTool
	rules []Rule
}

// Rule maps a page + context condition to a set of tool names.
type Rule struct {
	Page      string
	Condition func(ctx map[string]any) bool // nil means always match
	Tools     []string
}

func NewRegistry() *ChatRegistry {
	return &ChatRegistry{tools: make(map[string]*ChatTool)}
}

func (r *ChatRegistry) Register(t *ChatTool) {
	r.tools[t.Name] = t
}

func (r *ChatRegistry) AddRule(rule Rule) {
	r.rules = append(r.rules, rule)
}

// Resolve returns the chat tools available for the given page and context.
func (r *ChatRegistry) Resolve(page string, context map[string]any) []*ChatTool {
	names := make(map[string]bool)
	for _, rule := range r.rules {
		if rule.Page != page {
			continue
		}
		if rule.Condition != nil && !rule.Condition(context) {
			continue
		}
		for _, name := range rule.Tools {
			names[name] = true
		}
	}
	var result []*ChatTool
	for name := range names {
		if t, ok := r.tools[name]; ok {
			result = append(result, t)
		}
	}
	return result
}

// ToFunctionDeclarations converts chat tools to genai function declarations for the LLM.
func ToFunctionDeclarations(tools []*ChatTool) []*genai.FunctionDeclaration {
	decls := make([]*genai.FunctionDeclaration, len(tools))
	for i, t := range tools {
		decls[i] = &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  toSchema(t.Parameters),
		}
	}
	return decls
}

// ExecuteToolCall finds and executes a chat tool by name.
func (r *ChatRegistry) ExecuteToolCall(ctx context.Context, name string, args map[string]any) (any, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown chat tool: %s", name)
	}
	return t.Execute(ctx, args)
}
```

`toSchema` is a helper that converts `map[string]any` to `*genai.Schema` — follow the pattern used in `internal/tools/registry.go` for `ResolveToolSet`.

**Step 2: Run tests (no test yet, just verify it compiles)**

Run: `go build ./internal/chat/...`
Expected: BUILD OK

**Step 3: Commit**

```bash
git add internal/chat/
git commit -m "feat: add chat tool interface and registry"
```

---

### Task 2: Backend — Workflow Chat Tools

**Files:**
- Create: `internal/chat/workflow_tools.go`

Implement the chat tools for the Workflows page. Each tool wraps existing service layer calls.

**Step 1: Create workflow chat tools**

```go
// internal/chat/workflow_tools.go
package chat

import (
	"context"
	// imports for generator, services, etc.
)

// RegisterWorkflowTools adds workflow-related chat tools and rules.
// Dependencies are injected: generator for configure/generate, repo for CRUD.
func RegisterWorkflowTools(reg *ChatRegistry, deps WorkflowDeps) {
	reg.Register(configureNodeTool(deps))
	reg.Register(generateWorkflowTool(deps))
	reg.Register(addNodeTool(deps))
	reg.Register(removeNodeTool(deps))
	reg.Register(listNodesTool(deps))

	// Base tools for any workflows page
	reg.AddRule(Rule{
		Page:  "workflows",
		Tools: []string{"generate_workflow", "add_node", "remove_node", "list_nodes"},
	})
	// Additional tools when a node is selected
	reg.AddRule(Rule{
		Page:      "workflows",
		Condition: func(ctx map[string]any) bool { _, ok := ctx["selected_node_id"]; return ok },
		Tools:     []string{"configure_node"},
	})
}
```

Each tool function (e.g. `configureNodeTool`) returns a `*ChatTool` with:
- `Name`, `Description`, `Parameters` (JSON Schema for LLM)
- `Execute` that calls the existing service:
  - `configure_node` → `generator.ConfigureNode()` (reuse `internal/generate/configure.go`)
  - `generate_workflow` → `generator.Generate()` (reuse `internal/generate/generate.go`)
  - `add_node` / `remove_node` / `list_nodes` → workflow repository CRUD

The `configure_node` tool should accept `node_id`, `message`, and use the context's `selected_node` data (type, config, label, description, upstream_nodes) from the chat request context that the handler passes in.

**Step 2: Verify compilation**

Run: `go build ./internal/chat/...`
Expected: BUILD OK

**Step 3: Commit**

```bash
git add internal/chat/workflow_tools.go
git commit -m "feat: add workflow chat tools (configure, generate, add, remove, list)"
```

---

### Task 3: Backend — Chat Skill Prompt

**Files:**
- Create: `internal/skills/prompts/chat-workflows.md`

**Step 1: Write the chat-workflows skill**

Follow the format of `internal/skills/prompts/node-configure.md`. The skill should:
- Define an expert persona for a workflow assistant
- Explain that tools are available and the LLM should use them to fulfill requests
- Describe the context JSON structure (workflow_id, selected_node_id, selected_node)
- Instruct to use `configure_node` when user asks about a specific node, `generate_workflow` for workflow creation/editing, `add_node`/`remove_node` for structural changes
- Instruct to respond in Korean
- Instruct to always provide an explanation of what was done

**Step 2: Verify skill loads**

Run: `go test ./internal/skills/... -v -race -run TestRegistry`
Expected: PASS (existing tests should still pass; the new prompt file is auto-embedded)

**Step 3: Commit**

```bash
git add internal/skills/prompts/chat-workflows.md
git commit -m "feat: add chat-workflows skill prompt"
```

---

### Task 4: Backend — `/api/chat` SSE Handler

**Files:**
- Create: `internal/chat/handler.go`
- Modify: `internal/api/server.go` — add route + inject dependencies

This is the core handler: receives request, resolves tools, builds system prompt, runs LLM tool call loop, streams SSE events.

**Step 1: Create the chat handler**

```go
// internal/chat/handler.go
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/genai"
	// adkmodel, skills, llmutil imports
)

type ChatRequest struct {
	Message  string         `json:"message"`
	Page     string         `json:"page"`
	Context  map[string]any `json:"context"`
	History  []ChatMessage  `json:"history"`
	Model    string         `json:"model"`
	Thinking bool           `json:"thinking"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Handler struct {
	registry    *ChatRegistry
	skills      skills.Provider
	llmResolver ports.LLMResolver
	defaultLLM  func(ctx context.Context) (adkmodel.LLM, string, error)
}
```

**Handler.ServeHTTP flow:**

1. Decode `ChatRequest` from body
2. `registry.Resolve(req.Page, req.Context)` → available tools
3. Build system prompt:
   - `skills.GetPrompt("chat-" + req.Page)` as base
   - Append tool descriptions
   - Append `req.Context` as JSON block
4. Build contents from `req.History` + current `req.Message`
5. Set SSE headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`)
6. Tool call loop (max 10 turns), same pattern as `generateWithSkills`:
   - Call `llm.GenerateContent(ctx, llmReq, false)`
   - If tool calls found:
     - Stream `event: tool_call` for each call
     - Execute via `registry.ExecuteToolCall()`
     - Stream `event: tool_result` for each result
     - Append to contents, next turn
   - If no tool calls:
     - Stream `event: text_delta` with response text
     - Stream `event: done`
     - Return

Follow the SSE writing pattern from `internal/api/run.go:102-146` — use `http.Flusher`, `fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ...)`.

**Step 2: Register route in server.go**

In `internal/api/server.go`, add to the `Handler()` method in the `/api` route group:

```go
r.Post("/chat", s.chatHandler.ServeHTTP)
```

Add `chatHandler *chat.Handler` field to `Server` struct (line ~22) and a `SetChatHandler` setter method.

Wire it in `cmd/upal/main.go` where other services are injected.

**Step 3: Test with curl**

Run: `curl -N -X POST http://localhost:8080/api/chat -H 'Content-Type: application/json' -d '{"message":"hello","page":"workflows","context":{}}'`
Expected: SSE stream with `event: text_delta` and `event: done`

**Step 4: Commit**

```bash
git add internal/chat/handler.go internal/api/server.go cmd/upal/main.go
git commit -m "feat: add /api/chat SSE endpoint with tool call loop"
```

---

### Task 5: Backend — Write Tests

**Files:**
- Create: `internal/chat/handler_test.go`
- Create: `internal/chat/registry_test.go`

**Step 1: Write registry tests**

```go
// internal/chat/registry_test.go
func TestResolve_BaseTools(t *testing.T) {
	// Register tools, add rules, verify Resolve returns correct tools for page
}

func TestResolve_ConditionalTools(t *testing.T) {
	// Verify selected_node_id condition adds configure_node
}

func TestResolve_UnknownPage(t *testing.T) {
	// Verify empty result for unregistered page
}
```

**Step 2: Write handler tests**

Use `httptest.NewRecorder` and mock LLM (follow pattern from existing tests). Test:
- Valid request returns SSE stream with `event: done`
- Missing `page` field returns 400
- Tool call triggers `event: tool_call` + `event: tool_result` in stream

**Step 3: Run tests**

Run: `go test ./internal/chat/... -v -race`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add internal/chat/*_test.go
git commit -m "test: add chat handler and registry tests"
```

---

### Task 6: Frontend — Chat Store Redesign

**Files:**
- Modify: `web/src/entities/ui/model/chatStore.ts`

Replace the handler-based store with a context-based store that uses SSE.

**Step 1: Rewrite chatStore.ts**

Replace current `ChatHandler` / `registerHandler` pattern with:

```typescript
export type ChatContext = {
  page: string
  context: Record<string, unknown>
  applyResult: (toolName: string, result: unknown) => void
  placeholder: string
}

export type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
  isError?: boolean
  toolCalls?: { id: string; name: string; args: unknown; result?: unknown; success?: boolean }[]
}

type ChatBarState = {
  isOpen: boolean
  isLoading: boolean
  messages: ChatMessage[]
  chatContext: ChatContext | null

  // Position (draggable)
  position: { x: number; y: number } | null  // null = default center

  // Actions
  registerContext: (ctx: ChatContext) => void
  unregisterContext: () => void
  open: () => void
  close: () => void
  submit: (message: string) => Promise<void>
  setPosition: (pos: { x: number; y: number }) => void
}
```

The `submit` action should:
1. Add user message to `messages`
2. Open SSE connection to `/api/chat` via `fetch` + `ReadableStream` (not EventSource, because we need POST)
3. Parse SSE events from the stream:
   - `text_delta` → accumulate into assistant message content
   - `tool_call` → add to current message's `toolCalls` array, show in UI
   - `tool_result` → update the tool call entry, call `chatContext.applyResult(name, result)`
   - `done` → finalize assistant message, set `isLoading = false`
4. On error → add error message with `isError: true`

For SSE with POST, use the `fetch` API and parse the response body as a readable stream. Do NOT use `EventSource` (it only supports GET). Parse `event:` and `data:` lines from the stream text.

**Step 2: Verify TypeScript compilation**

Run: `cd web && npx tsc --noEmit`
Expected: No errors (some downstream errors expected from consumers not yet updated)

**Step 3: Commit**

```bash
git add web/src/entities/ui/model/chatStore.ts
git commit -m "feat: redesign chat store for SSE + page context"
```

---

### Task 7: Frontend — useRegisterChatContext Hook

**Files:**
- Modify: `web/src/shared/hooks/useRegisterChatHandler.ts` → rename to `useRegisterChatContext.ts`

**Step 1: Rewrite the hook**

```typescript
import { useEffect, useRef } from 'react'
import { useChatBarStore } from '@/entities/ui/model/chatStore'
import type { ChatContext } from '@/entities/ui/model/chatStore'

export function useRegisterChatContext(ctx: ChatContext | null) {
  const register = useChatBarStore((s) => s.registerContext)
  const unregister = useChatBarStore((s) => s.unregisterContext)
  const ctxRef = useRef(ctx)
  ctxRef.current = ctx

  useEffect(() => {
    if (ctx) {
      register(ctx)
    } else {
      unregister()
    }
    return () => unregister()
  }, [ctx?.page, ctx?.placeholder, register, unregister])
}
```

Use a ref for `ctx` so that `applyResult` and `context` are always fresh without re-registering on every render.

**Step 2: Update the export in shared/hooks/index.ts (if exists) or find consumers**

Search for all imports of `useRegisterChatHandler` and update to `useRegisterChatContext`.

**Step 3: Verify TypeScript compilation**

Run: `cd web && npx tsc --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add web/src/shared/hooks/
git commit -m "feat: replace useRegisterChatHandler with useRegisterChatContext"
```

---

### Task 8: Frontend — GlobalChatBar with SSE + Drag

**Files:**
- Modify: `web/src/app/GlobalChatBar.tsx`

**Step 1: Update GlobalChatBar to use new store**

Key changes:
- Read from `chatContext` instead of `handler`
- Show `chatContext.placeholder` in input
- Display `toolCalls` in message bubbles (small chip: "configure_node executed")
- Render `null` if `chatContext` is null

**Step 2: Add drag support**

- Add a drag handle (grip icon) to the input bar
- Track drag state with `onMouseDown` / `onMouseMove` / `onMouseUp`
- Store position in `localStorage` keyed by page: `chatbar-pos-{page}`
- Load saved position on mount, apply via `style={{ left, top }}` instead of `left-1/2 -translate-x-1/2`
- Clamp to viewport bounds on drag end
- Default position: bottom center (when no saved position)

**Step 3: Verify it renders**

Run: `cd web && npm run build`
Expected: BUILD OK

**Step 4: Commit**

```bash
git add web/src/app/GlobalChatBar.tsx
git commit -m "feat: update GlobalChatBar with SSE streaming and drag support"
```

---

### Task 9: Frontend — Workflows Page Context Registration

**Files:**
- Modify: `web/src/pages/workflows/index.tsx`

**Step 1: Replace useRegisterChatHandler with useRegisterChatContext**

Remove the old `nodeConfigureHandler` and `useRegisterChatHandler` call. Replace with:

```typescript
const chatContext = useMemo((): ChatContext | null => {
  const node = selectedNodeId ? nodes.find(n => n.id === selectedNodeId) : null
  return {
    page: 'workflows',
    context: {
      workflow_id: workflowId,
      ...(selectedNodeId && node ? {
        selected_node_id: selectedNodeId,
        selected_node: {
          type: node.data.nodeType,
          config: node.data.config,
          label: node.data.label,
          description: node.data.description,
        },
        upstream_nodes: computeUpstreamNodes(selectedNodeId, nodes, edges),
      } : {}),
    },
    applyResult: (toolName, result) => {
      const r = result as Record<string, any>
      switch (toolName) {
        case 'configure_node':
          if (r.config) updateNodeConfig(r.node_id, r.config)
          if (r.label) updateNodeLabel(r.node_id, r.label)
          if (r.description) updateNodeDescription(r.node_id, r.description)
          break
        case 'add_node':
          // add node to canvas
          break
        case 'remove_node':
          // remove node from canvas
          break
        case 'generate_workflow':
          // reload workflow
          break
      }
    },
    placeholder: selectedNodeId ? 'Ask about this node...' : 'Ask about this workflow...',
  }
}, [workflowId, selectedNodeId, nodes, edges])

useRegisterChatContext(chatContext)
```

**Step 2: Verify TypeScript compilation**

Run: `cd web && npx tsc --noEmit`
Expected: No errors

**Step 3: Commit**

```bash
git add web/src/pages/workflows/index.tsx
git commit -m "feat: register workflow chat context with applyResult callbacks"
```

---

### Task 10: Frontend — Remove CanvasPromptBar

**Files:**
- Modify: `web/src/widgets/workflow-canvas/ui/Canvas.tsx` — remove CanvasPromptBar import and usage
- Delete: `web/src/widgets/workflow-canvas/ui/CanvasPromptBar.tsx`

**Step 1: Remove CanvasPromptBar from Canvas.tsx**

Find the `<CanvasPromptBar>` usage in Canvas.tsx and remove it. Remove the import.

Workflow generation is now handled through the unified GlobalChatBar → `/api/chat` → `generate_workflow` tool path.

**Step 2: Delete CanvasPromptBar.tsx**

Remove the file entirely.

**Step 3: Verify build**

Run: `cd web && npm run build`
Expected: BUILD OK

**Step 4: Commit**

```bash
git add web/src/widgets/workflow-canvas/ui/Canvas.tsx
git rm web/src/widgets/workflow-canvas/ui/CanvasPromptBar.tsx
git commit -m "refactor: remove CanvasPromptBar, workflow generation via unified chat"
```

---

### Task 11: Backend — Wire Everything in main.go

**Files:**
- Modify: `cmd/upal/main.go`

**Step 1: Wire chat registry and handler**

In main.go, after existing service construction:

1. Create `chat.NewRegistry()`
2. Call `chat.RegisterWorkflowTools(registry, deps)` with generator, repo, etc.
3. Create `chat.NewHandler(registry, skills, llmResolver, defaultLLMFunc)`
4. Call `server.SetChatHandler(handler)`

**Step 2: Verify full build**

Run: `make build`
Expected: BUILD OK

**Step 3: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat: wire chat handler into server startup"
```

---

### Task 12: Integration Test — End-to-End

**Files:**
- Create: `internal/chat/integration_test.go`

**Step 1: Write integration test**

Test the full flow: HTTP request → SSE stream → tool call → tool result → done event.

Use `httptest.NewServer` with the chat handler wired to a mock LLM that returns a tool call, then a text response.

Verify:
1. SSE headers are set
2. `event: tool_call` is emitted
3. `event: tool_result` is emitted
4. `event: done` is emitted with final content

**Step 2: Run all tests**

Run: `make test`
Expected: ALL PASS

**Step 3: Run frontend build**

Run: `make build`
Expected: BUILD OK (frontend + backend)

**Step 4: Commit**

```bash
git add internal/chat/integration_test.go
git commit -m "test: add chat endpoint integration test"
```
