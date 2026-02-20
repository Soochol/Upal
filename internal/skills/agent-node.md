---
name: agent-node
description: Guide for configuring agent nodes — system_prompt, user prompt, model
---

## Objective

Configure an agent node that calls an AI model. You MUST fill ALL relevant fields comprehensively — do not leave fields empty when you can infer reasonable values.

## Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | Yes | Format: `"provider/model-name"` (e.g. `"anthropic/claude-sonnet-4-6"`) |
| `system_prompt` | string | Yes | Expert persona — see SYSTEM PROMPT FRAMEWORK below |
| `prompt` | string | Yes | User message template — see USER PROMPT FRAMEWORK below |
| `label` | string | Yes | Short human-readable label for the node (e.g. `"Summarizer"`, `"Code Reviewer"`) |
| `description` | string | Yes | Brief explanation of what this node does |
| `output` | string | Yes | Output format instruction appended to system_prompt (e.g. `"Respond in JSON with keys: title, summary, tags"`) |
| `tools` | array of strings | No | Tool names to enable for agentic tool-use loop (e.g. `["web_search", "python_exec"]`). Only use tools from the available tools list. |

{{include system-prompt}}

{{include prompt-framework}}

## Rules

1. ALWAYS set `label`, `model`, `system_prompt`, `prompt`, and `output`. Choose an appropriate model if the user doesn't specify one.
2. The `system_prompt` MUST follow the SYSTEM PROMPT FRAMEWORK above — generic or shallow prompts are not acceptable.
3. The `prompt` MUST follow the USER PROMPT FRAMEWORK above — always use `{{node_id}}` template references for upstream data.
4. For `model`, use the default model listed in "Available models" above unless the user specifies otherwise or a lighter model is clearly appropriate. Match model tier to task complexity.
5. For `output`, if the node feeds into another agent (mid-pipeline), prepend "You are working as part of an AI system, so no chit-chat and no explaining what you're doing and why." to enforce clean, parseable output. For user-facing final nodes, use natural tone.
