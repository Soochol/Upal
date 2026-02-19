# Full A2A Protocol Integration Design

**Date**: 2026-02-19
**Status**: Approved

## Overview

Upal의 모든 Agent 노드를 Google A2A (Agent-to-Agent) 프로토콜 기반 서버로 전환하고, DAG Runner를 A2A 클라이언트로 변환하여 노드 간 통신을 전부 JSON-RPC over HTTP로 통일한다. A2A의 Artifact 시스템을 도입하여 구조화된 데이터 타입 전달 문제를 해결한다.

## Decision Record

- **A2A 통신**: Full A2A (모든 노드가 A2A 서버, HTTP JSON-RPC 통신)
- **데이터 모델**: A2A Artifact 기반 (MIME type + Parts)
- **Transport**: 모두 HTTP (in-process 최적화 없음, 순수 A2A 준수)
- **테스트**: A2A 도입 후 통합 테스트

## Architecture

### 전환 구조

```
DAG Runner (A2A Client)
    ↓ a2a.sendMessage (JSON-RPC over HTTP)
Agent Node (A2A Server)  ←→  External A2A Agent
    ↓ Agent Card로 능력 발견
Provider.ChatCompletion()
    ↓
Artifact (MIME type + Parts)으로 결과 반환
    ↓ Task 라이프사이클 관리
다음 노드로 Artifact 전달
```

### 핵심 변경 사항

1. **DAG Runner** → A2A Client: 각 노드에 `a2a.sendMessage`로 태스크를 보내고 Task 상태 추적
2. **모든 Agent 노드** → A2A Server: Agent Card 보유, JSON-RPC 엔드포인트에서 요청 수신
3. **세션 상태** `map[string]any` → `map[string][]Artifact`: 노드 출력이 A2A Artifact로 표준화
4. **Input/Output/Tool 노드**도 A2A 래핑: 일관된 프로토콜 사용
5. **내부/외부 통신 모두 HTTP**: 순수 A2A 프로토콜 준수

## Data Model

### A2A Core Types

```go
type Part struct {
    Type     string `json:"type"`               // "text", "file", "data"
    Text     string `json:"text,omitempty"`
    MimeType string `json:"mimeType,omitempty"`
    Data     any    `json:"data,omitempty"`
    File     *File  `json:"file,omitempty"`
}

type Artifact struct {
    Name        string            `json:"name,omitempty"`
    Description string            `json:"description,omitempty"`
    Parts       []Part            `json:"parts"`
    Index       int               `json:"index"`
    Metadata    map[string]string `json:"metadata,omitempty"`
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
    Role  string `json:"role"`  // "user" or "agent"
    Parts []Part `json:"parts"`
}

type AgentCard struct {
    Name                string       `json:"name"`
    Description         string       `json:"description"`
    URL                 string       `json:"url"`
    Capabilities        Capabilities `json:"capabilities"`
    Skills              []Skill      `json:"skills"`
    DefaultInputModes   []string     `json:"defaultInputModes"`
    DefaultOutputModes  []string     `json:"defaultOutputModes"`
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

### 노드 타입별 Artifact 출력

| 노드 타입 | Artifact Parts |
|-----------|----------------|
| **input** | `[{type: "text", text: "사용자 입력값"}]` |
| **agent** (텍스트) | `[{type: "text", text: "LLM 응답", mimeType: "text/plain"}]` |
| **agent** (JSON) | `[{type: "data", data: {...}, mimeType: "application/json"}]` |
| **tool** | `[{type: "data", data: {...}, mimeType: "application/json"}]` or file |
| **output** | 모든 이전 노드의 Artifact 집계 |

### 템플릿 해석 변경

```
현재:  {{node_id}} → fmt.Sprintf("%v", state["node_id"])
A2A:   {{node_id}} → 해당 노드 Artifact의 첫 번째 text Part 내용
       {{node_id.data}} → JSON 직렬화된 data Part
       {{node_id.parts[0]}} → 특정 파트 접근
```

## Node Execution Flow

### Agent Card (per node)

```json
{
  "name": "research-agent",
  "description": "Research expert agent",
  "url": "http://localhost:8080/a2a/nodes/research",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false
  },
  "skills": [
    {
      "id": "research",
      "name": "Research Topic",
      "description": "Researches a given topic using LLM",
      "inputModes": ["text/plain"],
      "outputModes": ["text/plain", "application/json"]
    }
  ],
  "defaultInputModes": ["text/plain"],
  "defaultOutputModes": ["text/plain"]
}
```

### DAG Runner A2A Client Flow

```
For each node in topological order:

1. Wait for parent nodes' Task completion (status == "completed")

2. Collect data from parent Artifacts
   → Template resolution ({{parent_id}} → parent Artifact text)

3. JSON-RPC call: a2a.sendMessage
   POST http://localhost:8080/a2a/nodes/{node_id}
   {
     "jsonrpc": "2.0",
     "method": "a2a.sendMessage",
     "params": {
       "message": {
         "role": "user",
         "parts": [{"type": "text", "text": "resolved prompt"}]
       },
       "configuration": {
         "acceptedOutputModes": ["text/plain", "application/json"]
       }
     }
   }

4. Track Task state: created → working → completed (or failed)

5. On completion: receive Artifacts
   → session.Artifacts[node_id] = task.Artifacts

6. Publish events (EventBus → SSE → frontend)
```

### Agent Node Server Processing

```
On a2a.sendMessage:

1. Create Task (status: "created")
2. Extract Parts from Message
3. Transition to status: "working"
4. LLM call (existing Provider interface)
   - system_prompt + user message from Parts
   - If tools: agentic loop
5. Package response as Artifact
6. Transition to status: "completed" + return Artifact
7. JSON-RPC Response with Task object
```

### HTTP Endpoint Structure

```
Existing API (unchanged):
  /api/workflows/...            CRUD workflows
  /api/workflows/{name}/run     Execute workflow (SSE stream)

A2A endpoints (new):
  /a2a/nodes/{node_id}          Per-node A2A JSON-RPC endpoint
  /a2a/agent-card               Upal aggregate Agent Card
  /a2a/nodes/{node_id}/agent-card  Per-node Agent Card
```

## Error Handling

```
Task failure:
1. Node A2A server → status: "failed" + error message
2. DAG Runner →
   - Cancel downstream node Tasks (a2a.cancelTask)
   - Publish node.error event to EventBus
   - Continue independent parallel nodes

Task timeout:
- Per-node timeout in Agent Card
- DAG Runner manages via context.WithTimeout

Retry policy:
- Configurable retry count (default: 0)
- Exponential backoff
```

## Testing Plan

### Unit Tests
- A2A message serialization/deserialization
- Agent Card creation and discovery
- Artifact ↔ Part conversion
- Template resolution (Artifact-based)

### Integration Tests (Mock LLM)
- 2-node serial: input → agent → output
- 3-node parallel: input → [agent_a, agent_b] → output
- Error propagation: input → agent(fail) → output(cancel)
- Task lifecycle state transitions
- JSON-RPC request/response validation

### E2E Tests (Real LLM, optional)
- Simple pipeline: "topic input → research → summary"
- SSE streaming event verification
- Frontend integration

## Migration Strategy

### Phase 1: A2A Core Types
- `internal/a2a/types.go` — Task, Artifact, Part, Message, AgentCard
- `internal/a2a/server.go` — JSON-RPC handler
- `internal/a2a/client.go` — JSON-RPC client

### Phase 2: Node Executor A2A Wrapping
- Wrap each NodeExecutor as A2A Server
- Auto-generate Agent Cards from node definitions

### Phase 3: DAG Runner A2A Client Conversion
- `runner.go`: direct calls → HTTP A2A calls
- Session state: `map[string]any` → `map[string][]Artifact`

### Phase 4: External A2A Integration
- Add external agent nodes via Agent Card URL
- Publish Upal's aggregate Agent Card

### Phase 5: Integration Testing
- Execute test plan above
- Update frontend for Artifact display

## References

- [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/)
- [A2A GitHub Repository](https://github.com/a2aproject/A2A)
- [Google A2A Announcement](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/)
