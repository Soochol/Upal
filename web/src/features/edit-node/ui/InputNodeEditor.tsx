import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { InputNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function InputNodeEditor({ config, setConfig }: NodeEditorFieldProps<InputNodeConfig>) {
  return (
    <div className="space-y-1">
      <Label htmlFor="node-placeholder" className="text-xs">Placeholder</Label>
      <Input
        id="node-placeholder"
        className="h-7 text-xs"
        value={config.placeholder ?? ''}
        placeholder="Enter placeholder text..."
        onChange={(e) => setConfig('placeholder', e.target.value)}
      />
    </div>
  )
}
