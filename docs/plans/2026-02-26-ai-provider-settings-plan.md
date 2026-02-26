# AI Provider Settings Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Settings UI에서 AI API provider를 카테고리별(LLM/TTS/Image/Video)로 등록/관리하고, 기본 모델 선택이 워크플로우 생성 및 노드 설정에 자동 적용되도록 한다.

**Architecture:** 기존 Connection 시스템과 동일한 레이어 패턴(Domain → Repository → Service → API → Frontend)으로 별도 AI Provider 도메인을 구축. DB에 암호화 저장하고, GET /api/models를 등록된 provider 기준으로 전환.

**Tech Stack:** Go 1.24 / Chi / PostgreSQL / React 19 / TypeScript / Tailwind CSS v4 / Shadcn UI

---

### Task 1: Domain Model — AIProvider 타입 정의

**Files:**
- Create: `internal/upal/ai_provider.go`

**Step 1: Write the domain types**

```go
package upal

// AIProviderCategory classifies AI providers by capability.
type AIProviderCategory string

const (
	AICategoryLLM   AIProviderCategory = "llm"
	AICategoryTTS   AIProviderCategory = "tts"
	AICategoryImage AIProviderCategory = "image"
	AICategoryVideo AIProviderCategory = "video"
)

// AIProvider stores credentials and configuration for an AI model provider.
type AIProvider struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Category  AIProviderCategory `json:"category"`
	Type      string             `json:"type"`      // provider type: "anthropic", "openai", etc.
	APIKey    string             `json:"api_key,omitempty"` // encrypted at rest
	IsDefault bool               `json:"is_default"`
}

// AIProviderSafe is the API-safe view with secrets stripped.
type AIProviderSafe struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Category  AIProviderCategory `json:"category"`
	Type      string             `json:"type"`
	IsDefault bool               `json:"is_default"`
}

// Safe returns an AIProviderSafe view with the API key removed.
func (p *AIProvider) Safe() AIProviderSafe {
	return AIProviderSafe{
		ID:        p.ID,
		Name:      p.Name,
		Category:  p.Category,
		Type:      p.Type,
		IsDefault: p.IsDefault,
	}
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/upal/...`
Expected: success, no errors

**Step 3: Commit**

```bash
git add internal/upal/ai_provider.go
git commit -m "feat: add AIProvider domain model"
```

---

### Task 2: Repository Interface + Memory Implementation

**Files:**
- Create: `internal/repository/ai_provider.go` (interface)
- Create: `internal/repository/ai_provider_memory.go` (memory impl)

**Step 1: Write the repository interface**

```go
package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

type AIProviderRepository interface {
	Create(ctx context.Context, p *upal.AIProvider) error
	Get(ctx context.Context, id string) (*upal.AIProvider, error)
	List(ctx context.Context) ([]*upal.AIProvider, error)
	Update(ctx context.Context, p *upal.AIProvider) error
	Delete(ctx context.Context, id string) error
	// ClearDefault resets is_default for all providers in the given category.
	ClearDefault(ctx context.Context, category upal.AIProviderCategory) error
}
```

**Step 2: Write the memory implementation**

Follow the exact same pattern as `connection_memory.go`. Use `memstore.Store` with key function `func(p *upal.AIProvider) string { return p.ID }`.

Add `ClearDefault` by iterating all items and setting `IsDefault = false` where `Category` matches.

**Step 3: Verify it compiles**

Run: `go build ./internal/repository/...`
Expected: success

**Step 4: Commit**

```bash
git add internal/repository/ai_provider.go internal/repository/ai_provider_memory.go
git commit -m "feat: add AIProvider repository interface and memory implementation"
```

---

### Task 3: DB Layer — PostgreSQL Schema + Queries

**Files:**
- Modify: `internal/db/db.go` — add `ai_providers` table to migration SQL
- Create: `internal/db/ai_provider.go` — CRUD query methods

**Step 1: Add migration SQL**

Append to the `migrationSQL` const in `internal/db/db.go`, just before the final backtick:

```sql
CREATE TABLE IF NOT EXISTS ai_providers (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    category   TEXT NOT NULL,
    type       TEXT NOT NULL,
    api_key    TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT FALSE
);
```

**Step 2: Write DB query methods**

Create `internal/db/ai_provider.go` following the exact same pattern as `internal/db/connection.go`:

- `CreateAIProvider(ctx, p)` — INSERT
- `GetAIProvider(ctx, id)` — SELECT by id
- `ListAIProviders(ctx)` — SELECT all ORDER BY category, name
- `UpdateAIProvider(ctx, p)` — UPDATE by id
- `DeleteAIProvider(ctx, id)` — DELETE by id
- `ClearAIProviderDefault(ctx, category)` — `UPDATE ai_providers SET is_default = FALSE WHERE category = $1`

**Step 3: Verify it compiles**

Run: `go build ./internal/db/...`
Expected: success

**Step 4: Commit**

```bash
git add internal/db/db.go internal/db/ai_provider.go
git commit -m "feat: add ai_providers DB schema and queries"
```

---

### Task 4: Persistent Repository

**Files:**
- Create: `internal/repository/ai_provider_persistent.go`

**Step 1: Write the persistent repository**

Follow the exact same pattern as `connection_persistent.go`:
- Wraps `MemoryAIProviderRepository` + `*db.DB`
- Write-through: updates both memory and DB
- Graceful fallback: if DB fails, warn but succeed in-memory
- `List` reads from DB first, falls back to memory
- `ClearDefault` calls both memory and DB

**Step 2: Verify it compiles**

Run: `go build ./internal/repository/...`
Expected: success

**Step 3: Commit**

```bash
git add internal/repository/ai_provider_persistent.go
git commit -m "feat: add persistent AIProvider repository"
```

---

### Task 5: Service Layer

**Files:**
- Create: `internal/services/ai_provider.go`

**Step 1: Write the AIProvider service**

Follow the `ConnectionService` pattern. Key methods:

```go
type AIProviderService struct {
	repo repository.AIProviderRepository
	enc  *crypto.Encryptor
}

func NewAIProviderService(repo repository.AIProviderRepository, enc *crypto.Encryptor) *AIProviderService

func (s *AIProviderService) Create(ctx, p *AIProvider) error
  - Generate ID: upal.GenerateID("aip")
  - If p.IsDefault, call repo.ClearDefault(ctx, p.Category) first
  - Encrypt p.APIKey, then repo.Create

func (s *AIProviderService) Get(ctx, id) (*AIProvider, error)

func (s *AIProviderService) List(ctx) ([]AIProviderSafe, error)
  - repo.List → convert each to .Safe()

func (s *AIProviderService) Update(ctx, p *AIProvider) error
  - If p.IsDefault, call repo.ClearDefault(ctx, p.Category) first
  - Encrypt p.APIKey, then repo.Update

func (s *AIProviderService) Delete(ctx, id) error

func (s *AIProviderService) SetDefault(ctx, id) error
  - Get provider, ClearDefault for category, set IsDefault=true, Update

func (s *AIProviderService) Resolve(ctx, id) (*AIProvider, error)
  - Get + decrypt APIKey (for runtime use when building LLMs)

func (s *AIProviderService) ListAll(ctx) ([]*AIProvider, error)
  - repo.List (full objects, for internal use by models handler)
```

Encryption: only `APIKey` field needs encrypt/decrypt.

**Step 2: Verify it compiles**

Run: `go build ./internal/services/...`
Expected: success

**Step 3: Commit**

```bash
git add internal/services/ai_provider.go
git commit -m "feat: add AIProvider service with encryption"
```

---

### Task 6: API Handlers

**Files:**
- Create: `internal/api/ai_providers.go`
- Modify: `internal/api/server.go` — add field + setter + routes

**Step 1: Write API handlers**

Create `internal/api/ai_providers.go` following `connections.go` pattern:

```go
func (s *Server) createAIProvider(w, r)    // POST /api/ai-providers — validate name+type+category, create, return safe
func (s *Server) listAIProviders(w, r)     // GET /api/ai-providers — return []AIProviderSafe
func (s *Server) updateAIProvider(w, r)    // PUT /api/ai-providers/{id} — update, return safe
func (s *Server) deleteAIProvider(w, r)    // DELETE /api/ai-providers/{id} — 204
func (s *Server) setAIProviderDefault(w, r) // PUT /api/ai-providers/{id}/default — set default, return safe
```

**Step 2: Add to Server struct**

In `internal/api/server.go`:

- Add field: `aiProviderSvc *services.AIProviderService`
- Add setter: `func (s *Server) SetAIProviderService(svc *services.AIProviderService) { s.aiProviderSvc = svc }`
- Add routes in `Handler()`, after the connections block:

```go
if s.aiProviderSvc != nil {
    r.Route("/ai-providers", func(r chi.Router) {
        r.Post("/", s.createAIProvider)
        r.Get("/", s.listAIProviders)
        r.Put("/{id}", s.updateAIProvider)
        r.Delete("/{id}", s.deleteAIProvider)
        r.Put("/{id}/default", s.setAIProviderDefault)
    })
}
```

**Step 3: Verify it compiles**

Run: `go build ./internal/api/...`
Expected: success

**Step 4: Commit**

```bash
git add internal/api/ai_providers.go internal/api/server.go
git commit -m "feat: add AI provider API endpoints"
```

---

### Task 7: Main Wiring

**Files:**
- Modify: `cmd/upal/main.go`

**Step 1: Wire AIProvider service after connection service block (~line 216)**

```go
// AI provider management (persistent if DB is available).
memAIProviderRepo := repository.NewMemoryAIProviderRepository()
var aiProviderRepo repository.AIProviderRepository = memAIProviderRepo
if database != nil {
    aiProviderRepo = repository.NewPersistentAIProviderRepository(memAIProviderRepo, database)
}
aiProviderEnc, _ := upalcrypto.NewEncryptor(nil)
aiProviderSvc := services.NewAIProviderService(aiProviderRepo, aiProviderEnc)
srv.SetAIProviderService(aiProviderSvc)
```

**Step 2: Verify it compiles**

Run: `go build ./cmd/upal/...`
Expected: success

**Step 3: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat: wire AIProvider service in main"
```

---

### Task 8: Integrate GET /api/models with AI Providers

**Files:**
- Modify: `internal/api/models.go`
- Modify: `internal/api/server.go` (add helper)

**Step 1: Update listModels handler**

The `listModels` handler should:
1. If `aiProviderSvc` is available, list all providers via `ListAll(ctx)` (with decrypted keys)
2. Convert each `AIProvider` → `config.ProviderConfig` (Type from provider.Type, APIKey from decrypted key, URL inferred from known provider defaults)
3. Use these configs to call `AllStaticModels()` and `DiscoverOllamaModels()` as before
4. Fallback: if no AI providers in DB, use `s.providerConfigs` (existing config.yaml behavior)

Add a helper on Server:

```go
func (s *Server) effectiveProviderConfigs(ctx context.Context) map[string]config.ProviderConfig {
    if s.aiProviderSvc == nil {
        return s.providerConfigs
    }
    providers, err := s.aiProviderSvc.ListAll(ctx)
    if err != nil || len(providers) == 0 {
        return s.providerConfigs
    }
    configs := make(map[string]config.ProviderConfig, len(providers))
    for _, p := range providers {
        configs[p.Name] = config.ProviderConfig{
            Type:   p.Type,
            APIKey: p.APIKey,
            URL:    defaultURLForType(p.Type),
        }
    }
    return configs
}
```

Add `defaultURLForType` in `internal/model/catalog.go` or the models handler:

```go
func defaultURLForType(providerType string) string {
    switch providerType {
    case "anthropic":
        return "https://api.anthropic.com"
    case "openai", "openai-tts":
        return "https://api.openai.com"
    case "gemini", "gemini-image":
        return "" // uses Google SDK, no URL needed
    case "ollama":
        return "http://localhost:11434"
    default:
        return ""
    }
}
```

Update `listModels` to use `s.effectiveProviderConfigs(r.Context())` instead of `s.providerConfigs`.

**Step 2: Verify it compiles**

Run: `go build ./internal/api/...`
Expected: success

**Step 3: Commit**

```bash
git add internal/api/models.go internal/api/server.go
git commit -m "feat: integrate GET /api/models with AI providers"
```

---

### Task 9: Add Default Model Info to API Response

**Files:**
- Modify: `internal/api/ai_providers.go` — add endpoint for defaults
- Modify: `internal/api/server.go` — add route

**Step 1: Add defaults endpoint**

Add a `GET /api/ai-providers/defaults` endpoint that returns a map of category → default provider (safe view):

```go
func (s *Server) getAIProviderDefaults(w http.ResponseWriter, r *http.Request) {
    providers, err := s.aiProviderSvc.List(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defaults := make(map[string]upal.AIProviderSafe)
    for _, p := range providers {
        if p.IsDefault {
            defaults[string(p.Category)] = p
        }
    }
    writeJSON(w, defaults)
}
```

Add route: `r.Get("/defaults", s.getAIProviderDefaults)` inside the ai-providers route group.

**Step 2: Verify it compiles**

Run: `go build ./internal/api/...`
Expected: success

**Step 3: Commit**

```bash
git add internal/api/ai_providers.go internal/api/server.go
git commit -m "feat: add AI provider defaults endpoint"
```

---

### Task 10: Frontend — AI Provider Types and API Client

**Files:**
- Create: `web/src/entities/ai-provider/types.ts`
- Create: `web/src/entities/ai-provider/api.ts`
- Create: `web/src/entities/ai-provider/index.ts`

**Step 1: Write types**

```typescript
// types.ts
export type AIProviderCategory = 'llm' | 'tts' | 'image' | 'video'

export type AIProvider = {
  id: string
  name: string
  category: AIProviderCategory
  type: string
  is_default: boolean
}

export type AIProviderCreate = {
  name: string
  category: AIProviderCategory
  type: string
  api_key: string
}

export const PROVIDER_TYPES_BY_CATEGORY: Record<AIProviderCategory, { value: string; label: string }[]> = {
  llm: [
    { value: 'anthropic', label: 'Anthropic' },
    { value: 'openai', label: 'OpenAI' },
    { value: 'gemini', label: 'Google Gemini' },
    { value: 'ollama', label: 'Ollama' },
  ],
  tts: [
    { value: 'openai-tts', label: 'OpenAI TTS' },
  ],
  image: [
    { value: 'gemini-image', label: 'Gemini Image' },
    { value: 'zimage', label: 'Z-Image' },
  ],
  video: [],
}

export const CATEGORY_LABELS: Record<AIProviderCategory, string> = {
  llm: 'LLM',
  tts: 'TTS',
  image: 'Image',
  video: 'Video',
}
```

**Step 2: Write API client**

```typescript
// api.ts
import { apiFetch } from '@/shared/api/client'
import type { AIProvider, AIProviderCreate } from './types'

const BASE = '/api/ai-providers'

export async function listAIProviders(): Promise<AIProvider[]> {
  return apiFetch<AIProvider[]>(BASE)
}

export async function createAIProvider(data: AIProviderCreate): Promise<AIProvider> {
  return apiFetch<AIProvider>(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function updateAIProvider(id: string, data: Partial<AIProviderCreate>): Promise<AIProvider> {
  return apiFetch<AIProvider>(`${BASE}/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function deleteAIProvider(id: string): Promise<void> {
  await apiFetch<void>(`${BASE}/${id}`, { method: 'DELETE' })
}

export async function setAIProviderDefault(id: string): Promise<AIProvider> {
  return apiFetch<AIProvider>(`${BASE}/${id}/default`, { method: 'PUT' })
}
```

**Step 3: Write barrel export**

```typescript
// index.ts
export * from './types'
export * from './api'
```

**Step 4: Verify frontend compiles**

Run: `cd web && npx tsc -b`
Expected: success

**Step 5: Commit**

```bash
git add web/src/entities/ai-provider/
git commit -m "feat: add AI provider frontend types and API client"
```

---

### Task 11: Frontend — Settings AI Provider Section

**Files:**
- Modify: `web/src/pages/settings/index.tsx`

**Step 1: Build the AI Providers settings section**

Add below the Appearance section. The section should:

1. Fetch providers on mount via `listAIProviders()`
2. Display 4 category sections (LLM, TTS, Image, Video) using `CATEGORY_LABELS`
3. Each section shows registered providers as cards with:
   - Provider type badge + name
   - Star icon if default (clickable to set default)
   - Delete button (Trash2 icon)
4. "+ Add {Category} Provider" button at bottom of each section
5. Clicking add opens a modal/dialog for that category

For the add modal:
- Provider type selector (from `PROVIDER_TYPES_BY_CATEGORY[category]`)
- Name input (optional, defaults to provider type label)
- API Key input (password type)
- Save button → `createAIProvider()` → refresh list

For delete: confirm, then `deleteAIProvider()` → refresh list.
For set default: `setAIProviderDefault(id)` → refresh list.

Use existing Shadcn components: `Dialog`, `Input`, `Label`, `Button`, `Select`.

After any mutation (create/delete/setDefault), invalidate the model cache:
```typescript
// In useModels.ts, export a function:
export function invalidateModelsCache() { cachedModels = null }
```
Call this after provider mutations so `useModels()` refetches.

**Step 2: Verify frontend compiles**

Run: `cd web && npx tsc -b`
Expected: success

**Step 3: Commit**

```bash
git add web/src/pages/settings/index.tsx web/src/shared/api/useModels.ts
git commit -m "feat: add AI Provider settings UI with CRUD and default selection"
```

---

### Task 12: Frontend — Update ModelSelector Empty State

**Files:**
- Modify: `web/src/shared/ui/ModelSelector.tsx`

**Step 1: Update empty state message**

Change the "No models available" message from "Configure providers in config.yaml" to "Settings에서 AI Provider를 등록하세요" (or a neutral English version linking to Settings).

**Step 2: Verify frontend compiles**

Run: `cd web && npx tsc -b`
Expected: success

**Step 3: Commit**

```bash
git add web/src/shared/ui/ModelSelector.tsx
git commit -m "feat: update ModelSelector empty state to reference Settings"
```

---

### Task 13: Dynamic LLM Building from AI Providers

**Files:**
- Modify: `cmd/upal/main.go` — rebuild LLMs from DB providers on startup
- Modify: `internal/api/server.go` — add method to rebuild LLMs dynamically

**Step 1: On startup, merge config.yaml LLMs with DB providers**

After AI provider service is wired in main.go, add:

```go
// Merge DB-registered AI providers into LLM pool.
ctx := context.Background()
if dbProviders, err := aiProviderSvc.ListAll(ctx); err == nil && len(dbProviders) > 0 {
    for _, p := range dbProviders {
        // Decrypt API key for LLM building
        resolved, err := aiProviderSvc.Resolve(ctx, p.ID)
        if err != nil {
            continue
        }
        pc := config.ProviderConfig{
            Type:   resolved.Type,
            APIKey: resolved.APIKey,
        }
        if llm, ok := upalmodel.BuildLLM(resolved.Name, pc); ok {
            llms[resolved.Name] = llm
            providerTypes[resolved.Name] = resolved.Type
        }
    }
    // Re-pick default LLM if a DB provider is marked as default
    for _, p := range dbProviders {
        if p.IsDefault && p.Category == upal.AICategoryLLM {
            if llm, ok := llms[p.Name]; ok {
                defaultLLM = llm
                // Pick first known model for this provider type
                if models, exists := upalmodel.FirstModelForType(p.Type); exists {
                    defaultModelName = models
                }
            }
        }
    }
}
```

Add `FirstModelForType` helper in `internal/model/catalog.go`:

```go
func FirstModelForType(providerType string) (string, bool) {
    if models, ok := knownModels[providerType]; ok && len(models) > 0 {
        return models[0].Name, true
    }
    return "", false
}
```

**Step 2: Verify it compiles**

Run: `go build ./cmd/upal/...`
Expected: success

**Step 3: Commit**

```bash
git add cmd/upal/main.go internal/model/catalog.go
git commit -m "feat: build LLMs from DB providers on startup"
```

---

### Task 14: End-to-End Verification

**Step 1: Run backend tests**

Run: `go test ./... -v -race -count=1 2>&1 | tail -50`
Expected: all tests pass

**Step 2: Run frontend type check**

Run: `cd web && npx tsc -b`
Expected: no errors

**Step 3: Start dev servers and manually verify**

Run: `make dev-backend` (in terminal 1) and `make dev-frontend` (in terminal 2)

Verify:
1. Navigate to Settings page → AI Providers section visible
2. Add an LLM provider (e.g., Anthropic with API key)
3. Set as default
4. Check workflow editor → model selector shows only registered models
5. Create new agent node → model defaults to the registered LLM

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete AI provider settings with UI, API, and model integration"
```
