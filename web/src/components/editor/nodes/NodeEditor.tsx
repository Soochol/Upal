import { useEffect, useState } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData } from '@/stores/workflowStore'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { X, Inbox, Bot, Wrench, ArrowRightFromLine, Globe } from 'lucide-react'
import { listModels, type ModelInfo } from '@/lib/api'
import { PromptEditor } from '@/components/editor/PromptEditor'

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

  const [models, setModels] = useState<ModelInfo[]>([])

  useEffect(() => {
    listModels().then(setModels).catch(() => setModels([]))
  }, [])

  // Group models by provider for the Select dropdown
  const modelsByProvider = models.reduce<Record<string, ModelInfo[]>>((acc, m) => {
    ;(acc[m.provider] ??= []).push(m)
    return acc
  }, {})

  const config = data.config

  const setConfig = (key: string, value: unknown) => {
    updateNodeConfig(nodeId, { [key]: value })
  }

  const Icon = iconMap[data.nodeType]

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

      <div className="space-y-1">
        <Label htmlFor="node-description" className="text-xs">Description</Label>
        <Textarea
          id="node-description"
          className="min-h-[48px] resize-y text-xs"
          value={data.description ?? ''}
          placeholder="Describe what this node does..."
          onChange={(e) => updateNodeDescription(nodeId, e.target.value)}
        />
      </div>

      {data.nodeType === 'input' && (
        <>
          <Separator />
          <div className="space-y-1">
            <Label htmlFor="node-placeholder" className="text-xs">Placeholder</Label>
            <Input
              id="node-placeholder"
              className="h-7 text-xs"
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
          <div className="space-y-1">
            <Label className="text-xs">Model</Label>
            <Select
              value={(config.model as string) ?? ''}
              onValueChange={(v) => setConfig('model', v)}
            >
              <SelectTrigger className="h-7 text-xs w-full" size="sm">
                <SelectValue placeholder="Select a model..." />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(modelsByProvider).map(([provider, providerModels]) => (
                  <SelectGroup key={provider}>
                    <SelectLabel>{provider}</SelectLabel>
                    {providerModels.map((m) => (
                      <SelectItem key={m.id} value={m.id} className="text-xs">
                        {m.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                ))}
                {models.length === 0 && (
                  <div className="px-2 py-4 text-xs text-muted-foreground text-center">
                    No models available.<br />Configure providers in config.yaml
                  </div>
                )}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="node-system-prompt" className="text-xs">System Prompt</Label>
            <Textarea
              id="node-system-prompt"
              className="min-h-[60px] resize-y text-xs"
              value={(config.system_prompt as string) ?? ''}
              placeholder="You are a helpful assistant..."
              onChange={(e) => setConfig('system_prompt', e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">User Prompt</Label>
            <PromptEditor
              value={(config.prompt as string) ?? ''}
              onChange={(v) => setConfig('prompt', v)}
              nodeId={nodeId}
              placeholder="Type {{ to reference a node..."
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="node-max-turns" className="text-xs">Max Turns</Label>
            <Input
              id="node-max-turns"
              className="h-7 text-xs"
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
          <div className="space-y-1">
            <Label htmlFor="node-tool-name" className="text-xs">Tool Name</Label>
            <Input
              id="node-tool-name"
              className="h-7 text-xs"
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
          <div className="space-y-1">
            <Label htmlFor="node-endpoint-url" className="text-xs">Endpoint URL</Label>
            <Input
              id="node-endpoint-url"
              className="h-7 text-xs"
              value={(config.endpoint_url as string) ?? ''}
              placeholder="https://agent.example.com/a2a"
              onChange={(e) => setConfig('endpoint_url', e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="node-timeout" className="text-xs">Timeout (s)</Label>
            <Input
              id="node-timeout"
              className="h-7 text-xs"
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
