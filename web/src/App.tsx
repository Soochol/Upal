import { useState } from 'react'
import { Canvas } from '@/components/editor/Canvas'
import { RunDialog } from '@/components/dialogs/RunDialog'
import { GenerateDialog } from '@/components/dialogs/GenerateDialog'
import { RightPanel } from '@/components/panel/RightPanel'
import { Header } from '@/components/Header'
import { NodePalette } from '@/components/sidebar/NodePalette'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { serializeWorkflow, deserializeWorkflow } from '@/lib/serializer'
import { saveWorkflow, runWorkflow, generateWorkflow } from '@/lib/api'

function App() {
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

  const [showRunDialog, setShowRunDialog] = useState(false)
  const [showGenerateDialog, setShowGenerateDialog] = useState(false)
  const [isGenerating, setIsGenerating] = useState(false)

  const selectedNode = selectedNodeId
    ? nodes.find((n) => n.id === selectedNodeId)
    : null

  const handleAddNode = (type: 'input' | 'agent' | 'tool' | 'output') => {
    addNode(type, {
      x: 250,
      y: useWorkflowStore.getState().nodes.length * 150 + 50,
    })
  }

  const handleDropNode = (type: string, position: { x: number; y: number }) => {
    addNode(type as 'input' | 'agent' | 'tool' | 'output', position)
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
        const nodeId = event.data.node_id as string | undefined
        if (nodeId) {
          if (event.type === 'node.started') setNodeStatus(nodeId, 'running')
          else if (event.type === 'node.completed') setNodeStatus(nodeId, 'completed')
          else if (event.type === 'node.error') setNodeStatus(nodeId, 'error')
        }
      },
      (result) => {
        addRunEvent({ type: 'done', data: result })
        setIsRunning(false)
      },
      (error) => {
        addRunEvent({ type: 'error', data: { message: error.message } })
        setIsRunning(false)
      },
    )
  }

  const handleGenerate = async (description: string, model?: string) => {
    setIsGenerating(true)
    addRunEvent({ type: 'info', data: { message: 'Generating workflow from description...' } })
    try {
      const wf = await generateWorkflow(description, model)
      const { nodes: newNodes, edges: newEdges } = deserializeWorkflow(wf)
      useWorkflowStore.setState({ nodes: newNodes, edges: newEdges })
      setWorkflowName(wf.name)
      addRunEvent({ type: 'info', data: { message: `Workflow "${wf.name}" generated with ${wf.nodes.length} nodes.` } })
      setShowGenerateDialog(false)
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
    onGenerate: () => setShowGenerateDialog(true),
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
        onGenerate={() => setShowGenerateDialog(true)}
        isRunning={isRunning}
      />

      <div className="flex flex-1 overflow-hidden">
        <NodePalette onAddNode={handleAddNode} />

        <main className="flex-1">
          <Canvas
            onAddFirstNode={() => handleAddNode('input')}
            onGenerate={() => setShowGenerateDialog(true)}
            onDropNode={handleDropNode}
          />
        </main>

        <RightPanel
          selectedNode={selectedNode ? { id: selectedNode.id, data: selectedNode.data } : null}
          onCloseNode={() => selectNode(null)}
        />
      </div>

      <RunDialog
        open={showRunDialog}
        inputNodes={inputNodes}
        onRun={(inputs) => {
          setShowRunDialog(false)
          executeRun(inputs)
        }}
        onOpenChange={setShowRunDialog}
      />

      <GenerateDialog
        open={showGenerateDialog}
        onGenerate={handleGenerate}
        onOpenChange={setShowGenerateDialog}
        isGenerating={isGenerating}
      />
    </div>
  )
}

export default App
