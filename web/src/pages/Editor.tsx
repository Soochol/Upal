import { useState } from 'react'
import { Canvas } from '@/components/editor/Canvas'
import { RunDialog } from '@/components/dialogs/RunDialog'
import { RightPanel } from '@/components/panel/RightPanel'
import { Header } from '@/components/Header'
import { NodePalette } from '@/components/sidebar/NodePalette'
import { BottomConsole } from '@/components/console/BottomConsole'
import { useWorkflowStore, type NodeRunStatus } from '@/stores/workflowStore'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { serializeWorkflow, deserializeWorkflow } from '@/lib/serializer'
import { saveWorkflow, runWorkflow, generateWorkflow } from '@/lib/api'

export default function Editor() {
  const addNode = useWorkflowStore((s) => s.addNode)
  const nodes = useWorkflowStore((s) => s.nodes)
  const edges = useWorkflowStore((s) => s.edges)
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const setWorkflowName = useWorkflowStore((s) => s.setWorkflowName)
  const isRunning = useWorkflowStore((s) => s.isRunning)
  const setIsRunning = useWorkflowStore((s) => s.setIsRunning)
  const addRunEvent = useWorkflowStore((s) => s.addRunEvent)
  const clearRunEvents = useWorkflowStore((s) => s.clearRunEvents)

  const selectedNodeId = useWorkflowStore((s) => s.selectedNodeId)
  const selectNode = useWorkflowStore((s) => s.selectNode)
  const setNodeStatus = useWorkflowStore((s) => s.setNodeStatus)
  const clearNodeStatuses = useWorkflowStore((s) => s.clearNodeStatuses)
  const setSessionState = useWorkflowStore((s) => s.setSessionState)

  const [showRunDialog, setShowRunDialog] = useState(false)
  const [isGenerating, setIsGenerating] = useState(false)

  const selectedNode = selectedNodeId
    ? nodes.find((n) => n.id === selectedNodeId)
    : null

  const handleAddNode = (type: 'input' | 'agent' | 'tool' | 'output' | 'external') => {
    addNode(type, {
      x: 250,
      y: useWorkflowStore.getState().nodes.length * 150 + 50,
    })
  }

  const handleDropNode = (type: string, position: { x: number; y: number }) => {
    addNode(type as 'input' | 'agent' | 'tool' | 'output' | 'external', position)
  }

  const handleSave = async () => {
    let name = workflowName
    if (!name) {
      const input = window.prompt('Workflow name:')
      if (!input) return
      name = input
      setWorkflowName(name)
    }
    try {
      const wf = serializeWorkflow(name, nodes, edges)
      await saveWorkflow(wf)
      addRunEvent({ type: 'info', data: { message: `Workflow "${name}" saved.` } })
    } catch (err) {
      addRunEvent({
        type: 'error',
        data: { message: `Save failed: ${err instanceof Error ? err.message : String(err)}` },
      })
    }
  }

  const executeRun = async (inputs: Record<string, string>) => {
    let name = workflowName
    if (!name) {
      const input = window.prompt('Workflow name:')
      if (!input) return
      name = input
      setWorkflowName(name)
    }

    clearRunEvents()
    clearNodeStatuses()
    setIsRunning(true)
    addRunEvent({ type: 'info', data: { message: `Running workflow "${name}"...` } })

    await runWorkflow(
      name,
      inputs,
      (event) => {
        addRunEvent(event)
        // ADK events: author = node ID
        const nodeId = event.data.author as string | undefined
        if (nodeId) {
          // Any event from a node means it's running
          setNodeStatus(nodeId, 'running')
        }
      },
      (result) => {
        addRunEvent({ type: 'done', data: result })
        // Mark all running nodes as completed or error
        const statuses = useWorkflowStore.getState().nodeStatuses
        const finalStatus = result.status === 'failed' ? 'error' : 'completed'
        for (const [id, status] of Object.entries(statuses)) {
          if (status === 'running') {
            setNodeStatus(id, finalStatus as NodeRunStatus)
          }
        }
        if (result.state && typeof result.state === 'object') {
          setSessionState(result.state as Record<string, unknown>)
        }
        setIsRunning(false)
      },
      (error) => {
        addRunEvent({ type: 'error', data: { message: error.message } })
        setIsRunning(false)
      },
    )
  }

  const handlePromptSubmit = async (description: string) => {
    setIsGenerating(true)
    const currentNodes = useWorkflowStore.getState().nodes
    const currentEdges = useWorkflowStore.getState().edges
    const hasExisting = currentNodes.length > 0

    const action = hasExisting ? 'Editing' : 'Generating'
    addRunEvent({ type: 'info', data: { message: `${action} workflow...` } })

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
        data: { message: `Workflow "${wf.name}" ${hasExisting ? 'updated' : 'generated'} with ${wf.nodes.length} nodes.` },
      })
    } catch (err) {
      addRunEvent({
        type: 'error',
        data: { message: `Generate failed: ${err instanceof Error ? err.message : String(err)}` },
      })
    } finally {
      setIsGenerating(false)
    }
  }

  useKeyboardShortcuts({
    onSave: handleSave,
    onRun: () => !isRunning && setShowRunDialog(true),
  })

  const inputNodes = nodes
    .filter((n) => n.data.nodeType === 'input')
    .map((n) => ({ id: n.id, label: n.data.label }))

  return (
    <div className="h-screen flex flex-col bg-background text-foreground">
      <Header
        workflowName={workflowName}
        onWorkflowNameChange={setWorkflowName}
        onSave={handleSave}
        onRun={() => !isRunning && setShowRunDialog(true)}
        isRunning={isRunning}
      />

      <div className="flex flex-1 overflow-hidden">
        <NodePalette onAddNode={handleAddNode} />

        <main className="flex-1">
          <Canvas
            onAddFirstNode={() => handleAddNode('input')}
            onDropNode={handleDropNode}
            onPromptSubmit={handlePromptSubmit}
            isGenerating={isGenerating}
          />
        </main>

        <RightPanel
          selectedNode={selectedNode ?? null}
          onCloseNode={() => selectNode(null)}
        />
      </div>

      <BottomConsole />

      <RunDialog
        open={showRunDialog}
        inputNodes={inputNodes}
        onRun={(inputs) => {
          setShowRunDialog(false)
          executeRun(inputs)
        }}
        onOpenChange={setShowRunDialog}
      />

    </div>
  )
}
