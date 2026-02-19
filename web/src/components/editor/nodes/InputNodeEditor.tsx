import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import type { NodeEditorFieldProps } from './NodeEditor'

export function InputNodeEditor({ config, setConfig }: NodeEditorFieldProps) {
  return (
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
  )
}
