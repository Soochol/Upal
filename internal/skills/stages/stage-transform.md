---
name: stage-transform
description: Guide for configuring transform stages — apply an expression to reshape data
---

## "transform" stage — reshape data from a previous stage

```json
"config": {
  "expression": "jq-style expression or template string"
}
```

### Fields

- `expression`: a jq-style expression or template string that transforms the input data.

### Output fields available to downstream stages

| Field | Contents |
|-------|---------|
| `{{output}}` | The transformed result |

### When to use

- When upstream stage output needs reformatting before passing to the next stage.
- Extracting specific fields from structured data.
- Converting between data formats.

### Rules

- Use `{{output}}` in downstream `input_mapping` to reference the transformed result.
- Keep expressions simple and deterministic — no side effects.
