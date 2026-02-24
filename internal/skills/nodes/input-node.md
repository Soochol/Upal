---
name: input-node
description: Guide for configuring input nodes — user data entry points
---

## Objective

Configure an input node that collects data from the user before workflow execution.

## Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `label` | string | Yes | Short human-readable label (e.g. `"기사 URL"`, `"사용자 질문"`) |
| `description` | string | Yes | Brief explanation of what this input collects |
| `prompt` | string | Yes | Guiding text shown as placeholder in the input field when the user runs the workflow. Tells the user what to type |

## Rules

1. The `prompt` should clearly communicate what kind of input is expected.
   - BAD: "여기에 텍스트를 입력하세요"
   - GOOD: "분석할 기사의 URL을 붙여넣으세요..."
   - GOOD: "문서화할 제품 기능을 설명하세요..."
2. Make the prompt specific to the workflow context — it should help the user understand exactly what data this node needs.
3. If the input expects a particular format (URL, JSON, code snippet), mention it in the prompt.
