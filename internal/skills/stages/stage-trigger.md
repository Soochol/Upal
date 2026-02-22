---
name: stage-trigger
description: Guide for configuring trigger stages — wait for an external webhook event
---

## "trigger" stage — wait for an external webhook event

```json
"config": {
  "trigger_id": ""
}
```

### Fields

- `trigger_id`: always set to `""` — the user configures the trigger source after pipeline creation.

### Behavior

- The pipeline waits for an external system to send a webhook event before proceeding.
- This stage produces no meaningful output fields for downstream stages.

### Rules

- Always set `trigger_id` to `""`. Never invent a trigger ID.
- Trigger stages are typically the first stage in a pipeline (no `depends_on`).
- Use `schedule` instead if the pipeline should start on a time-based cron schedule.
