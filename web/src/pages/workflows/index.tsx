import { useCallback, useRef, useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ReactFlowProvider } from '@xyflow/react'
import { PanelRightOpen, ArrowLeft, Search } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { Canvas } from '@/widgets/workflow-canvas'
import { RightPanel } from '@/widgets/right-panel'
import { WorkflowHeader } from '@/widgets/workflow-header'
import { Console } from '@/widgets/bottom-console'
import { useResizeDrag } from '@/shared/lib/useResizeDrag'
import {
  useWorkflowStore, serializeWorkflow, deserializeWorkflow,
  loadWorkflow, listWorkflows, deleteWorkflow, generateWorkflow,
} from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { fetchRuns } from '@/entities/run'
import { useUIStore } from '@/entities/ui'
import { useKeyboardShortcuts, useAutoSave } from '@/features/manage-canvas'
import { useReconnectRun } from '@/features/execute-workflow'
import { useRegisterChatHandler } from '@/shared/hooks/useRegisterChatHandler'
import { configureNode, computeUpstreamNodes } from '@/features/edit-node'
import type { ChatSubmitParams } from '@/entities/ui/model/chatStore'
import type { TemplateDefinition } from '@/shared/lib/templates'
import { WorkflowSidebar } from './WorkflowSidebar'

export default function WorkflowsPage() {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()

  const selectedWorkflowName = searchParams.get('w')
  const isGenerateMode = searchParams.get('generate') === 'true'
  const hasWorkflowSelected = !!selectedWorkflowName

  const addNode = useWorkflowStore((s) => s.addNode)
  const nodes = useWorkflowStore((s) => s.nodes)
  const edges = useWorkflowStore((s) => s.edges)
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const setWorkflowName = useWorkflowStore((s) => s.setWorkflowName)
  const isTemplate = useWorkflowStore((s) => s.isTemplate)
  const setIsTemplate = useWorkflowStore((s) => s.setIsTemplate)

  const addRunEvent = useExecutionStore((s) => s.addRunEvent)
  const setIsRunning = useExecutionStore((s) => s.setIsRunning)
  const startLocalRun = useExecutionStore((s) => s.startRun)

  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const selectNode = useUIStore((s) => s.selectNode)
  const addToast = useUIStore((s) => s.addToast)

  const selectedNode = selectedNodeId
    ? nodes.find((n) => n.id === selectedNodeId) ?? null
    : null

  const [isGenerating, setIsGenerating] = useState(false)
  const [isRightPanelOpen, setIsRightPanelOpen] = useState(true)
  const [runningWorkflows, setRunningWorkflows] = useState<Set<string>>(new Set())
  const getViewportCenterRef = useRef<(() => { x: number; y: number }) | null>(null)
  const { size: rightPanelWidth, handleMouseDown: onRightPanelDrag } = useResizeDrag({
    direction: 'horizontal',
    min: 260,
    max: 700,
    initial: 320,
  })

  const { saveStatus, saveNow, markClean } = useAutoSave()
  useReconnectRun()
  useKeyboardShortcuts({ onSave: saveNow })

  // ─── Global chat bar: node configure handler ──────────────────────────
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)
  const updateNodeDescription = useWorkflowStore((s) => s.updateNodeDescription)

  const nodeConfigureHandler = useCallback(async (params: ChatSubmitParams) => {
    const { nodes: curNodes, edges: curEdges } = useWorkflowStore.getState()
    const nodeId = useUIStore.getState().selectedNodeId
    if (!nodeId) throw new Error('No node selected')
    const node = curNodes.find((n) => n.id === nodeId)
    if (!node) throw new Error('Node not found')

    const upstream = computeUpstreamNodes(nodeId, curNodes, curEdges)

    const response = await configureNode({
      node_type: node.data.nodeType,
      node_id: nodeId,
      current_config: node.data.config,
      label: node.data.label,
      description: node.data.description ?? '',
      message: params.message,
      model: params.model || undefined,
      thinking: params.thinking,
      history: params.history,
      upstream_nodes: upstream,
    })

    if (response.config && Object.keys(response.config).length > 0) {
      updateNodeConfig(nodeId, response.config)
    }
    if (response.label) updateNodeLabel(nodeId, response.label)
    const desc = response.description || (response.config?.description as string)
    if (desc) updateNodeDescription(nodeId, desc)

    return { explanation: response.explanation }
  }, [updateNodeConfig, updateNodeLabel, updateNodeDescription])

  useRegisterChatHandler(
    selectedNodeId ? nodeConfigureHandler : null,
    selectedNodeId ? 'Describe this node...' : '',
    selectedNodeId ? 'Node' : '',
  )

  const { data: workflows = [], isLoading } = useQuery({
    queryKey: ['workflows'],
    queryFn: listWorkflows,
  })

  useEffect(() => {
    fetchRuns(100, 0)
      .then(({ runs }) => {
        const running = new Set<string>()
        for (const r of runs ?? []) {
          if (r.status === 'running') running.add(r.workflow_name)
        }
        setRunningWorkflows(running) // eslint-disable-line react-hooks/set-state-in-effect
      })
      .catch((err) => {
        console.warn('Failed to fetch running workflows:', err)
      })
  }, [])

  useEffect(() => {
    if (!selectedWorkflowName) return

    loadWorkflow(selectedWorkflowName)
      .then((wf) => {
        const { nodes: n, edges: e } = deserializeWorkflow(wf)
        useWorkflowStore.setState({ nodes: n, edges: e, isTemplate: false, positionVersion: 0 })
        useWorkflowStore.getState().setWorkflowName(wf.name)
        useWorkflowStore.getState().setOriginalName(wf.name)
        useExecutionStore.getState().clearNodeStatuses()
        useExecutionStore.getState().clearRunEvents()
        // Defer markClean to after React re-renders with the new store state,
        // so the clean baseline captures the actual post-load snapshot.
        requestAnimationFrame(() => markClean())
      })
      .catch(() => {
        addToast(`Workflow "${selectedWorkflowName}" not found.`)
      })
  }, [selectedWorkflowName]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (selectedNodeId) {
      setIsRightPanelOpen(true)
    }
  }, [selectedNodeId])

  const handleRightPanelCollapse = useCallback(() => {
    setIsRightPanelOpen(false)
  }, [])

  const resetStores = useCallback((name: string, template = false) => {
    useWorkflowStore.setState({ nodes: [], edges: [], isTemplate: template, positionVersion: 0 })
    useWorkflowStore.getState().setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
  }, [])

  const handleSelect = useCallback(async (name: string) => {
    await saveNow()
    setSearchParams({ w: name })
  }, [saveNow, setSearchParams])

  const handleNew = useCallback(async () => {
    await saveNow()
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    resetStores(name)
    setSearchParams({ w: name })
  }, [saveNow, resetStores, setSearchParams])

  const handleGenerate = useCallback(async () => {
    await saveNow()
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    resetStores(name)
    setSearchParams({ w: name, generate: 'true' })
  }, [saveNow, resetStores, setSearchParams])

  const handleDelete = useCallback(async (name: string) => {
    if (!confirm(`Delete workflow "${name}"?`)) return
    try {
      await deleteWorkflow(name)
      queryClient.invalidateQueries({ queryKey: ['workflows'] })
      if (selectedWorkflowName === name) {
        setSearchParams({})
        resetStores('')
      }
    } catch (err) {
      addToast(`Failed to delete workflow: ${err instanceof Error ? err.message : 'unknown error'}`)
    }
  }, [queryClient, selectedWorkflowName, setSearchParams, resetStores, addToast])

  const handleTemplate = useCallback(async (tpl: TemplateDefinition) => {
    await saveNow()
    const { nodes: n, edges: e } = deserializeWorkflow(tpl.workflow)
    useWorkflowStore.setState({ nodes: n, edges: e, isTemplate: true })
    useWorkflowStore.getState().setWorkflowName(tpl.workflow.name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    setSearchParams({ w: tpl.workflow.name })
  }, [saveNow, setSearchParams])

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

  useEffect(() => {
    if (saveStatus === 'saved') {
      queryClient.invalidateQueries({ queryKey: ['workflows'] })
    }
  }, [saveStatus, queryClient])

  useEffect(() => {
    if (isGenerateMode) {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        next.delete('generate')
        return next
      })
    }
  }, [isGenerateMode, setSearchParams])

  const mobileLevel = selectedWorkflowName ? 'detail' : 'list'

  const goBackToList = useCallback(() => {
    setSearchParams({})
  }, [setSearchParams])

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
      bottomConsole={hasWorkflowSelected ? <Console /> : undefined}
    >
      <div className="flex h-full w-full overflow-hidden bg-background">

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

        <div className={cn(
          'flex-1 min-w-0 flex flex-col relative',
          mobileLevel === 'list' ? 'hidden md:flex' : 'flex',
        )}>
          {hasWorkflowSelected ? (
            <>
              <div className="md:hidden px-4 py-3 border-b border-border/50 bg-background/80 backdrop-blur-sm shrink-0 shadow-sm z-10 flex items-center">
                <button
                  onClick={goBackToList}
                  className="flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors shrink-0"
                >
                  <ArrowLeft className="w-4 h-4" />
                </button>
              </div>

              <div className="flex-1 min-h-0 overflow-hidden flex">
                <div className="flex-1 min-w-0 h-full relative">
                  <ReactFlowProvider>
                    <Canvas
                      onAddFirstNode={() => handleAddNode('input')}
                      onDropNode={handleDropNode}
                      onPromptSubmit={handlePromptSubmit}
                      isGenerating={isGenerating}
                      exposeGetViewportCenter={handleExposeViewportCenter}
                      onAddNode={handleAddNode}
                      readOnly={isTemplate}
                      autoFocusPrompt={isGenerateMode}
                    />
                  </ReactFlowProvider>

                  {!isRightPanelOpen && (
                    <button
                      onClick={() => setIsRightPanelOpen(true)}
                      className="hidden md:flex absolute top-3 right-3 z-20 p-2 rounded-xl border border-border bg-card/90 backdrop-blur-sm text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer shadow-sm"
                      title="Open Panel"
                    >
                      <PanelRightOpen className="w-4 h-4" />
                    </button>
                  )}
                </div>

                {isRightPanelOpen && (
                  <div className="hidden md:contents">
                    <div
                      onMouseDown={onRightPanelDrag}
                      className="w-1 shrink-0 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 transition-colors z-30 relative
                        after:absolute after:inset-y-0 after:-left-1 after:-right-1"
                    />
                    <aside
                      style={{ width: rightPanelWidth }}
                      className="border-l border-border bg-sidebar/95 backdrop-blur-md shadow-2xl z-30 flex flex-col shrink-0"
                    >
                      <RightPanel
                        selectedNode={selectedNode}
                        onCloseNode={() => selectNode(null)}
                        onCollapse={handleRightPanelCollapse}
                        onTogglePanel={() => setIsRightPanelOpen(false)}
                      />
                    </aside>
                  </div>
                )}
              </div>
            </>
          ) : (
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
