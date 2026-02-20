---
name: input-node
description: Guide for configuring input nodes — user data entry points
---

## Objective

Configure an input node that collects data from the user before workflow execution.

## Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `label` | string | Yes | Short human-readable label (e.g. `"Article URL"`, `"User Question"`) |
| `placeholder` | string | Yes | Hint text shown in the input field when empty |
| `description` | string | Yes | Brief explanation of what this input collects |

## Rules

1. The `placeholder` should clearly communicate what kind of input is expected.
   - BAD: "Enter text here"
   - GOOD: "Paste the article URL you want to analyze..."
   - GOOD: "Describe the product feature you want documented..."
2. Make the placeholder specific to the workflow context — it should help the user understand exactly what data this node needs.
3. If the input expects a particular format (URL, JSON, code snippet), mention it in the placeholder.
