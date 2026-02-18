# Upal Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a hybrid AI workflow platform (Google Opal + ADK inspired) with visual editor, multi-model support (local + cloud), and MCP tool integration.

**Architecture:** Workflow-First — DAG engine is core, agents are node types. JSON workflow definitions shared between visual editor and code. Event-driven execution with real-time WebSocket streaming.

**Tech Stack:** Go (chi, pgx, websocket) backend + React (Vite, React Flow, Zustand, Shadcn/ui, Tailwind) frontend + PostgreSQL

---

## Phase 1: Project Scaffolding

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `cmd/upal/main.go`
- Create: `Makefile`

**Step 1: Initialize Go module**

Run: `cd /home/dev/code/Upal && go mod init github.com/soochol/upal`
Expected: `go.mod` created

**Step 2: Create entry point**

Create `cmd/upal/main.go`:
```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("upal v0.1.0")
	os.Exit(0)
}
```

**Step 3: Create Makefile**

Create `Makefile`:
```makefile
.PHONY: build run test dev

build:
	go build -o bin/upal ./cmd/upal

run: build
	./bin/upal

test:
	go test ./... -v -race

dev:
	go run ./cmd/upal
```

**Step 4: Verify build**

Run: `make build && ./bin/upal`
Expected: `upal v0.1.0`

**Step 5: Commit**

```bash
git add go.mod cmd/ Makefile
git commit -m "scaffold: initialize Go module with entry point"
```

---

### Task 2: Set Up PostgreSQL with Docker Compose

**Files:**
- Create: `docker-compose.yaml`
- Create: `config.yaml`

**Step 1: Create docker-compose.yaml**

```yaml
services:
  postgres:
    image: postgres:17
    environment:
      POSTGRES_USER: upal
      POSTGRES_PASSWORD: upal
      POSTGRES_DB: upal
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U upal"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

**Step 2: Create config.yaml**

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  url: "postgres://upal:upal@localhost:5432/upal?sslmode=disable"

providers: {}

mcp_servers: {}
```

**Step 3: Verify PostgreSQL starts**

Run: `docker compose up -d && sleep 3 && docker compose exec postgres pg_isready -U upal`
Expected: `localhost:5432 - accepting connections`

Run: `docker compose down`

**Step 4: Commit**

```bash
git add docker-compose.yaml config.yaml
git commit -m "scaffold: add Docker Compose with PostgreSQL"
```

---

### Task 3: Set Up React Frontend

**Files:**
- Create: `web/` (entire React project via Vite)

**Step 1: Create React project with Vite**

Run:
```bash
cd /home/dev/code/Upal
npm create vite@latest web -- --template react-ts
cd web && npm install
```

**Step 2: Install core dependencies**

Run:
```bash
cd /home/dev/code/Upal/web
npm install @xyflow/react zustand
npm install -D tailwindcss @tailwindcss/vite
```

**Step 3: Configure Tailwind**

Replace `web/src/index.css`:
```css
@import "tailwindcss";
```

Update `web/vite.config.ts`:
```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})
```

**Step 4: Replace App.tsx with placeholder**

Replace `web/src/App.tsx`:
```tsx
function App() {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 flex items-center justify-center">
      <h1 className="text-4xl font-bold">Upal</h1>
    </div>
  )
}

export default App
```

**Step 5: Verify frontend builds**

Run: `cd /home/dev/code/Upal/web && npm run build`
Expected: Build succeeds, `web/dist/` created

**Step 6: Add web/ to .gitignore and commit**

Create `/home/dev/code/Upal/.gitignore`:
```
bin/
web/node_modules/
web/dist/
*.exe
.env
```

```bash
git add .gitignore web/
git commit -m "scaffold: add React frontend with Vite, Tailwind, React Flow"
```

---

### Task 4: Add Makefile dev target with concurrent Go + React

**Files:**
- Modify: `Makefile`

**Step 1: Update Makefile**

Add to Makefile:
```makefile
dev-frontend:
	cd web && npm run dev

dev-backend:
	go run ./cmd/upal serve

dev:
	@echo "Run 'make dev-backend' and 'make dev-frontend' in separate terminals"
```

**Step 2: Commit**

```bash
git add Makefile
git commit -m "scaffold: add dev targets to Makefile"
```

---

## Phase 2: Core Engine — Interfaces & Types

### Task 5: Define Core Types (Node, Edge, Workflow, Event, Session)

**Files:**
- Create: `internal/engine/types.go`
- Create: `internal/engine/types_test.go`

**Step 1: Write the test**

Create `internal/engine/types_test.go`:
```go
package engine

import (
	"encoding/json"
	"testing"
)

func TestWorkflowDefinition_JSONRoundTrip(t *testing.T) {
	wf := WorkflowDefinition{
		Name:    "test-workflow",
		Version: 1,
		Nodes: []NodeDefinition{
			{
				ID:   "input1",
				Type: NodeTypeInput,
				Config: map[string]any{
					"input_type": "text",
					"label":      "Enter topic",
				},
			},
			{
				ID:   "agent1",
				Type: NodeTypeAgent,
				Config: map[string]any{
					"model":         "ollama/llama3.2",
					"system_prompt": "You are a researcher.",
				},
			},
		},
		Edges: []EdgeDefinition{
			{From: "input1", To: "agent1"},
		},
	}

	data, err := json.Marshal(wf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got WorkflowDefinition
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != wf.Name {
		t.Errorf("name: got %q, want %q", got.Name, wf.Name)
	}
	if len(got.Nodes) != 2 {
		t.Errorf("nodes: got %d, want 2", len(got.Nodes))
	}
	if len(got.Edges) != 1 {
		t.Errorf("edges: got %d, want 1", len(got.Edges))
	}
	if got.Nodes[0].Type != NodeTypeInput {
		t.Errorf("node type: got %q, want %q", got.Nodes[0].Type, NodeTypeInput)
	}
}

func TestEventType_String(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{EventNodeStarted, "node.started"},
		{EventNodeCompleted, "node.completed"},
		{EventNodeError, "node.error"},
		{EventModelRequest, "model.request"},
		{EventModelResponse, "model.response"},
		{EventToolCall, "tool.call"},
		{EventToolResult, "tool.result"},
	}
	for _, tt := range tests {
		if got := string(tt.et); got != tt.want {
			t.Errorf("EventType: got %q, want %q", got, tt.want)
		}
	}
}

func TestSessionStatus(t *testing.T) {
	s := &Session{
		ID:         "sess-1",
		WorkflowID: "wf-1",
		State:      make(map[string]any),
		Status:     SessionRunning,
	}
	if s.Status != SessionRunning {
		t.Errorf("status: got %q, want %q", s.Status, SessionRunning)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v`
Expected: FAIL — types not defined

**Step 3: Write types implementation**

Create `internal/engine/types.go`:
```go
package engine

import "time"

// NodeType represents the type of a workflow node.
type NodeType string

const (
	NodeTypeInput  NodeType = "input"
	NodeTypeAgent  NodeType = "agent"
	NodeTypeTool   NodeType = "tool"
	NodeTypeOutput NodeType = "output"
)

// EventType represents the type of an execution event.
type EventType string

const (
	EventNodeStarted   EventType = "node.started"
	EventNodeCompleted EventType = "node.completed"
	EventNodeError     EventType = "node.error"
	EventModelRequest  EventType = "model.request"
	EventModelResponse EventType = "model.response"
	EventToolCall      EventType = "tool.call"
	EventToolResult    EventType = "tool.result"
)

// SessionStatus represents the status of a workflow session.
type SessionStatus string

const (
	SessionRunning   SessionStatus = "running"
	SessionCompleted SessionStatus = "completed"
	SessionFailed    SessionStatus = "failed"
	SessionPaused    SessionStatus = "paused"
)

// WorkflowDefinition is the JSON/YAML-serializable workflow definition.
type WorkflowDefinition struct {
	Name    string           `json:"name" yaml:"name"`
	Version int              `json:"version" yaml:"version"`
	Nodes   []NodeDefinition `json:"nodes" yaml:"nodes"`
	Edges   []EdgeDefinition `json:"edges" yaml:"edges"`
}

// NodeDefinition defines a single node in the workflow.
type NodeDefinition struct {
	ID     string         `json:"id" yaml:"id"`
	Type   NodeType       `json:"type" yaml:"type"`
	Config map[string]any `json:"config" yaml:"config"`
}

// EdgeDefinition defines a connection between nodes.
type EdgeDefinition struct {
	From string      `json:"from" yaml:"from"`
	To   string      `json:"to" yaml:"to"`
	Loop *LoopConfig `json:"loop,omitempty" yaml:"loop,omitempty"`
}

// LoopConfig defines loop behavior for back-edges.
type LoopConfig struct {
	MaxIterations int    `json:"max_iterations" yaml:"max_iterations"`
	ExitWhen      string `json:"exit_when" yaml:"exit_when"`
}

// Event represents an execution event.
type Event struct {
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflow_id"`
	SessionID  string    `json:"session_id"`
	NodeID     string    `json:"node_id"`
	Type       EventType `json:"type"`
	Payload    any       `json:"payload"`
	Timestamp  time.Time `json:"timestamp"`
}

// Session represents a workflow execution session.
type Session struct {
	ID         string         `json:"id"`
	WorkflowID string         `json:"workflow_id"`
	State      map[string]any `json:"state"`
	Events     []Event        `json:"events"`
	Status     SessionStatus  `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v`
Expected: PASS (3 tests)

**Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat(engine): add core types — Workflow, Node, Edge, Event, Session"
```

---

### Task 6: Implement DAG Builder (Topology Sort + Fan-out/Fan-in Detection)

**Files:**
- Create: `internal/engine/dag.go`
- Create: `internal/engine/dag_test.go`

**Step 1: Write the test**

Create `internal/engine/dag_test.go`:
```go
package engine

import (
	"testing"
)

func TestDAG_Build_LinearChain(t *testing.T) {
	wf := &WorkflowDefinition{
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}

	dag, err := BuildDAG(wf)
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}

	order := dag.TopologicalOrder()
	if len(order) != 3 {
		t.Fatalf("order length: got %d, want 3", len(order))
	}
	if order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Errorf("order: got %v, want [a b c]", order)
	}
}

func TestDAG_Build_FanOut(t *testing.T) {
	wf := &WorkflowDefinition{
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeAgent},
			{ID: "d", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
		},
	}

	dag, err := BuildDAG(wf)
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}

	// a should have fan-out to b and c
	children := dag.Children("a")
	if len(children) != 2 {
		t.Errorf("a children: got %d, want 2", len(children))
	}

	// d should have fan-in from b and c
	parents := dag.Parents("d")
	if len(parents) != 2 {
		t.Errorf("d parents: got %d, want 2", len(parents))
	}

	// Roots
	roots := dag.Roots()
	if len(roots) != 1 || roots[0] != "a" {
		t.Errorf("roots: got %v, want [a]", roots)
	}
}

func TestDAG_Build_InvalidNode(t *testing.T) {
	wf := &WorkflowDefinition{
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
		},
		Edges: []EdgeDefinition{
			{From: "a", To: "nonexistent"},
		},
	}

	_, err := BuildDAG(wf)
	if err == nil {
		t.Fatal("expected error for edge to nonexistent node")
	}
}

func TestDAG_DetectBackEdges(t *testing.T) {
	wf := &WorkflowDefinition{
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeAgent},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeAgent},
		},
		Edges: []EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "b", Loop: &LoopConfig{MaxIterations: 3, ExitWhen: "done"}},
		},
	}

	dag, err := BuildDAG(wf)
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}

	backEdges := dag.BackEdges()
	if len(backEdges) != 1 {
		t.Fatalf("back edges: got %d, want 1", len(backEdges))
	}
	if backEdges[0].From != "c" || backEdges[0].To != "b" {
		t.Errorf("back edge: got %s->%s, want c->b", backEdges[0].From, backEdges[0].To)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v -run TestDAG`
Expected: FAIL — BuildDAG not defined

**Step 3: Write DAG implementation**

Create `internal/engine/dag.go`:
```go
package engine

import "fmt"

// DAG represents a directed acyclic graph of workflow nodes.
type DAG struct {
	nodes     map[string]*NodeDefinition
	children  map[string][]string   // nodeID -> child nodeIDs
	parents   map[string][]string   // nodeID -> parent nodeIDs
	edges     map[string]EdgeDefinition // "from->to" -> edge
	backEdges []EdgeDefinition
	topoOrder []string
}

// BuildDAG constructs a DAG from a workflow definition.
func BuildDAG(wf *WorkflowDefinition) (*DAG, error) {
	dag := &DAG{
		nodes:    make(map[string]*NodeDefinition),
		children: make(map[string][]string),
		parents:  make(map[string][]string),
		edges:    make(map[string]EdgeDefinition),
	}

	// Index nodes
	for i := range wf.Nodes {
		n := &wf.Nodes[i]
		if _, exists := dag.nodes[n.ID]; exists {
			return nil, fmt.Errorf("duplicate node ID: %s", n.ID)
		}
		dag.nodes[n.ID] = n
	}

	// Build adjacency lists, separate back-edges
	for _, e := range wf.Edges {
		if _, ok := dag.nodes[e.From]; !ok {
			return nil, fmt.Errorf("edge references unknown node: %s", e.From)
		}
		if _, ok := dag.nodes[e.To]; !ok {
			return nil, fmt.Errorf("edge references unknown node: %s", e.To)
		}

		key := e.From + "->" + e.To
		dag.edges[key] = e

		if e.Loop != nil {
			dag.backEdges = append(dag.backEdges, e)
			continue
		}
		dag.children[e.From] = append(dag.children[e.From], e.To)
		dag.parents[e.To] = append(dag.parents[e.To], e.From)
	}

	// Topological sort (Kahn's algorithm)
	order, err := dag.topoSort()
	if err != nil {
		return nil, err
	}
	dag.topoOrder = order

	return dag, nil
}

func (d *DAG) topoSort() ([]string, error) {
	inDegree := make(map[string]int)
	for id := range d.nodes {
		inDegree[id] = 0
	}
	for _, children := range d.children {
		for _, c := range children {
			inDegree[c]++
		}
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)
		for _, c := range d.children[node] {
			inDegree[c]--
			if inDegree[c] == 0 {
				queue = append(queue, c)
			}
		}
	}

	if len(order) != len(d.nodes) {
		return nil, fmt.Errorf("cycle detected in workflow graph (excluding back-edges)")
	}
	return order, nil
}

// TopologicalOrder returns nodes in topological order.
func (d *DAG) TopologicalOrder() []string {
	return d.topoOrder
}

// Children returns child node IDs (excluding back-edges).
func (d *DAG) Children(nodeID string) []string {
	return d.children[nodeID]
}

// Parents returns parent node IDs (excluding back-edges).
func (d *DAG) Parents(nodeID string) []string {
	return d.parents[nodeID]
}

// Roots returns nodes with no parents (entry points).
func (d *DAG) Roots() []string {
	var roots []string
	for id := range d.nodes {
		if len(d.parents[id]) == 0 {
			roots = append(roots, id)
		}
	}
	return roots
}

// BackEdges returns edges marked as loops.
func (d *DAG) BackEdges() []EdgeDefinition {
	return d.backEdges
}

// Node returns a node definition by ID.
func (d *DAG) Node(id string) *NodeDefinition {
	return d.nodes[id]
}

// Edge returns the edge definition between two nodes.
func (d *DAG) Edge(from, to string) (EdgeDefinition, bool) {
	e, ok := d.edges[from+"->"+to]
	return e, ok
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v -run TestDAG`
Expected: PASS (4 tests)

**Step 5: Commit**

```bash
git add internal/engine/dag.go internal/engine/dag_test.go
git commit -m "feat(engine): add DAG builder with topology sort and back-edge detection"
```

---

### Task 7: Implement Event Bus

**Files:**
- Create: `internal/engine/eventbus.go`
- Create: `internal/engine/eventbus_test.go`

**Step 1: Write the test**

Create `internal/engine/eventbus_test.go`:
```go
package engine

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEventBus_Subscribe_Publish(t *testing.T) {
	bus := NewEventBus()

	var received []Event
	var mu sync.Mutex

	bus.Subscribe(func(e Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	ev := Event{
		ID:         "ev-1",
		WorkflowID: "wf-1",
		SessionID:  "sess-1",
		NodeID:     "node-1",
		Type:       EventNodeStarted,
		Timestamp:  time.Now(),
	}

	bus.Publish(ev)

	// Give subscriber goroutine time to process
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("received: got %d, want 1", len(received))
	}
	if received[0].ID != "ev-1" {
		t.Errorf("event ID: got %q, want ev-1", received[0].ID)
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()

	var count1, count2 int
	var mu sync.Mutex

	bus.Subscribe(func(e Event) {
		mu.Lock()
		count1++
		mu.Unlock()
	})
	bus.Subscribe(func(e Event) {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	bus.Publish(Event{ID: "ev-1", Type: EventNodeStarted, Timestamp: time.Now()})
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count1 != 1 || count2 != 1 {
		t.Errorf("counts: got %d/%d, want 1/1", count1, count2)
	}
}

func TestEventBus_Channel(t *testing.T) {
	bus := NewEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Channel(ctx, 10)

	bus.Publish(Event{ID: "ev-1", Type: EventNodeStarted, Timestamp: time.Now()})

	select {
	case ev := <-ch:
		if ev.ID != "ev-1" {
			t.Errorf("event ID: got %q, want ev-1", ev.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v -run TestEventBus`
Expected: FAIL — NewEventBus not defined

**Step 3: Write EventBus implementation**

Create `internal/engine/eventbus.go`:
```go
package engine

import (
	"context"
	"sync"
)

// EventHandler is a function that processes events.
type EventHandler func(Event)

// EventBus distributes events to subscribers.
type EventBus struct {
	mu       sync.RWMutex
	handlers []EventHandler
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{}
}

// Subscribe registers a handler for all events.
func (b *EventBus) Subscribe(handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

// Publish sends an event to all subscribers.
func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	handlers := make([]EventHandler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
}

// Channel returns a channel that receives events until ctx is cancelled.
func (b *EventBus) Channel(ctx context.Context, bufSize int) <-chan Event {
	ch := make(chan Event, bufSize)
	b.Subscribe(func(e Event) {
		select {
		case ch <- e:
		case <-ctx.Done():
		default:
			// Drop event if channel is full
		}
	})
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v -run TestEventBus -race`
Expected: PASS (3 tests)

**Step 5: Commit**

```bash
git add internal/engine/eventbus.go internal/engine/eventbus_test.go
git commit -m "feat(engine): add event bus with subscribe, publish, and channel"
```

---

### Task 8: Implement Session Manager

**Files:**
- Create: `internal/engine/session.go` (manager logic, separate from types)
- Create: `internal/engine/session_test.go`

**Step 1: Write the test**

Create `internal/engine/session_test.go`:
```go
package engine

import (
	"testing"
)

func TestSessionManager_Create(t *testing.T) {
	mgr := NewSessionManager()

	sess := mgr.Create("wf-1")
	if sess.WorkflowID != "wf-1" {
		t.Errorf("workflow ID: got %q, want wf-1", sess.WorkflowID)
	}
	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
	if sess.Status != SessionRunning {
		t.Errorf("status: got %q, want running", sess.Status)
	}
	if sess.State == nil {
		t.Error("state should be initialized")
	}
}

func TestSessionManager_Get(t *testing.T) {
	mgr := NewSessionManager()

	sess := mgr.Create("wf-1")
	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("session not found")
	}
	if got.ID != sess.ID {
		t.Errorf("ID: got %q, want %q", got.ID, sess.ID)
	}
}

func TestSessionManager_SetState(t *testing.T) {
	mgr := NewSessionManager()
	sess := mgr.Create("wf-1")

	mgr.SetState(sess.ID, "node1", "hello world")

	got, _ := mgr.Get(sess.ID)
	val, ok := got.State["node1"]
	if !ok {
		t.Fatal("state key 'node1' not found")
	}
	if val != "hello world" {
		t.Errorf("state value: got %q, want 'hello world'", val)
	}
}

func TestSessionManager_SetStatus(t *testing.T) {
	mgr := NewSessionManager()
	sess := mgr.Create("wf-1")

	mgr.SetStatus(sess.ID, SessionCompleted)

	got, _ := mgr.Get(sess.ID)
	if got.Status != SessionCompleted {
		t.Errorf("status: got %q, want completed", got.Status)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v -run TestSessionManager`
Expected: FAIL — NewSessionManager not defined

**Step 3: Write SessionManager implementation**

Create `internal/engine/session.go`:
```go
package engine

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// SessionManager manages in-memory workflow sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session for a workflow.
func (m *SessionManager) Create(workflowID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess := &Session{
		ID:         generateID("sess"),
		WorkflowID: workflowID,
		State:      make(map[string]any),
		Status:     SessionRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	m.sessions[sess.ID] = sess
	return sess
}

// Get retrieves a session by ID.
func (m *SessionManager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// SetState sets a key in the session state.
func (m *SessionManager) SetState(sessionID, key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		s.State[key] = value
		s.UpdatedAt = time.Now()
	}
}

// SetStatus updates the session status.
func (m *SessionManager) SetStatus(sessionID string, status SessionStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		s.Status = status
		s.UpdatedAt = time.Now()
	}
}

func generateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v -run TestSessionManager -race`
Expected: PASS (4 tests)

**Step 5: Commit**

```bash
git add internal/engine/session.go internal/engine/session_test.go
git commit -m "feat(engine): add in-memory session manager"
```

---

## Phase 3: Model Provider Layer

### Task 9: Define Model Provider Interface

**Files:**
- Create: `internal/provider/provider.go`
- Create: `internal/provider/provider_test.go`

**Step 1: Write the test**

Create `internal/provider/provider_test.go`:
```go
package provider

import (
	"testing"
)

func TestParseModelID(t *testing.T) {
	tests := []struct {
		input    string
		provider string
		model    string
		wantErr  bool
	}{
		{"openai/gpt-4o", "openai", "gpt-4o", false},
		{"ollama/llama3.2", "ollama", "llama3.2", false},
		{"anthropic/claude-sonnet-4-20250514", "anthropic", "claude-sonnet-4-20250514", false},
		{"invalid", "", "", true},
		{"", "", "", true},
	}
	for _, tt := range tests {
		p, m, err := ParseModelID(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseModelID(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			continue
		}
		if p != tt.provider || m != tt.model {
			t.Errorf("ParseModelID(%q): got (%q,%q), want (%q,%q)", tt.input, p, m, tt.provider, tt.model)
		}
	}
}

func TestMessage_Roles(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "You are helpful."},
		{Role: RoleUser, Content: "Hello"},
		{Role: RoleAssistant, Content: "Hi there!"},
	}
	if msgs[0].Role != RoleSystem {
		t.Errorf("role: got %q, want system", msgs[0].Role)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/provider/ -v`
Expected: FAIL — types not defined

**Step 3: Write provider types**

Create `internal/provider/provider.go`:
```go
package provider

import (
	"context"
	"fmt"
	"strings"
)

// Role represents a message role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a chat message.
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents an LLM tool call request.
type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolDefinition describes a tool the model can call.
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"` // JSON Schema
}

// ChatRequest is the input to a chat completion.
type ChatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	MaxTokens   *int             `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream"`
}

// ChatResponse is the output from a chat completion.
type ChatResponse struct {
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string   `json:"finish_reason"`
}

// StreamChunk is a single chunk from a streaming response.
type StreamChunk struct {
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Done      bool       `json:"done"`
}

// Provider is the interface all model providers implement.
type Provider interface {
	// Name returns the provider name (e.g., "openai", "ollama").
	Name() string

	// ChatCompletion sends a non-streaming chat request.
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatCompletionStream sends a streaming chat request.
	ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)
}

// ParseModelID splits "provider/model" into parts.
func ParseModelID(modelID string) (providerName, modelName string, err error) {
	parts := strings.SplitN(modelID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid model ID %q: expected format 'provider/model'", modelID)
	}
	return parts[0], parts[1], nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/provider/ -v`
Expected: PASS (2 tests)

**Step 5: Commit**

```bash
git add internal/provider/
git commit -m "feat(provider): add model provider interface and types"
```

---

### Task 10: Implement OpenAI-Compatible Provider

**Files:**
- Create: `internal/provider/openai.go`
- Create: `internal/provider/openai_test.go`

**Step 1: Write the test (with HTTP mock)**

Create `internal/provider/openai_test.go`:
```go
package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProvider_ChatCompletion(t *testing.T) {
	// Mock OpenAI API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth: %s", r.Header.Get("Authorization"))
		}

		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["model"] != "gpt-4o" {
			t.Errorf("unexpected model: %v", reqBody["model"])
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "Hello! How can I help?",
					},
					"finish_reason": "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAIProvider("openai", server.URL+"/v1", "test-key")

	resp, err := p.ChatCompletion(context.Background(), &ChatRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Content != "Hello! How can I help?" {
		t.Errorf("content: got %q", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("finish_reason: got %q", resp.FinishReason)
	}
}

func TestOpenAIProvider_ChatCompletion_ToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]any{
							{
								"id":   "call_1",
								"type": "function",
								"function": map[string]any{
									"name":      "web_search",
									"arguments": `{"query":"AI trends"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAIProvider("openai", server.URL+"/v1", "test-key")

	resp, err := p.ChatCompletion(context.Background(), &ChatRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: RoleUser, Content: "Search for AI trends"},
		},
		Tools: []ToolDefinition{
			{Name: "web_search", Description: "Search the web"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("tool calls: got %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "web_search" {
		t.Errorf("tool name: got %q", resp.ToolCalls[0].Name)
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	p := NewOpenAIProvider("ollama", "http://localhost:11434/v1", "")
	if p.Name() != "ollama" {
		t.Errorf("name: got %q, want ollama", p.Name())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/provider/ -v -run TestOpenAI`
Expected: FAIL — NewOpenAIProvider not defined

**Step 3: Write OpenAI provider implementation**

Create `internal/provider/openai.go`:
```go
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs.
type OpenAIProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAI-compatible provider.
func NewOpenAIProvider(name, baseURL, apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		name:    name,
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{},
	}
}

func (p *OpenAIProvider) Name() string { return p.name }

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := p.buildRequestBody(req)

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := apiResp.Choices[0]
	result := &ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
	}

	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return result, nil
}

func (p *OpenAIProvider) ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	// Streaming will be implemented in a later task
	return nil, fmt.Errorf("streaming not yet implemented")
}

func (p *OpenAIProvider) buildRequestBody(req *ChatRequest) map[string]any {
	messages := make([]map[string]any, len(req.Messages))
	for i, m := range req.Messages {
		msg := map[string]any{
			"role":    string(m.Role),
			"content": m.Content,
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]map[string]any, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				tcs[j] = map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": tc.Arguments,
					},
				}
			}
			msg["tool_calls"] = tcs
		}
		messages[i] = msg
	}

	body := map[string]any{
		"model":    req.Model,
		"messages": messages,
		"stream":   false,
	}

	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			}
		}
		body["tools"] = tools
	}

	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		body["max_tokens"] = *req.MaxTokens
	}

	return body
}

// OpenAI API response types (internal)
type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/provider/ -v -run TestOpenAI`
Expected: PASS (3 tests)

**Step 5: Commit**

```bash
git add internal/provider/openai.go internal/provider/openai_test.go
git commit -m "feat(provider): add OpenAI-compatible provider with tool call support"
```

---

### Task 11: Implement Provider Registry

**Files:**
- Create: `internal/provider/registry.go`
- Create: `internal/provider/registry_test.go`

**Step 1: Write the test**

Create `internal/provider/registry_test.go`:
```go
package provider

import (
	"context"
	"testing"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: "mock"}, nil
}
func (m *mockProvider) ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	return nil, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockProvider{name: "openai"})
	reg.Register(&mockProvider{name: "ollama"})

	p, ok := reg.Get("openai")
	if !ok {
		t.Fatal("openai not found")
	}
	if p.Name() != "openai" {
		t.Errorf("name: got %q", p.Name())
	}
}

func TestRegistry_Resolve(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockProvider{name: "openai"})

	p, model, err := reg.Resolve("openai/gpt-4o")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("provider: got %q", p.Name())
	}
	if model != "gpt-4o" {
		t.Errorf("model: got %q", model)
	}
}

func TestRegistry_Resolve_Unknown(t *testing.T) {
	reg := NewRegistry()
	_, _, err := reg.Resolve("unknown/model")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/provider/ -v -run TestRegistry`
Expected: FAIL

**Step 3: Write Registry implementation**

Create `internal/provider/registry.go`:
```go
package provider

import (
	"fmt"
	"sync"
)

// Registry manages registered model providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// Get retrieves a provider by name.
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// Resolve parses a model ID and returns the provider and model name.
func (r *Registry) Resolve(modelID string) (Provider, string, error) {
	providerName, modelName, err := ParseModelID(modelID)
	if err != nil {
		return nil, "", err
	}
	p, ok := r.Get(providerName)
	if !ok {
		return nil, "", fmt.Errorf("unknown provider: %q", providerName)
	}
	return p, modelName, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/provider/ -v -run TestRegistry`
Expected: PASS (3 tests)

**Step 5: Commit**

```bash
git add internal/provider/registry.go internal/provider/registry_test.go
git commit -m "feat(provider): add provider registry with model ID resolution"
```

---

## Phase 4: Tool System

### Task 12: Define Tool Interface & Registry

**Files:**
- Create: `internal/tools/tool.go`
- Create: `internal/tools/registry.go`
- Create: `internal/tools/registry_test.go`

**Step 1: Write the test**

Create `internal/tools/registry_test.go`:
```go
package tools

import (
	"context"
	"testing"
)

type echoTool struct{}

func (e *echoTool) Name() string                    { return "echo" }
func (e *echoTool) Description() string             { return "Echoes input" }
func (e *echoTool) InputSchema() map[string]any      { return map[string]any{"type": "object"} }
func (e *echoTool) Execute(ctx context.Context, input any) (any, error) {
	return input, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&echoTool{})

	tool, ok := reg.Get("echo")
	if !ok {
		t.Fatal("echo tool not found")
	}
	if tool.Name() != "echo" {
		t.Errorf("name: got %q", tool.Name())
	}
}

func TestRegistry_Execute(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&echoTool{})

	result, err := reg.Execute(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "hello" {
		t.Errorf("result: got %v, want hello", result)
	}
}

func TestRegistry_Execute_Unknown(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Execute(context.Background(), "unknown", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&echoTool{})

	tools := reg.List()
	if len(tools) != 1 {
		t.Fatalf("list: got %d, want 1", len(tools))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/tools/ -v`
Expected: FAIL

**Step 3: Write Tool interface and Registry**

Create `internal/tools/tool.go`:
```go
package tools

import "context"

// Tool is the interface all tools implement.
type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]any
	Execute(ctx context.Context, input any) (any, error)
}
```

Create `internal/tools/registry.go`:
```go
package tools

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages registered tools.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Execute runs a tool by name with the given input.
func (r *Registry) Execute(ctx context.Context, name string, input any) (any, error) {
	t, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown tool: %q", name)
	}
	return t.Execute(ctx, input)
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/tools/ -v`
Expected: PASS (4 tests)

**Step 5: Commit**

```bash
git add internal/tools/
git commit -m "feat(tools): add tool interface and registry"
```

---

## Phase 5: Node Implementations

### Task 13: Implement Input Node

**Files:**
- Create: `internal/nodes/nodes.go` (node executor interface)
- Create: `internal/nodes/input.go`
- Create: `internal/nodes/input_test.go`

**Step 1: Write the test**

Create `internal/nodes/input_test.go`:
```go
package nodes

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/engine"
)

func TestInputNode_Execute(t *testing.T) {
	node := &InputNode{}

	def := &engine.NodeDefinition{
		ID:   "input1",
		Type: engine.NodeTypeInput,
		Config: map[string]any{
			"input_type": "text",
			"label":      "Enter topic",
		},
	}

	// Simulate user input via session state
	state := map[string]any{
		"__user_input__input1": "AI trends 2026",
	}

	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "AI trends 2026" {
		t.Errorf("result: got %q, want 'AI trends 2026'", result)
	}
}

func TestInputNode_Execute_MissingInput(t *testing.T) {
	node := &InputNode{}

	def := &engine.NodeDefinition{
		ID:   "input1",
		Type: engine.NodeTypeInput,
	}

	_, err := node.Execute(context.Background(), def, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/nodes/ -v -run TestInput`
Expected: FAIL

**Step 3: Write Node executor interface + Input node**

Create `internal/nodes/nodes.go`:
```go
package nodes

import (
	"context"

	"github.com/soochol/upal/internal/engine"
)

// NodeExecutor executes a node with the given definition and session state.
type NodeExecutor interface {
	Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error)
}
```

Create `internal/nodes/input.go`:
```go
package nodes

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/engine"
)

// InputNode collects user input from session state.
type InputNode struct{}

func (n *InputNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	key := "__user_input__" + def.ID
	val, ok := state[key]
	if !ok {
		return nil, fmt.Errorf("no user input for node %q (expected state key %q)", def.ID, key)
	}
	return val, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/nodes/ -v -run TestInput`
Expected: PASS (2 tests)

**Step 5: Commit**

```bash
git add internal/nodes/
git commit -m "feat(nodes): add node executor interface and input node"
```

---

### Task 14: Implement Agent Node (LLM + Tool Loop)

**Files:**
- Create: `internal/nodes/agent.go`
- Create: `internal/nodes/agent_test.go`

**Step 1: Write the test**

Create `internal/nodes/agent_test.go`:
```go
package nodes

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/provider"
	"github.com/soochol/upal/internal/tools"
)

type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) ChatCompletion(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		Content:      "Generated response about: " + req.Messages[len(req.Messages)-1].Content,
		FinishReason: "stop",
	}, nil
}
func (m *mockProvider) ChatCompletionStream(ctx context.Context, req *provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	return nil, nil
}

func TestAgentNode_Execute_SimpleGenerate(t *testing.T) {
	providerReg := provider.NewRegistry()
	providerReg.Register(&mockProvider{})
	toolReg := tools.NewRegistry()

	node := NewAgentNode(providerReg, toolReg, engine.NewEventBus())

	def := &engine.NodeDefinition{
		ID:   "agent1",
		Type: engine.NodeTypeAgent,
		Config: map[string]any{
			"model":  "mock/test",
			"prompt": "Tell me about {{input1}}",
		},
	}

	state := map[string]any{
		"input1": "AI trends",
	}

	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("result type: got %T, want string", result)
	}
	if resultStr == "" {
		t.Error("result should not be empty")
	}
}

func TestAgentNode_Execute_WithSystemPrompt(t *testing.T) {
	providerReg := provider.NewRegistry()
	providerReg.Register(&mockProvider{})
	toolReg := tools.NewRegistry()

	node := NewAgentNode(providerReg, toolReg, engine.NewEventBus())

	def := &engine.NodeDefinition{
		ID:   "agent1",
		Type: engine.NodeTypeAgent,
		Config: map[string]any{
			"model":         "mock/test",
			"system_prompt": "You are a researcher.",
			"prompt":        "Research this topic",
		},
	}

	result, err := node.Execute(context.Background(), def, map[string]any{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == nil {
		t.Error("result should not be nil")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/nodes/ -v -run TestAgent`
Expected: FAIL

**Step 3: Write Agent node implementation**

Create `internal/nodes/agent.go`:
```go
package nodes

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/provider"
	"github.com/soochol/upal/internal/tools"
)

// AgentNode executes an LLM call, optionally with tool use loop.
type AgentNode struct {
	providers *provider.Registry
	tools     *tools.Registry
	eventBus  *engine.EventBus
}

// NewAgentNode creates a new agent node executor.
func NewAgentNode(providers *provider.Registry, tools *tools.Registry, eventBus *engine.EventBus) *AgentNode {
	return &AgentNode{
		providers: providers,
		tools:     tools,
		eventBus:  eventBus,
	}
}

func (n *AgentNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	modelID, _ := def.Config["model"].(string)
	if modelID == "" {
		return nil, fmt.Errorf("agent node %q: model is required", def.ID)
	}

	prov, modelName, err := n.providers.Resolve(modelID)
	if err != nil {
		return nil, fmt.Errorf("agent node %q: %w", def.ID, err)
	}

	// Build messages
	var messages []provider.Message

	if sp, ok := def.Config["system_prompt"].(string); ok && sp != "" {
		messages = append(messages, provider.Message{
			Role:    provider.RoleSystem,
			Content: resolveTemplate(sp, state),
		})
	}

	prompt, _ := def.Config["prompt"].(string)
	if prompt != "" {
		messages = append(messages, provider.Message{
			Role:    provider.RoleUser,
			Content: resolveTemplate(prompt, state),
		})
	}

	// Resolve tool definitions
	var toolDefs []provider.ToolDefinition
	if toolNames, ok := def.Config["tools"].([]any); ok {
		for _, tn := range toolNames {
			name, _ := tn.(string)
			if t, ok := n.tools.Get(name); ok {
				toolDefs = append(toolDefs, provider.ToolDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  t.InputSchema(),
				})
			}
		}
	}

	maxTurns := 1
	if mt, ok := def.Config["max_turns"].(float64); ok {
		maxTurns = int(mt)
	}
	if len(toolDefs) > 0 && maxTurns < 2 {
		maxTurns = 10 // Default for agent with tools
	}

	// Agent loop
	for turn := 0; turn < maxTurns; turn++ {
		n.eventBus.Publish(engine.Event{
			ID:        engine.GenerateID("ev"),
			NodeID:    def.ID,
			Type:      engine.EventModelRequest,
			Payload:   map[string]any{"turn": turn, "model": modelID},
			Timestamp: time.Now(),
		})

		req := &provider.ChatRequest{
			Model:    modelName,
			Messages: messages,
			Tools:    toolDefs,
		}

		resp, err := prov.ChatCompletion(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("agent node %q turn %d: %w", def.ID, turn, err)
		}

		n.eventBus.Publish(engine.Event{
			ID:        engine.GenerateID("ev"),
			NodeID:    def.ID,
			Type:      engine.EventModelResponse,
			Payload:   map[string]any{"content": resp.Content, "tool_calls": resp.ToolCalls},
			Timestamp: time.Now(),
		})

		// No tool calls — return content
		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		// Process tool calls
		messages = append(messages, provider.Message{
			Role:      provider.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			n.eventBus.Publish(engine.Event{
				ID:        engine.GenerateID("ev"),
				NodeID:    def.ID,
				Type:      engine.EventToolCall,
				Payload:   map[string]any{"tool": tc.Name, "args": tc.Arguments},
				Timestamp: time.Now(),
			})

			result, err := n.tools.Execute(ctx, tc.Name, tc.Arguments)

			var resultStr string
			if err != nil {
				resultStr = fmt.Sprintf("Error: %v", err)
			} else {
				resultStr = fmt.Sprintf("%v", result)
			}

			n.eventBus.Publish(engine.Event{
				ID:        engine.GenerateID("ev"),
				NodeID:    def.ID,
				Type:      engine.EventToolResult,
				Payload:   map[string]any{"tool": tc.Name, "result": resultStr},
				Timestamp: time.Now(),
			})

			messages = append(messages, provider.Message{
				Role:       provider.RoleTool,
				Content:    resultStr,
				ToolCallID: tc.ID,
			})
		}
	}

	return nil, fmt.Errorf("agent node %q: exceeded max turns (%d)", def.ID, maxTurns)
}

// resolveTemplate replaces {{key}} with values from state.
var templatePattern = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

func resolveTemplate(template string, state map[string]any) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.Trim(match, "{}")
		// Support simple dot notation: "node.output" → state["node"]
		parts := strings.SplitN(key, ".", 2)
		if val, ok := state[parts[0]]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match
	})
}
```

Note: This requires exporting `GenerateID` from the engine package. Update `internal/engine/session.go`:

Change `generateID` to `GenerateID`:
```go
// GenerateID generates a random ID with the given prefix.
func GenerateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}
```

And update the Create method to use `GenerateID`.

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/nodes/ -v -run TestAgent`
Expected: PASS (2 tests)

**Step 5: Commit**

```bash
git add internal/nodes/agent.go internal/nodes/agent_test.go internal/engine/session.go
git commit -m "feat(nodes): add agent node with LLM call and tool loop"
```

---

### Task 15: Implement Tool Node & Output Node

**Files:**
- Create: `internal/nodes/tool.go`
- Create: `internal/nodes/output.go`
- Create: `internal/nodes/tool_test.go`
- Create: `internal/nodes/output_test.go`

**Step 1: Write tests**

Create `internal/nodes/tool_test.go`:
```go
package nodes

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/tools"
)

type addTool struct{}

func (a *addTool) Name() string               { return "add" }
func (a *addTool) Description() string         { return "Adds numbers" }
func (a *addTool) InputSchema() map[string]any { return nil }
func (a *addTool) Execute(ctx context.Context, input any) (any, error) {
	return "result: 42", nil
}

func TestToolNode_Execute(t *testing.T) {
	toolReg := tools.NewRegistry()
	toolReg.Register(&addTool{})

	node := NewToolNode(toolReg)
	def := &engine.NodeDefinition{
		ID:   "tool1",
		Type: engine.NodeTypeTool,
		Config: map[string]any{
			"tool":  "add",
			"input": "{{input1}}",
		},
	}

	state := map[string]any{"input1": "some data"}
	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "result: 42" {
		t.Errorf("result: got %v", result)
	}
}
```

Create `internal/nodes/output_test.go`:
```go
package nodes

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/engine"
)

func TestOutputNode_Execute(t *testing.T) {
	node := &OutputNode{}
	def := &engine.NodeDefinition{
		ID:   "output1",
		Type: engine.NodeTypeOutput,
		Config: map[string]any{
			"output_type": "markdown",
		},
	}

	state := map[string]any{
		"agent1": "# Hello World\nThis is content.",
	}

	// Output node should collect all parent outputs
	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == nil {
		t.Error("result should not be nil")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /home/dev/code/Upal && go test ./internal/nodes/ -v -run "TestTool|TestOutput"`
Expected: FAIL

**Step 3: Write implementations**

Create `internal/nodes/tool.go`:
```go
package nodes

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/tools"
)

// ToolNode executes an external tool directly (no LLM).
type ToolNode struct {
	tools *tools.Registry
}

// NewToolNode creates a new tool node executor.
func NewToolNode(tools *tools.Registry) *ToolNode {
	return &ToolNode{tools: tools}
}

func (n *ToolNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	toolName, _ := def.Config["tool"].(string)
	if toolName == "" {
		return nil, fmt.Errorf("tool node %q: tool name is required", def.ID)
	}

	inputTemplate, _ := def.Config["input"].(string)
	input := resolveTemplate(inputTemplate, state)

	return n.tools.Execute(ctx, toolName, input)
}
```

Create `internal/nodes/output.go`:
```go
package nodes

import (
	"context"
	"fmt"
	"strings"

	"github.com/soochol/upal/internal/engine"
)

// OutputNode collects and formats the workflow output.
type OutputNode struct {
	// parentNodeIDs will be set by the runner based on DAG edges
}

func (n *OutputNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	// Collect all values from state (parent node outputs)
	// The runner will have stored parent outputs in state
	var parts []string
	for key, val := range state {
		// Skip internal keys
		if strings.HasPrefix(key, "__") {
			continue
		}
		if s, ok := val.(string); ok && s != "" {
			parts = append(parts, s)
		}
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("output node %q: no data to output", def.ID)
	}

	return strings.Join(parts, "\n\n"), nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /home/dev/code/Upal && go test ./internal/nodes/ -v -run "TestTool|TestOutput"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/nodes/tool.go internal/nodes/output.go internal/nodes/tool_test.go internal/nodes/output_test.go
git commit -m "feat(nodes): add tool node and output node"
```

---

## Phase 6: DAG Runner (Orchestration)

### Task 16: Implement DAG Runner

**Files:**
- Create: `internal/engine/runner.go`
- Create: `internal/engine/runner_test.go`

**Step 1: Write the test**

Create `internal/engine/runner_test.go`:
```go
package engine

import (
	"context"
	"fmt"
	"testing"
)

type mockNodeExecutor struct {
	results map[string]any
}

func (m *mockNodeExecutor) Execute(ctx context.Context, def *NodeDefinition, state map[string]any) (any, error) {
	if r, ok := m.results[def.ID]; ok {
		return r, nil
	}
	return fmt.Sprintf("output of %s", def.ID), nil
}

func TestRunner_LinearChain(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "test",
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}

	executor := &mockNodeExecutor{
		results: map[string]any{
			"a": "input data",
			"b": "processed data",
			"c": "final output",
		},
	}

	executors := map[NodeType]NodeExecutorInterface{
		NodeTypeInput:  executor,
		NodeTypeAgent:  executor,
		NodeTypeOutput: executor,
	}

	runner := NewRunner(NewEventBus(), NewSessionManager())
	result, err := runner.Run(context.Background(), wf, executors, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestRunner_FanOutFanIn(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "fan-test",
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeAgent},
			{ID: "d", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
		},
	}

	executed := make(map[string]bool)
	executor := &trackingExecutor{executed: executed}

	executors := map[NodeType]NodeExecutorInterface{
		NodeTypeInput:  executor,
		NodeTypeAgent:  executor,
		NodeTypeOutput: executor,
	}

	runner := NewRunner(NewEventBus(), NewSessionManager())
	_, err := runner.Run(context.Background(), wf, executors, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, id := range []string{"a", "b", "c", "d"} {
		if !executed[id] {
			t.Errorf("node %q was not executed", id)
		}
	}
}

type trackingExecutor struct {
	executed map[string]bool
}

func (e *trackingExecutor) Execute(ctx context.Context, def *NodeDefinition, state map[string]any) (any, error) {
	e.executed[def.ID] = true
	return fmt.Sprintf("output of %s", def.ID), nil
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v -run TestRunner`
Expected: FAIL

**Step 3: Write Runner implementation**

Add to `internal/engine/types.go`:
```go
// NodeExecutorInterface is the interface for executing nodes.
type NodeExecutorInterface interface {
	Execute(ctx context.Context, def *NodeDefinition, state map[string]any) (any, error)
}
```

Create `internal/engine/runner.go`:
```go
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Runner orchestrates workflow execution.
type Runner struct {
	eventBus *EventBus
	sessions *SessionManager
}

// NewRunner creates a new workflow runner.
func NewRunner(eventBus *EventBus, sessions *SessionManager) *Runner {
	return &Runner{
		eventBus: eventBus,
		sessions: sessions,
	}
}

// Run executes a workflow definition. userInputs maps "nodeID" → user input value.
func (r *Runner) Run(ctx context.Context, wf *WorkflowDefinition, executors map[NodeType]NodeExecutorInterface, userInputs map[string]any) (*Session, error) {
	dag, err := BuildDAG(wf)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	sess := r.sessions.Create(wf.Name)

	// Inject user inputs into state
	if userInputs != nil {
		for k, v := range userInputs {
			r.sessions.SetState(sess.ID, "__user_input__"+k, v)
		}
	}

	r.eventBus.Publish(Event{
		ID:         GenerateID("ev"),
		WorkflowID: wf.Name,
		SessionID:  sess.ID,
		Type:       EventNodeStarted,
		Payload:    map[string]any{"workflow": wf.Name},
		Timestamp:  time.Now(),
	})

	// Track completion: nodeID → done channel
	done := make(map[string]chan struct{})
	for _, n := range wf.Nodes {
		done[n.ID] = make(chan struct{})
	}

	// Execute nodes respecting DAG dependencies
	var wg sync.WaitGroup
	var execErr error
	var errOnce sync.Once

	for _, nodeID := range dag.TopologicalOrder() {
		nodeID := nodeID
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Wait for all parents to complete
			for _, parentID := range dag.Parents(nodeID) {
				select {
				case <-done[parentID]:
				case <-ctx.Done():
					return
				}
			}

			// Get current session state
			currentSess, _ := r.sessions.Get(sess.ID)
			stateCopy := make(map[string]any)
			for k, v := range currentSess.State {
				stateCopy[k] = v
			}

			nodeDef := dag.Node(nodeID)
			executor, ok := executors[nodeDef.Type]
			if !ok {
				errOnce.Do(func() {
					execErr = fmt.Errorf("no executor for node type %q", nodeDef.Type)
				})
				close(done[nodeID])
				return
			}

			r.eventBus.Publish(Event{
				ID:         GenerateID("ev"),
				WorkflowID: wf.Name,
				SessionID:  sess.ID,
				NodeID:     nodeID,
				Type:       EventNodeStarted,
				Timestamp:  time.Now(),
			})

			result, err := executor.Execute(ctx, nodeDef, stateCopy)
			if err != nil {
				r.eventBus.Publish(Event{
					ID:         GenerateID("ev"),
					WorkflowID: wf.Name,
					SessionID:  sess.ID,
					NodeID:     nodeID,
					Type:       EventNodeError,
					Payload:    map[string]any{"error": err.Error()},
					Timestamp:  time.Now(),
				})
				errOnce.Do(func() {
					execErr = fmt.Errorf("node %q: %w", nodeID, err)
				})
				close(done[nodeID])
				return
			}

			// Store result in session state
			r.sessions.SetState(sess.ID, nodeID, result)

			r.eventBus.Publish(Event{
				ID:         GenerateID("ev"),
				WorkflowID: wf.Name,
				SessionID:  sess.ID,
				NodeID:     nodeID,
				Type:       EventNodeCompleted,
				Payload:    map[string]any{"result": result},
				Timestamp:  time.Now(),
			})

			close(done[nodeID])
		}()
	}

	wg.Wait()

	if execErr != nil {
		r.sessions.SetStatus(sess.ID, SessionFailed)
		return nil, execErr
	}

	r.sessions.SetStatus(sess.ID, SessionCompleted)
	finalSess, _ := r.sessions.Get(sess.ID)
	return finalSess, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/engine/ -v -run TestRunner -race`
Expected: PASS (2 tests)

**Step 5: Commit**

```bash
git add internal/engine/runner.go internal/engine/runner_test.go internal/engine/types.go
git commit -m "feat(engine): add DAG runner with fan-out/fan-in parallel execution"
```

---

## Phase 7: HTTP API

### Task 17: Implement REST API (Workflow CRUD + Execute)

**Files:**
- Create: `internal/api/server.go`
- Create: `internal/api/workflow.go`
- Create: `internal/api/server_test.go`

**Step 1: Install dependencies**

Run:
```bash
cd /home/dev/code/Upal
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
```

**Step 2: Write the test**

Create `internal/api/server_test.go`:
```go
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/engine"
)

func TestAPI_CreateWorkflow(t *testing.T) {
	srv := NewServer(engine.NewEventBus(), engine.NewSessionManager())

	wf := engine.WorkflowDefinition{
		Name:    "test-wf",
		Version: 1,
		Nodes: []engine.NodeDefinition{
			{ID: "input1", Type: engine.NodeTypeInput},
		},
	}

	body, _ := json.Marshal(wf)
	req := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "test-wf" {
		t.Errorf("name: got %v", resp["name"])
	}
}

func TestAPI_ListWorkflows(t *testing.T) {
	srv := NewServer(engine.NewEventBus(), engine.NewSessionManager())

	// Create a workflow first
	wf := engine.WorkflowDefinition{Name: "wf1", Version: 1}
	body, _ := json.Marshal(wf)
	req := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	// List
	req = httptest.NewRequest("GET", "/api/workflows", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}

	var resp []map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 1 {
		t.Errorf("workflows: got %d, want 1", len(resp))
	}
}

func TestAPI_GetWorkflow(t *testing.T) {
	srv := NewServer(engine.NewEventBus(), engine.NewSessionManager())

	wf := engine.WorkflowDefinition{Name: "wf1", Version: 1}
	body, _ := json.Marshal(wf)
	req := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	req = httptest.NewRequest("GET", "/api/workflows/wf1", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
}
```

**Step 2b: Run test to verify it fails**

Run: `cd /home/dev/code/Upal && go test ./internal/api/ -v`
Expected: FAIL

**Step 3: Write API implementation**

Create `internal/api/server.go`:
```go
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/soochol/upal/internal/engine"
)

// Server is the HTTP API server.
type Server struct {
	eventBus *engine.EventBus
	sessions *engine.SessionManager
	workflows *WorkflowStore
}

// NewServer creates a new API server.
func NewServer(eventBus *engine.EventBus, sessions *engine.SessionManager) *Server {
	return &Server{
		eventBus:  eventBus,
		sessions:  sessions,
		workflows: NewWorkflowStore(),
	}
}

// Handler returns the HTTP handler.
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
		})
	})

	return r
}
```

Create `internal/api/workflow.go`:
```go
package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/engine"
)

// WorkflowStore is an in-memory workflow store (to be replaced with PostgreSQL).
type WorkflowStore struct {
	mu        sync.RWMutex
	workflows map[string]*engine.WorkflowDefinition
}

// NewWorkflowStore creates a new in-memory workflow store.
func NewWorkflowStore() *WorkflowStore {
	return &WorkflowStore{
		workflows: make(map[string]*engine.WorkflowDefinition),
	}
}

func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf engine.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.workflows.mu.Lock()
	s.workflows.workflows[wf.Name] = &wf
	s.workflows.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	s.workflows.mu.RLock()
	defer s.workflows.mu.RUnlock()

	result := make([]*engine.WorkflowDefinition, 0, len(s.workflows.workflows))
	for _, wf := range s.workflows.workflows {
		result = append(result, wf)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	s.workflows.mu.RLock()
	wf, ok := s.workflows.workflows[name]
	s.workflows.mu.RUnlock()

	if !ok {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var wf engine.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	wf.Name = name

	s.workflows.mu.Lock()
	s.workflows.workflows[name] = &wf
	s.workflows.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	s.workflows.mu.Lock()
	delete(s.workflows.workflows, name)
	s.workflows.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/dev/code/Upal && go test ./internal/api/ -v`
Expected: PASS (3 tests)

**Step 5: Commit**

```bash
git add internal/api/
git commit -m "feat(api): add REST API for workflow CRUD"
```

---

### Task 18: Wire Up Main Server

**Files:**
- Modify: `cmd/upal/main.go`

**Step 1: Update main.go**

Replace `cmd/upal/main.go`:
```go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/soochol/upal/internal/api"
	"github.com/soochol/upal/internal/engine"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serve()
		return
	}
	fmt.Println("upal v0.1.0")
	fmt.Println("Usage: upal serve")
}

func serve() {
	eventBus := engine.NewEventBus()
	sessions := engine.NewSessionManager()

	srv := api.NewServer(eventBus, sessions)

	addr := ":8080"
	slog.Info("starting upal server", "addr", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
```

**Step 2: Verify build and run**

Run: `cd /home/dev/code/Upal && go build -o bin/upal ./cmd/upal && ./bin/upal`
Expected: `upal v0.1.0` + usage message

**Step 3: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat: wire up main server with serve command"
```

---

## Phase 8: Frontend — Visual Workflow Editor

### Task 19: Set Up React Flow Canvas

**Files:**
- Create: `web/src/components/editor/Canvas.tsx`
- Create: `web/src/stores/workflowStore.ts`
- Modify: `web/src/App.tsx`

**Step 1: Create workflow store**

Create `web/src/stores/workflowStore.ts`:
```ts
import { create } from 'zustand'
import { Node, Edge, addEdge, applyNodeChanges, applyEdgeChanges, type OnNodesChange, type OnEdgesChange, type OnConnect } from '@xyflow/react'

export type NodeData = {
  label: string
  nodeType: 'input' | 'agent' | 'tool' | 'output'
  config: Record<string, unknown>
}

type WorkflowState = {
  nodes: Node<NodeData>[]
  edges: Edge[]
  onNodesChange: OnNodesChange<Node<NodeData>>
  onEdgesChange: OnEdgesChange
  onConnect: OnConnect
  addNode: (type: NodeData['nodeType'], position: { x: number; y: number }) => void
}

let nodeId = 0
const getId = () => `node_${++nodeId}`

export const useWorkflowStore = create<WorkflowState>((set, get) => ({
  nodes: [],
  edges: [],
  onNodesChange: (changes) => {
    set({ nodes: applyNodeChanges(changes, get().nodes) })
  },
  onEdgesChange: (changes) => {
    set({ edges: applyEdgeChanges(changes, get().edges) })
  },
  onConnect: (connection) => {
    set({ edges: addEdge(connection, get().edges) })
  },
  addNode: (type, position) => {
    const id = getId()
    const labels: Record<string, string> = {
      input: 'User Input',
      agent: 'Agent',
      tool: 'Tool',
      output: 'Output',
    }
    const newNode: Node<NodeData> = {
      id,
      type: 'upalNode',
      position,
      data: {
        label: labels[type] || type,
        nodeType: type,
        config: {},
      },
    }
    set({ nodes: [...get().nodes, newNode] })
  },
}))
```

**Step 2: Create Canvas component**

Create `web/src/components/editor/Canvas.tsx`:
```tsx
import { ReactFlow, Background, Controls, MiniMap } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useWorkflowStore } from '../../stores/workflowStore'
import { UpalNode } from './nodes/UpalNode'

const nodeTypes = {
  upalNode: UpalNode,
}

export function Canvas() {
  const { nodes, edges, onNodesChange, onEdgesChange, onConnect } = useWorkflowStore()

  return (
    <div className="h-full w-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        nodeTypes={nodeTypes}
        fitView
      >
        <Background color="#333" gap={20} />
        <Controls />
        <MiniMap />
      </ReactFlow>
    </div>
  )
}
```

**Step 3: Create UpalNode component**

Create `web/src/components/editor/nodes/UpalNode.tsx`:
```tsx
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import type { NodeData } from '../../../stores/workflowStore'

const colorMap: Record<string, string> = {
  input: 'border-yellow-500 bg-yellow-500/10',
  agent: 'border-blue-500 bg-blue-500/10',
  tool: 'border-red-500 bg-red-500/10',
  output: 'border-green-500 bg-green-500/10',
}

const iconMap: Record<string, string> = {
  input: '🟡',
  agent: '🔵',
  tool: '🔴',
  output: '🟢',
}

export function UpalNode({ data }: NodeProps<Node<NodeData>>) {
  return (
    <div className={`rounded-lg border-2 px-4 py-3 min-w-[200px] ${colorMap[data.nodeType] || 'border-zinc-600'}`}>
      <Handle type="target" position={Position.Top} className="!bg-zinc-400" />
      <div className="flex items-center gap-2">
        <span>{iconMap[data.nodeType]}</span>
        <span className="font-medium text-zinc-100">{data.label}</span>
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-zinc-400" />
    </div>
  )
}
```

**Step 4: Update App.tsx**

Replace `web/src/App.tsx`:
```tsx
import { Canvas } from './components/editor/Canvas'
import { useWorkflowStore } from './stores/workflowStore'

function App() {
  const addNode = useWorkflowStore((s) => s.addNode)

  const handleAddNode = (type: 'input' | 'agent' | 'tool' | 'output') => {
    addNode(type, { x: 250, y: useWorkflowStore.getState().nodes.length * 150 + 50 })
  }

  return (
    <div className="h-screen flex flex-col bg-zinc-950 text-zinc-100">
      {/* Header */}
      <header className="flex items-center justify-between px-4 py-2 border-b border-zinc-800">
        <h1 className="text-lg font-bold">Upal</h1>
        <div className="flex gap-2">
          <button className="px-3 py-1 bg-green-600 rounded text-sm hover:bg-green-700">
            ▶ Run
          </button>
        </div>
      </header>

      {/* Node Palette + Canvas */}
      <div className="flex flex-1 overflow-hidden">
        {/* Palette */}
        <aside className="w-48 border-r border-zinc-800 p-3 flex flex-col gap-2">
          <p className="text-xs text-zinc-500 uppercase font-medium">Add Step</p>
          {(['input', 'agent', 'tool', 'output'] as const).map((type) => (
            <button
              key={type}
              onClick={() => handleAddNode(type)}
              className="px-3 py-2 rounded border border-zinc-700 text-sm text-left hover:bg-zinc-800 capitalize"
            >
              {type === 'input' && '🟡 '}{type === 'agent' && '🔵 '}{type === 'tool' && '🔴 '}{type === 'output' && '🟢 '}{type}
            </button>
          ))}
        </aside>

        {/* Canvas */}
        <main className="flex-1">
          <Canvas />
        </main>
      </div>

      {/* Console */}
      <footer className="h-32 border-t border-zinc-800 p-3 overflow-y-auto">
        <p className="text-xs text-zinc-500 uppercase font-medium">Console</p>
        <p className="text-sm text-zinc-600 mt-1">Ready. Add nodes and connect them to build a workflow.</p>
      </footer>
    </div>
  )
}

export default App
```

**Step 5: Verify frontend builds**

Run: `cd /home/dev/code/Upal/web && npm run build`
Expected: Build succeeds

**Step 6: Commit**

```bash
git add web/src/
git commit -m "feat(web): add React Flow canvas with 4 node types and Opal-style layout"
```

---

### Task 20: Add Workflow Serialization (Canvas ↔ JSON)

**Files:**
- Create: `web/src/lib/serializer.ts`
- Modify: `web/src/stores/workflowStore.ts`

**Step 1: Create serializer**

Create `web/src/lib/serializer.ts`:
```ts
import type { Node, Edge } from '@xyflow/react'
import type { NodeData } from '../stores/workflowStore'

export type WorkflowDefinition = {
  name: string
  version: number
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
}

type WorkflowNode = {
  id: string
  type: string
  config: Record<string, unknown>
}

type WorkflowEdge = {
  from: string
  to: string
  loop?: { max_iterations: number; exit_when: string }
}

export function serializeWorkflow(
  name: string,
  nodes: Node<NodeData>[],
  edges: Edge[],
): WorkflowDefinition {
  return {
    name,
    version: 1,
    nodes: nodes.map((n) => ({
      id: n.id,
      type: n.data.nodeType,
      config: n.data.config,
    })),
    edges: edges.map((e) => ({
      from: e.source,
      to: e.target,
    })),
  }
}

export function deserializeWorkflow(
  wf: WorkflowDefinition,
): { nodes: Node<NodeData>[]; edges: Edge[] } {
  const labels: Record<string, string> = {
    input: 'User Input',
    agent: 'Agent',
    tool: 'Tool',
    output: 'Output',
  }

  const nodes: Node<NodeData>[] = wf.nodes.map((n, i) => ({
    id: n.id,
    type: 'upalNode',
    position: { x: 250, y: i * 150 + 50 },
    data: {
      label: labels[n.type] || n.type,
      nodeType: n.type as NodeData['nodeType'],
      config: n.config,
    },
  }))

  const edges: Edge[] = wf.edges.map((e, i) => ({
    id: `edge-${i}`,
    source: e.from,
    target: e.to,
  }))

  return { nodes, edges }
}
```

**Step 2: Verify build**

Run: `cd /home/dev/code/Upal/web && npm run build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add web/src/lib/serializer.ts
git commit -m "feat(web): add workflow serialization (Canvas ↔ JSON)"
```

---

## Phase 9: Integration & Docker

### Task 21: Embed Frontend in Go Binary

**Files:**
- Create: `internal/api/static.go`
- Modify: `cmd/upal/main.go`
- Modify: `Makefile`

**Step 1: Create static file handler**

Create `internal/api/static.go`:
```go
package api

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// StaticHandler serves the React frontend from a directory.
func StaticHandler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// SPA fallback: serve index.html for unmatched routes
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// StaticHandlerFS serves the React frontend from an embedded filesystem.
func StaticHandlerFS(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(fsys, r.URL.Path[1:]); os.IsNotExist(err) {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
```

**Step 2: Add static file serving to server**

Add to `internal/api/server.go` in the `Handler()` method, after the API routes:
```go
// Serve static files (frontend)
r.Handle("/*", StaticHandler("web/dist"))
```

**Step 3: Update Makefile**

```makefile
.PHONY: build run test dev build-frontend

build-frontend:
	cd web && npm run build

build: build-frontend
	go build -o bin/upal ./cmd/upal

run: build
	./bin/upal serve

test:
	go test ./... -v -race

test-frontend:
	cd web && npm test

dev-frontend:
	cd web && npm run dev

dev-backend:
	go run ./cmd/upal serve

dev:
	@echo "Run 'make dev-backend' and 'make dev-frontend' in separate terminals"
```

**Step 4: Verify full build**

Run: `cd /home/dev/code/Upal && make build`
Expected: Frontend builds, then Go binary builds

**Step 5: Commit**

```bash
git add internal/api/static.go Makefile
git commit -m "feat: embed frontend static files in Go server"
```

---

### Task 22: Create Dockerfile

**Files:**
- Create: `Dockerfile`
- Modify: `docker-compose.yaml`

**Step 1: Create Dockerfile**

```dockerfile
FROM node:22-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /upal ./cmd/upal

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend /upal /upal
COPY --from=backend /app/web/dist /web/dist
EXPOSE 8080
ENTRYPOINT ["/upal", "serve"]
```

**Step 2: Update docker-compose.yaml**

```yaml
services:
  upal:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DATABASE_URL: "postgres://upal:upal@postgres:5432/upal?sslmode=disable"
    volumes:
      - ./config.yaml:/config.yaml

  postgres:
    image: postgres:17
    environment:
      POSTGRES_USER: upal
      POSTGRES_PASSWORD: upal
      POSTGRES_DB: upal
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U upal"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

**Step 3: Commit**

```bash
git add Dockerfile docker-compose.yaml
git commit -m "feat: add Dockerfile and Docker Compose for full-stack deployment"
```

---

### Task 23: Run All Tests & Verify

**Step 1: Run all Go tests**

Run: `cd /home/dev/code/Upal && go test ./... -v -race`
Expected: All tests PASS

**Step 2: Run frontend build**

Run: `cd /home/dev/code/Upal/web && npm run build`
Expected: Build succeeds

**Step 3: Verify full build**

Run: `cd /home/dev/code/Upal && make build`
Expected: Binary created at `bin/upal`

**Step 4: Final commit**

```bash
git add -A
git commit -m "chore: verify all tests pass and build succeeds"
```

---

## Summary

| Phase | Tasks | What Gets Built |
|-------|-------|----------------|
| 1: Scaffolding | 1-4 | Go module, React project, Docker, Makefile |
| 2: Core Engine | 5-8 | Types, DAG Builder, Event Bus, Session Manager |
| 3: Model Provider | 9-11 | Provider interface, OpenAI driver, Registry |
| 4: Tool System | 12 | Tool interface, Registry |
| 5: Node Implementations | 13-15 | Input, Agent, Tool, Output nodes |
| 6: DAG Runner | 16 | Workflow orchestration with parallel execution |
| 7: HTTP API | 17-18 | REST API, Server wiring |
| 8: Frontend | 19-20 | React Flow canvas, Node components, Serialization |
| 9: Integration | 21-23 | Static embedding, Docker, Full test pass |

**Total: 23 tasks** — Each task is 5-15 minutes of implementation.

## Future Tasks (Post-MVP)

- PostgreSQL persistence (replace in-memory stores)
- WebSocket event streaming to frontend
- Anthropic/Google provider adapters
- Ollama/llama.cpp/vLLM provider adapters
- MCP client implementation
- Node inline editing UI
- Preview view
- Code view (YAML editor)
- Execution console with real-time events
- Natural language workflow generation
- Version history
- Workflow import/export
