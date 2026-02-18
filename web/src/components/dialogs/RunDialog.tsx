import { useState } from 'react'

export type RunDialogProps = {
  inputNodes: Array<{ id: string; label: string }>
  onRun: (inputs: Record<string, string>) => void
  onCancel: () => void
}

export function RunDialog({ inputNodes, onRun, onCancel }: RunDialogProps) {
  const [inputs, setInputs] = useState<Record<string, string>>({})

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onCancel()
    }
  }

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onKeyDown={handleKeyDown}
      onClick={onCancel}
    >
      <div
        className="bg-zinc-900 border border-zinc-700 rounded-lg p-6 w-full max-w-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-bold mb-4">Run Workflow</h2>
        <p className="text-sm text-zinc-400 mb-4">
          Provide input values for each input node in your workflow.
        </p>
        {inputNodes.map((node) => (
          <div key={node.id} className="mb-3">
            <label className="text-sm text-zinc-400 block mb-1">{node.label}</label>
            <textarea
              className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1.5 text-sm text-zinc-100 focus:outline-none focus:border-zinc-500 resize-y min-h-[60px]"
              value={inputs[node.id] ?? ''}
              onChange={(e) => setInputs({ ...inputs, [node.id]: e.target.value })}
              placeholder={`Enter value for ${node.label}...`}
              autoFocus={inputNodes[0]?.id === node.id}
            />
          </div>
        ))}
        {inputNodes.length === 0 && (
          <p className="text-sm text-zinc-500 mb-4">
            No input nodes found. The workflow will run with empty inputs.
          </p>
        )}
        <div className="flex justify-end gap-2 mt-4">
          <button
            onClick={onCancel}
            className="px-4 py-1.5 bg-zinc-700 rounded text-sm hover:bg-zinc-600"
          >
            Cancel
          </button>
          <button
            onClick={() => onRun(inputs)}
            className="px-4 py-1.5 bg-green-600 rounded text-sm hover:bg-green-700"
          >
            Run
          </button>
        </div>
      </div>
    </div>
  )
}
