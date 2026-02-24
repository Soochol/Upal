# Pipeline Model Override — Design

## Problem

`ContentCollector` uses a single `defaultLLM` + `defaultModelName` for all pipelines. Users cannot choose which LLM model analyzes their collected content on a per-pipeline basis.

## Scope

Analysis stage only. Workflow execution (Produce) uses each workflow's own model config.

## Design

### LLMResolver Port

New interface in `upal/ports/`:

```go
// LLMResolver resolves a "provider/model" ID to an LLM instance.
type LLMResolver interface {
    Resolve(modelID string) (model.LLM, string, error)
}
```

Implementation in `cmd/upal/` or a new `internal/llmutil/resolver.go`:

```go
type mapResolver struct {
    llms         map[string]model.LLM
    defaultLLM   model.LLM
    defaultModel string
}

func (r *mapResolver) Resolve(modelID string) (model.LLM, string, error) {
    if modelID == "" {
        return r.defaultLLM, r.defaultModel, nil
    }
    provider, modelName, _ := strings.Cut(modelID, "/")
    llm, ok := r.llms[provider]
    if !ok {
        return nil, "", fmt.Errorf("unknown provider %q", provider)
    }
    return llm, modelName, nil
}
```

### Domain

Add `Model` field to `Pipeline`:

```go
type Pipeline struct {
    // ... existing fields
    Model string `json:"model,omitempty"` // e.g. "anthropic/claude-sonnet-4-6"
}
```

### ContentCollector

Replace `llm adkmodel.LLM` + `model string` with `resolver ports.LLMResolver`:

```go
type ContentCollector struct {
    contentSvc   *ContentSessionService
    collectExec  *CollectStageExecutor
    workflowSvc  *WorkflowService
    workflowRepo repository.WorkflowRepository
    resolver     ports.LLMResolver
}
```

In `runAnalysis()`:

```go
llm, modelName, err := c.resolver.Resolve(pipeline.Model)
// use llm + modelName for LLMRequest
```

### Database

Add nullable `model TEXT` column to `pipelines` table. Existing rows get NULL → system default fallback.

### API

No handler changes needed — Pipeline CRUD already serializes the full struct. The `model` field flows through automatically.

### Frontend

Settings panel (`PipelineSettingsPanel`): add a "Model" collapsible section with a dropdown. Fetch options from `GET /api/models` (filter to text category). Empty selection = "System Default".

## Files Changed

| Layer | File | Change |
|-------|------|--------|
| Domain | `upal/pipeline.go` | Add `Model` field |
| Port | `upal/ports/llm.go` | New `LLMResolver` interface |
| Resolver | `llmutil/resolver.go` or inline in `main.go` | `mapResolver` implementation |
| Service | `services/content_collector.go` | Accept resolver, use in `runAnalysis` |
| Wiring | `cmd/upal/main.go` | Create resolver, pass to `NewContentCollector` |
| DB | `db/db.go` or migration | Add `model` column |
| Frontend | `PipelineDetail.tsx` | Model dropdown in Settings panel |

## Future Extension

The same `LLMResolver` can replace `defaultLLM` usage in `Generator`, `configure.go`, and other call sites that currently hardwire a single model.
