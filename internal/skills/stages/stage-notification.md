---
name: stage-notification
description: Guide for configuring notification stages — send a message and continue immediately
---

## "notification" stage — send a notification (does NOT pause)

```json
"config": {
  "connection_id": "",
  "message":       "알림 내용",
  "subject":       "제목 (optional, email only)"
}
```

### Fields

- `connection_id`: always set to `""` — the user configures the actual connection after pipeline creation.
- `message`: the notification body text (Korean).
- `subject`: optional, only meaningful for email connections.

### Output fields available to downstream stages

| Field | Contents |
|-------|---------|
| `{{sent}}` | Boolean — whether the notification was sent |
| `{{channel}}` | Channel name used for sending |

### Rules

- This stage does NOT pause the pipeline — execution continues immediately after sending.
- Always set `connection_id` to `""`. Never invent a connection ID.
- For approval/confirmation flows that should pause, use the `approval` stage instead.
