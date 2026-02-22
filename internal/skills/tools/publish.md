---
name: tool-publish
description: Publish content to external destinations — markdown_file or webhook
---

## Overview

Delivers final content to external destinations. Use as the last step in content pipelines when the output needs to be persisted or sent to an external system.

**Channels:**
- `markdown_file` — saves content as a Markdown file on the server with a YAML frontmatter header
- `webhook` — POSTs content as JSON to any HTTP endpoint (Slack, Discord, Zapier, custom APIs)

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `channel` | string | Yes | `"markdown_file"` or `"webhook"` |
| `content` | string | Yes | Content body to publish |
| `title` | string | No | Title — used as the file slug for `markdown_file`, sent as `title` field for `webhook` |
| `metadata` | object | No | Channel-specific options (see below) |

### `metadata` per channel

| Channel | Key | Type | Description |
|---------|-----|------|-------------|
| `markdown_file` | any key | string | Extra fields added to YAML frontmatter (e.g. `"tags"`, `"category"`, `"source"`) |
| `webhook` | `webhook_url` | string | **Required.** Destination URL for the POST request |

## Returns

**`markdown_file`** — `{ "status": "published", "path": string }`
- `path`: absolute path of the written file (e.g. `/data/output/2026-02-22-weekly-digest.md`)
- Filename format: `YYYY-MM-DD-{slug-of-title}.md`
- File format: YAML frontmatter block (title, date, metadata fields) followed by blank line and content

**`webhook`** — `{ "status": "published", "status_code": number }`
- Sends a POST request with `Content-Type: application/json`
- Payload shape: `{ "title": "...", "content": "..." }`
- Any HTTP 4xx/5xx causes an error

## Agent Prompt Patterns

**Save as Markdown report:**
```
Publish the final report:
  channel: "markdown_file"
  title: "{{report_title}}"
  content: [the complete report text in Markdown format]
  metadata: { "source": "{{pipeline_name}}", "category": "weekly-digest" }

Report the saved file path.
```

**Webhook delivery (generic):**
```
Deliver the result to the webhook:
  channel: "webhook"
  title: "Pipeline Result"
  content: {{analysis_result}}
  metadata: { "webhook_url": "{{target_webhook_url}}" }

Confirm the delivery with the response status code.
```

**Slack incoming webhook:**
```
Send the digest to Slack:
  channel: "webhook"
  title: "Weekly News Digest"
  content: [formatted digest as plain text — use *bold* for Slack markdown]
  metadata: { "webhook_url": "https://hooks.slack.com/services/..." }

Note: Slack receives {"title": "Weekly News Digest", "content": "..."}.
The content field will be displayed as-is.
```

**Conditional publish:**
```
After processing:
- If new items were found: publish a report using channel="markdown_file",
  title="digest-{{date}}", content=[summary of new items].
- If no new items: skip publishing and output "No new content to publish."
```

**With metadata tags:**
```
Publish the analysis:
  channel: "markdown_file"
  title: "{{topic}} Analysis"
  content: {{analysis_text}}
  metadata: {
    "topic": "{{topic}}",
    "model": "{{model_used}}",
    "items_processed": "{{item_count}}"
  }
```

## Pitfalls & Limitations

- **Webhook payload is fixed** — always `{ "title", "content" }`. Cannot customize shape. For custom payloads (e.g. Slack Block Kit, Discord embeds), use `http_request` instead.
- **File directory is server-controlled** — the agent cannot choose where the file is saved. Only the title (filename slug) is controllable.
- **`metadata.webhook_url` is required for `webhook`** — omitting it causes an error.
- **Content must be a string** — ensure all `{{node_id}}` references resolve to strings before publishing. Do not pass raw objects.
- **Webhook timeout** — 30 seconds. Slow webhook endpoints will fail.
- **No retry on failure** — if delivery fails, the error is returned to the agent. Instruct the LLM to report the failure clearly.
- **File slug** — non-alphanumeric characters in `title` are replaced with hyphens. Very long titles are not truncated, which can produce long filenames.
