import { useCallback, useRef, useState } from 'react'
import { ReactFlowProvider } from '@xyflow/react'
import { Canvas } from '@/widgets/workflow-canvas'
import { RightPanel } from '@/widgets/right-panel'
import { Header } from '@/shared/ui/Header'
import { NodePalette } from '@/widgets/node-palette'
import { Console } from '@/widgets/bottom-console'
import { useWorkflowStore } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { useUIStore } from '@/entities/ui'
import { useKeyboardShortcuts } from '@/features/manage-canvas'
import { useAutoSave } from '@/features/manage-canvas'
import { useReconnectRun } from '@/features/execute-workflow'
import type { NodeType } from '@/entities/node'
import { serializeWorkflow, deserializeWorkflow } from '@/entities/workflow'
import { generateWorkflow } from '@/entities/workflow'

export default function Editor() {
  const addNode = useWorkflowStore((s) => s.addNode)
  const nodes = useWorkflowStore((s) => s.nodes)
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const setWorkflowName = useWorkflowStore((s) => s.setWorkflowName)
  const addRunEvent = useExecutionStore((s) => s.addRunEvent)

  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const selectNode = useUIStore((s) => s.selectNode)

  const [isGenerating, setIsGenerating] = useState(false)
  const getViewportCenterRef = useRef<(() => { x: number; y: number }) | null>(null)

  const { saveStatus, saveNow } = useAutoSave()
  useReconnectRun()

  const selectedNode = selectedNodeId
    ? nodes.find((n) => n.id === selectedNodeId)
    : null

  const handleAddNode = (type: NodeType) => {
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

  useKeyboardShortcuts({
    onSave: saveNow,
  })

  return (
    <div className="h-screen flex flex-col bg-background text-foreground">
      <Header
        workflowName={workflowName}
        onWorkflowNameChange={setWorkflowName}
        saveStatus={saveStatus}
      />

      <div className="flex flex-1 overflow-hidden">
        <NodePalette onAddNode={handleAddNode} />

        <main className="flex-1">
          <ReactFlowProvider>
            <Canvas
              onAddFirstNode={() => handleAddNode('input')}
              onDropNode={handleDropNode}
              onPromptSubmit={handlePromptSubmit}
              isGenerating={isGenerating}
              exposeGetViewportCenter={handleExposeViewportCenter}
            />
          </ReactFlowProvider>
        </main>

        <RightPanel
          selectedNode={selectedNode ?? null}
          onCloseNode={() => selectNode(null)}
        />
      </div>

      <Console />
    </div>
  )
}
