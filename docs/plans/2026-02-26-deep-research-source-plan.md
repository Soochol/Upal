# Deep Research Source Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 파이프라인 소스에 `research` 타입을 추가하여 사용자가 LLM 기반 deep research를 소스 설정 UI에서 추가하고, 호환 모델을 선택할 수 있게 한다.

**Architecture:** 이미 구현된 `researchFetcher`와 `stage-research.md` 스킬을 파이프라인 소스 시스템에 연결한다. `PipelineSource`에 topic/depth/model 필드를 추가하고, `mapPipelineSources`에 research 매핑을 추가하고, `researchFetcher`를 collect executor에 등록한다. 프론트엔드에는 3번째 "AI Research" 카테고리를 AddSourceModal에 추가하고, 모델 선택 시 `supportsTools` 필터링을 적용한다.

**Tech Stack:** Go 1.24 (Chi, ADK), React 19, TypeScript, Tailwind CSS v4, Vite

**Design doc:** `docs/plans/2026-02-26-deep-research-source-design.md`

---

### Task 1: Backend — PipelineSource에 research 필드 추가

**Files:**
- Modify: `internal/upal/pipeline.go:94-111` (PipelineSource struct)

**Step 1: Add topic, depth, model fields**

`PipelineSource` struct의 `Limit` 필드(line 110) 뒤에 추가:

```go
	Limit     int      `json:"limit,omitempty"`
	// Research source fields
	Topic string `json:"topic,omitempty"` // research: subject to investigate
	Depth string `json:"depth,omitempty"` // research: "light" | "deep"
	Model string `json:"model,omitempty"` // research: LLM model ID ("provider/model")
```

**Step 2: Commit**

```bash
git add internal/upal/pipeline.go
git commit -m "feat: add research fields to PipelineSource"
```

---

### Task 2: Backend — mapPipelineSources에 research case 추가

**Files:**
- Modify: `internal/services/content_collector.go:780-849` (mapPipelineSources)
- Modify: `internal/services/content_collector.go:186-247` (convertToSourceItems)

**Step 1: Add research case to mapPipelineSources**

`mapPipelineSources` 함수의 `switch ps.Type` 블록에서 `case "social", "twitter":` 블록(line 827-831) 뒤, `default:` 앞에 추가:

```go
		case "research":
			cs.Type = "research"
			cs.Topic = ps.Topic
			cs.Model = ps.Model
			cs.Depth = ps.Depth
			if cs.Depth == "" {
				cs.Depth = "deep"
			}
```

**Step 2: Add research case to convertToSourceItems**

`convertToSourceItems` 함수의 `case "social":` 블록(line 243) 뒤, 마지막 `return nil`(line 246) 앞에 추가:

```go
	case "research":
		items, ok := data.([]map[string]any)
		if !ok {
			return nil
		}
		result := make([]upal.SourceItem, 0, len(items))
		for _, item := range items {
			result = append(result, upal.SourceItem{
				Title:   stringVal(item, "title"),
				URL:     stringVal(item, "url"),
				Content: stringVal(item, "summary"),
			})
		}
		return result
```

**Step 3: Commit**

```bash
git add internal/services/content_collector.go
git commit -m "feat: add research source mapping and item conversion"
```

---

### Task 3: Backend — researchFetcher를 CollectStageExecutor에 등록

**Files:**
- Modify: `internal/services/stage_collect.go:29-38` (NewCollectStageExecutor)
- Modify: `cmd/upal/main.go:259,302-304` (wiring)

**Step 1: Change NewCollectStageExecutor to accept resolver and skills**

`stage_collect.go`의 `NewCollectStageExecutor` 시그니처를 변경:

```go
func NewCollectStageExecutor(resolver ports.LLMResolver, skills skills.Provider) *CollectStageExecutor {
	e := &CollectStageExecutor{
		fetchers: make(map[string]SourceFetcher),
	}
	client := &http.Client{Timeout: 30 * time.Second}
	e.RegisterFetcher(&rssFetcher{client: client})
	e.RegisterFetcher(&httpFetcher{client: client})
	e.RegisterFetcher(&scrapeFetcher{client: client})
	e.RegisterFetcher(NewSocialFetcher(client))
	if resolver != nil {
		e.RegisterFetcher(NewResearchFetcher(resolver, skills))
	}
	return e
}
```

필요한 import 추가:
```go
	"github.com/soochol/upal/internal/skills"
	"github.com/soochol/upal/internal/upal/ports"
```

**Step 2: Update main.go call sites**

`cmd/upal/main.go`에서 `NewCollectStageExecutor()` 호출을 2곳 모두 업데이트.

Line 259 (pipeline runner):
```go
pipelineRunner.RegisterExecutor(services.NewCollectStageExecutor(resolver, skillReg))
```

Line 304 (content collector):
```go
services.NewCollectStageExecutor(resolver, skillReg),
```

**Step 3: Run backend tests**

```bash
make test
```

**Step 4: Commit**

```bash
git add internal/services/stage_collect.go cmd/upal/main.go
git commit -m "feat: register researchFetcher in CollectStageExecutor"
```

---

### Task 4: Frontend — PipelineSource 타입에 research 추가

**Files:**
- Modify: `web/src/entities/pipeline/types.ts:1-22`

**Step 1: Add 'research' to PipelineSourceType**

```typescript
export type PipelineSourceType =
  | 'rss'
  | 'hn'
  | 'reddit'
  | 'google_trends'
  | 'social'
  | 'http'
  | 'research'
```

**Step 2: Add research fields to PipelineSource**

```typescript
export type PipelineSource = {
  id: string
  type: PipelineSourceType
  source_type: 'static' | 'signal' | 'research'
  label: string
  // type-specific config
  url?: string
  subreddit?: string
  min_score?: number
  keywords?: string[]
  accounts?: string[]
  geo?: string
  limit?: number
  // research
  topic?: string
  depth?: 'light' | 'deep'
  model?: string
}
```

**Step 3: Commit**

```bash
git add web/src/entities/pipeline/types.ts
git commit -m "feat: add research type to PipelineSource"
```

---

### Task 5: Frontend — AddSourceModal에 AI Research 카테고리 추가

**Files:**
- Modify: `web/src/features/configure-pipeline-sources/AddSourceModal.tsx`

**Step 1: Add imports and RESEARCH_SOURCES array**

상단 import에 추가:
```typescript
import { Search } from 'lucide-react'
import { ModelSelector } from '@/shared/ui/ModelSelector'
import { useModels } from '@/shared/api/useModels'
```

`SIGNAL_SOURCES` 배열 뒤에 추가:

```typescript
export const RESEARCH_SOURCES: SourceTypeDef[] = [
  {
    type: 'research',
    source_type: 'research' as any,
    label: 'Deep Research',
    description: 'LLM-powered topic research with web search',
    icon: <Search className="h-4 w-4" />,
    accent: 'bg-[oklch(0.7_0.15_280)]/12',
    accentText: 'text-[oklch(0.6_0.15_280)]',
  },
]
```

**Step 2: Add 3rd category in the selection step**

`step === 'select'` 분기 내에서, Signal sources 섹션 뒤에 추가:

```tsx
{/* AI Research */}
<div>
  <div className="flex items-center gap-2 mb-2.5">
    <span className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">AI Research</span>
    <div className="flex-1 h-px bg-border" />
  </div>
  <div className="space-y-1.5">
    {RESEARCH_SOURCES.map((typeDef) => (
      <button
        key={typeDef.type}
        onClick={() => handleSelectType(typeDef)}
        className="group w-full flex items-center gap-3 rounded-xl border border-border p-3
          hover:border-foreground/15 hover:bg-muted/40 hover:shadow-sm
          active:scale-[0.99] transition-all text-left cursor-pointer"
      >
        <div className={`w-8 h-8 rounded-lg ${typeDef.accent} ${typeDef.accentText} flex items-center justify-center shrink-0
          group-hover:scale-110 transition-transform`}>
          {typeDef.icon}
        </div>
        <div className="min-w-0 flex-1">
          <span className="text-sm font-medium text-foreground">{typeDef.label}</span>
          <p className="text-[11px] text-muted-foreground leading-tight">{typeDef.description}</p>
        </div>
      </button>
    ))}
  </div>
</div>
```

**Step 3: Add research config fields**

`step === 'config'` 분기의 type-specific fields 영역에 추가:

```tsx
{draft.type === 'research' && (
  <>
    <div>
      <label className="block text-xs font-medium text-muted-foreground mb-1.5">Research topic</label>
      <input
        type="text"
        value={(draft as any).topic ?? ''}
        onChange={(e) => setDraft({ ...draft, topic: e.target.value } as any)}
        placeholder="e.g. EV battery technology trends 2026"
        className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
      />
    </div>
    <div>
      <label className="block text-xs font-medium text-muted-foreground mb-1.5">Depth</label>
      <div className="flex gap-4 mt-1">
        <label className="flex items-center gap-1.5 text-sm cursor-pointer">
          <input
            type="radio"
            name="depth"
            value="light"
            checked={((draft as any).depth ?? 'deep') === 'light'}
            onChange={() => setDraft({ ...draft, depth: 'light' } as any)}
            className="accent-primary"
          />
          <span>Light</span>
          <span className="text-[10px] text-muted-foreground">— quick scan</span>
        </label>
        <label className="flex items-center gap-1.5 text-sm cursor-pointer">
          <input
            type="radio"
            name="depth"
            value="deep"
            checked={((draft as any).depth ?? 'deep') === 'deep'}
            onChange={() => setDraft({ ...draft, depth: 'deep' } as any)}
            className="accent-primary"
          />
          <span>Deep</span>
          <span className="text-[10px] text-muted-foreground">— iterative research</span>
        </label>
      </div>
    </div>
    <ResearchModelSelector
      value={(draft as any).model ?? ''}
      onChange={(model) => setDraft({ ...draft, model } as any)}
    />
  </>
)}
```

**Step 4: Add ResearchModelSelector component**

같은 파일 하단(AddSourceModal 컴포넌트 위 또는 아래)에 추가:

```tsx
function ResearchModelSelector({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const models = useModels()
  const researchModels = models.filter((m) => m.supportsTools && m.category === 'text')

  return (
    <div>
      <label className="block text-xs font-medium text-muted-foreground mb-1.5">Model</label>
      <ModelSelector
        value={value}
        onChange={onChange}
        models={researchModels}
        placeholder="Select a model for research..."
      />
      <p className="text-[10px] text-muted-foreground mt-1">
        Only models with web search support are shown
      </p>
    </div>
  )
}
```

**Step 5: Update handleSelectType defaults for research**

`handleSelectType` 함수에서 research 타입일 때 기본값 설정. 기존 `setDraft` 호출을 research 타입에 맞게 확장:

```typescript
const handleSelectType = (typeDef: SourceTypeDef) => {
  setSelectedType(typeDef)
  setDraft({
    id: generateId(),
    type: typeDef.type,
    source_type: typeDef.source_type,
    label: typeDef.label,
    limit: typeDef.type === 'research' ? undefined : 20,
    keywords: [],
    accounts: [],
    ...(typeDef.type === 'research' ? { depth: 'deep', topic: '', model: '' } : {}),
  })
  setStep('config')
}
```

**Step 6: Run type check**

```bash
cd web && npx tsc -b
```

**Step 7: Commit**

```bash
git add web/src/features/configure-pipeline-sources/AddSourceModal.tsx
git commit -m "feat: add AI Research category to AddSourceModal with model selector"
```

---

### Task 6: Skills — stage-collect.md에 research 문서 추가

**Files:**
- Modify: `internal/skills/stages/stage-collect.md`

**Step 1: Add research source type**

`### Source type fields` 섹션의 마지막 항목(`social`) 뒤에 추가:

```markdown
- **research**: `topic` (string, required — subject to investigate), `depth` ("light" | "deep", default "deep"), `model` (string, required — "provider/model" format, must support web_search native tool). LLM-powered web research. Light mode does a single search pass; deep mode runs an iterative agent loop with sub-question decomposition. Not available on Ollama or other providers without native tool support.
```

`### Rules` 섹션의 마지막 항목 뒤에 추가:

```markdown
- `research` requires a model that supports native `web_search` tool (Anthropic, Gemini, OpenAI). Do not use with Ollama.
```

**Step 2: Commit**

```bash
git add internal/skills/stages/stage-collect.md
git commit -m "feat: add research source type to stage-collect skill"
```

---

### Task 7: Frontend Type Check + Backend Test

**Files:** (no new files)

**Step 1: Run frontend type check**

```bash
cd web && npx tsc -b
```

Fix any type errors.

**Step 2: Run frontend lint**

```bash
cd web && npm run lint
```

Fix any lint errors.

**Step 3: Run backend tests**

```bash
make test
```

Fix any failures.

**Step 4: Commit fixes if any**

```bash
git add -A
git commit -m "fix: resolve type check and test issues for research source"
```

---

### Task 8: useModels hook 확인 및 필요 시 수정

**Files:**
- Check: `web/src/shared/api/useModels.ts`

**Step 1: Verify useModels returns ModelInfo[] correctly**

`useModels` 훅이 `AddSourceModal`에서 사용 가능한 형태인지 확인. `ModelInfo.supportsTools` 필드가 API 응답에 이미 포함되어 있으므로 추가 작업 불필요할 수 있음.

Read the file and verify. If `useModels` doesn't export a synchronous array, adjust the `ResearchModelSelector` to use `listModels` directly with `useEffect` (like `ModelSelector` does).

**Step 2: Commit if needed**

```bash
git add web/src/shared/api/useModels.ts
git commit -m "fix: ensure useModels works for research model selector"
```
