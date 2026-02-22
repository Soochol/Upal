---
name: stage-schedule
description: Guide for configuring schedule stages — trigger pipeline on a cron schedule
---

## "schedule" stage — trigger on a cron schedule

```json
"config": {
  "cron":     "0 9 * * *",
  "timezone": "Asia/Seoul"
}
```

### Fields

- `cron`: standard 5-field cron expression (minute hour day-of-month month day-of-week).
- `timezone`: IANA timezone string.

### Common cron examples

| Expression | Meaning |
|-----------|---------|
| `"0 9 * * *"` | Every day at 9:00 AM |
| `"0 9 * * 1"` | Every Monday at 9:00 AM |
| `"0 */6 * * *"` | Every 6 hours |
| `"0 9 1 * *"` | First day of every month at 9:00 AM |
| `"*/30 * * * *"` | Every 30 minutes |

### Common timezones

- `"Asia/Seoul"` — KST (UTC+9)
- `"America/New_York"` — EST/EDT
- `"UTC"` — Universal

### Rules

- Schedule stages are typically the first stage in a pipeline (no `depends_on`).
- This stage produces no meaningful output fields for downstream stages.
- Use `trigger` instead if the pipeline should start on an external webhook event rather than a time schedule.
