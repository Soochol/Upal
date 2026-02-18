import { useWorkflowStore } from '../../../stores/workflowStore'
import type { NodeData } from '../../../stores/workflowStore'

const inputClass =
  'w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-sm text-zinc-100 focus:outline-none focus:border-zinc-500'
const textareaClass =
  'w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-sm text-zinc-100 focus:outline-none focus:border-zinc-500 resize-y min-h-[60px]'
const labelClass = 'text-xs text-zinc-400 mt-2 mb-0.5 block'

type NodeEditorProps = {
  nodeId: string
  data: NodeData
}

export function NodeEditor({ nodeId, data }: NodeEditorProps) {
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)

  const config = data.config

  const setConfig = (key: string, value: unknown) => {
    updateNodeConfig(nodeId, { [key]: value })
  }

  return (
    <div
      className="mt-2 pt-2 border-t border-zinc-700/50 flex flex-col gap-0.5"
      onClick={(e) => e.stopPropagation()}
    >
      {/* Label field -- shared by all node types */}
      <label className={labelClass}>Label</label>
      <input
        className={inputClass}
        value={data.label}
        onChange={(e) => updateNodeLabel(nodeId, e.target.value)}
      />

      {data.nodeType === 'input' && (
        <>
          <label className={labelClass}>Placeholder</label>
          <input
            className={inputClass}
            value={(config.placeholder as string) ?? ''}
            placeholder="Enter placeholder text..."
            onChange={(e) => setConfig('placeholder', e.target.value)}
          />
        </>
      )}

      {data.nodeType === 'agent' && (
        <>
          <label className={labelClass}>Model ID</label>
          <input
            className={inputClass}
            value={(config.model as string) ?? ''}
            placeholder="ollama/llama3"
            onChange={(e) => setConfig('model', e.target.value)}
          />
          <label className={labelClass}>System Prompt</label>
          <textarea
            className={textareaClass}
            value={(config.system_prompt as string) ?? ''}
            placeholder="You are a helpful assistant..."
            onChange={(e) => setConfig('system_prompt', e.target.value)}
          />
          <label className={labelClass}>User Prompt</label>
          <textarea
            className={textareaClass}
            value={(config.user_prompt as string) ?? ''}
            placeholder="{{input}}"
            onChange={(e) => setConfig('user_prompt', e.target.value)}
          />
          <label className={labelClass}>Max Turns</label>
          <input
            className={inputClass}
            type="number"
            min={1}
            value={(config.max_turns as number) ?? 1}
            onChange={(e) => setConfig('max_turns', parseInt(e.target.value) || 1)}
          />
        </>
      )}

      {data.nodeType === 'tool' && (
        <>
          <label className={labelClass}>Tool Name</label>
          <input
            className={inputClass}
            value={(config.tool_name as string) ?? ''}
            placeholder="web_search"
            onChange={(e) => setConfig('tool_name', e.target.value)}
          />
        </>
      )}

      {/* Output node only has the label field, which is already rendered above */}
    </div>
  )
}
