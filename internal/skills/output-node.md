---
name: output-node
description: Guide for configuring output nodes — display_mode, layout_prompt, layout_model
---

## Objective

Configure an output node that displays the final result of the workflow. The output node aggregates all upstream node results and optionally transforms them via LLM into a styled HTML layout.

## Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `label` | string | Yes | Short human-readable label for the node (e.g. `"Final Report"`, `"Dashboard"`) |
| `description` | string | Yes | Brief explanation of what this node does |
| `display_mode` | string | No | `"manual"` (default) or `"auto-layout"` |
| `prompt` | string | No | Template for selecting specific upstream outputs using `{{node_id}}` references. If omitted, all upstream results are concatenated. |
| `layout_prompt` | string | Conditional | User prompt for layout generation. Required when `display_mode` is `"manual"` and custom layout is desired. Uses `{{node_id}}` template references. |
| `layout_model` | string | No | Model to use for layout generation (e.g. `"anthropic/claude-sonnet-4-6"`). Falls back to default model if omitted. |

## Display Modes

### `"manual"` (default)
- Set `layout_prompt` to control how upstream data is presented.
- The `layout_prompt` is sent as the user prompt with a system prompt that instructs clean HTML generation.
- Use `{{node_id}}` references to pull specific upstream outputs into the layout.
- If `layout_prompt` is omitted, upstream results are simply concatenated as plain text.

**Example `layout_prompt`:**
```
Create an HTML report with the following sections:

## Research Findings
{{researcher}}

## Analysis
{{analyzer}}

Style it with clean typography and a professional color scheme.
```

### `"auto-layout"`
- The system automatically generates a styled HTML page from all upstream outputs.
- Uses **both system prompt and user prompt** — the system prompt instructs HTML/CSS best practices, the user prompt contains the aggregated content.
- No `layout_prompt` needed — the system constructs the prompt automatically.

## Rules

1. ALWAYS set `label`.
2. Set `display_mode` to `"auto-layout"` for rich visual output (reports, dashboards, presentations).
3. Use `"manual"` with `layout_prompt` when you need precise control over how upstream data is arranged.
4. In `layout_prompt`, ALWAYS use `{{node_id}}` template references — never hardcode placeholder text.
5. Omit `display_mode` entirely for simple plain-text concatenation of upstream results.
