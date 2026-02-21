import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { BranchNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function BranchNodeEditor({ config, setConfig }: NodeEditorFieldProps<BranchNodeConfig>) {
  const mode = config.mode ?? 'expression'

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label htmlFor="branch-mode" className="text-xs">Mode</Label>
        <select
          id="branch-mode"
          className="flex h-7 w-full rounded-md border border-input bg-transparent px-3 py-1 text-xs"
          value={mode}
          onChange={(e) => setConfig('mode', e.target.value)}
        >
          <option value="expression">Expression</option>
          <option value="llm">LLM Classification</option>
        </select>
      </div>

      {mode === 'expression' && (
        <div className="space-y-1">
          <Label htmlFor="branch-expr" className="text-xs">Expression</Label>
          <Input
            id="branch-expr"
            className="h-7 text-xs font-mono"
            value={config.expression ?? ''}
            placeholder="e.g. sentiment == 'positive'"
            onChange={(e) => setConfig('expression', e.target.value)}
          />
          <p className="text-[10px] text-muted-foreground">
            Use {'{{node_id}}'} to reference upstream values. Result stored as &quot;true&quot; or &quot;false&quot;.
          </p>
        </div>
      )}

      {mode === 'llm' && (
        <>
          <div className="space-y-1">
            <Label htmlFor="branch-model" className="text-xs">Model</Label>
            <Input
              id="branch-model"
              className="h-7 text-xs"
              value={config.model ?? ''}
              placeholder="e.g. gemini/gemini-2.0-flash"
              onChange={(e) => setConfig('model', e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="branch-prompt" className="text-xs">Prompt</Label>
            <Textarea
              id="branch-prompt"
              className="text-xs min-h-20"
              value={config.prompt ?? ''}
              placeholder="Classify the following text: {{upstream_node}}"
              onChange={(e) => setConfig('prompt', e.target.value)}
            />
          </div>
        </>
      )}
    </div>
  )
}
