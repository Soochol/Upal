import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData } from '@/stores/workflowStore'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Button } from '@/components/ui/button'
import { X, Inbox, Bot, Wrench, ArrowRightFromLine, Globe } from 'lucide-react'

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  input: Inbox,
  agent: Bot,
  tool: Wrench,
  output: ArrowRightFromLine,
  external: Globe,
}

type NodeEditorProps = {
  nodeId: string
  data: NodeData
  onClose: () => void
  embedded?: boolean
}

export function NodeEditor({ nodeId, data, onClose, embedded }: NodeEditorProps) {
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)
  const updateNodeDescription = useWorkflowStore((s) => s.updateNodeDescription)

  const config = data.config

  const setConfig = (key: string, value: unknown) => {
    updateNodeConfig(nodeId, { [key]: value })
  }

  const Icon = iconMap[data.nodeType]

  const formContent = (
    <div className="p-4 space-y-4">
      {/* Label field -- shared by all node types */}
      <div className="space-y-2">
        <Label htmlFor="node-label">Label</Label>
        <Input
          id="node-label"
          value={data.label}
          onChange={(e) => updateNodeLabel(nodeId, e.target.value)}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="node-description">Description</Label>
        <Textarea
          id="node-description"
          className="min-h-[60px] resize-y"
          value={data.description ?? ''}
          placeholder="Describe what this node does..."
          onChange={(e) => updateNodeDescription(nodeId, e.target.value)}
        />
      </div>

      {data.nodeType === 'input' && (
        <>
          <Separator />
          <div className="space-y-2">
            <Label htmlFor="node-placeholder">Placeholder</Label>
            <Input
              id="node-placeholder"
              value={(config.placeholder as string) ?? ''}
              placeholder="Enter placeholder text..."
              onChange={(e) => setConfig('placeholder', e.target.value)}
            />
          </div>
        </>
      )}

      {data.nodeType === 'agent' && (
        <>
          <Separator />
          <div className="space-y-2">
            <Label htmlFor="node-model">Model ID</Label>
            <Input
              id="node-model"
              value={(config.model as string) ?? ''}
              placeholder="ollama/llama3"
              onChange={(e) => setConfig('model', e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="node-system-prompt">System Prompt</Label>
            <Textarea
              id="node-system-prompt"
              className="min-h-[80px] resize-y"
              value={(config.system_prompt as string) ?? ''}
              placeholder="You are a helpful assistant..."
              onChange={(e) => setConfig('system_prompt', e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="node-user-prompt">User Prompt</Label>
            <Textarea
              id="node-user-prompt"
              value={(config.prompt as string) ?? ''}
              placeholder="Type {{ to reference a node..."
              onChange={(e) => setConfig('prompt', e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="node-max-turns">Max Turns</Label>
            <Input
              id="node-max-turns"
              type="number"
              min={1}
              value={(config.max_turns as number) ?? 1}
              onChange={(e) => setConfig('max_turns', parseInt(e.target.value) || 1)}
            />
          </div>
        </>
      )}

      {data.nodeType === 'tool' && (
        <>
          <Separator />
          <div className="space-y-2">
            <Label htmlFor="node-tool-name">Tool Name</Label>
            <Input
              id="node-tool-name"
              value={(config.tool_name as string) ?? ''}
              placeholder="web_search"
              onChange={(e) => setConfig('tool_name', e.target.value)}
            />
          </div>
        </>
      )}

      {data.nodeType === 'external' && (
        <>
          <Separator />
          <div className="space-y-2">
            <Label htmlFor="node-endpoint-url">Endpoint URL</Label>
            <Input
              id="node-endpoint-url"
              value={(config.endpoint_url as string) ?? ''}
              placeholder="https://agent.example.com/a2a"
              onChange={(e) => setConfig('endpoint_url', e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="node-timeout">Timeout (seconds)</Label>
            <Input
              id="node-timeout"
              type="number"
              min={1}
              value={(config.timeout as number) ?? 30}
              onChange={(e) => setConfig('timeout', parseInt(e.target.value) || 30)}
            />
          </div>
        </>
      )}
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
