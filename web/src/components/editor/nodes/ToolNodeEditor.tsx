import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import type { NodeEditorFieldProps } from './NodeEditor'

export function ToolNodeEditor({ config, setConfig }: NodeEditorFieldProps) {
  return (
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
  )
}
