import { useState } from 'react'
import { Canvas } from './components/editor/Canvas'
import { Console } from './components/console/Console'
import { RunDialog } from './components/dialogs/RunDialog'
import { useWorkflowStore } from './stores/workflowStore'
import { serializeWorkflow } from './lib/serializer'
import { saveWorkflow, runWorkflow } from './lib/api'

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

  const [showRunDialog, setShowRunDialog] = useState(false)

  const handleAddNode = (type: 'input' | 'agent' | 'tool' | 'output') => {
    addNode(type, {
      x: 250,
      y: useWorkflowStore.getState().nodes.length * 150 + 50,
    })
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

  const handleRun = () => {
    if (isRunning) return
    setShowRunDialog(true)
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
    setIsRunning(true)
    addRunEvent({ type: 'info', data: { message: `Running workflow "${name}"...` } })

    await runWorkflow(
      name,
      inputs,
      (event) => {
        addRunEvent(event)
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

  const inputNodes = nodes
    .filter((n) => n.data.nodeType === 'input')
    .map((n) => ({ id: n.id, label: n.data.label }))

  return (
    <div className="h-screen flex flex-col bg-zinc-950 text-zinc-100">
      {/* Header */}
      <header className="flex items-center justify-between px-4 py-2 border-b border-zinc-800">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-bold">Upal</h1>
          <input
            className="bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-sm text-zinc-100 w-48 focus:outline-none focus:border-zinc-500"
            placeholder="Workflow name..."
            value={workflowName}
            onChange={(e) => setWorkflowName(e.target.value)}
          />
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleSave}
            className="px-3 py-1 bg-zinc-700 rounded text-sm hover:bg-zinc-600"
          >
            Save
          </button>
          <button
            onClick={handleRun}
            disabled={isRunning}
            className={`px-3 py-1 rounded text-sm ${isRunning ? 'bg-zinc-700 text-zinc-400 cursor-not-allowed' : 'bg-green-600 hover:bg-green-700'}`}
          >
            {isRunning ? 'Running...' : '\u25B6 Run'}
          </button>
        </div>
      </header>

      {/* Node Palette + Canvas */}
      <div className="flex flex-1 overflow-hidden">
        {/* Palette */}
        <aside className="w-48 border-r border-zinc-800 p-3 flex flex-col gap-2">
          <p className="text-xs text-zinc-500 uppercase font-medium">
            Add Step
          </p>
          {(['input', 'agent', 'tool', 'output'] as const).map((type) => (
            <button
              key={type}
              onClick={() => handleAddNode(type)}
              className="px-3 py-2 rounded border border-zinc-700 text-sm text-left hover:bg-zinc-800 capitalize"
            >
              {type === 'input' && '\u{1F7E1} '}
              {type === 'agent' && '\u{1F535} '}
              {type === 'tool' && '\u{1F534} '}
              {type === 'output' && '\u{1F7E2} '}
              {type}
            </button>
          ))}
        </aside>

        {/* Canvas */}
        <main className="flex-1">
          <Canvas />
        </main>
      </div>

      {/* Console */}
      <footer className="h-32 border-t border-zinc-800 p-3 overflow-y-auto">
        <Console />
      </footer>

      {/* Run Dialog */}
      {showRunDialog && (
        <RunDialog
          inputNodes={inputNodes}
          onRun={(inputs) => {
            setShowRunDialog(false)
            executeRun(inputs)
          }}
          onCancel={() => setShowRunDialog(false)}
        />
      )}
    </div>
  )
}

export default App
