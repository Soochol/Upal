---
name: asset-node
description: Guide for understanding and referencing asset nodes — pre-uploaded file content injected into the workflow
---

## Objective

An asset node loads a pre-uploaded file from storage and exposes its content to downstream nodes via `{{node_id}}` template references. It performs no computation — it is a *source* node that makes file content available to the rest of the workflow.

**IMPORTANT**: Asset nodes CANNOT be created by generation. Files must be uploaded through the UI before a workflow runs. When editing a workflow that already contains asset nodes, preserve them exactly and reference their output correctly in downstream agent prompts.

---

## Schema

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `file_id` | string | Upload UI | UUID of the uploaded file — **required for execution** |
| `filename` | string | Upload UI | Original filename, for display only |
| `content_type` | string | Upload UI | MIME type (e.g. `"application/pdf"`, `"image/png"`), for display only |
| `preview_text` | string | Upload UI | First 300 characters of extracted text, for display only |

All fields are populated automatically when a file is uploaded through the workflow editor. **Never manually set or modify these fields in generated output.**

---

## Output: what `{{node_id}}` resolves to

The asset node stores its output in session state under the node's ID. Downstream nodes access it via `{{node_id}}`.

| File type | `{{node_id}}` contains |
|-----------|------------------------|
| Text files (`.txt`, `.csv`, `.json`, `.md`, …) | Raw file content as a string |
| PDF | Extracted plain text from all pages |
| DOCX / XLSX | Extracted plain text from the document |
| Images (`image/*`) | Base64 data URI: `data:image/png;base64,…` |
| Other / extraction failure | `[file: filename.ext]` (filename placeholder) |

**Image special behavior**: When an agent node's prompt contains a data URI (`data:image/…`), the platform automatically converts it to an inline image for vision-capable LLMs. The agent receives the actual image, not a text string. To analyze an image asset, simply reference it with `{{node_id}}` and use a vision-capable model.

---

## Usage Patterns

### Pattern 1 — Document analysis
An agent reads a PDF and produces a structured summary.
```
asset (PDF) ──→ analyzer_agent ──→ output
```
Agent prompt: `"다음 문서의 내용을 분석하고 핵심 포인트를 추출하세요:\n\n{{contract_pdf}}"`

### Pattern 2 — Image analysis (vision)
An agent with a vision model describes or classifies an image.
```
asset (image) ──→ vision_agent ──→ output
```
Agent config:
- `model`: a vision-capable model (e.g. `"anthropic/claude-sonnet-4-6"`, `"gemini/gemini-2.0-flash"`)
- `prompt`: `"다음 이미지를 분석하고 내용을 설명하세요:\n\n{{product_image}}"`

### Pattern 3 — Multi-document comparison
Two asset nodes feed one comparative agent.
```
asset_a ──┐
           ├──→ compare_agent ──→ output
asset_b ──┘
```
Agent prompt:
```
"다음 두 문서를 비교하고 주요 차이점을 분석하세요:\n\n## 문서 A\n{{doc_a}}\n\n## 문서 B\n{{doc_b}}"
```

### Pattern 4 — Data file + user question
An asset provides structured data; the user provides a question via an input node.
```
asset (CSV) ──┐
               ├──→ analyst_agent ──→ output
input ─────────┘
```
Agent prompt:
```
"다음 데이터를 참고하여 질문에 답하세요.\n\n데이터:\n{{data_csv}}\n\n질문: {{user_question}}"
```

---

## Rules

1. **NEVER generate a new asset node.** Files must be uploaded through the UI. If a user's request implies file ingestion, acknowledge that they need to upload the file through the editor and attach it to an asset node manually.
2. **When editing a workflow, preserve all existing asset nodes verbatim** — same `id`, same `config`. Do not modify or remove them unless the user explicitly asks.
3. **Reference asset output with `{{node_id}}`** in downstream agent `prompt` fields, exactly like any other upstream node.
4. **Asset nodes have no upstream inputs** — do not add incoming edges to an asset node.
5. **Asset nodes must connect to at least one downstream node** — add an outgoing edge to the agent that consumes the file.
6. **For image assets, use a vision-capable model** in the downstream agent. Check the "Available models" list for models that support image input.
7. **Do not hardcode file content** — always use `{{node_id}}` to reference the asset. Never paste or assume file content in a prompt.
