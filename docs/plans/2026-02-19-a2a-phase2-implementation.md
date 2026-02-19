# A2A Phase 2: Full Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Resolve the import cycle between `a2a` and `engine` packages, refactor A2ARunner to use typed Client, wire everything into main.go, add error propagation/cancellation, add external A2A node type, integrate template resolution, and add frontend A2A status display.

**Architecture:** Extract shared protocol types into `internal/a2atypes/` (zero dependencies), move A2A Client into `internal/a2aclient/` (depends only on a2atypes), keep server/router in `internal/a2a/` (depends on engine + a2atypes). This breaks the `a2a → engine → a2a` import cycle, allowing `engine` to use the typed Client.

**Tech Stack:** Go 1.23, Chi router, React 19, TypeScript, Zustand, React Flow, Tailwind CSS v4, Vite

**Design doc:** `docs/plans/2026-02-19-a2a-phase2-design.md`

---

### Task 1: Create `internal/a2atypes/` package — Protocol types + GenerateID

This task extracts all A2A protocol types and `GenerateID` into a new package with zero external dependencies. This is the foundation for breaking the import cycle.

**Files:**
- Create: `internal/a2atypes/types.go`
- Create: `internal/a2atypes/jsonrpc.go`
- Create: `internal/a2atypes/id.go`
- Create: `internal/a2atypes/template.go`
- Create: `internal/a2atypes/types_test.go`
- Create: `internal/a2atypes/template_test.go`

**Step 1: Write tests for `a2atypes` package**

Create `internal/a2atypes/types_test.go`. Copy test cases from `internal/a2a/types_test.go` but change the package to `a2atypes`. The tests should verify JSON round-trip for Part, Artifact, Task, Message, AgentCard, JSONRPCRequest, JSONRPCResponse.

Run: `go test ./internal/a2atypes/... -v -race`
Expected: FAIL — package doesn't exist yet

**Step 2: Create `internal/a2atypes/id.go`**

Move `GenerateID` from `internal/engine/session.go:112-116`:

```go
package a2atypes

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateID generates a random ID with the given prefix.
func GenerateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}
```

**Step 3: Create `internal/a2atypes/types.go`**

Move these types from `internal/a2a/types.go`:
- `Part`, `TextPart()`, `DataPart()`
- `Artifact`, `FirstText()`, `FirstData()`
- `TaskState` constants
- `Task`, `Message`, `NewTask()`
- `AgentCard`, `Capabilities`, `Skill`

Change `NewTask` to call `GenerateID("task")` (same package, no import needed).

```go
package a2atypes

import "encoding/json"

// Part represents a piece of content in a message or artifact.
type Part struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     any    `json:"data,omitempty"`
}

func TextPart(text string) Part {
	return Part{Type: "text", Text: text, MimeType: "text/plain"}
}

func DataPart(data any, mimeType string) Part {
	return Part{Type: "data", Data: data, MimeType: mimeType}
}

type Artifact struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Parts       []Part            `json:"parts"`
	Index       int               `json:"index"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func (a Artifact) FirstText() string {
	for _, p := range a.Parts {
		if p.Type == "text" {
			return p.Text
		}
	}
	return ""
}

func (a Artifact) FirstData() json.RawMessage {
	for _, p := range a.Parts {
		if p.Type == "data" {
			b, err := json.Marshal(p.Data)
			if err != nil {
				return nil
			}
			return b
		}
	}
	return nil
}

type TaskState string

const (
	TaskCreated       TaskState = "created"
	TaskWorking       TaskState = "working"
	TaskInputRequired TaskState = "input-required"
	TaskCompleted     TaskState = "completed"
	TaskFailed        TaskState = "failed"
	TaskCanceled      TaskState = "canceled"
)

type Task struct {
	ID        string     `json:"id"`
	ContextID string     `json:"contextId,omitempty"`
	Status    TaskState  `json:"status"`
	Messages  []Message  `json:"messages,omitempty"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
}

type Message struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

func NewTask(contextID string) *Task {
	return &Task{
		ID:        GenerateID("task"),
		ContextID: contextID,
		Status:    TaskCreated,
	}
}

type AgentCard struct {
	Name               string       `json:"name"`
	Description        string       `json:"description"`
	URL                string       `json:"url"`
	Capabilities       Capabilities `json:"capabilities"`
	Skills             []Skill      `json:"skills"`
	DefaultInputModes  []string     `json:"defaultInputModes"`
	DefaultOutputModes []string     `json:"defaultOutputModes"`
}

type Capabilities struct {
	Streaming         bool `json:"streaming"`
	PushNotifications bool `json:"pushNotifications"`
}

type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	InputModes  []string `json:"inputModes"`
	OutputModes []string `json:"outputModes"`
}
```

**Step 4: Create `internal/a2atypes/jsonrpc.go`**

Move JSON-RPC types from `internal/a2a/types.go`:

```go
package a2atypes

type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type SendMessageParams struct {
	Message       Message            `json:"message"`
	Configuration *SendMessageConfig `json:"configuration,omitempty"`
}

type SendMessageConfig struct {
	AcceptedOutputModes []string `json:"acceptedOutputModes,omitempty"`
	Blocking            bool     `json:"blocking,omitempty"`
}
```

**Step 5: Create `internal/a2atypes/template.go`**

Move from `internal/a2a/template.go`:

```go
package a2atypes

import (
	"encoding/json"
	"regexp"
	"strings"
)

var templatePattern = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

// ResolveTemplate resolves {{node_id}} references from artifact-based state.
func ResolveTemplate(template string, artifacts map[string][]Artifact) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.Trim(match, "{}")
		parts := strings.SplitN(key, ".", 2)
		nodeID := parts[0]

		arts, ok := artifacts[nodeID]
		if !ok || len(arts) == 0 {
			return match
		}
		art := arts[0]

		if len(parts) == 2 && parts[1] == "data" {
			data := art.FirstData()
			if data == nil {
				return match
			}
			return string(data)
		}

		text := art.FirstText()
		if text == "" {
			data := art.FirstData()
			if data != nil {
				return string(data)
			}
			b, _ := json.Marshal(art)
			return string(b)
		}
		return text
	})
}
```

**Step 6: Create `internal/a2atypes/template_test.go`**

Copy tests from `internal/a2a/template_test.go`, change package to `a2atypes`.

**Step 7: Run tests**

Run: `go test ./internal/a2atypes/... -v -race`
Expected: ALL PASS

**Step 8: Commit**

```bash
git add internal/a2atypes/
git commit -m "feat(a2a): extract protocol types into a2atypes package

Moves Part, Artifact, Task, Message, AgentCard, JSON-RPC types,
GenerateID, and ResolveTemplate into internal/a2atypes/ with zero
external dependencies. This is the foundation for resolving the
a2a <-> engine import cycle."
```

---

### Task 2: Create `internal/a2aclient/` package — Move A2A Client

Move the A2A HTTP client out of `internal/a2a/` into its own package that imports only `a2atypes` — NOT `engine`. This is the key piece that lets `engine` use the typed client without creating a cycle.

**Files:**
- Create: `internal/a2aclient/client.go`
- Create: `internal/a2aclient/client_test.go`

**Step 1: Write client test**

Create `internal/a2aclient/client_test.go`. Copy tests from `internal/a2a/client_test.go`, change package to `a2aclient`, update type imports from `a2a.` to `a2atypes.`.

Note: The existing client tests in `internal/a2a/client_test.go` use a `NodeHandler` from the same package as the mock server. For the new tests, create a standalone mock JSON-RPC server that doesn't depend on `a2a.NodeHandler`:

```go
package a2aclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/a2atypes"
)

func mockA2AServer(handler func(req a2atypes.JSONRPCRequest) (any, *a2atypes.JSONRPCError)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req a2atypes.JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		result, rpcErr := handler(req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
			JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr,
		})
	}))
}

func TestClient_SendMessage(t *testing.T) {
	server := mockA2AServer(func(req a2atypes.JSONRPCRequest) (any, *a2atypes.JSONRPCError) {
		task := a2atypes.Task{
			ID:     "task-1",
			Status: a2atypes.TaskCompleted,
			Artifacts: []a2atypes.Artifact{{
				Parts: []a2atypes.Part{a2atypes.TextPart("hello response")},
				Index: 0,
			}},
		}
		return task, nil
	})
	defer server.Close()

	client := NewClient(http.DefaultClient)
	msg := a2atypes.Message{Role: "user", Parts: []a2atypes.Part{a2atypes.TextPart("hello")}}
	task, err := client.SendMessage(context.Background(), server.URL, msg)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if task.Status != a2atypes.TaskCompleted {
		t.Errorf("status: got %q", task.Status)
	}
	if len(task.Artifacts) == 0 || task.Artifacts[0].FirstText() != "hello response" {
		t.Errorf("unexpected artifacts: %+v", task.Artifacts)
	}
}

func TestClient_SendMessage_ServerError(t *testing.T) {
	server := mockA2AServer(func(req a2atypes.JSONRPCRequest) (any, *a2atypes.JSONRPCError) {
		return nil, &a2atypes.JSONRPCError{Code: -32000, Message: "test error"}
	})
	defer server.Close()

	client := NewClient(http.DefaultClient)
	msg := a2atypes.Message{Role: "user", Parts: []a2atypes.Part{a2atypes.TextPart("hello")}}
	_, err := client.SendMessage(context.Background(), server.URL, msg)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_SendMessage_NetworkError(t *testing.T) {
	client := NewClient(http.DefaultClient)
	msg := a2atypes.Message{Role: "user", Parts: []a2atypes.Part{a2atypes.TextPart("hello")}}
	_, err := client.SendMessage(context.Background(), "http://localhost:1", msg)
	if err == nil {
		t.Fatal("expected error")
	}
}
```

Run: `go test ./internal/a2aclient/... -v -race`
Expected: FAIL — package doesn't exist yet

**Step 2: Create `internal/a2aclient/client.go`**

Copy from `internal/a2a/client.go`, change package to `a2aclient`, replace all `a2a.` type references with `a2atypes.`:

```go
package a2aclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/soochol/upal/internal/a2atypes"
)

type Client struct {
	httpClient *http.Client
	nextID     atomic.Int64
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

func (c *Client) SendMessage(ctx context.Context, url string, msg a2atypes.Message) (*a2atypes.Task, error) {
	reqBody := a2atypes.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID.Add(1),
		Method:  "a2a.sendMessage",
		Params: a2atypes.SendMessageParams{
			Message: msg,
			Configuration: &a2atypes.SendMessageConfig{
				AcceptedOutputModes: []string{"text/plain", "application/json"},
				Blocking:            true,
			},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp a2atypes.JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("a2a error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	resultData, err := json.Marshal(rpcResp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	var task a2atypes.Task
	if err := json.Unmarshal(resultData, &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &task, nil
}
```

**Step 3: Run tests**

Run: `go test ./internal/a2aclient/... -v -race`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add internal/a2aclient/
git commit -m "feat(a2a): extract A2A client into a2aclient package

Moves Client to internal/a2aclient/ which imports only a2atypes,
not engine. This enables engine to use the typed Client without
creating an import cycle."
```

---

### Task 3: Update `internal/a2a/` — Use a2atypes, remove moved files

Update `server.go` and `router.go` to import types from `a2atypes` instead of defining them locally. Remove `types.go`, `client.go`, and `template.go` (now in `a2atypes`/`a2aclient`).

**Files:**
- Delete: `internal/a2a/types.go`
- Delete: `internal/a2a/client.go`
- Delete: `internal/a2a/template.go`
- Delete: `internal/a2a/types_test.go`
- Delete: `internal/a2a/client_test.go`
- Delete: `internal/a2a/template_test.go`
- Modify: `internal/a2a/server.go`
- Modify: `internal/a2a/router.go`
- Modify: `internal/a2a/server_test.go`
- Modify: `internal/a2a/router_test.go`
- Modify: `internal/a2a/integration_test.go`

**Step 1: Update `internal/a2a/server.go`**

Add import for `a2atypes`. Replace all direct type references:
- `Part` → `a2atypes.Part`
- `TextPart(...)` → `a2atypes.TextPart(...)`
- `DataPart(...)` → `a2atypes.DataPart(...)`
- `Artifact` → `a2atypes.Artifact`
- `Task` → `a2atypes.Task` (only in `NewTask` call)
- `NewTask(...)` → `a2atypes.NewTask(...)`
- `TaskWorking` → `a2atypes.TaskWorking`
- `TaskCompleted` → `a2atypes.TaskCompleted`
- `TaskFailed` → `a2atypes.TaskFailed`
- `Message` → `a2atypes.Message`
- `AgentCard` → `a2atypes.AgentCard`
- `Capabilities` → `a2atypes.Capabilities`
- `Skill` → `a2atypes.Skill`
- `JSONRPCRequest` → `a2atypes.JSONRPCRequest`
- `JSONRPCResponse` → `a2atypes.JSONRPCResponse`
- `JSONRPCError` → `a2atypes.JSONRPCError`
- `SendMessageParams` → `a2atypes.SendMessageParams`

Keep `writeJSONRPCResponse` and `writeJSONRPCError` helper functions in server.go (they're server-specific).

**Step 2: Update `internal/a2a/router.go`**

Add import for `a2atypes`. Replace `AgentCard` → `a2atypes.AgentCard`, etc.

**Step 3: Delete moved files**

```bash
rm internal/a2a/types.go internal/a2a/client.go internal/a2a/template.go
rm internal/a2a/types_test.go internal/a2a/client_test.go internal/a2a/template_test.go
```

**Step 4: Update test files**

Update `internal/a2a/server_test.go`, `router_test.go`, `integration_test.go` to import `a2atypes` for type references. The mock executors in test files still use `engine.NodeExecutorInterface` (unchanged).

**Step 5: Run all tests**

Run: `go test ./internal/a2a/... ./internal/a2atypes/... ./internal/a2aclient/... -v -race`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add -A internal/a2a/ internal/a2atypes/ internal/a2aclient/
git commit -m "refactor(a2a): migrate a2a package to use a2atypes

Removes types.go, client.go, template.go from a2a package (now in
a2atypes/a2aclient). Updates server.go, router.go, and all tests
to import from a2atypes."
```

---

### Task 4: Update `internal/engine/` — Use a2atypes for GenerateID and Session.Artifacts

Replace `GenerateID` in engine with `a2atypes.GenerateID` and change `Session.Artifacts` from `map[string][]any` to `map[string][]a2atypes.Artifact`.

**Files:**
- Modify: `internal/engine/types.go`
- Modify: `internal/engine/session.go`
- Modify: `internal/engine/runner.go`
- Modify: `internal/engine/a2a_runner.go`

**Step 1: Update `internal/engine/types.go`**

Add import for `a2atypes`. Change `Session.Artifacts`:

```go
import (
	"context"
	"time"

	"github.com/soochol/upal/internal/a2atypes"
)

type Session struct {
	ID         string                          `json:"id"`
	WorkflowID string                          `json:"workflow_id"`
	State      map[string]any                  `json:"state"`
	Artifacts  map[string][]a2atypes.Artifact   `json:"artifacts,omitempty"`
	Events     []Event                         `json:"events"`
	Status     SessionStatus                   `json:"status"`
	CreatedAt  time.Time                       `json:"created_at"`
	UpdatedAt  time.Time                       `json:"updated_at"`
}
```

**Step 2: Update `internal/engine/session.go`**

- Remove `GenerateID` function (now in `a2atypes`)
- Add import for `a2atypes`
- Replace all `GenerateID(...)` calls with `a2atypes.GenerateID(...)`
- Update `Create()`: change `Artifacts: make(map[string][]any)` → `Artifacts: make(map[string][]a2atypes.Artifact)`
- Update `SetArtifacts` signature: `artifacts []any` → `artifacts []a2atypes.Artifact`
- Update `GetArtifacts` return type: `[]any` → `[]a2atypes.Artifact`
- Update `GetAllArtifacts` return type: `map[string][]any` → `map[string][]a2atypes.Artifact`

```go
func (m *SessionManager) SetArtifacts(sessionID, nodeID string, artifacts []a2atypes.Artifact) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		s.Artifacts[nodeID] = artifacts
		s.UpdatedAt = time.Now()
	}
}

func (m *SessionManager) GetArtifacts(sessionID, nodeID string) []a2atypes.Artifact {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil
	}
	src := s.Artifacts[nodeID]
	if src == nil {
		return nil
	}
	cp := make([]a2atypes.Artifact, len(src))
	copy(cp, src)
	return cp
}

func (m *SessionManager) GetAllArtifacts(sessionID string) map[string][]a2atypes.Artifact {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil
	}
	cp := make(map[string][]a2atypes.Artifact, len(s.Artifacts))
	for k, v := range s.Artifacts {
		cp[k] = v
	}
	return cp
}
```

**Step 3: Update `internal/engine/runner.go`**

Replace `GenerateID(...)` with `a2atypes.GenerateID(...)`. Add import for `a2atypes`.

**Step 4: Update `internal/engine/a2a_runner.go` (minimal — just GenerateID for now)**

Replace `GenerateID(...)` with `a2atypes.GenerateID(...)`. Add import for `a2atypes`.

Note: The full A2ARunner refactoring happens in Task 5. For now, the `sendMessage` and artifact storage code will temporarily break because `SetArtifacts` now expects `[]a2atypes.Artifact` instead of `[]any`. Fix the A2ARunner's artifact storage to create proper typed artifacts:

```go
// In Run(), replace the untyped artifact storage:
// OLD: r.sessions.SetArtifacts(sess.ID, "__user_input__"+k, []any{...})
// NEW:
r.sessions.SetArtifacts(sess.ID, "__user_input__"+k, []a2atypes.Artifact{{
	Parts: []a2atypes.Part{a2atypes.TextPart(textVal)},
	Index: 0,
}})
```

For the node result artifact storage, temporarily convert the `map[string]any` response to `a2atypes.Artifact`:

```go
// After sendMessage returns task result (still map[string]any):
if artifacts, ok := task["artifacts"].([]any); ok {
	var typed []a2atypes.Artifact
	// Marshal-unmarshal to convert map[string]any to typed artifacts
	data, _ := json.Marshal(artifacts)
	json.Unmarshal(data, &typed)
	r.sessions.SetArtifacts(sess.ID, nodeID, typed)
}
```

**Step 5: Update test files in engine/**

Update all tests that reference `GenerateID` to use `a2atypes.GenerateID`. Update any tests that create Sessions or use Artifacts to use typed `a2atypes.Artifact`.

**Step 6: Run all tests**

Run: `go test ./internal/engine/... ./internal/a2a/... ./internal/a2atypes/... ./internal/a2aclient/... -v -race`
Expected: ALL PASS

**Step 7: Commit**

```bash
git add internal/engine/
git commit -m "refactor(engine): use a2atypes for GenerateID and typed Artifacts

Session.Artifacts is now map[string][]a2atypes.Artifact instead of
map[string][]any. All GenerateID calls use a2atypes.GenerateID.
Resolves the import cycle foundation."
```

---

### Task 5: Refactor A2ARunner — Use a2aclient.Client + typed artifacts

Replace the duplicated `sendMessage()` method with `a2aclient.Client.SendMessage()`, and rewrite `buildMessageText()` to use typed `a2atypes.Artifact` instead of nested `map[string]any` type assertions.

**Files:**
- Modify: `internal/engine/a2a_runner.go`
- Modify: `internal/engine/a2a_runner_test.go`

**Step 1: Write updated test**

Update `internal/engine/a2a_runner_test.go`. The tests should still use httptest servers that return JSON-RPC responses, but now the runner uses `a2aclient.Client` internally.

The existing tests should continue working with minimal changes — the mock server format is the same (JSON-RPC). Add one new test verifying typed artifact access:

```go
func TestA2ARunner_TypedArtifacts(t *testing.T) {
	wf := &WorkflowDefinition{
		Name:  "typed-test",
		Nodes: []NodeDefinition{{ID: "a", Type: NodeTypeInput}, {ID: "b", Type: NodeTypeAgent}},
		Edges: []EdgeDefinition{{From: "a", To: "b"}},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		task := map[string]any{
			"id": "task-1", "status": "completed",
			"artifacts": []map[string]any{{
				"parts": []map[string]any{{"type": "text", "text": "typed output", "mimeType": "text/plain"}},
				"index": 0,
			}},
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": task})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), client)
	nodeURLs := map[string]string{
		"a": server.URL + "/a2a/nodes/a",
		"b": server.URL + "/a2a/nodes/b",
	}
	sess, err := runner.Run(context.Background(), wf, nodeURLs, map[string]any{"a": "test"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Verify typed artifact access
	arts := sess.Artifacts["b"]
	if len(arts) == 0 {
		t.Fatal("expected artifacts for node b")
	}
	if arts[0].FirstText() != "typed output" {
		t.Errorf("artifact text: got %q", arts[0].FirstText())
	}
}
```

Run: `go test ./internal/engine/... -v -race -run TestA2ARunner_TypedArtifacts`
Expected: FAIL (signature mismatch)

**Step 2: Rewrite `internal/engine/a2a_runner.go`**

Full rewrite:

```go
package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/soochol/upal/internal/a2aclient"
	"github.com/soochol/upal/internal/a2atypes"
)

type A2ARunner struct {
	eventBus *EventBus
	sessions *SessionManager
	client   *a2aclient.Client
}

func NewA2ARunner(eventBus *EventBus, sessions *SessionManager, client *a2aclient.Client) *A2ARunner {
	return &A2ARunner{eventBus: eventBus, sessions: sessions, client: client}
}

func (r *A2ARunner) Run(ctx context.Context, wf *WorkflowDefinition, nodeURLs map[string]string, userInputs map[string]any) (*Session, error) {
	dag, err := BuildDAG(wf)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	sess := r.sessions.Create(wf.Name)

	if userInputs != nil {
		for k, v := range userInputs {
			r.sessions.SetState(sess.ID, "__user_input__"+k, v)
			textVal := fmt.Sprintf("%v", v)
			r.sessions.SetArtifacts(sess.ID, "__user_input__"+k, []a2atypes.Artifact{{
				Parts: []a2atypes.Part{a2atypes.TextPart(textVal)},
				Index: 0,
			}})
		}
	}

	r.eventBus.Publish(Event{
		ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
		Type: EventNodeStarted, Payload: map[string]any{"workflow": wf.Name}, Timestamp: time.Now(),
	})

	done := make(map[string]chan struct{})
	for _, n := range wf.Nodes {
		done[n.ID] = make(chan struct{})
	}

	var wg sync.WaitGroup
	var execErr error
	var errOnce sync.Once

	for _, nodeID := range dag.TopologicalOrder() {
		nodeID := nodeID
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, parentID := range dag.Parents(nodeID) {
				select {
				case <-done[parentID]:
				case <-ctx.Done():
					return
				}
			}

			nodeURL, ok := nodeURLs[nodeID]
			if !ok {
				errOnce.Do(func() { execErr = fmt.Errorf("no URL for node %q", nodeID) })
				close(done[nodeID])
				return
			}

			r.eventBus.Publish(Event{
				ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
				NodeID: nodeID, Type: EventNodeStarted, Timestamp: time.Now(),
			})

			allArtifacts := r.sessions.GetAllArtifacts(sess.ID)
			messageText := buildTypedMessageText(nodeID, allArtifacts, dag)
			if messageText == "" {
				messageText = fmt.Sprintf("Execute node %s", nodeID)
			}

			msg := a2atypes.Message{
				Role:  "user",
				Parts: []a2atypes.Part{a2atypes.TextPart(messageText)},
			}
			task, err := r.client.SendMessage(ctx, nodeURL, msg)
			if err != nil {
				r.eventBus.Publish(Event{
					ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
					NodeID: nodeID, Type: EventNodeError, Payload: map[string]any{"error": err.Error()}, Timestamp: time.Now(),
				})
				errOnce.Do(func() { execErr = fmt.Errorf("node %q: %w", nodeID, err) })
				close(done[nodeID])
				return
			}

			if len(task.Artifacts) > 0 {
				r.sessions.SetArtifacts(sess.ID, nodeID, task.Artifacts)
			}
			// Legacy state compat
			if len(task.Artifacts) > 0 {
				text := task.Artifacts[0].FirstText()
				if text != "" {
					r.sessions.SetState(sess.ID, nodeID, text)
				}
			}

			r.eventBus.Publish(Event{
				ID: a2atypes.GenerateID("ev"), WorkflowID: wf.Name, SessionID: sess.ID,
				NodeID: nodeID, Type: EventNodeCompleted, Payload: map[string]any{"status": string(task.Status)}, Timestamp: time.Now(),
			})
			close(done[nodeID])
		}()
	}

	wg.Wait()

	if execErr != nil {
		r.sessions.SetStatus(sess.ID, SessionFailed)
		finalSess, _ := r.sessions.Get(sess.ID)
		return finalSess, execErr
	}
	r.sessions.SetStatus(sess.ID, SessionCompleted)
	finalSess, _ := r.sessions.Get(sess.ID)
	return finalSess, nil
}

func buildTypedMessageText(nodeID string, allArtifacts map[string][]a2atypes.Artifact, dag *DAG) string {
	parents := dag.Parents(nodeID)
	if len(parents) == 0 {
		return ""
	}
	var texts []string
	for _, parentID := range parents {
		arts, ok := allArtifacts[parentID]
		if !ok {
			arts = allArtifacts["__user_input__"+parentID]
		}
		if len(arts) == 0 {
			continue
		}
		text := arts[0].FirstText()
		if text != "" {
			texts = append(texts, text)
		}
	}
	if len(texts) == 0 {
		return ""
	}
	return strings.Join(texts, "\n\n")
}
```

**Step 3: Update existing tests**

Update `TestA2ARunner_LinearChain`, `TestA2ARunner_FanOutFanIn`, `TestA2ARunner_MissingNodeURL` to create `NewA2ARunner` with `a2aclient.NewClient(http.DefaultClient)` instead of passing `http.DefaultClient` directly.

**Step 4: Run tests**

Run: `go test ./internal/engine/... -v -race`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/engine/a2a_runner.go internal/engine/a2a_runner_test.go
git commit -m "refactor(engine): rewrite A2ARunner to use typed a2aclient.Client

Removes duplicated sendMessage() method (48 lines). Replaces
map[string]any type assertions with typed a2atypes.Artifact access.
buildTypedMessageText uses artifact.FirstText() directly."
```

---

### Task 6: Add error propagation with `context.WithCancel`

Add downstream cancellation — when a node fails, cancel all pending/running nodes immediately instead of letting them execute on failed parent output.

**Files:**
- Modify: `internal/engine/a2a_runner.go`
- Modify: `internal/engine/a2a_runner_test.go`

**Step 1: Write failing test**

Add to `internal/engine/a2a_runner_test.go`:

```go
func TestA2ARunner_ErrorCancelsDownstream(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "cancel-test",
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{{From: "a", To: "b"}, {From: "b", To: "c"}},
	}

	var cExecuted bool
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		nodeID := parts[len(parts)-1]
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if nodeID == "b" {
			// Node b fails
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": req["id"],
				"error": map[string]any{"code": -32000, "message": "node b failed"},
			})
			return
		}
		if nodeID == "c" {
			cExecuted = true
		}
		task := map[string]any{
			"id": "task-1", "status": "completed",
			"artifacts": []map[string]any{{
				"parts": []map[string]any{{"type": "text", "text": "ok", "mimeType": "text/plain"}},
				"index": 0,
			}},
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": task})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), client)
	nodeURLs := map[string]string{
		"a": server.URL + "/a2a/nodes/a",
		"b": server.URL + "/a2a/nodes/b",
		"c": server.URL + "/a2a/nodes/c",
	}
	sess, err := runner.Run(context.Background(), wf, nodeURLs, map[string]any{"a": "test"})
	if err == nil {
		t.Fatal("expected error")
	}
	if sess.Status != SessionFailed {
		t.Errorf("status: got %q, want failed", sess.Status)
	}
	if cExecuted {
		t.Error("node c should NOT have executed after b failed")
	}
}
```

Run: `go test ./internal/engine/... -v -race -run TestA2ARunner_ErrorCancelsDownstream`
Expected: FAIL — `cExecuted` is true because downstream isn't cancelled

**Step 2: Add `context.WithCancel` to A2ARunner.Run**

In `internal/engine/a2a_runner.go`, modify the `Run` method:

```go
func (r *A2ARunner) Run(ctx context.Context, wf *WorkflowDefinition, nodeURLs map[string]string, userInputs map[string]any) (*Session, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// ... same setup ...

	// In the goroutine, on error:
	errOnce.Do(func() {
		execErr = fmt.Errorf("node %q: %w", nodeID, err)
		cancel() // Cancel all pending/running nodes
	})
```

Also update the "no URL" error path to call `cancel()`.

**Step 3: Run test**

Run: `go test ./internal/engine/... -v -race -run TestA2ARunner_ErrorCancelsDownstream`
Expected: PASS

**Step 4: Run all tests**

Run: `go test ./internal/engine/... -v -race`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/engine/a2a_runner.go internal/engine/a2a_runner_test.go
git commit -m "feat(engine): add downstream cancellation on node failure

Uses context.WithCancel to immediately cancel all pending/running
nodes when any node fails, preventing execution on failed parent
output."
```

---

### Task 7: Add external node type (backend)

Add `NodeTypeExternal` for connecting to external A2A agents. The executor reads `endpoint_url` from node config and sends an A2A message to the external endpoint.

**Files:**
- Modify: `internal/engine/types.go`
- Create: `internal/nodes/external.go`
- Create: `internal/nodes/external_test.go`

**Step 1: Write failing test**

Create `internal/nodes/external_test.go`:

```go
package nodes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/a2aclient"
	"github.com/soochol/upal/internal/a2atypes"
	"github.com/soochol/upal/internal/engine"
)

func TestExternalNode_Execute(t *testing.T) {
	// Mock external A2A server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req a2atypes.JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		task := a2atypes.Task{
			ID:     "task-1",
			Status: a2atypes.TaskCompleted,
			Artifacts: []a2atypes.Artifact{{
				Parts: []a2atypes.Part{a2atypes.TextPart("external response")},
				Index: 0,
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
			JSONRPC: "2.0", ID: req.ID, Result: task,
		})
	}))
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	node := NewExternalNode(client)
	def := &engine.NodeDefinition{
		ID:   "ext1",
		Type: engine.NodeTypeExternal,
		Config: map[string]any{
			"endpoint_url": server.URL,
		},
	}
	state := map[string]any{
		"__a2a_message__": "translate this",
	}
	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	text, ok := result.(string)
	if !ok || text != "external response" {
		t.Errorf("result: got %v", result)
	}
}

func TestExternalNode_MissingURL(t *testing.T) {
	client := a2aclient.NewClient(http.DefaultClient)
	node := NewExternalNode(client)
	def := &engine.NodeDefinition{
		ID:     "ext1",
		Type:   engine.NodeTypeExternal,
		Config: map[string]any{},
	}
	_, err := node.Execute(context.Background(), def, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing endpoint_url")
	}
}
```

Run: `go test ./internal/nodes/... -v -race -run TestExternalNode`
Expected: FAIL — ExternalNode doesn't exist

**Step 2: Add `NodeTypeExternal` to `internal/engine/types.go`**

```go
const (
	NodeTypeInput    NodeType = "input"
	NodeTypeAgent    NodeType = "agent"
	NodeTypeTool     NodeType = "tool"
	NodeTypeOutput   NodeType = "output"
	NodeTypeExternal NodeType = "external"
)
```

**Step 3: Create `internal/nodes/external.go`**

```go
package nodes

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/a2aclient"
	"github.com/soochol/upal/internal/a2atypes"
	"github.com/soochol/upal/internal/engine"
)

type ExternalNode struct {
	client *a2aclient.Client
}

func NewExternalNode(client *a2aclient.Client) *ExternalNode {
	return &ExternalNode{client: client}
}

func (n *ExternalNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	endpointURL, ok := def.Config["endpoint_url"].(string)
	if !ok || endpointURL == "" {
		return nil, fmt.Errorf("external node %q: missing endpoint_url config", def.ID)
	}

	// Build message from state
	inputText := ""
	if msg, ok := state["__a2a_message__"].(string); ok {
		inputText = msg
	} else if input, ok := state["__user_input__"+def.ID]; ok {
		inputText = fmt.Sprintf("%v", input)
	}
	if inputText == "" {
		inputText = fmt.Sprintf("Execute external node %s", def.ID)
	}

	msg := a2atypes.Message{
		Role:  "user",
		Parts: []a2atypes.Part{a2atypes.TextPart(inputText)},
	}
	task, err := n.client.SendMessage(ctx, endpointURL, msg)
	if err != nil {
		return nil, fmt.Errorf("external agent call to %s: %w", endpointURL, err)
	}
	if task.Status == a2atypes.TaskFailed {
		return nil, fmt.Errorf("external agent %s failed", endpointURL)
	}
	if len(task.Artifacts) > 0 {
		text := task.Artifacts[0].FirstText()
		if text != "" {
			return text, nil
		}
		data := task.Artifacts[0].FirstData()
		if data != nil {
			return string(data), nil
		}
	}
	return "no output", nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/nodes/... -v -race -run TestExternalNode`
Expected: PASS

Run: `go test ./internal/... -v -race`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/engine/types.go internal/nodes/external.go internal/nodes/external_test.go
git commit -m "feat(nodes): add external A2A agent node type

New NodeTypeExternal allows connecting to external A2A-compatible
agents via endpoint_url config. Uses a2aclient.Client for typed
A2A communication."
```

---

### Task 8: Wire A2ARunner + external node into main.go and run.go

Connect A2ARunner to the server and use it for workflow execution. Add external node executor to the executor map.

**Files:**
- Modify: `cmd/upal/main.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/run.go`
- Modify: `internal/api/server_test.go`

**Step 1: Update `internal/api/server.go`**

Add `a2aRunner` field and update `NewServer`:

```go
import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	a2apkg "github.com/soochol/upal/internal/a2a"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/storage"
)

type Server struct {
	eventBus             *engine.EventBus
	sessions             *engine.SessionManager
	workflows            *WorkflowStore
	runner               *engine.Runner
	a2aRunner            *engine.A2ARunner
	executors            map[engine.NodeType]engine.NodeExecutorInterface
	generator            *generate.Generator
	defaultGenerateModel string
	storage              storage.Storage
	db                   *db.DB
}

func NewServer(eventBus *engine.EventBus, sessions *engine.SessionManager, runner *engine.Runner, a2aRunner *engine.A2ARunner, executors map[engine.NodeType]engine.NodeExecutorInterface) *Server {
	return &Server{
		eventBus:  eventBus,
		sessions:  sessions,
		workflows: NewWorkflowStore(),
		runner:    runner,
		a2aRunner: a2aRunner,
		executors: executors,
	}
}
```

**Step 2: Update `internal/api/run.go`**

Modify `runWorkflow` to mount dynamic A2A routes and use A2ARunner:

```go
func (s *Server) runWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	wf, ok := s.workflows.Get(name)
	if !ok && s.db != nil {
		row, err := s.db.GetWorkflow(r.Context(), name)
		if err == nil {
			wf = &row.Definition
			s.workflows.Put(wf)
			ok = true
		}
	}
	if !ok {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	var req RunRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			req.Inputs = nil
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	eventCh := make(chan engine.Event, 64)
	var sessionID string
	unsub := s.eventBus.Subscribe(func(e engine.Event) {
		if e.WorkflowID == name {
			select {
			case eventCh <- e:
			default:
			}
		}
	})
	defer unsub()

	type runResult struct {
		session *engine.Session
		err     error
	}
	doneCh := make(chan runResult, 1)

	// Use A2ARunner if available, otherwise fall back to Runner
	go func() {
		if s.a2aRunner != nil {
			// Build node URLs from workflow definition
			baseURL := getBaseURL(r)
			nodeURLs := make(map[string]string)
			for _, n := range wf.Nodes {
				// External nodes use their configured endpoint URL
				if n.Type == engine.NodeTypeExternal {
					if url, ok := n.Config["endpoint_url"].(string); ok {
						nodeURLs[n.ID] = url
						continue
					}
				}
				nodeURLs[n.ID] = fmt.Sprintf("%s/a2a/nodes/%s", baseURL, n.ID)
			}
			sess, err := s.a2aRunner.Run(r.Context(), wf, nodeURLs, req.Inputs)
			doneCh <- runResult{session: sess, err: err}
		} else {
			sess, err := s.runner.Run(r.Context(), wf, s.executors, req.Inputs)
			doneCh <- runResult{session: sess, err: err}
		}
	}()

	// ... rest of SSE streaming unchanged ...
```

Add helper function at the bottom of run.go:

```go
func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
```

**Step 3: Update `internal/api/server.go` Handler() — mount dynamic A2A routes**

In `Handler()`, replace `MountStaticA2ARoutes` with dynamic route mounting that accepts workflow context. For now, keep the static routes but also add the ability to register node handlers dynamically when a workflow runs:

The approach: Mount A2A routes as a sub-router that the server can dynamically populate when workflows are loaded.

Actually, the simpler approach: keep `MountStaticA2ARoutes` for static agent card, and in `runWorkflow`, mount node-specific routes before starting the run. Since Chi allows adding routes at startup, we need a different approach for dynamic routing.

**Better approach:** Create a dynamic A2A handler that looks up the node executor at request time:

In `internal/api/server.go`, update `Handler()`:

```go
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}))
	r.Route("/api", func(r chi.Router) {
		r.Route("/workflows", func(r chi.Router) {
			r.Post("/", s.createWorkflow)
			r.Get("/", s.listWorkflows)
			r.Get("/{name}", s.getWorkflow)
			r.Put("/{name}", s.updateWorkflow)
			r.Delete("/{name}", s.deleteWorkflow)
			r.Post("/{name}/run", s.runWorkflow)
		})
		r.Post("/generate", s.generateWorkflow)
		r.Post("/upload", s.uploadFile)
		r.Get("/files", s.listFiles)
	})

	// A2A protocol endpoints — dynamic node handlers
	r.Route("/a2a", func(r chi.Router) {
		r.Get("/agent-card", s.aggregateAgentCard)
		r.Post("/nodes/{nodeID}", s.handleA2ANode)
		r.Get("/nodes/{nodeID}/agent-card", s.handleA2ANodeCard)
	})

	// Serve static files (frontend)
	r.Handle("/*", StaticHandler("web/dist"))

	return r
}
```

Add new handler methods to server.go:

```go
func (s *Server) aggregateAgentCard(w http.ResponseWriter, r *http.Request) {
	card := a2atypes.AgentCard{
		Name:               "upal",
		Description:        "Upal visual AI workflow platform",
		URL:                getBaseURL(r) + "/a2a",
		Capabilities:       a2atypes.Capabilities{},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

func (s *Server) handleA2ANode(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")

	// Look up workflow that contains this node (search loaded workflows)
	var nodeDef *engine.NodeDefinition
	for _, wf := range s.workflows.List() {
		for i, n := range wf.Nodes {
			if n.ID == nodeID {
				nodeDef = &wf.Nodes[i]
				break
			}
		}
		if nodeDef != nil {
			break
		}
	}
	if nodeDef == nil {
		writeJSONRPCError(w, nil, -32001, fmt.Sprintf("node %q not registered", nodeID))
		return
	}

	executor, ok := s.executors[nodeDef.Type]
	if !ok {
		writeJSONRPCError(w, nil, -32001, fmt.Sprintf("no executor for node type %q", nodeDef.Type))
		return
	}

	handler := a2apkg.NewNodeHandler(executor, nodeDef)
	handler.ServeHTTP(w, r)
}

func (s *Server) handleA2ANodeCard(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")

	var nodeDef *engine.NodeDefinition
	for _, wf := range s.workflows.List() {
		for i, n := range wf.Nodes {
			if n.ID == nodeID {
				nodeDef = &wf.Nodes[i]
				break
			}
		}
		if nodeDef != nil {
			break
		}
	}
	if nodeDef == nil {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	card := a2apkg.AgentCardFromNodeDef(nodeDef, getBaseURL(r))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}
```

You will need `writeJSONRPCError` in server.go — add a local helper (or import from a2a):

```go
func writeJSONRPCError(w http.ResponseWriter, id any, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
		JSONRPC: "2.0", ID: id,
		Error: &a2atypes.JSONRPCError{Code: code, Message: message},
	})
}
```

Also add imports for `a2atypes`, `encoding/json`, `fmt`.

**Step 4: Update `cmd/upal/main.go`**

Add A2A client and runner creation:

```go
import (
	// ... existing imports ...
	"github.com/soochol/upal/internal/a2aclient"
)

func serve() {
	// ... existing setup ...

	a2aClient := a2aclient.NewClient(http.DefaultClient)
	runner := engine.NewRunner(eventBus, sessions)
	a2aRunner := engine.NewA2ARunner(eventBus, sessions, a2aClient)

	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeInput:    &nodes.InputNode{},
		engine.NodeTypeAgent:    nodes.NewAgentNode(providerReg, toolReg, eventBus),
		engine.NodeTypeTool:     nodes.NewToolNode(toolReg),
		engine.NodeTypeOutput:   &nodes.OutputNode{},
		engine.NodeTypeExternal: nodes.NewExternalNode(a2aClient),
	}

	srv := api.NewServer(eventBus, sessions, runner, a2aRunner, executors)

	// ... rest unchanged ...
}
```

**Step 5: Update `internal/api/server_test.go`**

Update `NewServer` calls to include `a2aRunner` parameter:

```go
srv := NewServer(engine.NewEventBus(), engine.NewSessionManager(), nil, nil, nil)
```

All existing tests should pass with `nil` for both `runner` and `a2aRunner`.

**Step 6: Check WorkflowStore has List() method**

If `WorkflowStore` doesn't have a `List()` method, add one. Check `internal/api/workflow.go` for the store implementation.

**Step 7: Run all tests**

Run: `go test ./... -v -race`
Expected: ALL PASS

**Step 8: Commit**

```bash
git add cmd/upal/main.go internal/api/server.go internal/api/run.go internal/api/server_test.go
git commit -m "feat(api): wire A2ARunner and external node into server

main.go creates a2aclient, A2ARunner, and ExternalNode executor.
Server uses A2ARunner for workflow execution with dynamic node
routing. External nodes route to configured endpoint URLs."
```

---

### Task 9: Integrate ResolveTemplate into A2ARunner

Use `a2atypes.ResolveTemplate()` to resolve `{{node_id}}` template references in node prompts before sending A2A messages.

**Files:**
- Modify: `internal/engine/a2a_runner.go`
- Modify: `internal/engine/a2a_runner_test.go`

**Step 1: Write failing test**

Add to `internal/engine/a2a_runner_test.go`:

```go
func TestA2ARunner_TemplateResolution(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "template-test",
		Nodes: []NodeDefinition{
			{ID: "input1", Type: NodeTypeInput},
			{ID: "agent1", Type: NodeTypeAgent, Config: map[string]any{
				"prompt": "Summarize: {{input1}}",
			}},
		},
		Edges: []EdgeDefinition{{From: "input1", To: "agent1"}},
	}

	var receivedMessage string
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req a2atypes.JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		// Extract message text from params
		paramsData, _ := json.Marshal(req.Params)
		var params a2atypes.SendMessageParams
		json.Unmarshal(paramsData, &params)
		for _, p := range params.Message.Parts {
			if p.Type == "text" {
				receivedMessage = p.Text
				break
			}
		}

		task := a2atypes.Task{
			ID: "task-1", Status: a2atypes.TaskCompleted,
			Artifacts: []a2atypes.Artifact{{
				Parts: []a2atypes.Part{a2atypes.TextPart("summary output")},
				Index: 0,
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: task})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), client)
	nodeURLs := map[string]string{
		"input1": server.URL + "/a2a/nodes/input1",
		"agent1": server.URL + "/a2a/nodes/agent1",
	}
	_, err := runner.Run(context.Background(), wf, nodeURLs, map[string]any{"input1": "hello world"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The agent node's prompt should have been resolved with input1's artifact text
	expected := "Summarize: hello world"
	if receivedMessage != expected {
		t.Errorf("message: got %q, want %q", receivedMessage, expected)
	}
}
```

Run: `go test ./internal/engine/... -v -race -run TestA2ARunner_TemplateResolution`
Expected: FAIL — template not resolved

**Step 2: Add template resolution to A2ARunner**

In `internal/engine/a2a_runner.go`, modify the goroutine's message building section. Before building the message, check if the node has a `prompt` config with templates:

```go
// In the goroutine, after getting allArtifacts:
allArtifacts := r.sessions.GetAllArtifacts(sess.ID)

// Check for prompt template in node config
nodeDef := dag.Node(nodeID)
var messageText string
if prompt, ok := nodeDef.Config["prompt"].(string); ok && prompt != "" {
	// Resolve {{node_id}} template references against typed artifacts
	messageText = a2atypes.ResolveTemplate(prompt, allArtifacts)
	// Also check for user input references (e.g., {{input1}} could match __user_input__input1)
	// Build a combined artifact map with user input aliases
	combinedArtifacts := make(map[string][]a2atypes.Artifact, len(allArtifacts))
	for k, v := range allArtifacts {
		combinedArtifacts[k] = v
		// Create alias without __user_input__ prefix
		if strings.HasPrefix(k, "__user_input__") {
			alias := strings.TrimPrefix(k, "__user_input__")
			if _, exists := combinedArtifacts[alias]; !exists {
				combinedArtifacts[alias] = v
			}
		}
	}
	messageText = a2atypes.ResolveTemplate(prompt, combinedArtifacts)
} else {
	messageText = buildTypedMessageText(nodeID, allArtifacts, dag)
}

if messageText == "" {
	messageText = fmt.Sprintf("Execute node %s", nodeID)
}
```

**Step 3: Run test**

Run: `go test ./internal/engine/... -v -race -run TestA2ARunner_TemplateResolution`
Expected: PASS

**Step 4: Run all tests**

Run: `go test ./internal/... -v -race`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/engine/a2a_runner.go internal/engine/a2a_runner_test.go
git commit -m "feat(engine): integrate ResolveTemplate into A2ARunner

Node prompts with {{node_id}} references are now resolved against
typed artifacts before sending A2A messages. User inputs are
aliased so {{input1}} matches __user_input__input1 artifacts."
```

---

### Task 10: Frontend — external node type + A2A event display

Add external node type support to the frontend and enhance the console with A2A-specific event formatting.

**Files:**
- Modify: `web/src/stores/workflowStore.ts`
- Modify: `web/src/components/editor/nodes/UpalNode.tsx`
- Modify: `web/src/components/editor/nodes/NodeEditor.tsx`
- Modify: `web/src/components/console/BottomConsole.tsx`

**Step 1: Update `web/src/stores/workflowStore.ts`**

Add `'external'` to `NodeData.nodeType`:

```typescript
export type NodeData = {
  label: string
  nodeType: 'input' | 'agent' | 'tool' | 'output' | 'external'
  description: string
  config: Record<string, unknown>
}
```

In `addNode`, add labels/descriptions for external:

```typescript
const labels: Record<string, string> = {
  input: 'User Input',
  agent: 'Agent',
  tool: 'Tool',
  output: 'Output',
  external: 'External Agent',
}
const descriptions: Record<string, string> = {
  input: 'User-provided data entry point',
  agent: 'AI model processing step',
  tool: 'External tool or function call',
  output: 'Workflow result endpoint',
  external: 'External A2A-compatible agent',
}
```

**Step 2: Update `web/src/components/editor/nodes/UpalNode.tsx`**

Add external node icon and color:

```typescript
import { Inbox, Bot, Wrench, ArrowRightFromLine, Globe, Loader2, Check, X } from 'lucide-react'

const iconMap: Record<string, ComponentType<{ className?: string }>> = {
  input: Inbox,
  agent: Bot,
  tool: Wrench,
  output: ArrowRightFromLine,
  external: Globe,
}

const colorMap: Record<string, string> = {
  // ... existing ...
  external: 'border-purple-500/50 bg-purple-500/10',
}

const accentMap: Record<string, string> = {
  // ... existing ...
  external: 'bg-purple-500 text-white',
}

const glowMap: Record<string, string> = {
  // ... existing ...
  external: 'shadow-[0_0_16px_oklch(0.553_0.235_303.4/0.4)]',
}
```

**Step 3: Update `web/src/components/editor/nodes/NodeEditor.tsx`**

Add external node config fields and icon:

```typescript
import { Inbox, Bot, Wrench, ArrowRightFromLine, Globe, X } from 'lucide-react'

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  input: Inbox,
  agent: Bot,
  tool: Wrench,
  output: ArrowRightFromLine,
  external: Globe,
}
```

Add external node config section after tool section:

```tsx
{data.nodeType === 'external' && (
  <>
    <Separator />
    <div className="space-y-2">
      <Label htmlFor="node-endpoint-url">Endpoint URL</Label>
      <Input
        id="node-endpoint-url"
        value={(config.endpoint_url as string) ?? ''}
        placeholder="https://agent.example.com/a2a"
        onChange={(e) => setConfig('endpoint_url', e.target.value)}
      />
    </div>
    <div className="space-y-2">
      <Label htmlFor="node-timeout">Timeout (seconds)</Label>
      <Input
        id="node-timeout"
        type="number"
        min={1}
        value={(config.timeout as number) ?? 30}
        onChange={(e) => setConfig('timeout', parseInt(e.target.value) || 30)}
      />
    </div>
  </>
)}
```

**Step 4: Update `web/src/components/console/BottomConsole.tsx`**

Add A2A event colors:

```typescript
const eventColorMap: Record<string, string> = {
  'node.started': 'text-node-agent',
  'node.completed': 'text-node-output',
  'node.error': 'text-destructive',
  'model.request': 'text-purple-500 dark:text-purple-400',
  'model.response': 'text-purple-400 dark:text-purple-300',
  'tool.call': 'text-node-tool',
  'tool.result': 'text-amber-600 dark:text-amber-300',
  'a2a.sending': 'text-blue-500 dark:text-blue-400',
  'a2a.received': 'text-blue-400 dark:text-blue-300',
  'workflow.failed': 'text-destructive font-semibold',
  done: 'text-node-output font-semibold',
  error: 'text-destructive',
  info: 'text-muted-foreground',
}
```

Update `formatEvent` to handle A2A events:

```typescript
function formatEvent(event: RunEvent): string {
  const data = event.data
  if (data.message && typeof data.message === 'string') return data.message
  const parts: string[] = []
  if (data.node_id) parts.push(`[${data.node_id}]`)
  if (data.node_type) parts.push(`(${data.node_type})`)
  if (data.model) parts.push(`model=${data.model}`)
  if (data.tool) parts.push(`tool=${data.tool}`)
  if (data.url) parts.push(`→ ${data.url}`)
  if (data.status) parts.push(`status=${data.status}`)
  if (data.error) parts.push(`error: ${data.error}`)
  if (data.result !== undefined)
    parts.push(
      `result: ${typeof data.result === 'string' ? data.result : JSON.stringify(data.result)}`,
    )
  if (data.output !== undefined)
    parts.push(
      `output: ${typeof data.output === 'string' ? data.output : JSON.stringify(data.output)}`,
    )
  if (parts.length === 0) return JSON.stringify(data)
  return parts.join(' ')
}
```

**Step 5: Check if sidebar has a palette for adding nodes**

If there's a node palette (sidebar), add 'external' to the list of draggable node types. Check `web/src/components/sidebar/` for the sidebar panel.

**Step 6: Run frontend type-check**

Run: `cd web && npx tsc -b`
Expected: No type errors

**Step 7: Commit**

```bash
git add web/src/stores/workflowStore.ts web/src/components/editor/nodes/UpalNode.tsx web/src/components/editor/nodes/NodeEditor.tsx web/src/components/console/BottomConsole.tsx
git commit -m "feat(web): add external node type and A2A event display

Adds 'external' node type with Globe icon, purple accent, endpoint
URL and timeout config fields. Console displays A2A-specific events
with blue color coding."
```

---

### Task 11: Integration tests

Write end-to-end integration tests verifying the full A2A execution path with typed client, template resolution, and external nodes.

**Files:**
- Modify: `internal/a2a/integration_test.go`

**Step 1: Update existing integration tests**

The existing tests in `internal/a2a/integration_test.go` use `engine.NewA2ARunner(eventBus, sessions, http.DefaultClient)`. Update to use `a2aclient.NewClient(http.DefaultClient)`:

```go
a2aClient := a2aclient.NewClient(http.DefaultClient)
a2aRunner := engine.NewA2ARunner(eventBus, sessions, a2aClient)
```

Add imports for `a2aclient` and `a2atypes`.

**Step 2: Add integration test for external node**

```go
func TestIntegration_ExternalNode(t *testing.T) {
	// Mock external A2A agent
	externalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req a2atypes.JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		task := a2atypes.Task{
			ID:     "ext-task-1",
			Status: a2atypes.TaskCompleted,
			Artifacts: []a2atypes.Artifact{{
				Parts: []a2atypes.Part{a2atypes.TextPart("translated: hello")},
				Index: 0,
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
			JSONRPC: "2.0", ID: req.ID, Result: task,
		})
	}))
	defer externalServer.Close()

	wf := &engine.WorkflowDefinition{
		Name: "ext-test",
		Nodes: []engine.NodeDefinition{
			{ID: "input", Type: engine.NodeTypeInput},
			{ID: "translate", Type: engine.NodeTypeExternal, Config: map[string]any{
				"endpoint_url": externalServer.URL,
			}},
			{ID: "output", Type: engine.NodeTypeOutput},
		},
		Edges: []engine.EdgeDefinition{
			{From: "input", To: "translate"},
			{From: "translate", To: "output"},
		},
	}

	// Set up local A2A server for input and output nodes
	executor := &mockAgentExecutor{}
	localRouter := chi.NewRouter()
	nodeDefs := make([]*engine.NodeDefinition, 0)
	for i, n := range wf.Nodes {
		if n.Type != engine.NodeTypeExternal {
			nodeDefs = append(nodeDefs, &wf.Nodes[i])
		}
	}
	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeInput:  executor,
		engine.NodeTypeOutput: executor,
	}
	MountA2ARoutes(localRouter, nodeDefs, executors, "http://localhost")
	localServer := httptest.NewServer(localRouter)
	defer localServer.Close()

	a2aClient := a2aclient.NewClient(http.DefaultClient)
	a2aRunner := engine.NewA2ARunner(engine.NewEventBus(), engine.NewSessionManager(), a2aClient)

	nodeURLs := map[string]string{
		"input":     localServer.URL + "/a2a/nodes/input",
		"translate": externalServer.URL,
		"output":    localServer.URL + "/a2a/nodes/output",
	}

	sess, err := a2aRunner.Run(context.Background(), wf, nodeURLs, map[string]any{"input": "hello"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sess.Status != engine.SessionCompleted {
		t.Errorf("status: got %q", sess.Status)
	}
	// Verify external node's artifacts are stored
	arts := sess.Artifacts["translate"]
	if len(arts) == 0 {
		t.Fatal("expected artifacts for translate node")
	}
	if arts[0].FirstText() != "translated: hello" {
		t.Errorf("translate output: got %q", arts[0].FirstText())
	}
}
```

**Step 3: Add integration test for template resolution**

```go
func TestIntegration_TemplateResolution(t *testing.T) {
	var agentReceivedPrompt string

	executor := &mockAgentExecutor{}
	localRouter := chi.NewRouter()

	wf := &engine.WorkflowDefinition{
		Name: "template-test",
		Nodes: []engine.NodeDefinition{
			{ID: "input", Type: engine.NodeTypeInput},
			{ID: "agent", Type: engine.NodeTypeAgent, Config: map[string]any{
				"prompt": "Summarize this: {{input}}",
			}},
		},
		Edges: []engine.EdgeDefinition{{From: "input", To: "agent"}},
	}

	// Custom handler that captures the received message
	localRouter.Post("/a2a/nodes/{nodeID}", func(w http.ResponseWriter, r *http.Request) {
		nodeID := chi.URLParam(r, "nodeID")
		var req a2atypes.JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		paramsData, _ := json.Marshal(req.Params)
		var params a2atypes.SendMessageParams
		json.Unmarshal(paramsData, &params)
		if nodeID == "agent" {
			for _, p := range params.Message.Parts {
				if p.Type == "text" {
					agentReceivedPrompt = p.Text
					break
				}
			}
		}
		// Use mock executor
		nodeDef := &wf.Nodes[0]
		for i := range wf.Nodes {
			if wf.Nodes[i].ID == nodeID {
				nodeDef = &wf.Nodes[i]
				break
			}
		}
		handler := NewNodeHandler(executor, nodeDef)
		handler.ServeHTTP(w, r)
	})
	localServer := httptest.NewServer(localRouter)
	defer localServer.Close()

	a2aClient := a2aclient.NewClient(http.DefaultClient)
	a2aRunner := engine.NewA2ARunner(engine.NewEventBus(), engine.NewSessionManager(), a2aClient)

	nodeURLs := map[string]string{
		"input": localServer.URL + "/a2a/nodes/input",
		"agent": localServer.URL + "/a2a/nodes/agent",
	}

	_, err := a2aRunner.Run(context.Background(), wf, nodeURLs, map[string]any{"input": "hello world"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	expected := "Summarize this: hello world"
	if agentReceivedPrompt != expected {
		t.Errorf("agent prompt: got %q, want %q", agentReceivedPrompt, expected)
	}
}
```

**Step 4: Run integration tests**

Run: `go test ./internal/a2a/... -v -race`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/a2a/integration_test.go
git commit -m "test(a2a): add integration tests for external nodes and templates

Tests verify external node routing to external A2A server,
template resolution with {{node_id}} references, and typed
artifact storage through the full execution pipeline."
```

---

### Task 12: Clean up old a2a router code

The `internal/a2a/router.go` still has `MountA2ARoutes` and `MountStaticA2ARoutes` which are now partially superseded by the dynamic routing in `server.go`. Clean up to avoid confusion.

**Files:**
- Modify: `internal/a2a/router.go`
- Modify: `internal/a2a/router_test.go`

**Step 1: Keep `MountA2ARoutes` for integration tests but remove `MountStaticA2ARoutes`**

`MountStaticA2ARoutes` is no longer needed since `server.go` handles routing dynamically. Remove it from `router.go`. Keep `MountA2ARoutes` — it's used by integration tests and could be useful for test utilities.

Also update `router.go` to import from `a2atypes` (if not already done in Task 3).

**Step 2: Remove references to `MountStaticA2ARoutes` from `server.go`**

The old `a2apkg.MountStaticA2ARoutes(r, s.executors, "")` call was already replaced in Task 8. Verify it's removed.

**Step 3: Run all tests**

Run: `go test ./... -v -race`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add internal/a2a/router.go internal/a2a/router_test.go internal/api/server.go
git commit -m "refactor(a2a): clean up router, remove MountStaticA2ARoutes

Dynamic routing in api/server.go supersedes static route mounting.
Keep MountA2ARoutes for integration test utility."
```

---

### Task 13: Full verification

Run all tests, type-checks, and build to verify everything works together.

**Step 1: Run all Go tests with race detector**

Run: `go test ./... -v -race`
Expected: ALL PASS (no race conditions, no failures)

**Step 2: Run frontend type-check**

Run: `cd web && npx tsc -b`
Expected: No errors

**Step 3: Build frontend**

Run: `cd web && npm run build`
Expected: Build succeeds

**Step 4: Build backend**

Run: `make build`
Expected: Produces `bin/upal` binary

**Step 5: Verify import graph — no cycles**

Run: `go vet ./...`
Expected: No errors, no import cycle warnings

**Step 6: Commit any remaining fixes**

If any test failures or build issues were found, fix and commit.

```bash
git commit -m "chore: final verification — all tests, builds, and type-checks pass"
```

---

## Dependency Graph (Final State)

```
a2atypes (zero deps)
  ├── types.go: Part, Artifact, Task, Message, AgentCard, JSON-RPC types
  ├── id.go: GenerateID
  └── template.go: ResolveTemplate

a2aclient → a2atypes
  └── client.go: Client.SendMessage

a2a → a2atypes + engine
  ├── server.go: NodeHandler, AgentCardFromNodeDef
  └── router.go: MountA2ARoutes

engine → a2atypes + a2aclient
  ├── types.go: Session.Artifacts = map[string][]a2atypes.Artifact
  ├── session.go: SessionManager (typed artifacts)
  ├── a2a_runner.go: A2ARunner (uses a2aclient.Client)
  └── runner.go: Runner (in-process, unchanged)

nodes → engine + a2aclient + a2atypes
  ├── external.go: ExternalNode (uses a2aclient.Client)
  └── agent.go, input.go, output.go, tool.go (unchanged)

api → a2a + engine + a2atypes
  ├── server.go: dynamic A2A routing
  └── run.go: A2ARunner execution path

cmd/upal → api + engine + a2aclient + nodes + ...
  └── main.go: wires everything together
```

No import cycles. Clean separation.
