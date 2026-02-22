import { useEffect, useState } from 'react'
import { Label } from '@/shared/ui/label'
import { Input } from '@/shared/ui/input'
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '@/shared/ui/collapsible'
import { ChevronRight } from 'lucide-react'
import { ModelSelector } from '@/shared/ui/ModelSelector'
import { ModelOptions } from '@/shared/ui/ModelOptions'
import { PromptEditor } from '@/shared/ui/PromptEditor'
import { TemplateText } from '@/shared/ui/TemplateText'
import { listTools, useModels } from '@/shared/api'
import type { ToolInfo } from '@/shared/types'
import { useUIStore } from '@/entities/ui'
import type { AgentNodeConfig } from '@/shared/lib/nodeConfigs'
import { fieldBox, type NodeEditorFieldProps } from './NodeEditor'

export function AgentNodeEditor({ nodeId, config, setConfig }: NodeEditorFieldProps<AgentNodeConfig>) {
  const [availableTools, setAvailableTools] = useState<ToolInfo[]>([])
  const [promptsOpen, setPromptsOpen] = useState(false)
  const [optionsOpen, setOptionsOpen] = useState(false)
  const [extractOpen, setExtractOpen] = useState(false)
  const selectedTools = config.tools ?? []
  const addToast = useUIStore((s) => s.addToast)
  const models = useModels()
  const selectedModel = models.find((m) => m.id === config.model)
  const toolsSupported = selectedModel?.supportsTools !== false

  useEffect(() => {
    listTools().then(setAvailableTools).catch((e) => addToast(`Failed to load tools: ${e.message}`))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-enable all tools for new agent nodes (config.tools not yet set).
  // Once the user toggles any tool, config.tools becomes an array and this
  // effect becomes a no-op, preserving the user's explicit choices.
  useEffect(() => {
    if (config.tools === undefined && availableTools.length > 0 && toolsSupported) {
      setConfig('tools', availableTools.map((t) => t.name))
    }
  }, [availableTools, config.tools, toolsSupported, setConfig])

  // Clear tools when switching to a model that doesn't support function calling.
  useEffect(() => {
    if (!toolsSupported && config.tools && config.tools.length > 0) {
      setConfig('tools', [])
    }
  }, [toolsSupported, config.tools, setConfig])

  const toggleTool = (name: string) => {
    const next = selectedTools.includes(name)
      ? selectedTools.filter((t) => t !== name)
      : [...selectedTools, name]
    setConfig('tools', next)
  }

  return (
    <div className="flex flex-col gap-3">
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

      {/* Tools toggles — hidden when selected model doesn't support function calling */}
      {toolsSupported && availableTools.length > 0 && (
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
      {!toolsSupported && selectedModel && (
        <p className="text-[11px] text-muted-foreground shrink-0">
          Tools are not supported by {selectedModel.name} (image generation model).
        </p>
      )}

      {/* Prompt — expands to fill remaining space */}
      <div className="flex flex-col gap-1">
        <Label className="text-xs shrink-0">Prompt</Label>
        <PromptEditor
          value={config.prompt ?? ''}
          onChange={(v) => setConfig('prompt', v)}
          nodeId={nodeId}
          placeholder="Use {{ to reference upstream node results..."
          className="min-h-40"
        />
      </div>

      {/* Collapsible: system prompt + output format (read-only) */}
      <Collapsible
        open={promptsOpen}
        onOpenChange={setPromptsOpen}
        className="shrink-0"
      >
        <CollapsibleTrigger className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors py-1">
          <ChevronRight className={`h-3 w-3 transition-transform ${promptsOpen ? 'rotate-90' : ''}`} />
          System Prompt / Output
        </CollapsibleTrigger>
        <CollapsibleContent className="flex flex-col gap-3 pt-1">
          <div className="flex flex-col gap-1">
            <Label className="text-xs shrink-0">System Prompt</Label>
            <div className={fieldBox}>
              <TemplateText text={config.system_prompt ?? ''} />
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <Label className="text-xs shrink-0">Output</Label>
            <div className={fieldBox}>
              <TemplateText text={config.output ?? ''} />
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>

      {/* Output Extraction — collapsible, closed by default */}
      <Collapsible
        open={extractOpen}
        onOpenChange={setExtractOpen}
        className="shrink-0"
      >
        <CollapsibleTrigger className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors py-1">
          <ChevronRight className={`h-3 w-3 transition-transform ${extractOpen ? 'rotate-90' : ''}`} />
          Output Extraction
        </CollapsibleTrigger>
        <CollapsibleContent className="flex flex-col gap-3 pt-1">
          <div className="flex flex-col gap-1">
            <Label className="text-xs">Mode</Label>
            <select
              value={config.output_extract?.mode ?? ''}
              onChange={(e) => {
                const m = e.target.value
                if (!m) {
                  setConfig('output_extract', undefined)
                } else {
                  setConfig('output_extract', { ...config.output_extract, mode: m as 'json' | 'tagged' })
                }
              }}
              className="h-8 w-full rounded-md border border-input bg-background px-2 text-xs"
            >
              <option value="">None (full response)</option>
              <option value="json">JSON key</option>
              <option value="tagged">XML tag</option>
            </select>
          </div>

          {config.output_extract?.mode === 'json' && (
            <div className="flex flex-col gap-1">
              <Label className="text-xs">JSON Key</Label>
              <Input
                value={config.output_extract?.key ?? ''}
                onChange={(e) =>
                  setConfig('output_extract', { mode: 'json', key: e.target.value })
                }
                placeholder="result"
                className="h-8 text-xs"
              />
              <p className="text-[11px] text-muted-foreground">
                LLM responds with{' '}
                <code className="font-mono">{`{"${config.output_extract?.key || 'result'}": ...}`}</code>
              </p>
            </div>
          )}

          {config.output_extract?.mode === 'tagged' && (
            <div className="flex flex-col gap-1">
              <Label className="text-xs">Tag Name</Label>
              <Input
                value={config.output_extract?.tag ?? ''}
                onChange={(e) =>
                  setConfig('output_extract', { mode: 'tagged', tag: e.target.value })
                }
                placeholder="artifact"
                className="h-8 text-xs"
              />
              <p className="text-[11px] text-muted-foreground">
                LLM wraps output in{' '}
                <code className="font-mono">{`<${config.output_extract?.tag || 'artifact'}>...</${config.output_extract?.tag || 'artifact'}>`}</code>
              </p>
            </div>
          )}
        </CollapsibleContent>
      </Collapsible>
    </div>
  )
}
