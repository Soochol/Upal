# Agent Output Extraction Design

**Date:** 2026-02-23
**Status:** Approved

## Problem

Agent nodes currently store the full LLM conversational response in session state
and pass it to downstream nodes via `{{node_id}}` template references. This means:

- Conversational preamble ("Sure! Here's your content: ...") pollutes downstream prompts
- Image generation agents mix data URI strings with surrounding text
- The Console displays the same full response

Users want only the meaningful "artifact" — the final produced output — to be passed
downstream and displayed in the Console.

## Goal

Decouple LLM response from the extracted artifact using a pluggable extraction strategy.
What the Console shows = what downstream nodes receive.

## Architecture

```
LLM Response
    ↓
OutputExtractor (strategy pattern)
    ├── RawExtractor     (default — current behavior)
    ├── JSONExtractor    (extracts a JSON key)
    ├── TaggedExtractor  (extracts XML-tagged block)
    └── ... future: RegexExtractor, JSONPathExtractor, LastBlockExtractor
    ↓
artifact string
    ├── state.Set(nodeID, artifact)       → downstream {{node_id}}
    └── stateDelta[nodeID] = artifact     → classifyEvent() → event.output → Console
```

`classifyEvent()` is changed to prefer `stateDelta[nodeID]` over `ExtractContent(LLMResponse)`.
This creates a single source of truth: what's stored in state = what's shown in the Console.

## Schema

### Go — `internal/upal/workflow.go`

```go
// OutputExtract defines how to extract an artifact from an LLM response.
type OutputExtract struct {
    Mode string `json:"mode"`           // "json" | "tagged"
    Key  string `json:"key,omitempty"`  // json mode: top-level JSON key
    Tag  string `json:"tag,omitempty"`  // tagged mode: XML tag name
    // Future additions (no schema migration needed):
    //   Pattern string `json:"pattern,omitempty"` // regex mode
    //   Path    string `json:"path,omitempty"`    // jsonpath mode
}
```

`OutputExtract *OutputExtract` is added to `NodeDefinition`.

### TypeScript — `web/src/shared/types/index.ts`

```typescript
interface OutputExtract {
  mode: 'json' | 'tagged'
  key?: string   // json mode
  tag?: string   // tagged mode
}
// Added to agent node config: output_extract?: OutputExtract
```

## Extraction Modes

| Mode | System Prompt Injected | Extraction | On Failure |
|------|------------------------|------------|------------|
| (none) | None | Full response (current behavior) | — |
| `json` | `Respond ONLY with valid JSON: {"<key>": <your output>}. No other text.` | JSON parse → `result[key]` | raw fallback |
| `tagged` | `Wrap your final output in <<tag>> tags: <<tag>>...</tag>. No other text outside the tags.` | regex `<tag>([\s\S]*?)</tag>` | raw fallback |

**Why `tagged` mode matters**: LLMs follow tag-wrapping instructions more reliably than pure JSON constraints, especially for multimodal or long outputs.

## Backend Changes

### `internal/agents/llm_builder.go`

1. Read `cfg.OutputExtract` from node definition
2. If set, append extraction instruction to system prompt
3. After `result = strings.TrimSpace(llmutil.ExtractContentSavingAudio(...))`:
   - Apply extractor: `artifact = extract(mode, result, key/tag)`
   - On failure: `artifact = result` (fallback)
4. `state.Set(nodeID, artifact)` — downstream receives artifact
5. `event.Actions.StateDelta[nodeID] = artifact` — passed to classifyEvent

```go
func applyOutputExtract(oe *upal.OutputExtract, raw string) string {
    if oe == nil { return raw }
    switch oe.Mode {
    case "json":
        // find JSON object in raw, parse, extract key
        if v, err := extractJSONKey(raw, oe.Key); err == nil { return v }
    case "tagged":
        // regex match <tag>...</tag>, extract inner content
        if v := extractTagged(raw, oe.Tag); v != "" { return v }
    }
    return raw // fallback
}
```

### `internal/services/workflow.go` — `classifyEvent()`

Use `stateDelta[nodeID]` as `output` when present:

```go
nodeID := event.Author
outputStr := llmutil.ExtractContent(&event.LLMResponse)
if a, ok := event.Actions.StateDelta[nodeID]; ok && a != nil {
    outputStr = fmt.Sprintf("%v", a)
}
payload["output"] = outputStr
```

This also benefits input/tool/asset nodes: they now show their actual output in
the Console (previously showed empty).

## Frontend Changes

### `web/src/features/edit-node/ui/AgentNodeEditor.tsx`

Add collapsible "Output Extraction" section below existing fields:

```
[ Output Extraction ]  (collapsed by default)
  Mode:  [ None ▾ ]    → None / JSON / Tagged
  Key:   [result    ]  (visible only in JSON mode)
  Tag:   [artifact  ]  (visible only in Tagged mode)
```

No changes to `PanelConsole.tsx`, `NodeOutputViewer.tsx`, or `detectOutputKind.ts`.
The Console already renders whatever string `event.output` contains.

## Future Extensibility

### Phase 1.x — New extraction modes (no schema change)
Add `Mode` values and corresponding backend cases:
- `"last_block"` — last non-empty paragraph of LLM response
- `"regex"` — with `Pattern string` field
- `"jsonpath"` — deep JSON path like `result.items[0].text`

### Phase 2 — Multi-output fields (separate feature)
```typescript
output_fields?: Array<{
  key: string    // referenced as {{node_id.key}}
  extract: OutputExtract
  type?: 'text' | 'image' | 'audio' | 'html'  // console rendering hint
}>
```
- `state[nodeID]` stored as JSON object `{"key1": "...", "key2": "data:..."}`
- `resolveTemplateFromState()` in `builders.go` gains dotpath support
- Console renders each field with type-appropriate component

## Files

| File | Change |
|------|--------|
| `internal/upal/workflow.go` | Add `OutputExtract` struct + field on `NodeDefinition` |
| `internal/agents/llm_builder.go` | Prompt injection + `applyOutputExtract()` + state storage |
| `internal/services/workflow.go` | `classifyEvent()` — prefer `stateDelta[nodeID]` for output |
| `web/src/shared/types/index.ts` | `OutputExtract` type + agent node config field |
| `web/src/features/edit-node/ui/AgentNodeEditor.tsx` | Output Extraction UI section |

No backend API changes required. `output_extract` flows through existing config
serialization (`NodeDefinition.Config` is a `map[string]any` or struct with JSON tags).
