# Quick Start Templates Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform 3 placeholder template cards into 6 fully functional quick-start templates with real workflow definitions, editor read-only mode, and Remix-to-edit UX.

**Architecture:** Templates are frontend-only `WorkflowDefinition` objects loaded into the Zustand workflow store with an `isTemplate` flag. The editor renders read-only when this flag is set (no drag, connect, delete, palette, prompt bar). A Remix button copies the template to a new editable workflow.

**Tech Stack:** React 19, TypeScript, Zustand, @xyflow/react, Tailwind v4, Lucide icons

---

### Task 1: Add `isTemplate` flag to workflow store

**Files:**
- Modify: `web/src/entities/workflow/model/store.ts:20-45`

**Step 1: Add `isTemplate` + `setIsTemplate` to `WorkflowState` type**

In `web/src/entities/workflow/model/store.ts`, add to the `WorkflowState` type (after line 38 `setOriginalName`):

```typescript
  // Template mode (read-only, no auto-save)
  isTemplate: boolean
  setIsTemplate: (v: boolean) => void
```

**Step 2: Add default values in the store creator**

In the `create<WorkflowState>` body (after `originalName: ''` around line 56), add:

```typescript
  isTemplate: false,
  setIsTemplate: (v) => set({ isTemplate: v }),
```

**Step 3: Verify type-check passes**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit`
Expected: PASS (no new errors)

**Step 4: Commit**

```bash
git add web/src/entities/workflow/model/store.ts
git commit -m "feat(templates): add isTemplate flag to workflow store"
```

---

### Task 2: Create template definitions

**Files:**
- Create: `web/src/shared/lib/templates.ts`

**Step 1: Create `web/src/shared/lib/templates.ts`**

```typescript
import { Search, FileText, BarChart3, GitBranch, Globe, PenLine } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { WorkflowDefinition } from '@/entities/workflow'

export type TemplateDefinition = {
  id: string
  title: string
  description: string
  icon: LucideIcon
  color: string        // Tailwind classes for card icon bg+text
  tags: string[]
  difficulty: 'Beginner' | 'Intermediate'
  workflow: WorkflowDefinition
}

export const templates: TemplateDefinition[] = [
  {
    id: 'basic-rag-agent',
    title: 'Basic RAG Agent',
    description: 'Retrieve relevant web content and generate contextual responses.',
    icon: Search,
    color: 'bg-teal-500/10 text-teal-500',
    tags: ['RAG', 'Web'],
    difficulty: 'Beginner',
    workflow: {
      name: 'Basic RAG Agent',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'User Query',
            prompt: 'Enter your question',
            description: 'Accepts the user question to research',
          },
        },
        {
          id: 'rag_agent',
          type: 'agent',
          config: {
            label: 'RAG Agent',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt:
              'You are a helpful research assistant. Use the get_webpage tool to retrieve relevant information, then synthesize a clear, well-sourced answer.',
            prompt: '{{user_input}}',
            tools: ['get_webpage'],
            description: 'Fetches web content and generates a contextual answer',
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Response',
            description: 'Final answer with sourced information',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'rag_agent' },
        { from: 'rag_agent', to: 'output' },
      ],
    },
  },
  {
    id: 'content-summarizer',
    title: 'Content Summarizer',
    description: 'Extract key points from long articles or documents.',
    icon: FileText,
    color: 'bg-blue-500/10 text-blue-500',
    tags: ['NLP', 'Writing'],
    difficulty: 'Beginner',
    workflow: {
      name: 'Content Summarizer',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Content Input',
            prompt: 'Paste article or document text',
            description: 'The text content to summarize',
          },
        },
        {
          id: 'summarizer',
          type: 'agent',
          config: {
            label: 'Summarizer',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt:
              'You are an expert summarizer. Extract the key points, main arguments, and conclusions from the provided text. Format your summary with bullet points for key takeaways and a brief narrative overview.',
            prompt: 'Summarize the following content:\n\n{{user_input}}',
            tools: [],
            description: 'Extracts key points and generates a structured summary',
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Summary',
            description: 'Structured summary with key takeaways',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'summarizer' },
        { from: 'summarizer', to: 'output' },
      ],
    },
  },
  {
    id: 'sentiment-analyzer',
    title: 'Sentiment Analyzer',
    description: 'Classify tone and sentiment from text input.',
    icon: BarChart3,
    color: 'bg-rose-500/10 text-rose-500',
    tags: ['NLP', 'Analysis'],
    difficulty: 'Beginner',
    workflow: {
      name: 'Sentiment Analyzer',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Text Input',
            prompt: 'Enter text to analyze',
            description: 'Text for sentiment analysis',
          },
        },
        {
          id: 'analyzer',
          type: 'agent',
          config: {
            label: 'Sentiment Analyzer',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt:
              'You are a sentiment analysis expert. Analyze the given text and return a JSON object with: sentiment (positive/negative/neutral/mixed), confidence (0-1), key_phrases (array of influential phrases), and explanation (brief reasoning).',
            prompt: 'Analyze the sentiment of this text:\n\n{{user_input}}',
            tools: [],
            description: 'Classifies sentiment and extracts key emotional signals',
            output_extract: { mode: 'json' as const, key: 'sentiment_result' },
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Analysis Result',
            description: 'Structured sentiment analysis result',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'analyzer' },
        { from: 'analyzer', to: 'output' },
      ],
    },
  },
  {
    id: 'data-pipeline',
    title: 'Data Pipeline',
    description: 'Process and classify incoming data streams.',
    icon: GitBranch,
    color: 'bg-indigo-500/10 text-indigo-500',
    tags: ['Data', 'Classification'],
    difficulty: 'Intermediate',
    workflow: {
      name: 'Data Pipeline',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Raw Data',
            prompt: 'Paste raw data (JSON, CSV, or plain text)',
            description: 'Raw data to be processed and classified',
          },
        },
        {
          id: 'classifier',
          type: 'agent',
          config: {
            label: 'Data Classifier',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt:
              'You are a data processing pipeline. Parse the input data, identify its format and structure, classify each record into categories, and output a clean structured JSON result with: format_detected, record_count, categories (array), and classified_records (array of objects with original data + assigned category + confidence).',
            prompt: 'Process and classify the following data:\n\n{{user_input}}',
            tools: [],
            description: 'Parses, classifies, and structures raw data',
            output_extract: { mode: 'json' as const, key: 'pipeline_result' },
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Classified Data',
            description: 'Structured and classified data output',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'classifier' },
        { from: 'classifier', to: 'output' },
      ],
    },
  },
  {
    id: 'web-research-agent',
    title: 'Web Research Agent',
    description: 'Search the web, gather sources, and synthesize findings.',
    icon: Globe,
    color: 'bg-violet-500/10 text-violet-500',
    tags: ['Research', 'Multi-step'],
    difficulty: 'Intermediate',
    workflow: {
      name: 'Web Research Agent',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Research Topic',
            prompt: 'Enter your research question or topic',
            description: 'The topic to research across the web',
          },
        },
        {
          id: 'researcher',
          type: 'agent',
          config: {
            label: 'Web Researcher',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt:
              'You are a thorough web researcher. Use the available tools to find relevant sources on the given topic. Retrieve at least 2-3 different web pages. For each source, note the URL and extract the key relevant information.',
            prompt: 'Research the following topic and gather information from multiple sources:\n\n{{user_input}}',
            tools: ['get_webpage', 'http_request'],
            description: 'Searches and retrieves content from multiple web sources',
          },
        },
        {
          id: 'synthesizer',
          type: 'agent',
          config: {
            label: 'Synthesizer',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt:
              'You are a research synthesizer. Take the raw research findings and create a well-organized research brief. Include: an executive summary, key findings organized by theme, source citations, and areas of consensus or disagreement between sources.',
            prompt: 'Synthesize these research findings into a comprehensive brief:\n\n{{researcher}}',
            tools: [],
            description: 'Synthesizes raw research into a structured brief',
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Research Brief',
            description: 'Comprehensive research brief with citations',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'researcher' },
        { from: 'researcher', to: 'synthesizer' },
        { from: 'synthesizer', to: 'output' },
      ],
    },
  },
  {
    id: 'multi-step-writer',
    title: 'Multi-Step Writer',
    description: 'Generate structured long-form content in stages.',
    icon: PenLine,
    color: 'bg-amber-500/10 text-amber-500',
    tags: ['Writing', 'Chain'],
    difficulty: 'Intermediate',
    workflow: {
      name: 'Multi-Step Writer',
      version: 1,
      nodes: [
        {
          id: 'user_input',
          type: 'input',
          config: {
            label: 'Writing Brief',
            prompt: 'Describe what you want written (topic, audience, tone, length)',
            description: 'The writing brief with topic and requirements',
          },
        },
        {
          id: 'outliner',
          type: 'agent',
          config: {
            label: 'Outliner',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt:
              'You are a content strategist. Create a detailed outline for the requested piece. Include: a compelling title, section headers, key points for each section, suggested word counts per section, and the overall narrative arc.',
            prompt: 'Create a detailed outline for:\n\n{{user_input}}',
            tools: [],
            description: 'Creates a structured outline with sections and key points',
          },
        },
        {
          id: 'writer',
          type: 'agent',
          config: {
            label: 'Writer',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt:
              'You are a skilled writer. Follow the provided outline exactly and write the full content. Match the requested tone and audience. Write engaging, clear prose with smooth transitions between sections.',
            prompt: 'Write the full content following this outline:\n\n{{outliner}}',
            tools: [],
            description: 'Writes the full draft following the outline',
          },
        },
        {
          id: 'editor_agent',
          type: 'agent',
          config: {
            label: 'Editor',
            model: 'anthropic/claude-sonnet-4-20250514',
            system_prompt:
              'You are a professional editor. Review the draft for clarity, grammar, flow, and engagement. Fix any issues and polish the text. Preserve the author\'s voice while improving quality. Output the final edited version.',
            prompt: 'Edit and polish this draft:\n\n{{writer}}',
            tools: [],
            description: 'Reviews, polishes, and finalizes the written content',
          },
        },
        {
          id: 'output',
          type: 'output',
          config: {
            label: 'Final Content',
            description: 'Polished, publication-ready content',
          },
        },
      ],
      edges: [
        { from: 'user_input', to: 'outliner' },
        { from: 'outliner', to: 'writer' },
        { from: 'writer', to: 'editor_agent' },
        { from: 'editor_agent', to: 'output' },
      ],
    },
  },
]
```

**Step 2: Verify type-check passes**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/shared/lib/templates.ts
git commit -m "feat(templates): add 6 quick-start template definitions"
```

---

### Task 3: Add `readOnly` prop to Canvas

**Files:**
- Modify: `web/src/widgets/workflow-canvas/ui/Canvas.tsx`

**Step 1: Add `readOnly` to CanvasProps type**

At `web/src/widgets/workflow-canvas/ui/Canvas.tsx:28-35`, update:

```typescript
type CanvasProps = {
  onAddFirstNode: () => void
  onDropNode: (type: string, position: { x: number; y: number }) => void
  onPromptSubmit: (description: string) => void
  isGenerating: boolean
  exposeGetViewportCenter?: (fn: () => { x: number; y: number }) => void
  onAddNode: (type: NodeType) => void
  readOnly?: boolean
}
```

**Step 2: Destructure `readOnly` in the Canvas function**

Update the function signature at line 70:

```typescript
export function Canvas({ onAddFirstNode, onDropNode, onPromptSubmit, isGenerating, exposeGetViewportCenter, onAddNode, readOnly }: CanvasProps) {
```

**Step 3: Conditionally render palette, prompt bar, empty state and set ReactFlow read-only props**

In the JSX return (line 191 onwards), update:

```typescript
  return (
    <div className="h-full w-full relative bg-background" onDrop={readOnly ? undefined : onDrop} onDragOver={readOnly ? undefined : onDragOver}>
      {!readOnly && <NodePalette onAddNode={onAddNode} />}
      {isEmpty && !readOnly && (
        <EmptyState onAddNode={onAddFirstNode} />
      )}
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={readOnly ? undefined : onNodesChange}
        onEdgesChange={readOnly ? undefined : onEdgesChange}
        onConnect={readOnly ? undefined : onConnect}
        onConnectEnd={readOnly ? undefined : onConnectEnd}
        isValidConnection={isValidConnection}
        connectionRadius={80}
        nodeTypes={nodeTypes}
        fitView
        className="bg-background"
        multiSelectionKeyCode={readOnly ? null : ['Shift', 'Control', 'Meta']}
        selectionOnDrag={!readOnly}
        selectionMode={SelectionMode.Partial}
        panOnDrag={[1, 2]}
        deleteKeyCode={readOnly ? null : ['Delete', 'Backspace']}
        nodesDraggable={!readOnly}
        nodesConnectable={!readOnly}
        elementsSelectable={!readOnly}
        proOptions={{ hideAttribution: true }}
      >
        {!readOnly && <SelectionGrouper />}
        <Background color="var(--border)" gap={20} size={1} />
        <Controls className="!bg-card !border-border !shadow-sm" />
        {!isEmpty && (
          <MiniMap
            nodeColor="var(--muted)"
            maskColor="var(--background)"
            className="!bg-card !border-border !rounded-lg !shadow-sm"
          />
        )}
      </ReactFlow>
      {!readOnly && (
        <CanvasPromptBar
          onSubmit={onPromptSubmit}
          isGenerating={isGenerating}
          hasNodes={!isEmpty}
        />
      )}
    </div>
  )
```

**Step 4: Verify type-check passes**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/widgets/workflow-canvas/ui/Canvas.tsx
git commit -m "feat(templates): add readOnly prop to Canvas component"
```

---

### Task 4: Update WorkflowHeader for template mode

**Files:**
- Modify: `web/src/widgets/workflow-header/ui/WorkflowHeader.tsx`

**Step 1: Add template props to WorkflowHeaderProps**

```typescript
type WorkflowHeaderProps = {
  workflowName?: string
  onWorkflowNameChange?: (name: string) => void
  saveStatus?: SaveStatus
  onRun?: () => void
  isTemplate?: boolean
  templateName?: string
  onRemix?: () => void
}
```

**Step 2: Update the WorkflowHeader component**

Replace the component body:

```typescript
export function WorkflowHeader({ workflowName, onWorkflowNameChange, saveStatus, onRun, isTemplate, templateName, onRemix }: WorkflowHeaderProps) {
  return (
    <div className="flex items-center justify-between flex-1 gap-3">
      <div className="flex items-center gap-2">
        {isTemplate ? (
          <>
            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-primary/10 border border-primary/20 text-xs font-medium text-primary">
              Template
            </span>
            <span className="text-sm font-semibold text-foreground truncate max-w-[200px]">
              {templateName || workflowName}
            </span>
          </>
        ) : (
          <>
            {workflowName !== undefined && onWorkflowNameChange && (
              <Input
                className="h-8 w-44 text-sm"
                placeholder="Workflow name..."
                value={workflowName}
                onChange={(e) => onWorkflowNameChange(e.target.value)}
              />
            )}
            <SaveStatusIndicator saveStatus={saveStatus} />
          </>
        )}
      </div>
      <div className="flex items-center gap-2">
        {isTemplate && onRemix && (
          <button
            onClick={onRemix}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg
              border border-primary/30 text-primary hover:bg-primary/10
              transition-colors duration-150"
          >
            <Copy className="size-3.5" />
            Remix
          </button>
        )}
        <RunWorkflowButton onRun={onRun ?? (() => {})} />
        <ThemeToggle />
      </div>
    </div>
  )
}
```

**Step 3: Add the Copy import at the top of the file**

```typescript
import { Loader2, Check, AlertCircle, Clock, Copy } from 'lucide-react'
```

**Step 4: Verify type-check passes**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/widgets/workflow-header/ui/WorkflowHeader.tsx
git commit -m "feat(templates): add template banner and Remix button to WorkflowHeader"
```

---

### Task 5: Update Editor for template mode

**Files:**
- Modify: `web/src/pages/Editor.tsx`

**Step 1: Read `isTemplate` from store and add remix handler**

Update the Editor component. After the existing store reads (around line 18-26), add:

```typescript
  const isTemplate = useWorkflowStore((s) => s.isTemplate)
  const setIsTemplate = useWorkflowStore((s) => s.setIsTemplate)
```

**Step 2: Conditionally call `useAutoSave`**

Replace line 30 (`const { saveStatus, saveNow } = useAutoSave()`) with:

```typescript
  const { saveStatus, saveNow } = useAutoSave()
```

The auto-save hook (`useAutoSave`) already skips saving when `nodes.length === 0`. We need to also skip when `isTemplate`. Modify `web/src/features/manage-canvas/model/useAutoSave.ts` — in the `flushSave` function, add at the start:

```typescript
  if (useWorkflowStore.getState().isTemplate) return
```

**Step 3: Add `handleRemix` function**

After `handleRun` (around line 109), add:

```typescript
  const handleRemix = () => {
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    setIsTemplate(false)
    setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
  }
```

**Step 4: Update the JSX to pass template props**

Update the `<WorkflowHeader>` in the return:

```typescript
        <WorkflowHeader
          workflowName={workflowName}
          onWorkflowNameChange={isTemplate ? undefined : setWorkflowName}
          saveStatus={isTemplate ? undefined : saveStatus}
          onRun={handleRun}
          isTemplate={isTemplate}
          templateName={workflowName}
          onRemix={handleRemix}
        />
```

Update the `<Canvas>`:

```typescript
            <Canvas
              onAddFirstNode={() => handleAddNode('input')}
              onDropNode={handleDropNode}
              onPromptSubmit={handlePromptSubmit}
              isGenerating={isGenerating}
              exposeGetViewportCenter={handleExposeViewportCenter}
              onAddNode={handleAddNode}
              readOnly={isTemplate}
            />
```

**Step 5: Verify type-check passes**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit`
Expected: PASS

**Step 6: Commit**

```bash
git add web/src/pages/Editor.tsx web/src/features/manage-canvas/model/useAutoSave.ts
git commit -m "feat(templates): add read-only mode and remix handler to Editor"
```

---

### Task 6: Update Landing page — template cards and handler

**Files:**
- Modify: `web/src/pages/Landing.tsx`

**Step 1: Add imports**

At the top of Landing.tsx, add:

```typescript
import { templates } from '@/shared/lib/templates'
import { WorkflowMiniGraph } from '@/shared/ui/WorkflowMiniGraph'
```

**Step 2: Add `openTemplate` handler**

After `openNew` function (around line 70), add:

```typescript
  const openTemplate = (tpl: (typeof templates)[number]) => {
    const { nodes, edges } = deserializeWorkflow(tpl.workflow)
    useWorkflowStore.setState({ nodes, edges, isTemplate: true })
    useWorkflowStore.getState().setWorkflowName(tpl.workflow.name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    navigate('/editor')
  }
```

**Step 3: Replace the Quick Start Templates section**

Replace lines 176-197 (the entire Quick Start Templates section) with:

```tsx
          {/* ─── Quick Start Templates ─── */}
          <div className="mb-10">
            <h2 className="text-lg font-semibold mb-4 tracking-tight">Quick Start Templates</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {templates.map((tpl) => {
                const Icon = tpl.icon
                return (
                  <button
                    key={tpl.id}
                    onClick={() => openTemplate(tpl)}
                    className="group text-left rounded-2xl overflow-hidden glass-panel border border-white/5
                      hover:border-white/10 hover:bg-white/5 hover:shadow-[0_12px_24px_rgba(0,0,0,0.2)]
                      hover:-translate-y-1 transition-all duration-300 cursor-pointer"
                  >
                    {/* Mini graph preview */}
                    <div className="relative h-[68px] border-b border-white/5 bg-black/10 dark:bg-black/20 overflow-hidden">
                      <WorkflowMiniGraph
                        nodes={tpl.workflow.nodes}
                        edges={tpl.workflow.edges}
                        uid={tpl.id}
                      />
                    </div>

                    {/* Card body */}
                    <div className="px-4 pt-3 pb-3.5">
                      <div className="flex items-center gap-2 mb-1.5">
                        <div className={`size-6 rounded-md flex items-center justify-center ${tpl.color}`}>
                          <Icon className="size-3.5" />
                        </div>
                        <h3 className="font-semibold text-sm text-foreground group-hover:text-primary transition-colors truncate">
                          {tpl.title}
                        </h3>
                      </div>
                      <p className="text-xs text-muted-foreground/80 leading-relaxed line-clamp-1 mb-2">
                        {tpl.description}
                      </p>
                      <div className="flex items-center gap-2">
                        <span className="text-[10px] text-muted-foreground/50 tabular-nums">
                          {tpl.workflow.nodes.length} nodes
                        </span>
                        <span className={`text-[10px] px-1.5 py-0.5 rounded-full ${
                          tpl.difficulty === 'Beginner'
                            ? 'bg-success/10 text-success'
                            : 'bg-warning/10 text-warning'
                        }`}>
                          {tpl.difficulty}
                        </span>
                      </div>
                    </div>
                  </button>
                )
              })}
            </div>
          </div>
```

**Step 4: Remove unused icon imports**

The old template section used `Search`, `Clock`, `GitBranch` as template icons — `Clock` and `Search` may no longer be needed in Landing.tsx (check if they're used elsewhere in the file before removing). `GitBranch` is still used by the dashboard stats. Remove only unused imports.

**Step 5: Verify type-check passes**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit`
Expected: PASS

**Step 6: Verify dev server renders correctly**

Run: `cd /home/dev/code/Upal/web && npm run dev`
Manual check: Open http://localhost:5173/workflows — should see 6 template cards with mini graph previews

**Step 7: Commit**

```bash
git add web/src/pages/Landing.tsx
git commit -m "feat(templates): upgrade Landing page with 6 functional template cards"
```

---

### Task 7: Reset `isTemplate` on `openNew` and `openWorkflow`

**Files:**
- Modify: `web/src/pages/Landing.tsx`

**Step 1: Add `isTemplate: false` reset to `openNew`**

In the `openNew` function, add `isTemplate: false` to the setState call:

```typescript
  const openNew = () => {
    useWorkflowStore.setState({ nodes: [], edges: [], isTemplate: false })
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    useWorkflowStore.getState().setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    navigate('/editor')
  }
```

**Step 2: Add `isTemplate: false` reset to `openWorkflow`**

In the `openWorkflow` function:

```typescript
  const openWorkflow = (wf: WorkflowDefinition) => {
    const { nodes, edges } = deserializeWorkflow(wf)
    useWorkflowStore.setState({ nodes, edges, isTemplate: false })
    useWorkflowStore.getState().setWorkflowName(wf.name)
    useWorkflowStore.getState().setOriginalName(wf.name)
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    navigate('/editor')
  }
```

**Step 3: Verify type-check passes**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/pages/Landing.tsx
git commit -m "fix(templates): reset isTemplate flag when opening blank or saved workflows"
```

---

### Task 8: End-to-end manual verification

**Step 1: Verify template card rendering**

Open http://localhost:5173/workflows
- All 6 template cards visible with mini graph previews
- Cards show: icon, title, description, node count, difficulty badge
- Hover effects: lift, border highlight, title color change

**Step 2: Verify template opens in read-only mode**

Click "Basic RAG Agent" template
- Editor opens with 3 nodes (User Query → RAG Agent → Response)
- Top banner shows "Template" badge + "Basic RAG Agent" name
- "Remix" button visible next to Run
- No NodePalette sidebar visible
- No CanvasPromptBar at bottom
- Cannot drag nodes, delete nodes, or create edges
- Pan/zoom works normally

**Step 3: Verify Remix flow**

Click "Remix" button
- Banner disappears, replaced by editable name Input ("Untitled-xxxx")
- Save status appears
- NodePalette and CanvasPromptBar appear
- Nodes become draggable and editable
- Can save as new workflow

**Step 4: Verify other flows still work**

- Click "Blank Workflow" → blank editor, no template banner
- Click existing saved workflow → opens normally, no template banner
- Auto-save works on non-template workflows

**Step 5: Verify type-check and lint**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit && npm run lint`
Expected: Both PASS
