import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import type { InputNodeConfig } from '@/shared/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function InputNodeEditor({ config, setConfig }: NodeEditorFieldProps<InputNodeConfig>) {
  return (
    <div className="space-y-1">
      <Label htmlFor="node-prompt" className="text-xs">Prompt</Label>
      <Input
        id="node-prompt"
        className="h-7 text-xs"
        value={config.prompt ?? ''}
        placeholder="Enter prompt text..."
        onChange={(e) => setConfig('prompt', e.target.value)}
      />
    </div>
  )
}
