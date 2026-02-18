import { Canvas } from './components/editor/Canvas'
import { useWorkflowStore } from './stores/workflowStore'

function App() {
  const addNode = useWorkflowStore((s) => s.addNode)

  const handleAddNode = (type: 'input' | 'agent' | 'tool' | 'output') => {
    addNode(type, {
      x: 250,
      y: useWorkflowStore.getState().nodes.length * 150 + 50,
    })
  }

  return (
    <div className="h-screen flex flex-col bg-zinc-950 text-zinc-100">
      {/* Header */}
      <header className="flex items-center justify-between px-4 py-2 border-b border-zinc-800">
        <h1 className="text-lg font-bold">Upal</h1>
        <div className="flex gap-2">
          <button className="px-3 py-1 bg-green-600 rounded text-sm hover:bg-green-700">
            {'\u25B6'} Run
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
        <p className="text-xs text-zinc-500 uppercase font-medium">Console</p>
        <p className="text-sm text-zinc-600 mt-1">
          Ready. Add nodes and connect them to build a workflow.
        </p>
      </footer>
    </div>
  )
}

export default App
