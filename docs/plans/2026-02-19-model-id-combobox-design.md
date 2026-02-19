# Model ID Combobox Design

## Summary
Replace the plain text Input for Model ID in the node editor with a Select dropdown populated by available models from configured providers + locally installed Ollama models.

## Backend: `GET /api/models`

New endpoint returns available models grouped by provider.

### Model discovery logic
- **gemini**: hardcoded popular models (gemini-2.0-flash, gemini-2.5-pro, etc.)
- **anthropic**: hardcoded popular models (claude-sonnet-4, claude-haiku, etc.)
- **openai**: hardcoded popular models (gpt-4o, gpt-4o-mini, etc.)
- **ollama** (type=openai, URL contains `11434`): dynamic discovery via `GET http://{host}/api/tags`

### Response format
```json
[
  { "id": "gemini/gemini-2.0-flash", "provider": "gemini", "name": "gemini-2.0-flash" },
  { "id": "ollama/llama3.2", "provider": "ollama", "name": "llama3.2" }
]
```

## Frontend: Radix Select

- Use existing `Select` component (no new deps)
- Group by provider using `SelectGroup` + `SelectLabel`
- Fetch on mount via `GET /api/models`
- Restricted to discovered models only (no custom input)

## Files to modify
- `internal/api/server.go` — add route
- `internal/api/models.go` — new handler (model discovery logic)
- `web/src/lib/api.ts` — add `listModels()` function
- `web/src/components/editor/nodes/NodeEditor.tsx` — replace Input with Select
