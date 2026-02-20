import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData } from '@/stores/workflowStore'
import { Button } from '@/components/ui/button'
import { X } from 'lucide-react'
import { NODE_TYPES, type NodeType } from '@/lib/nodeTypes'
import { cn } from '@/lib/utils'
import { InputNodeEditor } from './InputNodeEditor'
import { AgentNodeEditor } from './AgentNodeEditor'
import { OutputNodeEditor } from './OutputNodeEditor'

/** Shared read-only field style used by Agent and Output node editors. */
const fieldBoxBase = "text-xs overflow-y-auto cursor-default select-text rounded-md border border-input bg-transparent px-3 py-2 whitespace-pre-wrap break-words"
export const fieldBox = fieldBoxBase + " max-h-32"
/** Expandable variant — grows via flex-1 to fill available vertical space. */
export const fieldBoxExpand = fieldBoxBase + " flex-1 min-h-24"

export type NodeEditorFieldProps<C = Record<string, unknown>> = {
  nodeId: string
  config: C
  setConfig: (key: string, value: unknown) => void
}

type NodeEditorProps = {
  nodeId: string
  data: NodeData
  onClose: () => void
  embedded?: boolean
}

const nodeEditors: Record<string, React.ComponentType<NodeEditorFieldProps>> = {
  input: InputNodeEditor,
  agent: AgentNodeEditor,
  output: OutputNodeEditor,
}


export function NodeEditor({ nodeId, data, onClose, embedded }: NodeEditorProps) {
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)

  const config = data.config

  const setConfig = (key: string, value: unknown) => {
    updateNodeConfig(nodeId, { [key]: value })
  }

  const ntCfg = NODE_TYPES[data.nodeType as NodeType]
  const Icon = ntCfg?.icon
  const TypeEditor = nodeEditors[data.nodeType]

  if (embedded) {
    return (
      <div className="flex flex-col flex-1 min-h-0">
        {/* Header: icon + inline-editable label + close */}
        <div className="flex items-center gap-2 px-3 py-2.5 border-b border-border">
          {Icon && (
            <div className={cn(
              'h-6 w-6 rounded-md flex items-center justify-center shrink-0',
              ntCfg?.accent,
            )}>
              <Icon className="h-3.5 w-3.5" />
            </div>
          )}
          <input
            className="flex-1 min-w-0 text-sm font-semibold bg-transparent border-none outline-none focus:ring-1 focus:ring-ring rounded px-1"
            value={data.label}
            onChange={(e) => updateNodeLabel(nodeId, e.target.value)}
          />
          <Button variant="ghost" size="icon" className="h-6 w-6 shrink-0" onClick={onClose}>
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
        {/* Form body — description editing intentionally omitted (set via AI Chat / generation) */}
        <div className="p-2 flex-1 flex flex-col gap-3 min-h-0">
          {TypeEditor && <TypeEditor nodeId={nodeId} config={config} setConfig={setConfig} />}
        </div>
      </div>
    )
  }

  return (
    <aside className="w-80 border-l border-border bg-background p-0 overflow-y-auto">
      {/* Panel header */}
      <div className="flex items-center justify-between p-4 border-b border-border">
        <div className="flex items-center gap-2">
          {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
          <h3 className="text-sm font-semibold">Properties</h3>
        </div>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>
      <div className="p-2 space-y-3">
        {TypeEditor && <TypeEditor nodeId={nodeId} config={config} setConfig={setConfig} />}
      </div>
    </aside>
  )
}
