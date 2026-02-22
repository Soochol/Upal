---
name: tool-node
description: Guide for configuring tool nodes — direct tool execution without LLM
---

## Objective

Configure a tool node that executes a registered tool directly — no LLM call, no token cost. Use this for deterministic transformations where the inputs are fully known from upstream nodes.

## Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `label` | string | Yes | Short human-readable label (e.g. `"TTS 제작"`, `"영상 편집"`) |
| `description` | string | Yes | Brief explanation of what this node does |
| `tool` | string | Yes | Registered tool name (e.g. `"tts"`, `"shell_exec"`, `"http_request"`) |
| `input` | object | No | Key-value parameters passed to the tool. Values support `{{node_id}}` template references. |

## Input Template Syntax

String values in `input` support `{{node_id}}` to inject outputs from upstream nodes:

```json
"input": {
  "text": "{{llm_analysis}}",
  "voice": "Rachel",
  "output_path": "/tmp/narration.mp3"
}
```

At runtime, `{{llm_analysis}}` is replaced with the output stored in `session.State["llm_analysis"]`.

## When to use tool vs agent

| Use `tool` node | Use `agent` node |
|-----------------|------------------|
| Inputs fully determined by upstream nodes | LLM needs to reason about what to call |
| One specific tool, always called | LLM may call tools conditionally |
| No reasoning needed | Content generation, analysis, decisions |

## Rules

1. `tool` MUST match a name from the "Available tools" list. Never invent tool names.
2. `input` keys must match the tool's expected parameter names.
3. Only reference upstream nodes in `{{node_id}}` — never downstream nodes.
4. The tool's output is stored in session state under this node's `id` and can be referenced as `{{this_node_id}}` by downstream nodes.
