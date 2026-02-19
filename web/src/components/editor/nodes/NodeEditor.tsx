import { useEffect, useState } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData } from '@/stores/workflowStore'
import { Input } from '@/components/ui/input'
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
import { X } from 'lucide-react'
import { listModels, type ModelInfo } from '@/lib/api'
import { groupModelsByProvider } from '@/lib/utils'
import { PromptEditor } from '@/components/editor/PromptEditor'
import { nodeIconMap } from '@/lib/nodeTypes'

type NodeEditorProps = {
  nodeId: string
  data: NodeData
  onClose: () => void
  embedded?: boolean
}

export function NodeEditor({ nodeId, data, onClose, embedded }: NodeEditorProps) {
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)

  const [models, setModels] = useState<ModelInfo[]>([])

  useEffect(() => {
    listModels().then(setModels).catch(() => setModels([]))
  }, [])

  const modelsByProvider = groupModelsByProvider(models)

  const config = data.config

  const setConfig = (key: string, value: unknown) => {
    updateNodeConfig(nodeId, { [key]: value })
  }

  const Icon = nodeIconMap[data.nodeType]

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

      {data.nodeType === 'output' && (
        <>
          <Separator />
          <div className="space-y-1">
            <Label className="text-xs">Display Mode</Label>
            <Select
              value={(config.display_mode as string) ?? 'manual'}
              onValueChange={(v) => setConfig('display_mode', v)}
            >
              <SelectTrigger className="h-7 text-xs w-full" size="sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="manual" className="text-xs">
                  Manual layout
                </SelectItem>
                <SelectItem value="auto-layout" className="text-xs">
                  Webpage with auto-layout
                </SelectItem>
              </SelectContent>
            </Select>
            <p className="text-[10px] text-muted-foreground">
              {(config.display_mode as string) === 'auto-layout'
                ? 'Layout automatically generated by AI'
                : 'Custom layout generated from your instructions'}
            </p>
          </div>

          {/* Manual layout: user-provided layout prompt + model selector */}
          {(config.display_mode as string) !== 'auto-layout' && (
            <>
              <div className="space-y-1">
                <Label className="text-xs">Layout Prompt</Label>
                <PromptEditor
                  value={(config.layout_prompt as string) ?? ''}
                  onChange={(v) => setConfig('layout_prompt', v)}
                  nodeId={nodeId}
                  placeholder="Describe your page layout, style, and component structure. Use {{ to reference upstream node outputs..."
                />
                <p className="text-[10px] text-muted-foreground">
                  Describe layout, style, and components. Reference node outputs with {'{{node_id}}'}.
                </p>
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Layout Model</Label>
                <Select
                  value={(config.layout_model as string) ?? ''}
                  onValueChange={(v) => setConfig('layout_model', v)}
                >
                  <SelectTrigger className="h-7 text-xs w-full" size="sm">
                    <SelectValue placeholder="Default model" />
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
            </>
          )}

          {(config.display_mode as string) === 'auto-layout' && (
            <div className="space-y-1">
              <Label className="text-xs">Layout Model</Label>
              <Select
                value={(config.layout_model as string) ?? ''}
                onValueChange={(v) => setConfig('layout_model', v)}
              >
                <SelectTrigger className="h-7 text-xs w-full" size="sm">
                  <SelectValue placeholder="Default model" />
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
          )}
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
