---
name: tool-content_store
description: Persistent key-value store — survives across pipeline runs
---

## Overview

A persistent key-value store backed by a local JSON file on the server. Unlike session state (`{{node_id}}`), which is ephemeral and scoped to a single run, `content_store` values survive indefinitely across pipeline executions until explicitly deleted.

**Primary use cases**: deduplication (tracking seen item IDs/URLs), last-run timestamps for incremental collection, run counters, and any cross-run state.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | Yes | Operation to perform: `get`, `set`, `list`, `delete` |
| `key` | string | For get/set/delete | Key to operate on |
| `value` | string | For `set` | String value to store. **Values must be strings** — serialize objects as JSON. |
| `prefix` | string | For `list` | Filter keys by prefix. Omit to list all keys. |

## Returns by Action

**`get`** — `{ "value": string | null, "found": boolean }`
- `found: false` when the key does not exist (never throws an error)
- `value` is `null` when not found

**`set`** — `{ "status": "ok" }`

**`list`** — `{ "keys": string[], "count": number }`
- Keys sorted alphabetically
- Use `prefix` to narrow scope; listing all keys without a prefix is expensive in large stores

**`delete`** — `{ "status": "ok" }`

## Key Naming Conventions

Use namespaced, hyphen-separated keys to prevent collisions between pipelines:

```
{pipeline-name}:{purpose}
```

Examples:
- `"news-monitor:last-run"` — ISO 8601 timestamp of last successful run
- `"news-monitor:seen-article-links"` — JSON array of processed URLs
- `"github-digest:seen-issue-ids"` — JSON array of processed issue IDs
- `"report-counter:total"` — integer stored as string

## Agent Prompt Patterns

**Last-run timestamp (incremental fetch):**
```
1. Call content_store: action="get", key="{{pipeline_name}}:last-run"
2. If result.found is false, set since_date to 7 days ago in ISO 8601 (e.g. "2026-02-15T00:00:00Z").
   Otherwise, use result.value as since_date.
3. [perform fetch with since_date]
4. After processing, call content_store: action="set", key="{{pipeline_name}}:last-run",
   value=[current UTC time as ISO 8601 string, e.g. "2026-02-22T10:30:00Z"]
```

**Seen-items deduplication (JSON array):**
```
1. Call content_store: action="get", key="{{pipeline_name}}:seen-ids"
2. Parse result.value as JSON array. If result.found is false, use an empty array [].
3. Filter incoming items: keep only those whose ID is NOT in the seen array.
4. Process the new items.
5. Build updated seen list = old seen list + new IDs (keep last 1000 to prevent unbounded growth).
6. Call content_store: action="set", key="{{pipeline_name}}:seen-ids",
   value=[JSON.stringify of the updated array]
```

**Counter:**
```
1. Call content_store: action="get", key="{{pipeline_name}}:count"
2. Parse as integer. Default to 0 if not found.
3. Add the number of processed items.
4. Call content_store: action="set", key="{{pipeline_name}}:count", value=[new total as string]
```

**Reset / cleanup:**
```
To reset state for a fresh run:
Call content_store: action="delete", key="{{pipeline_name}}:last-run"
Call content_store: action="delete", key="{{pipeline_name}}:seen-ids"
```

## Pitfalls & Limitations

- **Values must be strings** — numbers, arrays, and objects must be serialized. Use JSON string format: `"[\"id1\",\"id2\"]"`. Always parse on read, serialize on write.
- **No TTL or auto-expiry** — values persist forever. Design explicit cleanup logic for stores that can grow unbounded (e.g. cap seen-IDs list at 1000 entries).
- **Shared global store** — all agents across all workflows share the same store. Namespacing keys is not optional — it is required to prevent silent data corruption.
- **No transactions** — two concurrent pipeline runs may race on the same key. Not suitable for high-concurrency workloads.
- **`list` without prefix** — returns all keys from all pipelines. Always filter with a pipeline-specific prefix.
- **String comparison for JSON arrays** — the stored value is an opaque string. The LLM must treat it as JSON text and parse/stringify explicitly.
