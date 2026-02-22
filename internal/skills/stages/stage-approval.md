---
name: stage-approval
description: Guide for configuring approval stages — pause pipeline and wait for human approval
---

## "approval" stage — pause and wait for human approval

```json
"config": {
  "message":       "승인 요청 메시지",
  "connection_id": "",
  "timeout":       3600
}
```

### Fields

- `message`: the approval request message shown to the approver (Korean).
- `connection_id`: always set to `""` — the user configures the actual connection after pipeline creation.
- `timeout`: seconds to wait before auto-rejecting (default 3600 = 1 hour; use 86400 for 24 hours).

### Behavior

- The pipeline **pauses** at this stage until a human approves or rejects, or the timeout expires.
- If rejected or timed out, the pipeline stops.
- If approved, execution continues to the next stage.

### Rules

- Always set `connection_id` to `""`. Never invent a connection ID.
- Choose `timeout` based on urgency: 3600 (1h) for same-day, 86400 (24h) for next-day review.
- For non-blocking notifications that don't pause, use the `notification` stage instead.
