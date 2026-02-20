import { useEffect, useState } from 'react'
import { Label } from '@/components/ui/label'
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '@/components/ui/collapsible'
import { ChevronRight } from 'lucide-react'
import { ModelSelector } from '@/components/editor/ModelSelector'
import { ModelOptions } from '@/components/editor/ModelOptions'
import { TemplateText } from '@/components/editor/TemplateText'
import { listTools, type ToolInfo } from '@/lib/api'
import { useModels } from '@/hooks/useModels'
import { useUIStore } from '@/stores/uiStore'
import type { AgentNodeConfig } from '@/lib/nodeConfigs'
import { fieldBoxExpand, type NodeEditorFieldProps } from './NodeEditor'

export function AgentNodeEditor({ config, setConfig }: NodeEditorFieldProps<AgentNodeConfig>) {
  const [availableTools, setAvailableTools] = useState<ToolInfo[]>([])
  const [promptsOpen, setPromptsOpen] = useState(false)
  const [optionsOpen, setOptionsOpen] = useState(false)
  const selectedTools = config.tools ?? []
  const addToast = useUIStore((s) => s.addToast)
  const models = useModels()
  const selectedModel = models.find((m) => m.id === config.model)

  useEffect(() => {
    listTools().then(setAvailableTools).catch((e) => addToast(`Failed to load tools: ${e.message}`))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-enable all tools for new agent nodes (config.tools not yet set).
  // Once the user toggles any tool, config.tools becomes an array and this
  // effect becomes a no-op, preserving the user's explicit choices.
  useEffect(() => {
    if (config.tools === undefined && availableTools.length > 0) {
      setConfig('tools', availableTools.map((t) => t.name))
    }
  }, [availableTools, config.tools, setConfig])

  const toggleTool = (name: string) => {
    const next = selectedTools.includes(name)
      ? selectedTools.filter((t) => t !== name)
      : [...selectedTools, name]
    setConfig('tools', next)
  }

  return (
    <div className="flex flex-col flex-1 min-h-0 gap-3">
      {/* Model selector — fixed height */}
      <div className="space-y-1 shrink-0">
        <Label className="text-xs">Model</Label>
        <ModelSelector
          value={config.model ?? ''}
          onChange={(v) => setConfig('model', v)}
          models={models}
        />
      </div>

      {/* Model-specific options — collapsible */}
      {selectedModel && selectedModel.options?.length > 0 && (
        <Collapsible
          open={optionsOpen}
          onOpenChange={setOptionsOpen}
          className="shrink-0"
        >
          <CollapsibleTrigger className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors py-1">
            <ChevronRight className={`h-3 w-3 transition-transform ${optionsOpen ? 'rotate-90' : ''}`} />
            Model Options
          </CollapsibleTrigger>
          <CollapsibleContent className="pt-1">
            <ModelOptions
              options={selectedModel.options}
              values={config as Record<string, unknown>}
              onChange={setConfig}
            />
          </CollapsibleContent>
        </Collapsible>
      )}

      {/* Tools toggles — fixed height */}
      {availableTools.length > 0 && (
        <div className="space-y-1.5 shrink-0">
          <Label className="text-xs">Tools</Label>
          <div className="flex flex-wrap gap-1.5">
            {availableTools.map((tool) => {
              const active = selectedTools.includes(tool.name)
              return (
                <button
                  key={tool.name}
                  type="button"
                  onClick={() => toggleTool(tool.name)}
                  title={tool.description}
                  className={`px-2 py-0.5 rounded-md text-[11px] border transition-colors ${
                    active
                      ? 'bg-primary text-primary-foreground border-primary'
                      : 'bg-transparent text-muted-foreground border-border hover:border-foreground/30'
                  }`}
                >
                  {tool.name}
                </button>
              )
            })}
          </div>
        </div>
      )}

      {/* Prompt — expands to fill remaining space */}
      <div className="flex-1 flex flex-col min-h-24 gap-1">
        <Label className="text-xs shrink-0">Prompt</Label>
        <div className={fieldBoxExpand}>
          <TemplateText text={config.prompt ?? ''} />
        </div>
      </div>

      {/* Collapsible: system prompt + output format (read-only) */}
      <Collapsible
        open={promptsOpen}
        onOpenChange={setPromptsOpen}
        className={promptsOpen ? 'flex-1 flex flex-col min-h-0 shrink-0' : 'shrink-0'}
      >
        <CollapsibleTrigger className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors py-1">
          <ChevronRight className={`h-3 w-3 transition-transform ${promptsOpen ? 'rotate-90' : ''}`} />
          System Prompt / Output
        </CollapsibleTrigger>
        <CollapsibleContent className="flex-1 flex flex-col gap-3 min-h-0 pt-1">
          <div className="flex-1 flex flex-col min-h-20 gap-1">
            <Label className="text-xs shrink-0">System Prompt</Label>
            <div className={fieldBoxExpand}>
              <TemplateText text={config.system_prompt ?? ''} />
            </div>
          </div>
          <div className="flex-1 flex flex-col min-h-20 gap-1">
            <Label className="text-xs shrink-0">Output</Label>
            <div className={fieldBoxExpand}>
              <TemplateText text={config.output ?? ''} />
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  )
}
