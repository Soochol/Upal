import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import type { NodeEditorFieldProps } from './NodeEditor'

export function ExternalNodeEditor({ config, setConfig }: NodeEditorFieldProps) {
  return (
    <>
      <Separator />
      <div className="space-y-1">
        <Label htmlFor="node-endpoint-url" className="text-xs">Endpoint URL</Label>
        <Input
          id="node-endpoint-url"
          className="h-7 text-xs"
          value={(config.endpoint_url as string) ?? ''}
          placeholder="https://agent.example.com/a2a"
          onChange={(e) => setConfig('endpoint_url', e.target.value)}
        />
      </div>
      <div className="space-y-1">
        <Label htmlFor="node-timeout" className="text-xs">Timeout (s)</Label>
        <Input
          id="node-timeout"
          className="h-7 text-xs"
          type="number"
          min={1}
          value={(config.timeout as number) ?? 30}
          onChange={(e) => setConfig('timeout', parseInt(e.target.value) || 30)}
        />
      </div>
    </>
  )
}
