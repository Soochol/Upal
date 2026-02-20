import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PromptEditor } from '@/components/editor/PromptEditor'
import { ModelSelector } from '@/components/editor/ModelSelector'
import { OUTPUT_FORMATS, type OutputFormatId } from '@/lib/outputFormats'
import type { OutputNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function OutputNodeEditor({ nodeId, config, setConfig }: NodeEditorFieldProps<OutputNodeConfig>) {
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

      {/* Model — HTML only */}
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

      {/* System Prompt — HTML only */}
      {formatDef.editorFields.includes('system_prompt') && (
        <div className="space-y-1 shrink-0">
          <Label className="text-xs">System Prompt</Label>
          <PromptEditor
            value={config.system_prompt ?? ''}
            onChange={(v) => setConfig('system_prompt', v)}
            nodeId={nodeId}
            placeholder="Design direction: theme, layout, typography, colors..."
          />
          <p className="text-[10px] text-muted-foreground">
            Visual design direction for the HTML output page.
          </p>
        </div>
      )}

      {/* Prompt */}
      {formatDef.editorFields.includes('prompt') && (
        <div className="space-y-1 shrink-0">
          <Label className="text-xs">Prompt</Label>
          <PromptEditor
            value={config.prompt ?? ''}
            onChange={(v) => setConfig('prompt', v)}
            nodeId={nodeId}
            placeholder="Use {{ to reference upstream node results..."
          />
          <p className="text-[10px] text-muted-foreground">
            Select and arrange upstream data. Leave empty to collect all.
          </p>
        </div>
      )}
    </div>
  )
}
