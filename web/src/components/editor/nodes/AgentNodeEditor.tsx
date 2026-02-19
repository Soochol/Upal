import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { ModelSelector } from '@/components/editor/ModelSelector'
import type { NodeEditorFieldProps } from './NodeEditor'

export function AgentNodeEditor({ config, setConfig }: NodeEditorFieldProps) {
  return (
    <>
      <Separator />
      <div className="space-y-1">
        <Label className="text-xs">Model</Label>
        <ModelSelector
          value={(config.model as string) ?? ''}
          onChange={(v) => setConfig('model', v)}
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
  )
}
