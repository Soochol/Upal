# Workflows Inbox-Style 3-Column Layout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Merge `/workflows` (dashboard) and `/editor` (canvas) into a single 3-column inbox-style page: WorkflowSidebar + Canvas + RightPanel.

**Architecture:** Replace Landing.tsx (card grid) and Editor.tsx (standalone canvas) with a unified WorkflowsPage that uses the same sidebar+detail pattern as Pipelines. The left sidebar lists workflows; selecting one loads it into the central canvas area. RightPanel moves to MainLayout's `rightPanel` prop with collapse/expand controlled by the parent page.

**Tech Stack:** React 19, React Flow, Zustand, React Router, Tailwind CSS v4, TanStack Query

---

### Task 1: Create `WorkflowSidebar` Component

**Files:**
- Create: `web/src/pages/workflows/WorkflowSidebar.tsx`

**Step 1: Create the sidebar component**

Reference `web/src/pages/pipelines/PipelineSidebar.tsx` for layout structure (search, tabs, list items). Reference `web/src/pages/Landing.tsx` for workflow data fetching patterns (`listWorkflows`, `deleteWorkflow`, running status via `fetchRuns`).

```tsx
// web/src/pages/workflows/WorkflowSidebar.tsx
import { useState } from 'react'
import { Search, Plus, Sparkles, Loader2, GitBranch, Clock, Trash2 } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import type { WorkflowDefinition } from '@/entities/workflow'
import { templates } from '@/shared/lib/templates'

interface WorkflowSidebarProps {
  workflows: WorkflowDefinition[]
  selectedName: string | null
  onSelect: (name: string) => void
  onNew: () => void
  onGenerate: () => void
  onDelete: (name: string) => void
  onTemplate: (tpl: typeof templates[number]) => void
  isLoading: boolean
  runningWorkflows: Set<string>
}

export function WorkflowSidebar({
  workflows, selectedName, onSelect, onNew, onGenerate, onDelete,
  onTemplate, isLoading, runningWorkflows,
}: WorkflowSidebarProps) {
  const [search, setSearch] = useState('')
  const filtered = workflows.filter((w) =>
    w.name.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <div className="flex flex-col h-full animate-in fade-in duration-300">
      {/* Header: search + action buttons */}
      <div className="p-4 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md shadow-sm z-10 space-y-3">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="search"
            placeholder="Search workflows..."
            className="w-full h-9 pl-9 pr-4 rounded-lg bg-background border border-input text-sm outline-none focus:ring-1 focus:ring-ring transition-shadow"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onNew}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
          >
            <Plus className="h-3 w-3" /> New
          </button>
          <button
            onClick={onGenerate}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
          >
            <Sparkles className="h-3 w-3" /> Generate
          </button>
        </div>
      </div>

      {/* Workflow list */}
      <div className="flex-1 overflow-y-auto p-3 space-y-1.5">
        {isLoading ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
            <Loader2 className="w-5 h-5 animate-spin text-primary/50" />
            <span className="text-sm font-medium">Loading workflows...</span>
          </div>
        ) : filtered.length === 0 ? (
          workflows.length === 0 ? (
            <div className="flex flex-col items-center justify-center text-muted-foreground p-6 gap-4 text-center pt-16">
              <div className="w-14 h-14 rounded-2xl bg-muted/20 flex items-center justify-center">
                <GitBranch className="w-6 h-6 opacity-30" />
              </div>
              <div>
                <p className="font-medium text-foreground">No workflows yet</p>
                <p className="text-xs mt-1">Create your first workflow to get started.</p>
              </div>
            </div>
          ) : (
            <div className="text-center py-12 px-4">
              <p className="text-sm text-muted-foreground">
                No workflows matching &ldquo;{search}&rdquo;
              </p>
            </div>
          )
        ) : (
          filtered.map((wf) => {
            const isSelected = selectedName === wf.name
            const isRunning = runningWorkflows.has(wf.name)
            return (
              <button
                key={wf.name}
                onClick={() => onSelect(wf.name)}
                className={cn(
                  'group w-full text-left p-3.5 rounded-xl border transition-all duration-200 cursor-pointer flex flex-col gap-1.5',
                  isSelected
                    ? 'bg-primary/5 border-primary/40 shadow-sm ring-1 ring-primary/20'
                    : 'bg-card border-border/60 hover:border-primary/40 hover:bg-muted/50',
                )}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <div className={cn(
                      'w-7 h-7 rounded-lg bg-card border border-white/5 flex items-center justify-center shrink-0',
                      isRunning && 'ring-2 ring-node-agent/50 animate-pulse',
                    )}>
                      <GitBranch className="w-3.5 h-3.5 text-blue-400" />
                    </div>
                    <span className="text-sm font-semibold truncate">{wf.name}</span>
                  </div>
                  <button
                    onClick={(e) => { e.stopPropagation(); onDelete(wf.name) }}
                    className="p-1 rounded-md text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
                    title="Delete"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
                <div className="flex items-center gap-2 text-[10px] text-muted-foreground/60">
                  <span>{wf.nodes.length} nodes</span>
                  {isRunning && (
                    <>
                      <span className="text-muted-foreground/30">·</span>
                      <span className="inline-flex items-center gap-0.5 text-node-agent font-bold">
                        <span className="w-1.5 h-1.5 rounded-full bg-node-agent animate-pulse" />
                        Running
                      </span>
                    </>
                  )}
                </div>
              </button>
            )
          })
        )}
      </div>

      {/* Templates section */}
      {templates.length > 0 && (
        <div className="border-t border-border/50 p-3 shrink-0">
          <p className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60 px-1 mb-2">
            Templates
          </p>
          <div className="space-y-1">
            {templates.slice(0, 3).map((tpl) => {
              const Icon = tpl.icon
              return (
                <button
                  key={tpl.id}
                  onClick={() => onTemplate(tpl)}
                  className="w-full text-left px-3 py-2 rounded-lg text-xs hover:bg-muted/50 transition-colors cursor-pointer flex items-center gap-2"
                >
                  <Icon className={`w-3.5 h-3.5 ${tpl.color}`} />
                  <span className="truncate font-medium">{tpl.title}</span>
                </button>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
```

**Step 2: Verify type-check passes**

Run: `cd web && npx tsc --noEmit`
Expected: No errors related to WorkflowSidebar

**Step 3: Commit**

```bash
git add web/src/pages/workflows/WorkflowSidebar.tsx
git commit -m "feat: create WorkflowSidebar component for inbox-style layout"
```

---

### Task 2: Refactor RightPanel for MainLayout `rightPanel` Prop

**Files:**
- Modify: `web/src/widgets/right-panel/ui/RightPanel.tsx`

Currently RightPanel manages its own `<aside>` wrapper, resize handle, and collapse/expand state. Refactor so:
1. Remove outer `<aside>` and resize handle — MainLayout handles these
2. Expose collapse/expand state via a callback to parent
3. When collapsed, return `null` (parent hides the panel)

**Step 1: Refactor RightPanel to remove self-managed aside wrapper**

The existing RightPanel (at `web/src/widgets/right-panel/ui/RightPanel.tsx`) returns either:
- Collapsed: `<aside className="...w-12...">` (icon strip)
- Expanded: `<aside style={{ width }} ...>` (full panel with resize)

Change it to:
- Accept `onCollapse?: () => void` callback
- Always return the inner content (no `<aside>` wrapper)
- Remove `useResizeDrag` (MainLayout handles resize)
- Collapsed icon strip moves into the parent or becomes a toggle button in WorkflowHeader

```tsx
// web/src/widgets/right-panel/ui/RightPanel.tsx
import { useState, useRef, useEffect } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { NodeEditor } from '@/features/edit-node'
import { PanelPreview } from './PanelPreview'
import { PanelConsole } from './PanelConsole'
import { GroupEditor } from './GroupEditor'
import { AIChatEditor } from '@/features/edit-node'
import { Settings2, Terminal, Eye } from 'lucide-react'
import type { NodeData } from '@/entities/workflow'
import { useUIStore } from '@/entities/ui'
import type { Node } from '@xyflow/react'

type RightPanelProps = {
  selectedNode: Node<NodeData> | null
  onCloseNode: () => void
  onCollapse?: () => void
}

const tabs = [
  { value: 'properties', label: 'Properties', icon: Settings2 },
  { value: 'console', label: 'Console', icon: Terminal },
  { value: 'preview', label: 'Preview', icon: Eye },
] as const

export function RightPanel({ selectedNode, onCloseNode, onCollapse }: RightPanelProps) {
  const [activeTab, setActiveTab] = useState('properties')

  // Track node selection for auto-tab switching
  const prevNodeIdRef = useRef<string | null>(selectedNode?.id ?? null)
  useEffect(() => {
    if (selectedNode) {
      setActiveTab('properties')
    }
    prevNodeIdRef.current = selectedNode?.id ?? null
  }, [selectedNode?.id])

  // Force Preview tab from store signal
  useEffect(() => {
    const unsub = useUIStore.subscribe(
      (state, prevState) => {
        if (state.forcePreviewTab && !prevState.forcePreviewTab) {
          setActiveTab('preview')
          useUIStore.getState().setForcePreviewTab(false)
        }
      },
    )
    return unsub
  }, [])

  const showAIChat = selectedNode && selectedNode.type !== 'groupNode'

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-col flex-1 min-h-0 gap-0 h-full">
      <div className="flex items-center justify-between border-b border-border px-1">
        <TabsList className="h-10 bg-transparent p-0 gap-0">
          {tabs.map((tab) => {
            const Icon = tab.icon
            return (
              <TabsTrigger
                key={tab.value}
                value={tab.value}
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 flex items-center gap-1.5 text-xs font-medium"
              >
                <Icon className="h-3.5 w-3.5 shrink-0" />
                {tab.label}
              </TabsTrigger>
            )
          })}
        </TabsList>
      </div>

      <TabsContent value="properties" className="flex-1 min-h-0 flex flex-col mt-0">
        {selectedNode && selectedNode.type === 'groupNode' ? (
          <GroupEditor groupId={selectedNode.id} data={selectedNode.data as NodeData} onClose={onCloseNode} />
        ) : selectedNode ? (
          <NodeEditor
            nodeId={selectedNode.id}
            data={selectedNode.data as NodeData}
            onClose={onCloseNode}
            embedded
          />
        ) : (
          <div className="flex items-center justify-center h-32 text-xs text-muted-foreground p-3">
            Select a node to edit its properties.
          </div>
        )}
      </TabsContent>

      <TabsContent value="console" className="flex-1 min-h-0 overflow-hidden mt-0">
        <PanelConsole />
      </TabsContent>

      <TabsContent value="preview" className="flex-1 min-h-0 overflow-hidden mt-0">
        <PanelPreview />
      </TabsContent>

      {showAIChat && (
        <div className="mt-auto shrink-0 border-t border-white/10 bg-black/20 dark:bg-white/5 backdrop-blur-md">
          <AIChatEditor nodeId={selectedNode.id} data={selectedNode.data as NodeData} />
        </div>
      )}
    </Tabs>
  )
}
```

**Step 2: Verify type-check passes**

Run: `cd web && npx tsc --noEmit`
Expected: No errors in RightPanel or its consumers

**Step 3: Commit**

```bash
git add web/src/widgets/right-panel/ui/RightPanel.tsx
git commit -m "refactor: remove RightPanel self-managed aside wrapper for MainLayout integration"
```

---

### Task 3: Create Unified `WorkflowsPage`

**Files:**
- Create: `web/src/pages/workflows/index.tsx`

This is the main page component that merges the logic from:
- `web/src/pages/Landing.tsx` (workflow listing, template opening, deletion)
- `web/src/pages/Editor.tsx` (canvas, generate, run, auto-save, keyboard shortcuts)

**Step 1: Create WorkflowsPage**

Key patterns:
- URL params: `?w=WorkflowName` for selection (similar to Pipelines' `?p=id`)
- WorkflowSidebar on left (like PipelineSidebar)
- Canvas + Console in center (from Editor.tsx)
- RightPanel via MainLayout's `rightPanel` prop
- WorkflowHeader in MainLayout's `headerContent`
- Mobile: show sidebar or canvas, not both

```tsx
// web/src/pages/workflows/index.tsx
import { useCallback, useRef, useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ReactFlowProvider } from '@xyflow/react'
import {
  PanelRightClose, PanelRightOpen, ArrowLeft, Search,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { Canvas } from '@/widgets/workflow-canvas'
import { RightPanel } from '@/widgets/right-panel'
import { WorkflowHeader } from '@/widgets/workflow-header'
import { Console } from '@/widgets/bottom-console'
import {
  useWorkflowStore, serializeWorkflow, deserializeWorkflow,
  loadWorkflow, listWorkflows, deleteWorkflow, generateWorkflow,
} from '@/entities/workflow'
import type { WorkflowDefinition } from '@/entities/workflow'
import { useExecutionStore, fetchRuns } from '@/entities/run'
import { useUIStore } from '@/entities/ui'
import { useKeyboardShortcuts, useAutoSave } from '@/features/manage-canvas'
import { useReconnectRun } from '@/features/execute-workflow'
import { templates } from '@/shared/lib/templates'
import { WorkflowSidebar } from './WorkflowSidebar'

export default function WorkflowsPage() {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedWorkflowName = searchParams.get('w')

  // ─── Workflow list ─────────────────────────────────────────────────
  const { data: workflows = [], isLoading } = useQuery({
    queryKey: ['workflows'],
    queryFn: listWorkflows,
  })

  const [runningWorkflows, setRunningWorkflows] = useState<Set<string>>(new Set())
  useEffect(() => {
    fetchRuns(100, 0)
      .then(({ runs }) => {
        const running = new Set<string>()
        for (const r of runs ?? []) {
          if (r.status === 'running') running.add(r.workflow_name)
        }
        setRunningWorkflows(running)
      })
      .catch(() => {})
  }, [])

  // ─── Workflow store ────────────────────────────────────────────────
  const addNode = useWorkflowStore((s) => s.addNode)
  const nodes = useWorkflowStore((s) => s.nodes)
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const setWorkflowName = useWorkflowStore((s) => s.setWorkflowName)
  const edges = useWorkflowStore((s) => s.edges)
  const isTemplate = useWorkflowStore((s) => s.isTemplate)
  const setIsTemplate = useWorkflowStore((s) => s.setIsTemplate)
  const addRunEvent = useExecutionStore((s) => s.addRunEvent)
  const setIsRunning = useExecutionStore((s) => s.setIsRunning)
  const startLocalRun = useExecutionStore((s) => s.startRun)
  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const selectNode = useUIStore((s) => s.selectNode)

  const selectedNode = selectedNodeId
    ? nodes.find((n) => n.id === selectedNodeId) ?? null
    : null

  const [isGenerating, setIsGenerating] = useState(false)
  const getViewportCenterRef = useRef<(() => { x: number; y: number }) | null>(null)

  const { saveStatus, saveNow } = useAutoSave()
  useReconnectRun()

  // ─── Right panel toggle ────────────────────────────────────────────
  const [isRightPanelOpen, setIsRightPanelOpen] = useState(true)

  // Auto-open right panel when node selected
  useEffect(() => {
    if (selectedNodeId) {
      setIsRightPanelOpen(true)
    }
  }, [selectedNodeId])

  // ─── Load workflow on URL param change ─────────────────────────────
  useEffect(() => {
    if (!selectedWorkflowName) return
    loadWorkflow(selectedWorkflowName)
      .then((wf) => {
        const { nodes: n, edges: e } = deserializeWorkflow(wf)
        useWorkflowStore.setState({ nodes: n, edges: e, isTemplate: false })
        useWorkflowStore.getState().setWorkflowName(wf.name)
        useWorkflowStore.getState().setOriginalName(wf.name)
        useExecutionStore.getState().clearNodeStatuses()
        useExecutionStore.getState().clearRunEvents()
      })
      .catch(() => {
        // Workflow not found — stay on empty canvas
      })
  }, [selectedWorkflowName])

  // ─── Sidebar actions ───────────────────────────────────────────────
  const handleSelect = useCallback((name: string) => {
    // Auto-save current workflow before switching
    saveNow()
    setSearchParams({ w: name })
  }, [saveNow, setSearchParams])

  const handleNew = useCallback(() => {
    saveNow()
    useWorkflowStore.setState({ nodes: [], edges: [], isTemplate: false })
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    useWorkflowStore.getState().setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    setSearchParams({ w: name })
  }, [saveNow, setSearchParams])

  const handleGenerate = useCallback(() => {
    saveNow()
    useWorkflowStore.setState({ nodes: [], edges: [], isTemplate: false })
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    useWorkflowStore.getState().setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    setSearchParams({ w: name, generate: 'true' })
  }, [saveNow, setSearchParams])

  const handleDelete = useCallback(async (name: string) => {
    if (!confirm(`Delete workflow "${name}"?`)) return
    try {
      await deleteWorkflow(name)
      queryClient.invalidateQueries({ queryKey: ['workflows'] })
      if (selectedWorkflowName === name) {
        setSearchParams({})
        useWorkflowStore.setState({ nodes: [], edges: [] })
      }
    } catch { /* ignore */ }
  }, [selectedWorkflowName, setSearchParams, queryClient])

  const handleTemplate = useCallback((tpl: typeof templates[number]) => {
    saveNow()
    const { nodes: n, edges: e } = deserializeWorkflow(tpl.workflow)
    useWorkflowStore.setState({ nodes: n, edges: e, isTemplate: true })
    useWorkflowStore.getState().setWorkflowName(tpl.workflow.name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    setSearchParams({ w: tpl.workflow.name })
  }, [saveNow, setSearchParams])

  // ─── Canvas actions (from Editor.tsx) ──────────────────────────────
  const handleAddNode = (type: Parameters<typeof addNode>[0]) => {
    const center = getViewportCenterRef.current?.() ?? { x: 250, y: 150 }
    addNode(type, {
      x: center.x + (Math.random() - 0.5) * 60,
      y: center.y + (Math.random() - 0.5) * 40,
    })
  }

  const handleExposeViewportCenter = useCallback(
    (fn: () => { x: number; y: number }) => { getViewportCenterRef.current = fn },
    [],
  )

  const handleDropNode = (type: string, position: { x: number; y: number }) => {
    addNode(type as 'input' | 'agent' | 'output', position)
  }

  const handlePromptSubmit = async (description: string) => {
    setIsGenerating(true)
    const currentNodes = useWorkflowStore.getState().nodes
    const currentEdges = useWorkflowStore.getState().edges
    const hasExisting = currentNodes.length > 0
    const action = hasExisting ? 'Editing' : 'Generating'
    addRunEvent({ type: 'info', message: `${action} workflow...` })

    try {
      const existingWf = hasExisting
        ? serializeWorkflow(workflowName || 'untitled', currentNodes, currentEdges)
        : undefined
      const wf = await generateWorkflow(description, undefined, existingWf)
      const { nodes: newNodes, edges: newEdges } = deserializeWorkflow(wf)
      useWorkflowStore.setState({ nodes: newNodes, edges: newEdges })
      if (wf.name) setWorkflowName(wf.name)
      addRunEvent({
        type: 'info',
        message: `Workflow "${wf.name}" ${hasExisting ? 'updated' : 'generated'} with ${wf.nodes.length} nodes.`,
      })
    } catch (err) {
      addRunEvent({
        type: 'error',
        message: `Generate failed: ${err instanceof Error ? err.message : String(err)}`,
      })
    } finally {
      setIsGenerating(false)
    }
  }

  // Support auto-generating from ?generate=true URL param
  useEffect(() => {
    const generate = searchParams.get('generate')
    if (generate === 'true' && !isGenerating && nodes.length === 0) {
      // Clear the param to avoid re-triggering
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        next.delete('generate')
        return next
      })
      // The Canvas component will show the generate prompt
    }
  }, [searchParams]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleRun = async () => {
    if (nodes.length === 0) {
      addRunEvent({ type: 'error', message: 'Canvas is empty. Add nodes before running.' })
      return
    }
    const inputs: Record<string, string> = { prompt: 'Hello world from manual run!' }
    setIsRunning(true)
    try {
      const wf = serializeWorkflow(workflowName || 'manual-run', nodes, edges)
      await startLocalRun(workflowName || 'manual-run', inputs, wf)
    } catch (err) {
      addRunEvent({
        type: 'error',
        message: `Run failed to start: ${err instanceof Error ? err.message : String(err)}`,
      })
      setIsRunning(false)
    }
  }

  const handleRemix = () => {
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    setIsTemplate(false)
    setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
  }

  useKeyboardShortcuts({ onSave: saveNow })

  // ─── Mobile level ──────────────────────────────────────────────────
  const mobileLevel: 'list' | 'detail' = selectedWorkflowName ? 'detail' : 'list'
  const goBackToList = () => {
    saveNow()
    setSearchParams({})
  }

  const toggleRightPanel = useCallback(() => {
    setIsRightPanelOpen((v) => !v)
  }, [])

  // ─── Render ────────────────────────────────────────────────────────
  return (
    <MainLayout
      headerContent={
        <WorkflowHeader
          workflowName={workflowName}
          onWorkflowNameChange={isTemplate ? undefined : setWorkflowName}
          saveStatus={isTemplate ? undefined : saveStatus}
          onRun={handleRun}
          isTemplate={isTemplate}
          templateName={workflowName}
          onRemix={handleRemix}
        />
      }
      bottomConsole={<Console />}
      rightPanel={
        isRightPanelOpen ? (
          <RightPanel
            selectedNode={selectedNode}
            onCloseNode={() => selectNode(null)}
          />
        ) : null
      }
    >
      <div className="flex h-full w-full overflow-hidden bg-background">
        {/* ── Left: Workflow Sidebar ── */}
        <div className={cn(
          'w-full md:w-[340px] 2xl:w-[400px] shrink-0 md:border-r border-border',
          'bg-sidebar/30 backdrop-blur-xl z-20 flex flex-col',
          'md:shadow-[4px_0_24px_-12px_rgba(0,0,0,0.5)]',
          mobileLevel === 'list' ? 'flex' : 'hidden md:flex',
        )}>
          <WorkflowSidebar
            workflows={workflows}
            selectedName={selectedWorkflowName}
            onSelect={handleSelect}
            onNew={handleNew}
            onGenerate={handleGenerate}
            onDelete={handleDelete}
            onTemplate={handleTemplate}
            isLoading={isLoading}
            runningWorkflows={runningWorkflows}
          />
        </div>

        {/* ── Right: Canvas area ── */}
        <div className={cn(
          'flex-1 min-w-0 flex flex-col relative',
          mobileLevel === 'list' ? 'hidden md:flex' : 'flex',
        )}>
          {/* Mobile back + panel toggle header */}
          <div className="flex items-center justify-between px-4 py-2 border-b border-border/50 bg-background/80 backdrop-blur-sm shrink-0 shadow-sm z-10 md:justify-end">
            <button
              onClick={goBackToList}
              className="md:hidden flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
            >
              <ArrowLeft className="w-4 h-4" />
              Workflows
            </button>
            <button
              onClick={toggleRightPanel}
              className="hidden md:flex p-2 rounded-xl border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
              title="Toggle Panel"
            >
              {isRightPanelOpen ? <PanelRightClose className="w-4 h-4" /> : <PanelRightOpen className="w-4 h-4" />}
            </button>
          </div>

          {selectedWorkflowName ? (
            <div className="flex-1 flex flex-col overflow-hidden">
              <ReactFlowProvider>
                <Canvas
                  onAddFirstNode={() => handleAddNode('input')}
                  onDropNode={handleDropNode}
                  onPromptSubmit={handlePromptSubmit}
                  isGenerating={isGenerating}
                  exposeGetViewportCenter={handleExposeViewportCenter}
                  onAddNode={handleAddNode}
                  readOnly={isTemplate}
                />
              </ReactFlowProvider>
            </div>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-8">
              <div className="w-16 h-16 rounded-3xl glass-panel flex items-center justify-center mb-4 shadow-xl">
                <Search className="w-6 h-6 opacity-50" />
              </div>
              <p className="text-sm font-medium">No workflow selected</p>
              <p className="text-xs opacity-60 mt-1 max-w-[250px] text-center">
                Select a workflow from the list or create a new one.
              </p>
            </div>
          )}
        </div>
      </div>
    </MainLayout>
  )
}
```

**Step 2: Verify type-check passes**

Run: `cd web && npx tsc --noEmit`
Expected: No errors

**Step 3: Commit**

```bash
git add web/src/pages/workflows/index.tsx
git commit -m "feat: create unified WorkflowsPage with inbox-style 3-column layout"
```

---

### Task 4: Update Router and All `/editor` References

**Files:**
- Modify: `web/src/app/router.tsx` — remove `/editor` route, point `/workflows` to new page
- Modify: `web/src/App.tsx` — same changes (duplicate router file)
- Modify: `web/src/shared/ui/Header.tsx:27` — remove Editor nav link
- Modify: `web/src/pages/ProductLanding.tsx:111` — `navigate('/editor?generate=true')` → `navigate('/workflows?generate=true')`
- Modify: `web/src/widgets/pipeline-editor/ui/PipelineEditor.tsx:84` — `navigate('/editor')` → `navigate('/workflows?w=...')`
- Modify: `web/src/pages/pipelines/PipelineSettingsPanel.tsx:205` — `/editor?name=` → `/workflows?w=`
- Modify: `web/src/pages/pipelines/session/stages/AnalyzeStage.tsx:395` — `/editor?name=` → `/workflows?w=`
- Modify: `web/src/widgets/run-detail/ui/RunDetail.tsx:139` — `/editor?name=` → `/workflows?w=`

**Step 1: Update router.tsx**

In `web/src/app/router.tsx`:
- Remove the `import EditorPage` line
- Remove `<Route path="/editor" ...>` line
- Change `LandingPage` import to new WorkflowsPage:

```tsx
// Before
import LandingPage from '@/pages/landing'
import EditorPage from '@/pages/Editor'
// ...
<Route path="/workflows" element={<LandingPage />} />
<Route path="/editor" element={<EditorPage />} />

// After
import WorkflowsPage from '@/pages/workflows'
// ...
<Route path="/workflows" element={<WorkflowsPage />} />
// /editor removed
```

**Step 2: Update App.tsx** (same changes)

```tsx
// Before
import Landing from '@/pages/Landing'
import Editor from '@/pages/Editor'
// ...
<Route path="/workflows" element={<Landing />} />
<Route path="/editor" element={<Editor />} />

// After
import WorkflowsPage from '@/pages/workflows'
// ...
<Route path="/workflows" element={<WorkflowsPage />} />
// /editor removed
```

**Step 3: Update Header.tsx**

In `web/src/shared/ui/Header.tsx:25-31`, remove the Editor nav link:

```tsx
// Before
const navLinks: NavLink[] = [
  { to: "/workflows", label: "Workflows" },
  { to: "/editor", label: "Editor" },
  { to: "/runs", label: "Runs" },
  // ...
];

// After
const navLinks: NavLink[] = [
  { to: "/workflows", label: "Workflows" },
  { to: "/runs", label: "Runs" },
  // ...
];
```

**Step 4: Update all `/editor` navigation references**

Each file that navigates to `/editor` or links to `/editor?name=X` needs updating:

| File | Old | New |
|------|-----|-----|
| `ProductLanding.tsx:111` | `navigate('/editor?generate=true')` | `navigate('/workflows?generate=true')` |
| `PipelineEditor.tsx:84` | `navigate('/editor')` | `navigate('/workflows?w=' + encodeURIComponent(name))` |
| `PipelineSettingsPanel.tsx:205` | `` href={`/editor?name=${...}`} `` | `` href={`/workflows?w=${...}`} `` |
| `AnalyzeStage.tsx:395` | `` href={`/editor?name=${...}`} `` | `` href={`/workflows?w=${...}`} `` |
| `RunDetail.tsx:139` | `` to={`/editor?name=${...}`} `` | `` to={`/workflows?w=${...}`} `` |

**Step 5: Verify type-check passes**

Run: `cd web && npx tsc --noEmit`
Expected: No errors. No remaining references to `/editor`.

**Step 6: Verify no remaining references**

Run: `grep -r "/editor" web/src/ --include="*.tsx" --include="*.ts"`
Expected: No matches (or only comments)

**Step 7: Commit**

```bash
git add -A
git commit -m "refactor: remove /editor route, update all navigation to /workflows?w="
```

---

### Task 5: Clean Up Old Files

**Files:**
- Delete: `web/src/pages/Editor.tsx`
- Delete: `web/src/pages/Landing.tsx`
- Optionally keep: `web/src/pages/landing/WorkflowCard.tsx` (if still referenced elsewhere; check first)

**Step 1: Check for remaining imports of old files**

Run: `grep -r "pages/Editor\|pages/Landing\|pages/landing'" web/src/ --include="*.tsx" --include="*.ts"`

If no references remain, delete the files.

**Step 2: Delete old files**

```bash
rm web/src/pages/Editor.tsx
rm web/src/pages/Landing.tsx
```

**Step 3: Check if WorkflowCard.tsx is still used**

Run: `grep -r "WorkflowCard" web/src/ --include="*.tsx" --include="*.ts"`

If only referenced from deleted Landing.tsx, delete it too. If referenced elsewhere, keep it.

**Step 4: Verify type-check passes**

Run: `cd web && npx tsc --noEmit`
Expected: Clean build with no errors

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove deprecated Editor.tsx and Landing.tsx"
```

---

### Task 6: Verify Full Build and Runtime

**Step 1: Run full type check**

Run: `cd web && npx tsc --noEmit`
Expected: 0 errors

**Step 2: Run ESLint**

Run: `cd web && npm run lint`
Expected: No new errors

**Step 3: Build frontend**

Run: `cd web && npm run build`
Expected: Successful build

**Step 4: Manual smoke test**

Start dev server (`make dev-frontend`) and verify:

1. `/workflows` shows 3-column layout: sidebar + canvas + right panel
2. Selecting a workflow loads it in the canvas
3. Creating new workflow works
4. Delete workflow works from sidebar
5. RightPanel toggle (PanelRightOpen/Close icon) works
6. Node selection opens RightPanel Properties tab
7. Auto-save works when editing
8. Run button executes workflow
9. Template selection loads template in read-only mode
10. Mobile responsive: sidebar/canvas toggle
11. Links from Pipelines, RunDetail, ProductLanding navigate correctly to `/workflows?w=...`

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat: complete workflows inbox-style layout migration"
```
