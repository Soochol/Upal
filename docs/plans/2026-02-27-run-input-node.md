# Run Input Node Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `run_input` node type that receives pipeline run briefs, separate from user input nodes.

**Architecture:** New node type `run_input` follows the existing builder pattern. Backend builder reads from `__run_input__` prefix in session state. Pipeline's `buildProductionInputsV2` targets `run_input` nodes instead of `input` nodes. Frontend registers the node with its own icon, color, and read-only editor.

**Tech Stack:** Go (backend builder, registry), React/TypeScript (frontend node registration), Tailwind CSS (styling)

---

### Task 1: Backend — Add NodeType constant

**Files:**
- Modify: `internal/upal/workflow.go:5-11`

**Step 1: Add constant**

```go
const (
	NodeTypeInput    NodeType = "input"
	NodeTypeRunInput NodeType = "run_input"
	NodeTypeAgent    NodeType = "agent"
	NodeTypeOutput   NodeType = "output"
	NodeTypeAsset    NodeType = "asset"
	NodeTypeTool     NodeType = "tool"
)
```

---

### Task 2: Backend — Create RunInputNodeBuilder

**Files:**
- Create: `internal/agents/run_input_builder.go`

**Step 1: Create builder**

Identical to `InputNodeBuilder` but reads from `__run_input__` prefix instead of `__user_input__`.

```go
package agents

import (
	"fmt"
	"iter"

	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type RunInputNodeBuilder struct{}

func (b *RunInputNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeRunInput }

func (b *RunInputNodeBuilder) Build(nd *upal.NodeDefinition, _ BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Run input node %s — receives pipeline brief", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()
				key := "__run_input__" + nodeID
				val, err := state.Get(key)
				if err != nil || val == nil {
					val = ""
				}

				_ = state.Set(nodeID, val)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(fmt.Sprintf("%v", val))},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = val
				yield(event, nil)
			}
		},
	})
}
```

---

### Task 3: Backend — Register builder in DefaultRegistry

**Files:**
- Modify: `internal/agents/registry.go:69-76`

**Step 1: Add to DefaultRegistry**

```go
func DefaultRegistry() *NodeRegistry {
	r := NewNodeRegistry()
	r.Register(&InputNodeBuilder{})
	r.Register(&RunInputNodeBuilder{})
	r.Register(&OutputNodeBuilder{})
	r.Register(&LLMNodeBuilder{})
	r.Register(&ToolNodeBuilder{})
	return r
}
```

---

### Task 4: Backend — Update WorkflowService.Run for run_input state

**Files:**
- Modify: `internal/services/workflow.go:100-103`

**Step 1: Support both input prefixes**

The `inputs` map now has two kinds of keys. We need to distinguish between user input and run input when setting up session state.

```go
inputState := make(map[string]any)
for k, v := range inputs {
	inputState["__user_input__"+k] = v
}
if runInputs, ok := inputs["__run_inputs__"].(map[string]any); ok {
	for k, v := range runInputs {
		inputState["__run_input__"+k] = v
	}
	delete(inputState, "__user_input____run_inputs__")
}
```

---

### Task 5: Backend — Update buildProductionInputsV2 to target run_input nodes

**Files:**
- Modify: `internal/services/content_collector.go` (buildProductionInputsV2)

**Step 1: Target run_input nodes, keep input node fallback**

```go
// At the end of buildProductionInputsV2, replace the input population loop:
inputs := make(map[string]any)

// Primary: populate run_input nodes with the brief.
runInputs := make(map[string]any)
for _, node := range wf.Nodes {
	if node.Type == upal.NodeTypeRunInput {
		runInputs[node.ID] = brief
	}
}

if len(runInputs) > 0 {
	inputs["__run_inputs__"] = runInputs
} else {
	// Fallback: if no run_input node, populate user input nodes (backward compat).
	for _, node := range wf.Nodes {
		if node.Type == upal.NodeTypeInput {
			inputs[node.ID] = brief
		}
	}
}

return inputs
```

---

### Task 6: Frontend — Add NodeType + config type

**Files:**
- Modify: `web/src/entities/node/types.ts:3`
- Modify: `web/src/shared/lib/nodeConfigs.ts`

**Step 1: Add type**

```typescript
export type NodeType = 'input' | 'run_input' | 'agent' | 'output' | 'asset' | 'tool'
```

**Step 2: Add config type**

```typescript
export type RunInputNodeConfig = {
  description?: string
}
```

---

### Task 7: Frontend — Add CSS variables

**Files:**
- Modify: `web/src/index.css`

**Step 1: Add run_input color tokens**

In `:root` (light mode) after `--node-input-foreground`:
```css
--node-run-input: oklch(0.65 0.15 280);
--node-run-input-foreground: oklch(0.985 0 0);
```

In `.dark` after `--node-input-foreground`:
```css
--node-run-input: oklch(0.65 0.15 280);
--node-run-input-foreground: oklch(0.985 0 0);
```

In `@theme inline` section, add:
```css
--color-node-run-input: var(--node-run-input);
--color-node-run-input-foreground: var(--node-run-input-foreground);
```

---

### Task 8: Frontend — Register node in registry

**Files:**
- Modify: `web/src/entities/node/model/registry.ts`

**Step 1: Add run_input registration after input registration**

```typescript
import { Inbox, Bot, ArrowRightFromLine, FileBox, Wrench, Zap } from 'lucide-react'

registerNode({
  type: 'run_input',
  label: 'Run Input',
  description: 'Receives data from pipeline runs',
  icon: Zap,
  border: 'border-node-run-input/20',
  borderSelected: 'border-node-run-input/60',
  headerBg: 'bg-node-run-input/10',
  accent: 'bg-node-run-input text-node-run-input-foreground',
  glow: 'shadow-[0_0_20px_var(--color-node-run-input)/0.25]',
  paletteBg: 'bg-node-run-input/10 text-node-run-input border-node-run-input/20 hover:bg-node-run-input/20',
  cssVar: 'var(--node-run-input)',
})
```

---

### Task 9: Frontend — Create RunInputNodeEditor

**Files:**
- Create: `web/src/features/edit-node/ui/RunInputNodeEditor.tsx`

**Step 1: Read-only editor showing description**

```tsx
import type { RunInputNodeConfig } from '@/shared/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function RunInputNodeEditor({ }: NodeEditorFieldProps<RunInputNodeConfig>) {
  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">
        This node receives data from pipeline runs automatically.
        When the workflow is triggered by a pipeline, the run brief is injected here.
      </p>
    </div>
  )
}
```

---

### Task 10: Frontend — Register editor + update NodePalette colors

**Files:**
- Modify: `web/src/features/edit-node/model/registerEditors.ts`
- Modify: `web/src/widgets/node-palette/ui/NodePalette.tsx`

**Step 1: Register editor**

```typescript
import { RunInputNodeEditor } from '../ui/RunInputNodeEditor'
registerNodeEditor('run_input', RunInputNodeEditor)
```

**Step 2: Add palette colors**

```typescript
const iconColor: Record<string, string> = {
  '--node-input': 'text-node-input',
  '--node-run-input': 'text-node-run-input',
  ...
}

const hoverAccent: Record<string, string> = {
  '--node-input': 'hover:bg-node-input/10',
  '--node-run-input': 'hover:bg-node-run-input/10',
  ...
}
```

---

### Task 11: Backend — Add run_input-node skill

**Files:**
- Create: `internal/skills/nodes/run_input-node.md`

---

### Task 12: Build & test

Run: `go build ./... && cd web && npx tsc -b && cd .. && go test ./... -v -race`

---
