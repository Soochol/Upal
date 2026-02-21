import { useEffect, useState } from 'react'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '@/components/ui/collapsible'
import { ChevronRight } from 'lucide-react'
import { PromptEditor } from '@/components/editor/PromptEditor'
import { ModelSelector } from '@/components/editor/ModelSelector'
import { OUTPUT_FORMATS, type OutputFormatId } from '@/lib/outputFormats'
import type { OutputNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function OutputNodeEditor({ nodeId, config, setConfig }: NodeEditorFieldProps<OutputNodeConfig>) {
  const [systemPromptOpen, setSystemPromptOpen] = useState(false)

  // Persist default output_format to config so backend and result renderer see it
  useEffect(() => {
    if (config.output_format === undefined) {
      setConfig('output_format', 'html')
    }
  }, [config.output_format, setConfig])

  const formatId = (config.output_format ?? 'html') as OutputFormatId
  const formatDef = OUTPUT_FORMATS[formatId]

  return (
    <div className="flex flex-col flex-1 min-h-0 gap-3">
      {/* Output Format */}
      <div className="space-y-1 shrink-0">
        <Label className="text-xs">Output Format</Label>
        <Select value={formatId} onValueChange={(v) => setConfig('output_format', v)}>
          <SelectTrigger className="h-7 text-xs w-full" size="sm">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {Object.values(OUTPUT_FORMATS).map((f) => (
              <SelectItem key={f.id} value={f.id} className="text-xs">
                {f.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Model */}
      {formatDef.editorFields.includes('model') && (
        <div className="space-y-1 shrink-0">
          <Label className="text-xs">Model</Label>
          <ModelSelector
            value={config.model ?? ''}
            onChange={(v) => setConfig('model', v)}
            placeholder="Default model"
          />
        </div>
      )}

      {/* Prompt — main expandable area */}
      {formatDef.editorFields.includes('prompt') && (
        <div className="flex-1 flex flex-col min-h-24 gap-1">
          <Label className="text-xs shrink-0">Prompt</Label>
          <PromptEditor
            value={config.prompt ?? ''}
            onChange={(v) => setConfig('prompt', v)}
            nodeId={nodeId}
            placeholder="Use {{ to reference upstream node results..."
            className="flex-1 min-h-24"
          />
          <p className="text-[10px] text-muted-foreground shrink-0">
            Select and arrange upstream data. Leave empty to collect all.
          </p>
        </div>
      )}

      {/* Collapsible: System Prompt */}
      {formatDef.editorFields.includes('system_prompt') && (
        <Collapsible
          open={systemPromptOpen}
          onOpenChange={setSystemPromptOpen}
          className={systemPromptOpen ? 'flex-1 flex flex-col min-h-0 shrink-0' : 'shrink-0'}
        >
          <CollapsibleTrigger className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors py-1">
            <ChevronRight className={`h-3 w-3 transition-transform ${systemPromptOpen ? 'rotate-90' : ''}`} />
            System Prompt
          </CollapsibleTrigger>
          <CollapsibleContent className="flex-1 flex flex-col gap-1 min-h-0 pt-1">
            <PromptEditor
              value={config.system_prompt ?? ''}
              onChange={(v) => setConfig('system_prompt', v)}
              nodeId={nodeId}
              placeholder="Design direction: theme, layout, typography, colors..."
              className="flex-1 min-h-20"
            />
            <p className="text-[10px] text-muted-foreground shrink-0">
              Visual design direction for the HTML output page.
            </p>
          </CollapsibleContent>
        </Collapsible>
      )}
    </div>
  )
}