import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData } from '@/stores/workflowStore'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { X } from 'lucide-react'
import { nodeIconMap } from '@/lib/nodeTypes'
import { InputNodeEditor } from './InputNodeEditor'
import { AgentNodeEditor } from './AgentNodeEditor'
import { ToolNodeEditor } from './ToolNodeEditor'
import { OutputNodeEditor } from './OutputNodeEditor'
import { ExternalNodeEditor } from './ExternalNodeEditor'

export type NodeEditorFieldProps = {
  nodeId: string
  config: Record<string, unknown>
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
  tool: ToolNodeEditor,
  output: OutputNodeEditor,
  external: ExternalNodeEditor,
}

export function NodeEditor({ nodeId, data, onClose, embedded }: NodeEditorProps) {
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)

  const config = data.config

  const setConfig = (key: string, value: unknown) => {
    updateNodeConfig(nodeId, { [key]: value })
  }

  const Icon = nodeIconMap[data.nodeType]
  const TypeEditor = nodeEditors[data.nodeType]

  const formContent = (
    <div className="p-2 space-y-3">
      {/* Label field -- shared by all node types */}
      <div className="space-y-1">
        <Label htmlFor="node-label" className="text-xs">Label</Label>
        <Input
          id="node-label"
          className="h-7 text-xs"
          value={data.label}
          onChange={(e) => updateNodeLabel(nodeId, e.target.value)}
        />
      </div>

      {TypeEditor && <TypeEditor nodeId={nodeId} config={config} setConfig={setConfig} />}
    </div>
  )

  if (embedded) {
    return formContent
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
      {formContent}
    </aside>
  )
}
