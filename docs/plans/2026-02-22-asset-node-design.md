# Asset Node Feature Design

Date: 2026-02-22

## Overview

Allow users to upload files from their PC and place them as nodes on the workflow canvas. Asset nodes extract file content at upload time and inject it into downstream agent prompts at runtime via `{{node_id}}` template references.

## Decisions

- **Runtime behavior**: Text extraction at upload time (Option A — eager). Extracted text stored in FileInfo; asset node executor reads it at runtime with zero latency.
- **File formats**: text family, PDF, DOCX, XLSX, images (JPG/PNG/WebP)
- **UX**: drag-and-drop onto canvas + sidebar button
- **Node deletion**: deletes the uploaded file from storage

## Data Flow

```
User drags file onto canvas (or clicks sidebar button)
  → POST /api/upload
  → Storage.Save()          stores raw file
  → extract.Extract()       extracts text (format-specific)
  → FileInfo saved with ExtractedText + PreviewText
  → Frontend creates asset node: { file_id, filename, preview_text, is_image }

Workflow run:
  → AssetNodeBuilder.Run()
  → state[nodeID] = ExtractedText   (or base64 data URI for images)
  → {{asset_1}} in agent prompt → resolved to file content
  → llm_builder detects data:image/ prefix → adds as multimodal image part
```

## Backend

### New package: `internal/extract/`

| File | Responsibility |
|------|---------------|
| `extractor.go` | `Extract(contentType string, r io.Reader) (string, error)` dispatcher |
| `text.go` | `text/*` → UTF-8 string |
| `pdf.go` | `application/pdf` → text via `github.com/ledongthuc/pdf` |
| `office.go` | DOCX (ZIP+XML parse), XLSX via `github.com/xuri/excelize/v2` |
| `image.go` | `image/*` → `data:image/...;base64,...` data URI |

### Modified files

**`internal/storage/storage.go`** — extend `FileInfo`:
```go
ExtractedText string `json:"extracted_text,omitempty"` // full extracted content
PreviewText   string `json:"preview_text,omitempty"`   // first 300 chars for UI
```

**`internal/api/upload.go`** — after `Storage.Save()`, call `extract.Extract()`, persist updated FileInfo.

**`internal/api/server.go`** — add routes:
- `GET /api/files/{id}/serve` — stream raw file (for image preview)
- `DELETE /api/files/{id}` — delete file from storage

**`internal/upal/types.go`** — add `NodeTypeAsset NodeType = "asset"`

**`internal/agents/asset_builder.go`** — `AssetNodeBuilder`:
- Config: `file_id string`
- Run: loads FileInfo → sets `state[nodeID] = ExtractedText`

**`internal/agents/registry.go`** — register `AssetNodeBuilder`

**`internal/agents/llm_builder.go`** — when resolving `{{node_id}}`, if value starts with `data:image/`, add as multimodal image part instead of inline text

## Frontend

### New files

| File | Purpose |
|------|---------|
| `web/src/components/editor/nodes/AssetNodeEditor.tsx` | Read-only panel: filename, size, type, preview_text |

### Modified files

**`web/src/lib/nodeTypes.ts`** — add `asset` type:
- Icon: `Paperclip`
- Color: `--node-asset` CSS variable

**`web/src/lib/nodeConfigs.ts`** — add `AssetNodeConfig`:
```ts
type AssetNodeConfig = {
  file_id: string
  filename: string
  content_type: string
  preview_text: string
  is_image: boolean
}
```

**`web/src/stores/workflowStore.ts`**:
- Add `'asset'` to `NodeData.nodeType` union
- In `onNodesChange`: when an asset node is removed, call `DELETE /api/files/{file_id}`

**`web/src/components/editor/nodes/NodeEditor.tsx`** — register `AssetNodeEditor`

**`web/src/components/editor/Canvas.tsx`** — add `onDrop` / `onDragOver`:
- Detect `e.dataTransfer.files`
- Call `uploadFile()` → create asset node at drop position

**`web/src/components/editor/UpalNode.tsx`** — asset node preview:
- Images: `<img src="/api/files/{id}/serve" />` thumbnail
- Others: file icon + filename + preview_text snippet

**Sidebar palette** — add "Asset" entry that opens file picker dialog

## CSS

Add `--node-asset` token in `index.css` (light + dark), consistent with other node-type variables.

## Libraries

| Library | Purpose | License |
|---------|---------|---------|
| `github.com/ledongthuc/pdf` | PDF text extraction | MIT |
| `github.com/xuri/excelize/v2` | XLSX cell extraction | BSD-3 |

DOCX extraction uses only stdlib (`archive/zip` + `encoding/xml`) — no new dependency.
