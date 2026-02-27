# Unified GlobalChatBar — Design Document

**Date:** 2026-02-27
**Status:** Approved

## Overview

범용 AI 챗을 모든 페이지에서 사용할 수 있도록 GlobalChatBar를 재설계한다. 백엔드에 중앙 `/api/chat` 엔드포인트를 만들고, 페이지 컨텍스트와 함께 LLM이 tool call로 액션을 수행하는 MCP 스타일 아키텍처.

## Architecture

```
┌─ Frontend ──────────────────────────────────────────┐
│                                                      │
│  GlobalChatBar (플로팅, 드래그 가능)                   │
│       │                                              │
│  useRegisterChatContext (페이지별 등록)                │
│    ├─ page: 페이지 식별자                              │
│    ├─ context: 현재 선택된 엔티티 정보 (동적)           │
│    ├─ applyResult: tool 결과를 UI에 반영하는 콜백       │
│    └─ placeholder: 동적 안내문                         │
│                                                      │
└──────────────┬───────────────────────────────────────┘
               │ POST /api/chat (SSE stream)
               ▼
┌─ Backend ───────────────────────────────────────────┐
│                                                      │
│  ChatHandler                                         │
│    ├─ 시스템 프롬프트: skills("chat-{page}") + 컨텍스트 │
│    ├─ Tool set: page + context 조건으로 결정            │
│    ├─ LLM 호출 (tool call 루프, 최대 10턴)              │
│    └─ SSE 스트림으로 결과 반환                          │
│                                                      │
│  Chat Tools (기존 서비스 래핑)                          │
│    ├─ configure_node → generator.ConfigureNode        │
│    ├─ generate_workflow → generator.Generate           │
│    ├─ add_node / remove_node → workflow CRUD           │
│    ├─ configure_stage → pipeline service               │
│    └─ summarize_session → session service              │
│                                                      │
└──────────────────────────────────────────────────────┘
```

## Design Decisions

### History: 프론트엔드 관리

- 매 요청에 history 배열 전송, 서버는 stateless
- 페이지 이동 시 자연스럽게 리셋
- 대화가 길어지면 프론트엔드에서 최근 N개만 전송하여 제어
- 이유: 보안이 아닌 UI 상태 문제이므로 클라이언트가 관리하는 것이 구조적

### Available Tools: 백엔드 결정

- 프론트엔드는 `page` + `context`만 전송
- 백엔드가 page와 context 조건을 보고 사용 가능한 tool set을 결정
- 이유: tool 목록은 권한 경계이므로 서버가 결정해야 함

## API Contract

### Request

```
POST /api/chat
Content-Type: application/json
Accept: text/event-stream
```

```json
{
  "message": "이 노드에 웹 검색 툴 추가해줘",
  "page": "workflows",
  "context": {
    "workflow_id": "blog-post-writer",
    "selected_node_id": "research-agent",
    "selected_node": { "type": "agent", "config": {}, "label": "..." }
  },
  "history": [
    { "role": "user", "content": "이전 메시지" },
    { "role": "assistant", "content": "이전 응답" }
  ],
  "model": "",
  "thinking": false
}
```

### SSE Response

```
event: text_delta
data: {"chunk": "웹 검색 툴을 추가하겠"}

event: text_delta
data: {"chunk": "습니다."}

event: tool_call
data: {"id": "tc_1", "name": "configure_node", "args": {...}}

event: tool_result
data: {"id": "tc_1", "success": true, "result": {...}}

event: done
data: {"content": "웹 검색 툴을 추가했습니다."}
```

## Backend: Chat Tools

### Tool Resolution (page + context → tool set)

| Page | Context 조건 | Tools |
|---|---|---|
| `workflows` | 기본 | `generate_workflow`, `add_node`, `remove_node`, `list_nodes` |
| `workflows` | + `selected_node_id` | + `configure_node`, `describe_node` |
| `pipelines` | 기본 | `generate_pipeline`, `list_stages` |
| `pipelines` | + `selected_stage_id` | + `configure_stage`, `add_stage`, `remove_stage` |
| `sessions` | 기본 | `list_sessions`, `summarize_session` |
| `sessions` | + `selected_session_id` | + `analyze_session` |
| `runs` | 기본 | `list_runs`, `explain_run_error` |

### Chat Tool Interface

```go
type ChatTool struct {
    Name        string
    Description string
    Schema      map[string]any
    Execute     func(ctx context.Context, args map[string]any) (any, error)
}
```

기존 서비스 레이어(generator, workflowService, pipelineService)를 직접 호출하는 어댑터 구조. 새 비즈니스 로직을 만들지 않는다.

### System Prompt Construction

```
skills("chat-{page}")          ← 페이지별 skill 파일
+ 모델 카탈로그                  ← 기존 패턴 재활용
+ resolved tool descriptions   ← 활성화된 tool의 name + description
+ context JSON                 ← 프론트엔드가 보낸 현재 상태
```

### Tool Call Loop

기존 `generateWithSkills`와 동일한 패턴. 최대 10턴. SSE로 각 단계를 실시간 스트리밍.

## Frontend: PageContext + ChatBar

### useRegisterChatContext

`useRegisterChatHandler`를 대체하는 새 훅:

```tsx
useRegisterChatContext({
  page: 'workflows',
  context: {
    workflow_id: workflowId,
    selected_node_id: selectedNodeId,
    selected_node: selectedNode,
  },
  applyResult: (toolName, result) => {
    if (toolName === 'configure_node') updateNodeConfig(result.node_id, result.config)
    if (toolName === 'add_node') addNode(result.node)
  },
  placeholder: selectedNodeId ? 'Ask about this node...' : 'Ask about this workflow...',
})
```

### ChatBar UI

- 플로팅 위치, 드래그 가능 (localStorage에 페이지별 저장)
- 기본 위치: 하단 중앙
- 입력바 항상 표시, 메시지 영역은 확장/축소
- tool_call/tool_result 이벤트를 메시지 영역에 표시
- handler 미등록 페이지에서는 숨김

## Implementation Phases

### Phase 1 — 기반 (Workflows)

- 백엔드: `/api/chat` SSE 엔드포인트 + tool call 루프
- 백엔드: ChatTool 인터페이스 + `configure_node`, `generate_workflow`, `add_node`, `remove_node`
- 백엔드: page→tool 매핑 resolver
- 백엔드: `chat-workflows` skill
- 프론트엔드: `useRegisterChatContext` 훅
- 프론트엔드: GlobalChatBar SSE + tool_call/tool_result 표시
- 프론트엔드: 드래그 가능한 플로팅 위치
- 프론트엔드: Workflows 페이지 context 등록
- CanvasPromptBar 제거 (워크플로우 생성/수정도 ChatBar로 통합)
- 기존 `/api/nodes/configure` API는 유지 (ChatTool이 내부적으로 호출)

### Phase 2 — Pipelines

- `chat-pipelines` skill
- `configure_stage`, `add_stage`, `generate_pipeline` Chat Tools
- Pipelines 페이지 context 등록

### Phase 3 — Sessions + Runs

- `chat-sessions`, `chat-runs` skills
- `summarize_session`, `analyze_session`, `explain_run_error` Chat Tools
- 각 페이지 context 등록

### Phase 4 — 고도화

- 스트리밍 텍스트 응답 (text_delta 청크별 렌더링)
- thinking 모드 지원
- tool call 실행 중 취소 기능
