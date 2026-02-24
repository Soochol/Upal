# Skill Prompt Extraction Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 하드코딩된 LLM 시스템 프롬프트 4개를 `internal/skills/prompts/` md 파일로 분리하고, skills 접근 인터페이스를 통일한다.

**Architecture:** `skills.Provider` 인터페이스에 `GetPrompt()` 추가하여 모든 패키지가 동일한 인터페이스로 프롬프트에 접근. 4개 하드코딩 문자열을 md 파일로 이동하고, Go 코드에서 `GetPrompt("name")` 호출로 교체.

**Tech Stack:** Go 1.24, embedded filesystem (`//go:embed`), skills registry

**Design doc:** `docs/plans/2026-02-24-skill-prompt-extraction-design.md`

---

### Task 1: Interface Unification — `skills.Provider`에 `GetPrompt()` 추가

**Files:**
- Modify: `internal/skills/provider.go`
- Modify: `internal/generate/generate.go:32-36, 42, 53`

**Step 1: Update `skills.Provider` interface**

In `internal/skills/provider.go`, add `GetPrompt` method:

```go
package skills

// Provider abstracts read access to skill and prompt content.
// Registry satisfies this interface implicitly.
type Provider interface {
	Get(name string) string
	GetPrompt(name string) string
}
```

**Step 2: Remove local `skillProvider` from generate package**

In `internal/generate/generate.go`:
- Delete lines 32-36 (the local `skillProvider` interface)
- Add `"github.com/soochol/upal/internal/skills"` to imports
- Change `skills` field type from `skillProvider` to `skills.Provider` (line 42)
- Change `New()` parameter from `skills skillProvider` to `skills skills.Provider` (line 53)

The `skills` field and parameter name stays the same, only the type changes.

**Step 3: Run tests to verify**

Run: `go build ./...`
Expected: PASS (no compilation errors)

Run: `go test ./internal/generate/... ./internal/api/... -v -race -count=1`
Expected: PASS — `noopSkills` in `api/generate_test.go` already implements both `Get()` and `GetPrompt()`

**Step 4: Commit**

```
feat(skills): unify Provider interface with GetPrompt method
```

---

### Task 2: Create 4 skill prompt files

**Files:**
- Create: `internal/skills/prompts/content-analyze.md`
- Create: `internal/skills/prompts/node-configure.md`
- Create: `internal/skills/prompts/workflow-name.md`
- Create: `internal/skills/prompts/html-layout.md`

**Step 1: Create `prompts/content-analyze.md`**

Copy exact text from `content_collector.go:302-318` into markdown with frontmatter:

```markdown
---
name: content-analyze
description: System prompt for LLM content analysis in pipeline sessions
---

You are a content analyst. Analyze the collected data and return a JSON object with these fields:
- summary: 2-3 sentence overview of the collected content
- insights: array of up to 5 key findings as strings
- suggested_angles: array of objects with:
  - "format": content format (e.g. blog, shorts, newsletter, longform, video, thread)
  - "headline": short compelling title for this angle
  - "workflow_name": exact name of the best matching workflow from the available list (empty string "" if none match well)
  - "rationale": one sentence explaining why this workflow is the best fit for this angle
- overall_score: 0-100 relevance score based on how well the content matches the pipeline context

When choosing workflow_name:
- Select the workflow whose purpose and node structure best match the content format and production goal
- Only use exact names from the available workflow list
- If no workflow is a good fit, set workflow_name to empty string ""
- Prefer workflows from the pipeline's preferred list when quality of match is similar

Only return valid JSON, no markdown fences, no commentary.
```

**Step 2: Create `prompts/node-configure.md`**

Copy exact text from `configure.go:52-69` (`configureBasePrompt` var):

```markdown
---
name: node-configure
description: Base system prompt for AI-assisted node configuration
---

You are an AI assistant that fully configures nodes in the Upal visual workflow platform.
When the user describes what a node should do, you MUST fill in ALL relevant config fields — not just one or two.
Be proactive: infer and set every field that makes sense given the user's description.

You MUST also set "label" (short name for the node) and "description" (brief explanation of its purpose).

Template syntax: {{node_id}} references the output of an upstream node at runtime. This is how data flows between nodes in the DAG.

IMPORTANT RULES:
1. ALWAYS set label and description based on the user's intent.
2. CRITICAL — upstream node references: When upstream nodes exist, you MUST use {{node_id}} template references to receive their output. NEVER write hardcoded placeholder text like "다음 내용을 분석해줘: [여기에 입력]" — instead write "다음 내용을 분석해줘:\n\n{{upstream_node_id}}". The {{node_id}} gets replaced with the actual upstream node's output at runtime.
3. Fill in ALL fields comprehensively — do not leave fields empty when you can infer reasonable values.
4. LANGUAGE: ALL user-facing text (label, description, system_prompt, prompt, output, explanation) MUST be written in Korean (한국어).

Return JSON format:
{"config": {ALL relevant fields}, "label": "설명적 이름", "description": "이 노드가 하는 일", "explanation": "변경된 필드 한 줄 요약, 예: '모델 설정, 페르소나 프롬프트 작성, 업스트림 참조 추가'"}

Return ONLY valid JSON, no markdown fences, no extra text.
```

**Step 3: Create `prompts/workflow-name.md`**

Copy exact text from `name.go:16-22` (`nameSystemPrompt` var):

```markdown
---
name: workflow-name
description: System prompt for workflow name suggestion
---

You name workflows. Given a workflow definition JSON, produce a short descriptive slug-style name.
Rules:
- lowercase letters and hyphens only
- max 4 words
- descriptive of what the workflow does
Examples: "content-pipeline", "multi-model-compare", "code-review-agent", "research-summarizer"
Respond with ONLY a JSON object: {"name": "the-slug-name"}
```

**Step 4: Create `prompts/html-layout.md`**

Copy exact text from `formatter.go:68-81` (`BaseLayoutConstraints` var):

```markdown
---
name: html-layout
description: Base layout constraints for HTML output generation via LLM
---

You are an AI Web Developer. Your task is to generate a single, self-contained HTML document for rendering in an iframe, based on the provided content data from a workflow.

**Libraries:**
* Use Tailwind for CSS via CDN: `<script src="https://cdn.tailwindcss.com"></script>`
* Google Fonts are allowed for typography imports.
* **Tailwind Configuration**: Extend the Tailwind configuration within a `<script>` block to define custom font families and color palettes that match the theme.

**Constraints:**
* The output must be a complete and valid HTML document with no placeholder content.
* **Media Restriction:** ONLY use media URLs that are explicitly present in the input data. Do NOT generate or hallucinate any media URLs.
* **Render All Media:** You MUST render ALL media (images, videos, audio) that are present in the data. Every provided media URL must appear in the final HTML output.
* **Navigation Restriction:** Do NOT generate unneeded fake links or buttons to sub-pages (e.g. "About", "Contact", "Learn More") unless the data explicitly calls for them.
* **Footer Restriction:** NEVER generate any footer content, including legal footers like "All rights reserved" or "Copyright".
* Output ONLY the HTML document, no explanation or markdown fences.
```

**Step 5: Verify files are loaded by registry**

Run: `go build ./...`
Expected: PASS — the `//go:embed prompts/*.md` glob in `skills.go:15` automatically picks up new files

**Step 6: Commit**

```
feat(skills): add 4 prompt skill files for content-analyze, node-configure, workflow-name, html-layout
```

---

### Task 3: Wire `content_collector.go` to use skills registry

**Files:**
- Modify: `internal/services/content_collector.go:23-46, 226, 302-318`
- Modify: `cmd/upal/main.go:291-297`

**Step 1: Add skills field to ContentCollector**

In `internal/services/content_collector.go`, add `skills` field and constructor parameter:

```go
import (
	// ... existing imports ...
	"github.com/soochol/upal/internal/skills"
)

type ContentCollector struct {
	contentSvc   *ContentSessionService
	collectExec  *CollectStageExecutor
	workflowSvc  *WorkflowService
	workflowRepo repository.WorkflowRepository
	resolver     ports.LLMResolver
	generator    ports.WorkflowGenerator
	skills       skills.Provider
}

func NewContentCollector(
	contentSvc *ContentSessionService,
	collectExec *CollectStageExecutor,
	workflowSvc *WorkflowService,
	workflowRepo repository.WorkflowRepository,
	resolver ports.LLMResolver,
	skills skills.Provider,
) *ContentCollector {
	return &ContentCollector{
		contentSvc:   contentSvc,
		collectExec:  collectExec,
		workflowSvc:  workflowSvc,
		workflowRepo: workflowRepo,
		resolver:     resolver,
		skills:       skills,
	}
}
```

**Step 2: Replace hardcoded system prompt in `buildAnalysisPrompt`**

In `runAnalysis()`, pass the skill prompt to `buildAnalysisPrompt`:

```go
// In runAnalysis(), change line 226 from:
systemPrompt, userPrompt := buildAnalysisPrompt(pipeline, fetches, allWorkflows)
// To:
systemPrompt, userPrompt := buildAnalysisPrompt(c.skills.GetPrompt("content-analyze"), pipeline, fetches, allWorkflows)
```

Update `buildAnalysisPrompt` signature and remove the hardcoded string:

```go
func buildAnalysisPrompt(systemPromptBase string, pipeline *upal.Pipeline, fetches []*upal.SourceFetch, workflows []*upal.WorkflowDefinition) (systemPrompt, userPrompt string) {
	systemPrompt = systemPromptBase
	// ... rest of function unchanged (starts at var b strings.Builder) ...
```

Delete the old hardcoded `systemPrompt = \`You are a content analyst...` block (lines 302-318).

**Step 3: Update `main.go` wiring**

In `cmd/upal/main.go`, add `skillReg` to `NewContentCollector` call. Change lines 291-297 from:

```go
collector = services.NewContentCollector(
    contentSvc,
    services.NewCollectStageExecutor(),
    workflowSvc,
    repo,
    resolver,
)
```

To:

```go
collector = services.NewContentCollector(
    contentSvc,
    services.NewCollectStageExecutor(),
    workflowSvc,
    repo,
    resolver,
    skillReg,
)
```

Note: `skillReg` is created at line 302, but `collector` is created at line 291. Move the `skillReg := skills.New()` line to BEFORE the collector block (e.g. around line 288, before `var collector`). Also move `srv.SetSkills(skillReg)` accordingly.

**Step 4: Build and test**

Run: `go build ./...`
Expected: PASS

Run: `go test ./internal/services/... -v -race -count=1`
Expected: PASS

**Step 5: Commit**

```
refactor: extract content-analyze prompt to skill file
```

---

### Task 4: Wire `api/configure.go` to use `GetPrompt("node-configure")`

**Files:**
- Modify: `internal/api/configure.go:50-69, 131`

**Step 1: Remove `configureBasePrompt` var and use `GetPrompt`**

Delete the `configureBasePrompt` var declaration (lines 50-69).

In `configureNode()`, change line 131 from:

```go
sysPrompt := configureBasePrompt
```

To:

```go
sysPrompt := ""
if s.skills != nil {
    sysPrompt = s.skills.GetPrompt("node-configure")
}
```

The existing node-type skill injection block (lines 132-135) stays unchanged — it appends to `sysPrompt`.

**Step 2: Build and test**

Run: `go build ./...`
Expected: PASS

Run: `go test ./internal/api/... -v -race -count=1`
Expected: PASS

**Step 3: Commit**

```
refactor: extract node-configure prompt to skill file
```

---

### Task 5: Wire `api/name.go` to use `GetPrompt("workflow-name")`

**Files:**
- Modify: `internal/api/name.go:16-22, 56`

**Step 1: Remove `nameSystemPrompt` var and use `GetPrompt`**

Delete the `nameSystemPrompt` var declaration (lines 16-22).

In `suggestWorkflowName()`, change line 56 from:

```go
SystemInstruction: genai.NewContentFromText(nameSystemPrompt, genai.RoleUser),
```

To:

```go
SystemInstruction: genai.NewContentFromText(s.skills.GetPrompt("workflow-name"), genai.RoleUser),
```

**Step 2: Build and test**

Run: `go build ./...`
Expected: PASS

Run: `go test ./internal/api/... -v -race -count=1`
Expected: PASS

**Step 3: Commit**

```
refactor: extract workflow-name prompt to skill file
```

---

### Task 6: Wire `output/formatter.go` to use `GetPrompt("html-layout")`

**Files:**
- Modify: `internal/output/formatter.go:66-81, 87, 106`
- Modify: `internal/agents/output_builder.go:26`
- Modify: `internal/agents/registry.go:28-32`
- Modify: `internal/services/workflow.go:53`
- Modify: `cmd/upal/main.go` (BuildDeps wiring)

**Step 1: Add `basePrompt` parameter to `NewFormatter`**

In `internal/output/formatter.go`:
- Delete the `BaseLayoutConstraints` var (lines 66-81) and its comment
- Change `NewFormatter` signature to accept `basePrompt string`:

```go
func NewFormatter(config map[string]any, llms map[string]adkmodel.LLM, resolveLLM func(string, map[string]adkmodel.LLM) (adkmodel.LLM, string), basePrompt string) Formatter {
```

- Change line 106 from:

```go
SystemPrompt: BaseLayoutConstraints + "\n\n" + systemPrompt,
```

To:

```go
SystemPrompt: basePrompt + "\n\n" + systemPrompt,
```

**Step 2: Add `HTMLLayoutPrompt` to `BuildDeps`**

In `internal/agents/registry.go`, add field:

```go
type BuildDeps struct {
	LLMs             map[string]adkmodel.LLM
	ToolReg          *tools.Registry
	OutputDir        string // directory for saving media outputs (audio, video)
	HTMLLayoutPrompt string // base prompt for HTML output formatting
}
```

**Step 3: Update `output_builder.go` to pass `HTMLLayoutPrompt`**

In `internal/agents/output_builder.go`, change line 26 from:

```go
formatter := output.NewFormatter(nd.Config, deps.LLMs, resolveLLM)
```

To:

```go
formatter := output.NewFormatter(nd.Config, deps.LLMs, resolveLLM, deps.HTMLLayoutPrompt)
```

**Step 4: Wire in `workflow.go` and `main.go`**

In `internal/services/workflow.go`, change `NewWorkflowService` signature to accept `htmlLayoutPrompt string`:

```go
func NewWorkflowService(
	repo repository.WorkflowRepository,
	llms map[string]adkmodel.LLM,
	sessionService session.Service,
	toolReg *tools.Registry,
	nodeRegistry *agents.NodeRegistry,
	outputDir string,
	htmlLayoutPrompt string,
) *WorkflowService {
	return &WorkflowService{
		repo:           repo,
		llms:           llms,
		sessionService: sessionService,
		toolReg:        toolReg,
		nodeRegistry:   nodeRegistry,
		buildDeps:      agents.BuildDeps{LLMs: llms, ToolReg: toolReg, OutputDir: outputDir, HTMLLayoutPrompt: htmlLayoutPrompt},
	}
}
```

In `cmd/upal/main.go`, update `NewWorkflowService` call (line 169):

```go
workflowSvc := services.NewWorkflowService(repo, llms, sessionService, toolReg, nodeReg, outputDir, skillReg.GetPrompt("html-layout"))
```

Note: `skillReg` must be created BEFORE `workflowSvc`. After Task 3's changes, `skillReg` is already created early. Verify the order.

**Step 5: Build and test**

Run: `go build ./...`
Expected: PASS

Run: `go test ./... -v -race -count=1`
Expected: PASS

**Step 6: Commit**

```
refactor: extract html-layout prompt to skill file
```

---

### Task 7: Final verification and cleanup

**Step 1: Verify no hardcoded prompts remain**

Run: `grep -rn 'configureBasePrompt\|nameSystemPrompt\|BaseLayoutConstraints' internal/`
Expected: No matches

Run: `grep -rn 'You are a content analyst' internal/`
Expected: No matches in `.go` files (only in `prompts/content-analyze.md`)

**Step 2: Verify all prompts load correctly**

Run: `go test ./internal/skills/... -v -race -count=1`
Expected: PASS

**Step 3: Full test suite**

Run: `go test ./... -v -race -count=1`
Expected: PASS

Run: `go build ./...`
Expected: PASS

**Step 4: Commit**

```
chore: verify skill prompt extraction complete
```
