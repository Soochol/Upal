// web/src/pages/workflows/index.tsx — Unified Workflows page (inbox-style layout)
// Merges Landing.tsx (listing) + Editor.tsx (canvas editing) into a single 3-column view.

import { useCallback, useRef, useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ReactFlowProvider } from '@xyflow/react'
import { PanelRightClose, PanelRightOpen, ArrowLeft, Search } from 'lucide-react'
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
import { useExecutionStore } from '@/entities/run'
import { fetchRuns } from '@/entities/run'
import { useUIStore } from '@/entities/ui'
import { useKeyboardShortcuts, useAutoSave } from '@/features/manage-canvas'
import { useReconnectRun } from '@/features/execute-workflow'
import { templates } from '@/shared/lib/templates'
import type { TemplateDefinition } from '@/shared/lib/templates'
import { WorkflowSidebar } from './WorkflowSidebar'

// ─── Page ────────────────────────────────────────────────────────────────────

export default function WorkflowsPage() {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()

  const selectedWorkflowName = searchParams.get('w')
  const isGenerateMode = searchParams.get('generate') === 'true'

  // ─── Workflow store ─────────────────────────────────────────────────────

  const addNode = useWorkflowStore((s) => s.addNode)
  const nodes = useWorkflowStore((s) => s.nodes)
  const edges = useWorkflowStore((s) => s.edges)
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const setWorkflowName = useWorkflowStore((s) => s.setWorkflowName)
  const isTemplate = useWorkflowStore((s) => s.isTemplate)
  const setIsTemplate = useWorkflowStore((s) => s.setIsTemplate)

  // ─── Execution store ────────────────────────────────────────────────────

  const addRunEvent = useExecutionStore((s) => s.addRunEvent)
  const setIsRunning = useExecutionStore((s) => s.setIsRunning)
  const startLocalRun = useExecutionStore((s) => s.startRun)

  // ─── UI store ───────────────────────────────────────────────────────────

  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const selectNode = useUIStore((s) => s.selectNode)
  const addToast = useUIStore((s) => s.addToast)

  const selectedNode = selectedNodeId
    ? nodes.find((n) => n.id === selectedNodeId) ?? null
    : null

  // ─── Local state ────────────────────────────────────────────────────────

  const [isGenerating, setIsGenerating] = useState(false)
  const [isRightPanelOpen, setIsRightPanelOpen] = useState(false)
  const [runningWorkflows, setRunningWorkflows] = useState<Set<string>>(new Set())
  const getViewportCenterRef = useRef<(() => { x: number; y: number }) | null>(null)

  // ─── Auto-save + reconnect ──────────────────────────────────────────────

  const { saveStatus, saveNow } = useAutoSave()
  useReconnectRun()
  useKeyboardShortcuts({ onSave: saveNow })

  // ─── Data fetching ──────────────────────────────────────────────────────

  const { data: workflows = [], isLoading } = useQuery({
    queryKey: ['workflows'],
    queryFn: listWorkflows,
  })

  // Detect running workflows on mount
  useEffect(() => {
    fetchRuns(100, 0)
      .then(({ runs }) => {
        const running = new Set<string>()
        for (const r of runs ?? []) {
          if (r.status === 'running') running.add(r.workflow_name)
        }
        setRunningWorkflows(running) // eslint-disable-line react-hooks/set-state-in-effect
      })
      .catch(() => {})
  }, [])

  // ─── Load workflow when ?w= param changes ──────────────────────────────

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
        addToast(`Workflow "${selectedWorkflowName}" not found.`)
      })
  }, [selectedWorkflowName]) // eslint-disable-line react-hooks/exhaustive-deps

  // ─── Auto-open right panel when node selected ─────────────────────────

  useEffect(() => {
    if (selectedNodeId) {
      setIsRightPanelOpen(true)
    }
  }, [selectedNodeId])

  const handleRightPanelCollapse = useCallback(() => {
    setIsRightPanelOpen(false)
  }, [])

  // ─── Sidebar callbacks ─────────────────────────────────────────────────

  const handleSelect = useCallback((name: string) => {
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
      // If the deleted workflow is currently selected, clear selection
      if (selectedWorkflowName === name) {
        setSearchParams({})
        useWorkflowStore.setState({ nodes: [], edges: [], isTemplate: false })
        useWorkflowStore.getState().setWorkflowName('')
        useWorkflowStore.getState().setOriginalName('')
      }
    } catch (err) {
      addToast(`Failed to delete workflow: ${err instanceof Error ? err.message : 'unknown error'}`)
    }
  }, [queryClient, selectedWorkflowName, setSearchParams, addToast])

  const handleTemplate = useCallback((tpl: TemplateDefinition) => {
    saveNow()
    const { nodes: n, edges: e } = deserializeWorkflow(tpl.workflow)
    useWorkflowStore.setState({ nodes: n, edges: e, isTemplate: true })
    useWorkflowStore.getState().setWorkflowName(tpl.workflow.name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    setSearchParams({ w: tpl.workflow.name })
  }, [saveNow, setSearchParams])

  // ─── Canvas callbacks ──────────────────────────────────────────────────

  const handleAddNode = useCallback((type: Parameters<typeof addNode>[0]) => {
    const center = getViewportCenterRef.current?.() ?? { x: 250, y: 150 }
    addNode(type, {
      x: center.x + (Math.random() - 0.5) * 60,
      y: center.y + (Math.random() - 0.5) * 40,
    })
  }, [addNode])

  const handleExposeViewportCenter = useCallback(
    (fn: () => { x: number; y: number }) => { getViewportCenterRef.current = fn },
    [],
  )

  const handleDropNode = useCallback((type: string, position: { x: number; y: number }) => {
    addNode(type as 'input' | 'agent' | 'output', position)
  }, [addNode])

  const handlePromptSubmit = useCallback(async (description: string) => {
    setIsGenerating(true)
    const currentNodes = useWorkflowStore.getState().nodes
    const currentEdges = useWorkflowStore.getState().edges
    const currentName = useWorkflowStore.getState().workflowName
    const hasExisting = currentNodes.length > 0

    const action = hasExisting ? 'Editing' : 'Generating'
    addRunEvent({ type: 'info', message: `${action} workflow...` })

    try {
      const existingWf = hasExisting
        ? serializeWorkflow(currentName || 'untitled', currentNodes, currentEdges)
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
  }, [addRunEvent, setWorkflowName])

  const handleRun = useCallback(async () => {
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
  }, [nodes, edges, workflowName, addRunEvent, setIsRunning, startLocalRun])

  const handleRemix = useCallback(() => {
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    setIsTemplate(false)
    setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
  }, [setIsTemplate, setWorkflowName])

  // ─── Generate mode: auto-focus prompt bar ──────────────────────────────

  useEffect(() => {
    if (isGenerateMode) {
      // Clear the generate flag from URL so it doesn't re-trigger
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        next.delete('generate')
        return next
      })
    }
  }, [isGenerateMode, setSearchParams])

  // ─── Mobile level ──────────────────────────────────────────────────────

  type MobileLevel = 'list' | 'detail'
  const mobileLevel: MobileLevel = selectedWorkflowName ? 'detail' : 'list'

  const goBackToList = useCallback(() => {
    setSearchParams({})
  }, [setSearchParams])

  // ─── Render ────────────────────────────────────────────────────────────

  const hasWorkflowSelected = !!selectedWorkflowName

  return (
    <MainLayout
      headerContent={
        hasWorkflowSelected ? (
          <WorkflowHeader
            workflowName={workflowName}
            onWorkflowNameChange={isTemplate ? undefined : setWorkflowName}
            saveStatus={isTemplate ? undefined : saveStatus}
            onRun={handleRun}
            isTemplate={isTemplate}
            templateName={workflowName}
            onRemix={handleRemix}
          />
        ) : (
          <span className="font-semibold tracking-tight">Workflows</span>
        )
      }
      rightPanel={
        isRightPanelOpen && hasWorkflowSelected ? (
          <RightPanel
            selectedNode={selectedNode}
            onCloseNode={() => selectNode(null)}
            onCollapse={handleRightPanelCollapse}
          />
        ) : null
      }
      bottomConsole={hasWorkflowSelected ? <Console /> : undefined}
    >
      <div className="flex h-full w-full overflow-hidden bg-background">

        {/* ── Left sidebar: Workflow list ── */}
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

        {/* ── Center: Canvas area ── */}
        <div className={cn(
          'flex-1 min-w-0 flex flex-col relative',
          mobileLevel === 'list' ? 'hidden md:flex' : 'flex',
        )}>
          {hasWorkflowSelected ? (
            <>
              {/* Sub-header strip */}
              <div className="px-4 md:px-6 py-2 border-b border-border/50 bg-background/80 backdrop-blur-sm shrink-0 shadow-sm z-10 flex items-center justify-between gap-3">
                {/* Mobile back button */}
                <button
                  onClick={goBackToList}
                  className="md:hidden flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors shrink-0"
                >
                  <ArrowLeft className="w-4 h-4" />
                </button>

                <div className="flex-1 min-w-0" />

                {/* Right panel toggle */}
                <button
                  onClick={() => setIsRightPanelOpen((v) => !v)}
                  className="hidden md:flex p-2 rounded-xl border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
                  title="Toggle Panel"
                >
                  {isRightPanelOpen ? <PanelRightClose className="w-4 h-4" /> : <PanelRightOpen className="w-4 h-4" />}
                </button>
              </div>

              {/* Canvas */}
              <div className="flex-1 min-h-0 overflow-hidden">
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
            </>
          ) : (
            /* Empty state — no workflow selected */
            <div className="flex-1 flex items-center justify-center text-muted-foreground flex-col gap-3">
              <div className="size-14 rounded-full bg-muted/30 flex items-center justify-center shrink-0 border border-border/50">
                <Search className="w-6 h-6 opacity-30" />
              </div>
              <div className="text-center">
                <p className="font-medium text-foreground">No workflow selected</p>
                <p className="text-sm">Select a workflow from the list or create a new one.</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </MainLayout>
  )
}
