# Go ADK Full Adoption — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace all engine/nodes/provider/a2a code with Google ADK Go SDK, unifying runner paths and enabling multimodal support.

**Architecture:** DAGAgent (ADK Custom Agent) orchestrates workflow execution via topological sort. Each node becomes an ADK Agent (LLMAgent, Custom, or RemoteAgent). All data flows through ADK session.State as genai.Content. SSE streams ADK Events directly.

**Tech Stack:** Go 1.24+, google.golang.org/adk, google.golang.org/genai, github.com/a2aproject/a2a-go

---

### Task 1: Upgrade Go version and add ADK dependency

**Files:**
- Modify: `go.mod`

**Step 1: Upgrade Go toolchain to 1.24+**

```bash
go mod edit -go=1.24
```

**Step 2: Add ADK dependency**

```bash
go get google.golang.org/adk@latest
go get google.golang.org/genai@latest
```

**Step 3: Verify it compiles**

```bash
go build ./...
```

Expected: SUCCESS (existing code unchanged, new deps available)

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: upgrade to Go 1.24 and add ADK dependency"
```

---

### Task 2: Create foundation types package

**Files:**
- Create: `internal/upal/types.go`

**Step 1: Write test**

```go
// internal/upal/types_test.go
package upal

import (
	"encoding/json"
	"testing"
)

func TestWorkflowDefinitionJSON(t *testing.T) {
	wf := WorkflowDefinition{
		Name:    "test-wf",
		Version: 1,
		Nodes: []NodeDefinition{
			{ID: "input1", Type: NodeTypeInput, Config: map[string]any{}},
			{ID: "agent1", Type: NodeTypeAgent, Config: map[string]any{"model": "anthropic/claude-sonnet-4-20250514"}},
		},
		Edges: []EdgeDefinition{{From: "input1", To: "agent1"}},
	}
	data, err := json.Marshal(wf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got WorkflowDefinition
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != wf.Name || len(got.Nodes) != 2 || len(got.Edges) != 1 {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/upal/... -v -race -run TestWorkflowDefinitionJSON
```

Expected: FAIL (package doesn't exist)

**Step 3: Write implementation**

```go
// internal/upal/types.go
package upal

type NodeType string

const (
	NodeTypeInput    NodeType = "input"
	NodeTypeAgent    NodeType = "agent"
	NodeTypeTool     NodeType = "tool"
	NodeTypeOutput   NodeType = "output"
	NodeTypeExternal NodeType = "external"
)

type WorkflowDefinition struct {
	Name    string            `json:"name" yaml:"name"`
	Version int               `json:"version" yaml:"version"`
	Nodes   []NodeDefinition  `json:"nodes" yaml:"nodes"`
	Edges   []EdgeDefinition  `json:"edges" yaml:"edges"`
	Groups  []GroupDefinition `json:"groups,omitempty" yaml:"groups,omitempty"`
}

type NodeDefinition struct {
	ID     string         `json:"id" yaml:"id"`
	Type   NodeType       `json:"type" yaml:"type"`
	Config map[string]any `json:"config" yaml:"config"`
	Group  string         `json:"group,omitempty" yaml:"group,omitempty"`
}

type EdgeDefinition struct {
	From string      `json:"from" yaml:"from"`
	To   string      `json:"to" yaml:"to"`
	Loop *LoopConfig `json:"loop,omitempty" yaml:"loop,omitempty"`
}

type LoopConfig struct {
	MaxIterations int    `json:"max_iterations" yaml:"max_iterations"`
	ExitWhen      string `json:"exit_when" yaml:"exit_when"`
}

type GroupDefinition struct {
	ID    string `json:"id" yaml:"id"`
	Label string `json:"label" yaml:"label"`
	Color string `json:"color,omitempty" yaml:"color,omitempty"`
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/upal/... -v -race
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/upal/
git commit -m "feat: add upal types package (WorkflowDef, NodeDef)"
```

---

### Task 3: Move DAG to standalone package

**Files:**
- Create: `internal/dag/dag.go`
- Create: `internal/dag/dag_test.go`

**Step 1: Write test**

```go
// internal/dag/dag_test.go
package dag

import (
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestBuildDAG(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: "input"},
			{ID: "b", Type: "agent"},
			{ID: "c", Type: "output"},
		},
		Edges: []upal.EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}
	d, err := Build(wf)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	order := d.TopologicalOrder()
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(order))
	}
	// a must come before b, b before c
	idx := map[string]int{}
	for i, id := range order {
		idx[id] = i
	}
	if idx["a"] >= idx["b"] || idx["b"] >= idx["c"] {
		t.Fatalf("wrong order: %v", order)
	}
}

func TestBuildDAGCycleDetection(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Nodes: []upal.NodeDefinition{
			{ID: "a"}, {ID: "b"},
		},
		Edges: []upal.EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
		},
	}
	_, err := Build(wf)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/dag/... -v -race
```

Expected: FAIL

**Step 3: Write implementation**

Copy `internal/engine/dag.go` to `internal/dag/dag.go`, change package to `dag`, update types to use `upal.NodeDefinition`/`upal.EdgeDefinition`, rename `BuildDAG` → `Build`.

```go
// internal/dag/dag.go
package dag

import (
	"fmt"
	"sort"

	"github.com/soochol/upal/internal/upal"
)

type DAG struct {
	nodes     map[string]*upal.NodeDefinition
	children  map[string][]string
	parents   map[string][]string
	edges     map[string]upal.EdgeDefinition
	backEdges []upal.EdgeDefinition
	topoOrder []string
}

func Build(wf *upal.WorkflowDefinition) (*DAG, error) {
	d := &DAG{
		nodes:    make(map[string]*upal.NodeDefinition),
		children: make(map[string][]string),
		parents:  make(map[string][]string),
		edges:    make(map[string]upal.EdgeDefinition),
	}
	for i := range wf.Nodes {
		n := &wf.Nodes[i]
		if _, exists := d.nodes[n.ID]; exists {
			return nil, fmt.Errorf("duplicate node ID: %s", n.ID)
		}
		d.nodes[n.ID] = n
	}
	for _, e := range wf.Edges {
		if _, ok := d.nodes[e.From]; !ok {
			return nil, fmt.Errorf("edge references unknown node: %s", e.From)
		}
		if _, ok := d.nodes[e.To]; !ok {
			return nil, fmt.Errorf("edge references unknown node: %s", e.To)
		}
		key := e.From + "->" + e.To
		d.edges[key] = e
		if e.Loop != nil {
			d.backEdges = append(d.backEdges, e)
			continue
		}
		d.children[e.From] = append(d.children[e.From], e.To)
		d.parents[e.To] = append(d.parents[e.To], e.From)
	}
	order, err := d.topoSort()
	if err != nil {
		return nil, err
	}
	d.topoOrder = order
	return d, nil
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
	sort.Strings(queue)
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
		sort.Strings(queue)
	}
	if len(order) != len(d.nodes) {
		return nil, fmt.Errorf("cycle detected in workflow graph (excluding back-edges)")
	}
	return order, nil
}

func (d *DAG) TopologicalOrder() []string              { return d.topoOrder }
func (d *DAG) Children(nodeID string) []string          { return d.children[nodeID] }
func (d *DAG) Parents(nodeID string) []string           { return d.parents[nodeID] }
func (d *DAG) Node(id string) *upal.NodeDefinition     { return d.nodes[id] }
func (d *DAG) Edge(from, to string) (upal.EdgeDefinition, bool) {
	e, ok := d.edges[from+"->"+to]
	return e, ok
}
func (d *DAG) BackEdges() []upal.EdgeDefinition { return d.backEdges }
func (d *DAG) Roots() []string {
	var roots []string
	for id := range d.nodes {
		if len(d.parents[id]) == 0 {
			roots = append(roots, id)
		}
	}
	return roots
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/dag/... -v -race
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/dag/
git commit -m "feat: extract DAG package from engine"
```

---

### Task 4: Implement model.LLM for Anthropic

**Files:**
- Create: `internal/model/anthropic.go`
- Create: `internal/model/anthropic_test.go`

**Step 1: Write test**

```go
// internal/model/anthropic_test.go
package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestAnthropicLLM_Name(t *testing.T) {
	llm := NewAnthropicLLM("test-key")
	if llm.Name() != "anthropic" {
		t.Fatalf("expected 'anthropic', got %q", llm.Name())
	}
}

func TestAnthropicLLM_GenerateContent(t *testing.T) {
	// Mock Anthropic API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing API key header")
		}
		resp := map[string]any{
			"content":    []map[string]any{{"type": "text", "text": "Hello from Claude"}},
			"stop_reason": "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-key", WithAnthropicBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "claude-sonnet-4-20250514",
		Contents: []*genai.Content{
			genai.NewContentFromText("Hello", genai.RoleUser),
		},
	}

	var got *adkmodel.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		got = resp
	}
	if got == nil || got.Content == nil {
		t.Fatal("expected non-nil response")
	}
	text := got.Content.Parts[0].Text
	if text != "Hello from Claude" {
		t.Fatalf("expected 'Hello from Claude', got %q", text)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/model/... -v -race -run TestAnthropic
```

Expected: FAIL

**Step 3: Write implementation**

```go
// internal/model/anthropic.go
package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

type AnthropicLLM struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

type AnthropicOption func(*AnthropicLLM)

func WithAnthropicBaseURL(url string) AnthropicOption {
	return func(a *AnthropicLLM) { a.baseURL = url }
}

func NewAnthropicLLM(apiKey string, opts ...AnthropicOption) *AnthropicLLM {
	a := &AnthropicLLM{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com",
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *AnthropicLLM) Name() string { return "anthropic" }

func (a *AnthropicLLM) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		body := a.buildRequestBody(req)
		jsonData, err := json.Marshal(body)
		if err != nil {
			yield(nil, fmt.Errorf("marshal request: %w", err))
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/v1/messages", bytes.NewReader(jsonData))
		if err != nil {
			yield(nil, fmt.Errorf("create request: %w", err))
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", a.apiKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")

		resp, err := a.client.Do(httpReq)
		if err != nil {
			yield(nil, fmt.Errorf("do request: %w", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			yield(nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(respBody)))
			return
		}

		var apiResp anthropicAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			yield(nil, fmt.Errorf("decode response: %w", err))
			return
		}

		content := a.toGenaiContent(&apiResp)
		yield(&adkmodel.LLMResponse{
			Content:      content,
			TurnComplete: true,
		}, nil)
	}
}

func (a *AnthropicLLM) buildRequestBody(req *adkmodel.LLMRequest) map[string]any {
	var systemPrompt string
	var messages []map[string]any

	for _, c := range req.Contents {
		if c.Role == "system" {
			for _, p := range c.Parts {
				systemPrompt += p.Text
			}
			continue
		}

		role := "user"
		if c.Role == genai.RoleModel {
			role = "assistant"
		}

		var contentBlocks []map[string]any
		for _, p := range c.Parts {
			if p.Text != "" {
				contentBlocks = append(contentBlocks, map[string]any{"type": "text", "text": p.Text})
			}
			if p.FunctionCall != nil {
				contentBlocks = append(contentBlocks, map[string]any{
					"type":  "tool_use",
					"id":    p.FunctionCall.ID,
					"name":  p.FunctionCall.Name,
					"input": p.FunctionCall.Args,
				})
			}
			if p.FunctionResponse != nil {
				contentBlocks = append(contentBlocks, map[string]any{
					"type":        "tool_result",
					"tool_use_id": p.FunctionResponse.ID,
					"content":     fmt.Sprintf("%v", p.FunctionResponse.Response),
				})
			}
		}
		if len(contentBlocks) == 1 && contentBlocks[0]["type"] == "text" {
			messages = append(messages, map[string]any{"role": role, "content": contentBlocks[0]["text"]})
		} else if len(contentBlocks) > 0 {
			messages = append(messages, map[string]any{"role": role, "content": contentBlocks})
		}
	}

	body := map[string]any{
		"model":      req.Model,
		"messages":   messages,
		"max_tokens": 4096,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}
	if req.Config != nil {
		if req.Config.Temperature != nil {
			body["temperature"] = *req.Config.Temperature
		}
		if req.Config.MaxOutputTokens != nil {
			body["max_tokens"] = *req.Config.MaxOutputTokens
		}
	}

	// Tool definitions from Config
	if req.Config != nil && len(req.Config.Tools) > 0 {
		var tools []map[string]any
		for _, t := range req.Config.Tools {
			if t.FunctionDeclarations != nil {
				for _, fd := range t.FunctionDeclarations {
					tools = append(tools, map[string]any{
						"name":         fd.Name,
						"description":  fd.Description,
						"input_schema": fd.Parameters,
					})
				}
			}
		}
		if len(tools) > 0 {
			body["tools"] = tools
		}
	}

	return body
}

func (a *AnthropicLLM) toGenaiContent(resp *anthropicAPIResponse) *genai.Content {
	content := &genai.Content{Role: genai.RoleModel}
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			content.Parts = append(content.Parts, genai.NewPartFromText(block.Text))
		case "tool_use":
			argsMap, _ := block.Input.(map[string]any)
			content.Parts = append(content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   block.ID,
					Name: block.Name,
					Args: argsMap,
				},
			})
		}
	}
	return content
}

type anthropicAPIResponse struct {
	Content    []anthropicBlock `json:"content"`
	StopReason string           `json:"stop_reason"`
}

type anthropicBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/model/... -v -race -run TestAnthropic
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/model/anthropic.go internal/model/anthropic_test.go
git commit -m "feat: implement model.LLM for Anthropic"
```

---

### Task 5: Implement model.LLM for OpenAI

**Files:**
- Create: `internal/model/openai.go`
- Create: `internal/model/openai_test.go`

**Step 1: Write test**

```go
// internal/model/openai_test.go
package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestOpenAILLM_Name(t *testing.T) {
	llm := NewOpenAILLM("test-key")
	if llm.Name() != "openai" {
		t.Fatalf("expected 'openai', got %q", llm.Name())
	}
}

func TestOpenAILLM_GenerateContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{{
				"message":       map[string]any{"role": "assistant", "content": "Hello from GPT"},
				"finish_reason": "stop",
			}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("test-key", WithOpenAIBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "gpt-4o",
		Contents: []*genai.Content{
			genai.NewContentFromText("Hello", genai.RoleUser),
		},
	}

	var got *adkmodel.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		got = resp
	}
	if got == nil || got.Content == nil {
		t.Fatal("expected non-nil response")
	}
	text := got.Content.Parts[0].Text
	if text != "Hello from GPT" {
		t.Fatalf("expected 'Hello from GPT', got %q", text)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/model/... -v -race -run TestOpenAI
```

Expected: FAIL

**Step 3: Write implementation**

```go
// internal/model/openai.go
package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

type OpenAILLM struct {
	name    string
	apiKey  string
	baseURL string
	client  *http.Client
}

type OpenAIOption func(*OpenAILLM)

func WithOpenAIBaseURL(url string) OpenAIOption {
	return func(o *OpenAILLM) { o.baseURL = url }
}

func WithOpenAIName(name string) OpenAIOption {
	return func(o *OpenAILLM) { o.name = name }
}

func NewOpenAILLM(apiKey string, opts ...OpenAIOption) *OpenAILLM {
	o := &OpenAILLM{
		name:    "openai",
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *OpenAILLM) Name() string { return o.name }

func (o *OpenAILLM) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		body := o.buildRequestBody(req)
		jsonData, err := json.Marshal(body)
		if err != nil {
			yield(nil, fmt.Errorf("marshal request: %w", err))
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/chat/completions", bytes.NewReader(jsonData))
		if err != nil {
			yield(nil, fmt.Errorf("create request: %w", err))
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if o.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
		}

		resp, err := o.client.Do(httpReq)
		if err != nil {
			yield(nil, fmt.Errorf("do request: %w", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			yield(nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(respBody)))
			return
		}

		var apiResp openAIAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			yield(nil, fmt.Errorf("decode response: %w", err))
			return
		}

		if len(apiResp.Choices) == 0 {
			yield(nil, fmt.Errorf("no choices in response"))
			return
		}

		content := o.toGenaiContent(&apiResp.Choices[0])
		yield(&adkmodel.LLMResponse{
			Content:      content,
			TurnComplete: true,
		}, nil)
	}
}

func (o *OpenAILLM) buildRequestBody(req *adkmodel.LLMRequest) map[string]any {
	var messages []map[string]any
	for _, c := range req.Contents {
		role := string(c.Role)
		if c.Role == genai.RoleModel {
			role = "assistant"
		}
		for _, p := range c.Parts {
			if p.Text != "" {
				messages = append(messages, map[string]any{"role": role, "content": p.Text})
			}
			if p.FunctionCall != nil {
				argsJSON, _ := json.Marshal(p.FunctionCall.Args)
				messages = append(messages, map[string]any{
					"role": "assistant",
					"tool_calls": []map[string]any{{
						"id": p.FunctionCall.ID, "type": "function",
						"function": map[string]any{"name": p.FunctionCall.Name, "arguments": string(argsJSON)},
					}},
				})
			}
			if p.FunctionResponse != nil {
				respJSON, _ := json.Marshal(p.FunctionResponse.Response)
				messages = append(messages, map[string]any{
					"role": "tool", "tool_call_id": p.FunctionResponse.ID, "content": string(respJSON),
				})
			}
		}
	}

	body := map[string]any{"model": req.Model, "messages": messages, "stream": false}
	if req.Config != nil {
		if req.Config.Temperature != nil {
			body["temperature"] = *req.Config.Temperature
		}
		if req.Config.MaxOutputTokens != nil {
			body["max_tokens"] = *req.Config.MaxOutputTokens
		}
		if len(req.Config.Tools) > 0 {
			var tools []map[string]any
			for _, t := range req.Config.Tools {
				if t.FunctionDeclarations != nil {
					for _, fd := range t.FunctionDeclarations {
						tools = append(tools, map[string]any{
							"type": "function",
							"function": map[string]any{
								"name": fd.Name, "description": fd.Description, "parameters": fd.Parameters,
							},
						})
					}
				}
			}
			if len(tools) > 0 {
				body["tools"] = tools
			}
		}
	}
	return body
}

func (o *OpenAILLM) toGenaiContent(choice *openAIAPIChoice) *genai.Content {
	content := &genai.Content{Role: genai.RoleModel}
	if choice.Message.Content != "" {
		content.Parts = append(content.Parts, genai.NewPartFromText(choice.Message.Content))
	}
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		content.Parts = append(content.Parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				ID: tc.ID, Name: tc.Function.Name, Args: args,
			},
		})
	}
	return content
}

type openAIAPIResponse struct {
	Choices []openAIAPIChoice `json:"choices"`
}
type openAIAPIChoice struct {
	Message      openAIAPIMessage `json:"message"`
	FinishReason string           `json:"finish_reason"`
}
type openAIAPIMessage struct {
	Role      string              `json:"role"`
	Content   string              `json:"content"`
	ToolCalls []openAIAPIToolCall `json:"tool_calls,omitempty"`
}
type openAIAPIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
```

**Step 4: Run tests**

```bash
go test ./internal/model/... -v -race
```

Expected: PASS (both Anthropic and OpenAI tests)

**Step 5: Commit**

```bash
git add internal/model/openai.go internal/model/openai_test.go
git commit -m "feat: implement model.LLM for OpenAI"
```

---

### Task 6: Create ADK tool adapter

**Files:**
- Create: `internal/agents/tooladapter.go`
- Create: `internal/agents/tooladapter_test.go`

Bridge `internal/tools.Tool` → ADK `tool.Tool` so existing Upal tools work with LLMAgent.

**Step 1: Write test**

```go
// internal/agents/tooladapter_test.go
package agents

import (
	"context"
	"testing"
)

type mockTool struct{}

func (m *mockTool) Name() string                                  { return "test_tool" }
func (m *mockTool) Description() string                           { return "A test tool" }
func (m *mockTool) InputSchema() map[string]any                   { return map[string]any{"type": "string"} }
func (m *mockTool) Execute(ctx context.Context, input any) (any, error) { return "result", nil }

func TestADKToolAdapter(t *testing.T) {
	adapter := NewADKTool(&mockTool{})
	if adapter.Name() != "test_tool" {
		t.Fatalf("expected 'test_tool', got %q", adapter.Name())
	}
	if adapter.Description() != "A test tool" {
		t.Fatalf("expected 'A test tool', got %q", adapter.Description())
	}
	if adapter.IsLongRunning() {
		t.Fatal("expected not long running")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/agents/... -v -race -run TestADKToolAdapter
```

Expected: FAIL

**Step 3: Write implementation**

```go
// internal/agents/tooladapter.go
package agents

import (
	"github.com/soochol/upal/internal/tools"
	adktool "google.golang.org/adk/tool"
)

// ADKTool wraps a Upal tools.Tool to satisfy ADK's tool.Tool interface.
type ADKTool struct {
	inner tools.Tool
}

func NewADKTool(t tools.Tool) *ADKTool {
	return &ADKTool{inner: t}
}

func (a *ADKTool) Name() string        { return a.inner.Name() }
func (a *ADKTool) Description() string { return a.inner.Description() }
func (a *ADKTool) IsLongRunning() bool { return false }

// AdaptTools converts a slice of Upal tools to ADK tools.
func AdaptTools(upalTools []tools.Tool) []adktool.Tool {
	result := make([]adktool.Tool, len(upalTools))
	for i, t := range upalTools {
		result[i] = NewADKTool(t)
	}
	return result
}
```

**Step 4: Run tests**

```bash
go test ./internal/agents/... -v -race
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/agents/
git commit -m "feat: add ADK tool adapter for Upal tools"
```

---

### Task 7: Implement agent builders

**Files:**
- Create: `internal/agents/builders.go`
- Create: `internal/agents/builders_test.go`

Factory functions that create ADK Agents from WorkflowDefinition NodeDefinitions.

**Step 1: Write test**

```go
// internal/agents/builders_test.go
package agents

import (
	"testing"

	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
)

func TestBuildAgent_Input(t *testing.T) {
	nd := &upal.NodeDefinition{ID: "input1", Type: upal.NodeTypeInput}
	a, err := BuildAgent(nd, nil, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if a.Name() != "input1" {
		t.Fatalf("expected name 'input1', got %q", a.Name())
	}
}

func TestBuildAgent_Output(t *testing.T) {
	nd := &upal.NodeDefinition{ID: "output1", Type: upal.NodeTypeOutput}
	a, err := BuildAgent(nd, nil, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if a.Name() != "output1" {
		t.Fatalf("expected name 'output1', got %q", a.Name())
	}
}

func TestBuildAgent_UnknownType(t *testing.T) {
	nd := &upal.NodeDefinition{ID: "x", Type: "unknown"}
	_, err := BuildAgent(nd, nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/agents/... -v -race -run TestBuildAgent
```

Expected: FAIL

**Step 3: Write implementation**

```go
// internal/agents/builders.go
package agents

import (
	"fmt"
	"iter"
	"regexp"
	"sort"
	"strings"

	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/remoteagent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// BuildAgent creates an ADK Agent from a NodeDefinition.
// llms maps provider name → model.LLM. toolReg is the tool registry.
func BuildAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM, toolReg *tools.Registry) (agent.Agent, error) {
	switch nd.Type {
	case upal.NodeTypeInput:
		return buildInputAgent(nd)
	case upal.NodeTypeOutput:
		return buildOutputAgent(nd)
	case upal.NodeTypeTool:
		return buildToolAgent(nd, toolReg)
	case upal.NodeTypeAgent:
		return buildLLMAgent(nd, llms, toolReg)
	case upal.NodeTypeExternal:
		return buildRemoteAgent(nd)
	default:
		return nil, fmt.Errorf("unknown node type %q for node %q", nd.Type, nd.ID)
	}
}

func buildInputAgent(nd *upal.NodeDefinition) (agent.Agent, error) {
	return agent.New(agent.Config{
		Name:        nd.ID,
		Description: "Reads user input from session state",
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				key := "__user_input__" + nd.ID
				val, err := ctx.Session().State().Get(key)
				if err != nil || val == nil {
					yield(nil, fmt.Errorf("no user input for node %q (expected state key %q)", nd.ID, key))
					return
				}
				// Store the input value under the node's ID
				ctx.Session().State().Set(nd.ID, val)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nd.ID
				text := fmt.Sprintf("%v", val)
				event.LLMResponse.Content = genai.NewContentFromText(text, genai.RoleModel)
				yield(event, nil)
			}
		},
	})
}

func buildOutputAgent(nd *upal.NodeDefinition) (agent.Agent, error) {
	return agent.New(agent.Config{
		Name:        nd.ID,
		Description: "Collects all node outputs into final result",
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				var keys []string
				for k, _ := range ctx.Session().State().All() {
					if !strings.HasPrefix(k, "__") {
						keys = append(keys, k)
					}
				}
				sort.Strings(keys)

				var parts []string
				for _, k := range keys {
					v, _ := ctx.Session().State().Get(k)
					if s, ok := v.(string); ok && s != "" {
						parts = append(parts, s)
					}
				}

				result := strings.Join(parts, "\n\n")
				ctx.Session().State().Set(nd.ID, result)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nd.ID
				event.LLMResponse.Content = genai.NewContentFromText(result, genai.RoleModel)
				yield(event, nil)
			}
		},
	})
}

func buildToolAgent(nd *upal.NodeDefinition, toolReg *tools.Registry) (agent.Agent, error) {
	toolName, _ := nd.Config["tool"].(string)
	if toolName == "" {
		return nil, fmt.Errorf("tool node %q: tool name is required", nd.ID)
	}
	return agent.New(agent.Config{
		Name:        nd.ID,
		Description: fmt.Sprintf("Executes tool %q", toolName),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				inputTemplate, _ := nd.Config["input"].(string)
				input := resolveTemplateFromState(inputTemplate, ctx.Session().State())

				result, err := toolReg.Execute(ctx, toolName, input)
				if err != nil {
					yield(nil, fmt.Errorf("tool node %q: %w", nd.ID, err))
					return
				}

				resultStr := fmt.Sprintf("%v", result)
				ctx.Session().State().Set(nd.ID, resultStr)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nd.ID
				event.LLMResponse.Content = genai.NewContentFromText(resultStr, genai.RoleModel)
				yield(event, nil)
			}
		},
	})
}

func buildLLMAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM, toolReg *tools.Registry) (agent.Agent, error) {
	modelID, _ := nd.Config["model"].(string)
	if modelID == "" {
		return nil, fmt.Errorf("agent node %q: model is required", nd.ID)
	}

	parts := strings.SplitN(modelID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("agent node %q: invalid model ID %q (expected provider/model)", nd.ID, modelID)
	}
	providerName, modelName := parts[0], parts[1]
	llm, ok := llms[providerName]
	if !ok {
		return nil, fmt.Errorf("agent node %q: unknown provider %q", nd.ID, providerName)
	}

	instruction, _ := nd.Config["system_prompt"].(string)

	cfg := llmagent.Config{
		Name:        nd.ID,
		Description: fmt.Sprintf("LLM agent using %s", modelID),
		Model:       llm,
		Instruction: instruction,
		OutputKey:   nd.ID,
	}

	// Resolve model name override (ADK uses LLMRequest.Model)
	cfg.GenerateContentConfig = &genai.GenerateContentConfig{}
	_ = modelName // model name is passed via LLMRequest by ADK runner

	// Add tools if configured
	if toolNames, ok := nd.Config["tools"].([]any); ok && toolReg != nil {
		for _, tn := range toolNames {
			name, _ := tn.(string)
			if t, found := toolReg.Get(name); found {
				cfg.Tools = append(cfg.Tools, NewADKTool(t))
			}
		}
	}

	return llmagent.New(cfg)
}

func buildRemoteAgent(nd *upal.NodeDefinition) (agent.Agent, error) {
	url, _ := nd.Config["endpoint_url"].(string)
	if url == "" {
		return nil, fmt.Errorf("external node %q: endpoint_url is required", nd.ID)
	}
	return remoteagent.NewA2A(remoteagent.A2AConfig{
		Name:            nd.ID,
		Description:     fmt.Sprintf("Remote A2A agent at %s", url),
		AgentCardSource: url,
	})
}

var templatePattern = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

func resolveTemplateFromState(template string, state session.State) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.Trim(match, "{}")
		val, err := state.Get(key)
		if err != nil || val == nil {
			return match
		}
		return fmt.Sprintf("%v", val)
	})
}
```

**Step 4: Run tests**

```bash
go test ./internal/agents/... -v -race
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/agents/
git commit -m "feat: add agent builders for all node types"
```

---

### Task 8: Implement DAGAgent

**Files:**
- Create: `internal/agents/dag.go`
- Create: `internal/agents/dag_test.go`

**Step 1: Write test**

```go
// internal/agents/dag_test.go
package agents

import (
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestNewDAGAgent(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "test-wf",
		Nodes: []upal.NodeDefinition{
			{ID: "input1", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "output1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "input1", To: "output1"}},
	}
	a, err := NewDAGAgent(wf, nil, nil)
	if err != nil {
		t.Fatalf("new dag agent: %v", err)
	}
	if a.Name() != "test-wf" {
		t.Fatalf("expected 'test-wf', got %q", a.Name())
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/agents/... -v -race -run TestNewDAGAgent
```

Expected: FAIL

**Step 3: Write implementation**

```go
// internal/agents/dag.go
package agents

import (
	"fmt"
	"iter"
	"sync"

	"github.com/soochol/upal/internal/dag"
	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
)

// NewDAGAgent creates an ADK Custom Agent that executes a workflow DAG.
func NewDAGAgent(wf *upal.WorkflowDefinition, llms map[string]adkmodel.LLM, toolReg *tools.Registry) (agent.Agent, error) {
	// Pre-build DAG and agents at creation time
	d, err := dag.Build(wf)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	agents := make(map[string]agent.Agent)
	for i := range wf.Nodes {
		nd := &wf.Nodes[i]
		a, err := BuildAgent(nd, llms, toolReg)
		if err != nil {
			return nil, fmt.Errorf("build agent for node %q: %w", nd.ID, err)
		}
		agents[nd.ID] = a
	}

	return agent.New(agent.Config{
		Name:        wf.Name,
		Description: fmt.Sprintf("Workflow %q with %d nodes", wf.Name, len(wf.Nodes)),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				done := make(map[string]chan struct{})
				for _, n := range wf.Nodes {
					done[n.ID] = make(chan struct{})
				}

				var wg sync.WaitGroup
				var execErr error
				var errOnce sync.Once

				for _, nodeID := range d.TopologicalOrder() {
					nodeID := nodeID
					wg.Add(1)
					go func() {
						defer wg.Done()
						// Wait for all parents to complete
						for _, parentID := range d.Parents(nodeID) {
							select {
							case <-done[parentID]:
							case <-ctx.Done():
								return
							}
						}

						// Check for prior errors
						if execErr != nil {
							close(done[nodeID])
							return
						}

						// Run the node's agent
						nodeAgent := agents[nodeID]
						for event, err := range nodeAgent.Run(ctx) {
							if err != nil {
								errOnce.Do(func() { execErr = fmt.Errorf("node %q: %w", nodeID, err) })
								close(done[nodeID])
								return
							}
							if !yield(event, nil) {
								close(done[nodeID])
								return
							}
						}
						close(done[nodeID])
					}()
				}

				wg.Wait()
				if execErr != nil {
					yield(nil, execErr)
				}
			}
		},
	})
}
```

**Step 4: Run tests**

```bash
go test ./internal/agents/... -v -race
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/agents/dag.go internal/agents/dag_test.go
git commit -m "feat: implement DAGAgent for workflow execution"
```

---

### Task 9: Update API layer

**Files:**
- Modify: `internal/api/server.go`
- Modify: `internal/api/run.go`

**Step 1: Rewrite server.go**

Remove old engine/nodes/provider dependencies. New Server struct:

```go
// internal/api/server.go — key changes
type Server struct {
	workflows            *WorkflowStore
	llms                 map[string]model.LLM
	sessionService       session.Service
	toolReg              *tools.Registry
	generator            *generate.Generator
	defaultGenerateModel string
	storage              storage.Storage
	db                   *db.DB
	providerConfigs      map[string]config.ProviderConfig
}

func NewServer(llms map[string]model.LLM, sessionService session.Service, toolReg *tools.Registry) *Server {
	return &Server{
		workflows:      NewWorkflowStore(),
		llms:           llms,
		sessionService: sessionService,
		toolReg:        toolReg,
	}
}
```

Remove: eventBus, sessions, runner, a2aRunner, executors fields.
Remove: aggregateAgentCard, handleA2ANode, handleA2ANodeCard handlers.
Remove: `/a2a` route group.
Update: Handler() to remove a2a routes.
Update: imports to remove engine, a2a, a2atypes, nodes packages.

**Step 2: Rewrite run.go**

```go
// internal/api/run.go — complete rewrite
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
		json.NewDecoder(r.Body).Decode(&req)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Build DAGAgent from workflow definition
	dagAgent, err := agents.NewDAGAgent(wf, s.llms, s.toolReg)
	if err != nil {
		writeSSEError(w, flusher, err)
		return
	}

	// Create ADK Runner
	adkRunner, err := runner.New(runner.Config{
		AppName:        wf.Name,
		Agent:          dagAgent,
		SessionService: s.sessionService,
	})
	if err != nil {
		writeSSEError(w, flusher, err)
		return
	}

	// Build user content with inputs
	userContent := genai.NewContentFromText("run", genai.RoleUser)
	// Set inputs in session state via initial state
	// ... (inputs are set via session state before run)

	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	userID := "default"

	// Pre-populate session with user inputs
	sess, err := s.sessionService.Create(r.Context(), &session.CreateRequest{
		AppName:   wf.Name,
		UserID:    userID,
		SessionID: sessionID,
		State:     prefixInputs(req.Inputs),
	})
	if err != nil {
		writeSSEError(w, flusher, err)
		return
	}
	_ = sess

	// Run and stream ADK events directly
	for event, err := range adkRunner.Run(r.Context(), userID, sessionID, userContent, agent.RunConfig{}) {
		if err != nil {
			writeSSEError(w, flusher, err)
			return
		}
		if event == nil {
			continue
		}
		jsonData, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	// Send done event
	fmt.Fprintf(w, "event: done\ndata: {\"status\":\"completed\",\"session_id\":%q}\n\n", sessionID)
	flusher.Flush()
}

func prefixInputs(inputs map[string]any) map[string]any {
	state := make(map[string]any)
	for k, v := range inputs {
		state["__user_input__"+k] = v
	}
	return state
}

func writeSSEError(w http.ResponseWriter, flusher http.Flusher, err error) {
	data, _ := json.Marshal(map[string]string{"status": "failed", "error": err.Error()})
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
	flusher.Flush()
}
```

**Step 3: Verify compilation**

```bash
go build ./internal/api/...
```

Note: This will fail until old code is deleted and main.go is updated (Task 11). That's expected — we'll wire everything together in Task 11.

**Step 4: Commit**

```bash
git add internal/api/server.go internal/api/run.go
git commit -m "feat: rewrite API layer for ADK Runner"
```

---

### Task 10: Update generate package

**Files:**
- Modify: `internal/generate/generate.go`

The generate package currently depends on `provider.Registry` and `provider.ChatRequest`. Replace with `model.LLM` and `genai.Content`.

```go
// internal/generate/generate.go — key changes
type Generator struct {
	llm   model.LLM
	model string // e.g. "claude-sonnet-4-20250514"
}

func New(llm model.LLM, modelName string) *Generator {
	return &Generator{llm: llm, model: modelName}
}

func (g *Generator) Generate(ctx context.Context, description string, existingWorkflow *upal.WorkflowDefinition) (*upal.WorkflowDefinition, error) {
	// Build genai.Content messages
	contents := []*genai.Content{
		{Role: "system", Parts: []*genai.Part{genai.NewPartFromText(sysPrompt)}},
		{Role: genai.RoleUser, Parts: []*genai.Part{genai.NewPartFromText(userContent)}},
	}
	req := &model.LLMRequest{Model: g.model, Contents: contents}

	var resp *model.LLMResponse
	for r, err := range g.llm.GenerateContent(ctx, req, false) {
		if err != nil { return nil, err }
		resp = r
	}
	// Parse resp.Content.Parts[0].Text as JSON → WorkflowDefinition
}
```

**Commit:**

```bash
git add internal/generate/generate.go
git commit -m "refactor: update generate package to use model.LLM"
```

---

### Task 11: Delete old code and rewrite main.go

**Files:**
- Delete: `internal/engine/` (entire directory)
- Delete: `internal/nodes/` (entire directory)
- Delete: `internal/provider/` (entire directory)
- Delete: `internal/a2atypes/` (entire directory)
- Delete: `internal/a2a/` (entire directory)
- Delete: `internal/a2aclient/` (entire directory)
- Rewrite: `cmd/upal/main.go`

**Step 1: Delete old packages**

```bash
rm -rf internal/engine/ internal/nodes/ internal/provider/ internal/a2atypes/ internal/a2a/ internal/a2aclient/
```

**Step 2: Rewrite main.go**

```go
// cmd/upal/main.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/soochol/upal/internal/api"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/generate"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/storage"
	"github.com/soochol/upal/internal/tools"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/session"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serve()
		return
	}
	fmt.Println("upal v0.2.0")
	fmt.Println("Usage: upal serve")
}

func serve() {
	cfg, err := config.LoadDefault()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	// Build model.LLM instances
	llms := make(map[string]adkmodel.LLM)
	var defaultLLM adkmodel.LLM
	var defaultModelName string

	for name, pc := range cfg.Providers {
		switch pc.Type {
		case "anthropic":
			llms[name] = upalmodel.NewAnthropicLLM(pc.APIKey)
			if defaultLLM == nil {
				defaultLLM = llms[name]
				defaultModelName = "claude-sonnet-4-20250514"
			}
		case "gemini":
			g, err := gemini.NewModel(context.Background(), "gemini-2.0-flash", nil)
			if err != nil {
				slog.Warn("gemini init failed", "err", err)
				continue
			}
			llms[name] = g
			if defaultLLM == nil {
				defaultLLM = g
				defaultModelName = "gemini-2.0-flash"
			}
		default:
			llms[name] = upalmodel.NewOpenAILLM(pc.APIKey, upalmodel.WithOpenAIBaseURL(pc.URL), upalmodel.WithOpenAIName(name))
			if defaultLLM == nil {
				defaultLLM = llms[name]
				defaultModelName = "gpt-4o"
			}
		}
	}

	toolReg := tools.NewRegistry()
	sessionService := session.InMemoryService()

	srv := api.NewServer(llms, sessionService, toolReg)

	// Optional: PostgreSQL
	if cfg.Database.URL != "" {
		database, err := db.New(context.Background(), cfg.Database.URL)
		if err != nil {
			slog.Warn("database unavailable, using in-memory storage", "err", err)
		} else {
			defer database.Close()
			if err := database.Migrate(context.Background()); err != nil {
				slog.Error("database migration failed", "err", err)
				os.Exit(1)
			}
			srv.SetDB(database)
			slog.Info("database connected")
		}
	}

	// Configure generator
	if defaultLLM != nil {
		gen := generate.New(defaultLLM, defaultModelName)
		srv.SetGenerator(gen, defaultModelName)
	}
	srv.SetProviderConfigs(cfg.Providers)

	// File storage
	store, err := storage.NewLocalStorage("./uploads")
	if err != nil {
		slog.Error("storage error", "err", err)
		os.Exit(1)
	}
	srv.SetStorage(store)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	slog.Info("starting upal server", "addr", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
```

**Step 3: Fix compilation errors**

```bash
go build ./...
```

Iteratively fix any remaining import issues, type mismatches, or missing methods.

**Step 4: Run all tests**

```bash
go test ./... -v -race
```

Expected: All new tests pass, old tests deleted with old code.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: complete ADK adoption — delete old engine/nodes/provider, rewrite main.go"
```

---

### Task 12: Update frontend SSE parsing

**Files:**
- Modify: `web/src/lib/api.ts`
- Modify: `web/src/stores/workflowStore.ts`

**Step 1: Update api.ts RunEvent type and parsing**

ADK events come as JSON lines (no SSE `event:` prefix). Update the SSE parser to handle ADK `session.Event` format:

```typescript
// web/src/lib/api.ts — update RunEvent type
export type ADKEventPart = {
  text?: string
  functionCall?: { id: string; name: string; args: Record<string, unknown> }
  functionResponse?: { id: string; name: string; response: unknown }
}

export type RunEvent = {
  type: string      // kept for backward compat in onEvent callback
  data: {
    id?: string
    author?: string
    invocation_id?: string
    content?: { role: string; parts: ADKEventPart[] }
    actions?: { state_delta: Record<string, unknown> }
    [key: string]: unknown
  }
}
```

The SSE parsing loop itself remains the same (it already handles `data:` lines). The `event: done` format is preserved from the server side.

**Step 2: Update workflowStore.ts event handling**

Map `event.data.author` → nodeId for status updates. Map `event.data.content.parts[0].text` → node result display.

**Step 3: Verify frontend builds**

```bash
cd web && npm run build
```

Expected: SUCCESS

**Step 4: Commit**

```bash
git add web/src/lib/api.ts web/src/stores/workflowStore.ts
git commit -m "feat: update frontend to consume ADK events"
```

---

### Task 13: Integration test

**Files:**
- Create: `internal/agents/integration_test.go`

**Step 1: Write end-to-end test**

```go
// internal/agents/integration_test.go
package agents_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/agents"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

func TestDAGAgent_EndToEnd(t *testing.T) {
	// Mock LLM server
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content":     []map[string]any{{"type": "text", "text": "Generated response"}},
			"stop_reason": "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	llms := map[string]adkmodel.LLM{
		"test": upalmodel.NewAnthropicLLM("key", upalmodel.WithAnthropicBaseURL(mockLLM.URL)),
	}

	wf := &upal.WorkflowDefinition{
		Name: "test-workflow",
		Nodes: []upal.NodeDefinition{
			{ID: "input1", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "agent1", Type: upal.NodeTypeAgent, Config: map[string]any{
				"model":         "test/claude-sonnet-4-20250514",
				"system_prompt": "You are helpful",
				"prompt":        "Process: {{input1}}",
			}},
			{ID: "output1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "input1", To: "agent1"},
			{From: "agent1", To: "output1"},
		},
	}

	dagAgent, err := agents.NewDAGAgent(wf, llms, nil)
	if err != nil {
		t.Fatalf("new dag agent: %v", err)
	}

	sessionSvc := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:        "test-workflow",
		Agent:          dagAgent,
		SessionService: sessionSvc,
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	// Create session with user input
	_, err = sessionSvc.Create(context.Background(), &session.CreateRequest{
		AppName:   "test-workflow",
		UserID:    "user1",
		SessionID: "sess1",
		State:     map[string]any{"__user_input__input1": "Hello world"},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	userMsg := genai.NewContentFromText("run", genai.RoleUser)
	var events int
	for event, err := range r.Run(context.Background(), "user1", "sess1", userMsg, agent.RunConfig{}) {
		if err != nil {
			t.Fatalf("run error: %v", err)
		}
		if event != nil {
			events++
			t.Logf("event: author=%s", event.Author)
		}
	}
	if events == 0 {
		t.Fatal("expected at least one event")
	}
}
```

**Step 2: Run integration test**

```bash
go test ./internal/agents/... -v -race -run TestDAGAgent_EndToEnd
```

Expected: PASS

**Step 3: Commit**

```bash
git add internal/agents/integration_test.go
git commit -m "test: add DAGAgent end-to-end integration test"
```

---

### Task 14: Full build and manual verification

**Step 1: Build backend**

```bash
go build -o bin/upal ./cmd/upal
```

Expected: SUCCESS

**Step 2: Build frontend**

```bash
cd web && npm run build
```

Expected: SUCCESS

**Step 3: Run all tests**

```bash
go test ./... -v -race
```

Expected: All PASS

**Step 4: Manual smoke test**

```bash
./bin/upal serve
```

Open browser, create a simple input→agent→output workflow, run it. Verify SSE events stream correctly.

**Step 5: Final commit**

```bash
git add -A
git commit -m "chore: ADK adoption complete — full build verified"
```
