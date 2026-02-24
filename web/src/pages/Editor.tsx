import { useCallback, useRef, useState } from 'react'
import { ReactFlowProvider } from '@xyflow/react'
import { Canvas } from '@/widgets/workflow-canvas'
import { RightPanel } from '@/widgets/right-panel'
import { WorkflowHeader } from '@/widgets/workflow-header'
import { MainLayout } from '@/app/layout'
import { Console } from '@/widgets/bottom-console'
import { useWorkflowStore } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { useUIStore } from '@/entities/ui'
import { useKeyboardShortcuts } from '@/features/manage-canvas'
import { useAutoSave } from '@/features/manage-canvas'
import { useReconnectRun } from '@/features/execute-workflow'
import { serializeWorkflow, deserializeWorkflow, loadWorkflow } from '@/entities/workflow'
import { generateWorkflow } from '@/entities/workflow'
import { useSearchParams } from 'react-router-dom'
import { useEffect } from 'react'

export default function Editor() {
  const [searchParams] = useSearchParams()
  const returnTo = searchParams.get('returnTo')

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

  const [isGenerating, setIsGenerating] = useState(false)
  const getViewportCenterRef = useRef<(() => { x: number; y: number }) | null>(null)

  const { saveStatus, saveNow } = useAutoSave()
  useReconnectRun()

  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const selectNode = useUIStore((s) => s.selectNode)

  const selectedNode = selectedNodeId
    ? nodes.find((n) => n.id === selectedNodeId)
    : null

  // Support opening a saved workflow by name via ?name= URL parameter
  useEffect(() => {
    const name = searchParams.get('name')
    if (!name) return
    searchParams.delete('name')
    window.history.replaceState(null, '', `?${searchParams.toString()}`)
    loadWorkflow(name).then((wf) => {
      const { nodes: n, edges: e } = deserializeWorkflow(wf)
      useWorkflowStore.setState({ nodes: n, edges: e, isTemplate: false })
      useWorkflowStore.getState().setWorkflowName(wf.name)
      useWorkflowStore.getState().setOriginalName(wf.name)
      useExecutionStore.getState().clearNodeStatuses()
      useExecutionStore.getState().clearRunEvents()
    }).catch(() => {
      // Workflow not found — ignore, stay on empty editor
    })
  }, [])

  // Support generating a workflow immediately from a prompt URL parameter
  useEffect(() => {
    const prompt = searchParams.get('prompt')
    if (prompt && !isGenerating && nodes.length === 0 && !workflowName) {
      // Clear the prompt from URL to avoid re-generating on reload
      searchParams.delete('prompt')
      window.history.replaceState(null, '', `?${searchParams.toString()}`)
      handlePromptSubmit(prompt)
    }
  }, [searchParams])

  const handleAddNode = (type: Parameters<typeof addNode>[0]) => {
    const center = getViewportCenterRef.current?.() ?? { x: 250, y: 150 }
    // Small random offset so consecutive clicks don't stack exactly
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

  const handleRun = async () => {
    if (nodes.length === 0) {
      addRunEvent({ type: 'error', message: 'Canvas is empty. Add nodes before running.' })
      return
    }

    // In the future this might open a modal to prompt for starting inputs
    const inputs: Record<string, string> = { "prompt": "Hello world from manual run!" }

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

  useKeyboardShortcuts({
    onSave: saveNow,
  })

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
          returnTo={returnTo}
        />
      }
      bottomConsole={<Console />}
    >
      <div className="flex flex-1 overflow-hidden w-full relative">
        <main className="flex-1">
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
        </main>
        <RightPanel
          selectedNode={selectedNode ?? null}
          onCloseNode={() => selectNode(null)}
        />
      </div>
    </MainLayout>
  )
}
