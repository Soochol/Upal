# Output Layout Modes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Google Opal-style output node layout modes — "Manual layout" (plain text, current behavior) and "Webpage with auto-layout" (LLM generates styled HTML from upstream results).

**Architecture:** The output node gains a `display_mode` config field (`"manual"` | `"auto-layout"`). In auto-layout mode, `buildOutputAgent` makes an LLM call using the generator's default model to transform aggregated upstream results into a self-contained HTML page. The frontend Preview panel detects HTML content and renders it in a sandboxed iframe.

**Tech Stack:** Go (ADK LLM API), React, TypeScript, Tailwind CSS, shadcn/ui Select component

---

### Task 1: Backend — Pass LLMs to buildOutputAgent

**Files:**
- Modify: `internal/agents/builders.go` (lines 22-37, 76-123)
- Modify: `internal/agents/dag.go` (lines 33-36)

**Step 1: Update BuildAgent signature to pass LLMs map to buildOutputAgent**

In `builders.go`, change `buildOutputAgent(nd)` to `buildOutputAgent(nd, llms)`:

```go
case upal.NodeTypeOutput:
    return buildOutputAgent(nd, llms)
```

Update `buildOutputAgent` signature:

```go
func buildOutputAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM) (agent.Agent, error) {
```

**Step 2: Verify DAG builder already passes llms**

In `dag.go` line 35, `BuildAgent(nd, llms, toolReg)` already passes llms — no change needed.

**Step 3: Run backend tests**

Run: `make test`
Expected: PASS (no behavior change yet)

---

### Task 2: Backend — Implement auto-layout LLM call in buildOutputAgent

**Files:**
- Modify: `internal/agents/builders.go` (buildOutputAgent function)

**Step 1: Add auto-layout system prompt and LLM call logic**

Replace the `buildOutputAgent` function body to check `display_mode` config. When `"auto-layout"`, make an LLM call to generate HTML:

```go
func buildOutputAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM) (agent.Agent, error) {
	nodeID := nd.ID
	displayMode, _ := nd.Config["display_mode"].(string)
	layoutModel, _ := nd.Config["layout_model"].(string)

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Output node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				// Collect all non-__ prefixed state values (existing logic)
				var keys []string
				for k := range state.All() {
					if !strings.HasPrefix(k, "__") {
						keys = append(keys, k)
					}
				}
				sort.Strings(keys)

				var parts []string
				for _, k := range keys {
					if k == nodeID {
						continue
					}
					v, err := state.Get(k)
					if err != nil || v == nil {
						continue
					}
					parts = append(parts, fmt.Sprintf("%v", v))
				}

				result := strings.Join(parts, "\n\n")

				// Auto-layout: use LLM to generate styled HTML page
				if displayMode == "auto-layout" && llms != nil {
					htmlResult, err := generateAutoLayout(ctx, result, layoutModel, llms)
					if err == nil && htmlResult != "" {
						result = htmlResult
					}
					// On error, fall back to plain text result
				}

				_ = state.Set(nodeID, result)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(result)},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = result
				yield(event, nil)
			}
		},
	})
}
```

**Step 2: Add generateAutoLayout helper function**

```go
var autoLayoutSystemPrompt = `You are a web page layout designer. Given content data from a workflow, create a beautiful, self-contained HTML page that presents the information clearly.

Rules:
- Output a COMPLETE HTML document with inline CSS (no external dependencies)
- Use modern, clean design with good typography and spacing
- Use a professional color scheme (subtle grays, blues, or the appropriate mood)
- Make it responsive with max-width container
- Structure the content logically with headings, sections, and visual hierarchy
- If the content contains code, wrap it in <pre><code> blocks with styling
- If the content has multiple sections, use cards or bordered sections
- Do NOT include any JavaScript
- Output ONLY the HTML document, no explanation or markdown fences`

func generateAutoLayout(ctx agent.InvocationContext, content string, layoutModel string, llms map[string]adkmodel.LLM) (string, error) {
	// Resolve LLM: use layout_model config, or fall back to first available
	var llm adkmodel.LLM
	var modelName string

	if layoutModel != "" {
		parts := strings.SplitN(layoutModel, "/", 2)
		if len(parts) == 2 {
			if l, ok := llms[parts[0]]; ok {
				llm = &namedLLM{LLM: l, name: parts[1]}
				modelName = parts[1]
			}
		}
	}

	// Fallback: use the first available LLM
	if llm == nil {
		for providerName, l := range llms {
			llm = l
			modelName = providerName
			break
		}
	}

	if llm == nil {
		return "", fmt.Errorf("no LLM available for auto-layout")
	}

	req := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(autoLayoutSystemPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(
				fmt.Sprintf("Create a styled HTML page presenting the following content:\n\n%s", content),
				genai.RoleUser,
			),
		},
	}

	var resp *adkmodel.LLMResponse
	for r, err := range llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("auto-layout LLM call failed: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty response from LLM")
	}

	var text string
	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			text += p.Text
		}
	}

	// Strip markdown code fences if present
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```html")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	return text, nil
}
```

**Step 3: Run backend tests**

Run: `make test`
Expected: PASS

---

### Task 3: Frontend — Add display_mode config UI to NodeEditor

**Files:**
- Modify: `web/src/components/editor/nodes/NodeEditor.tsx`

**Step 1: Add output node configuration section**

After the `data.nodeType === 'tool'` block (~line 173), add an output section:

```tsx
{data.nodeType === 'output' && (
  <>
    <Separator />
    <div className="space-y-1">
      <Label className="text-xs">Display Mode</Label>
      <Select
        value={(config.display_mode as string) ?? 'manual'}
        onValueChange={(v) => setConfig('display_mode', v)}
      >
        <SelectTrigger className="h-7 text-xs w-full" size="sm">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="manual" className="text-xs">
            <div>
              <div className="font-medium">Manual layout</div>
              <div className="text-muted-foreground">Content displayed as-is</div>
            </div>
          </SelectItem>
          <SelectItem value="auto-layout" className="text-xs">
            <div>
              <div className="font-medium">Webpage with auto-layout</div>
              <div className="text-muted-foreground">Layout generated by AI</div>
            </div>
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    {(config.display_mode as string) === 'auto-layout' && (
      <div className="space-y-1">
        <Label className="text-xs">Layout Model</Label>
        <Select
          value={(config.layout_model as string) ?? ''}
          onValueChange={(v) => setConfig('layout_model', v)}
        >
          <SelectTrigger className="h-7 text-xs w-full" size="sm">
            <SelectValue placeholder="Default model" />
          </SelectTrigger>
          <SelectContent>
            {Object.entries(modelsByProvider).map(([provider, providerModels]) => (
              <SelectGroup key={provider}>
                <SelectLabel>{provider}</SelectLabel>
                {providerModels.map((m) => (
                  <SelectItem key={m.id} value={m.id} className="text-xs">
                    {m.name}
                  </SelectItem>
                ))}
              </SelectGroup>
            ))}
          </SelectContent>
        </Select>
      </div>
    )}
  </>
)}
```

**Step 2: Type-check frontend**

Run: `make test-frontend`
Expected: PASS

---

### Task 4: Frontend — Render HTML in PanelPreview with sandboxed iframe

**Files:**
- Modify: `web/src/components/panel/PanelPreview.tsx`

**Step 1: Add HTML detection and iframe rendering**

Replace PanelPreview to detect HTML content (starts with `<!DOCTYPE` or `<html`) and render in sandboxed iframe:

```tsx
import { useWorkflowStore } from '@/stores/workflowStore'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Eye } from 'lucide-react'

function isHtmlContent(text: string): boolean {
  const trimmed = text.trimStart().toLowerCase()
  return trimmed.startsWith('<!doctype') || trimmed.startsWith('<html')
}

function HtmlPreview({ html }: { html: string }) {
  return (
    <iframe
      srcDoc={html}
      sandbox="allow-same-origin"
      className="w-full h-full border-0 rounded-lg bg-white"
      title="Auto-layout preview"
    />
  )
}

export function PanelPreview() {
  const runEvents = useWorkflowStore((s) => s.runEvents)
  const sessionState = useWorkflowStore((s) => s.sessionState)

  const doneEvent = runEvents.find((e) => e.type === 'done')
  const outputEvents = runEvents.filter(
    (e) => e.type === 'node.completed' && e.data.output,
  )

  // Check session state for HTML output from auto-layout output nodes
  const htmlOutput = Object.values(sessionState).find(
    (v) => typeof v === 'string' && isHtmlContent(v),
  ) as string | undefined

  if (!doneEvent && outputEvents.length === 0 && !htmlOutput) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-6">
        <Eye className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm text-center">
          Run a workflow to see results here.
        </p>
      </div>
    )
  }

  // If there's HTML output, show it full-height in an iframe
  if (htmlOutput) {
    return (
      <div className="h-full p-2">
        <HtmlPreview html={htmlOutput} />
      </div>
    )
  }

  // Fallback: original text-based preview
  return (
    <ScrollArea className="h-full">
      <div className="p-4 space-y-4">
        {outputEvents.map((event, i) => (
          <div key={i} className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground">
              {(event.data.node_id as string) || `Step ${i + 1}`}
            </p>
            <div className="rounded-lg border border-border bg-card p-3 text-sm whitespace-pre-wrap">
              {typeof event.data.output === 'string'
                ? event.data.output
                : JSON.stringify(event.data.output, null, 2)}
            </div>
          </div>
        ))}
        {doneEvent && (
          <div className="space-y-1">
            <p className="text-xs font-medium text-node-output">Final Result</p>
            <div className="rounded-lg border border-node-output/30 bg-node-output/5 p-3 text-sm whitespace-pre-wrap">
              {typeof doneEvent.data.result === 'string'
                ? doneEvent.data.result
                : JSON.stringify(doneEvent.data, null, 2)}
            </div>
          </div>
        )}
      </div>
    </ScrollArea>
  )
}
```

**Step 2: Type-check frontend**

Run: `make test-frontend`
Expected: PASS

---

### Task 5: Build verification

**Step 1: Full build**

Run: `make build`
Expected: PASS — both backend and frontend compile cleanly

**Step 2: Manual verification**

1. Start dev servers (`make dev-backend` + `make dev-frontend`)
2. Create a workflow with an output node
3. Click on output node → Properties panel should show "Display Mode" dropdown
4. Select "Webpage with auto-layout" → "Layout Model" picker appears
5. Run workflow → Preview panel shows styled HTML page in iframe
6. Switch to "Manual layout" → runs show plain text as before
