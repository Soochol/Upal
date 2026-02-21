import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { SensorNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function SensorNodeEditor({ config, setConfig }: NodeEditorFieldProps<SensorNodeConfig>) {
  const mode = config.mode ?? 'poll'

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label htmlFor="sensor-mode" className="text-xs">Mode</Label>
        <select
          id="sensor-mode"
          className="flex h-7 w-full rounded-md border border-input bg-transparent px-3 py-1 text-xs"
          value={mode}
          onChange={(e) => setConfig('mode', e.target.value)}
        >
          <option value="poll">HTTP Polling</option>
          <option value="webhook">Webhook (wait for resume)</option>
        </select>
      </div>

      {mode === 'poll' && (
        <>
          <div className="space-y-1">
            <Label htmlFor="sensor-url" className="text-xs">URL</Label>
            <Input
              id="sensor-url"
              className="h-7 text-xs font-mono"
              value={config.url ?? ''}
              placeholder="https://api.example.com/status"
              onChange={(e) => setConfig('url', e.target.value)}
            />
          </div>

          <div className="space-y-1">
            <Label htmlFor="sensor-condition" className="text-xs">Condition</Label>
            <Input
              id="sensor-condition"
              className="h-7 text-xs font-mono"
              value={config.condition ?? ''}
              placeholder='e.g. status == "ready"'
              onChange={(e) => setConfig('condition', e.target.value)}
            />
            <p className="text-[10px] text-muted-foreground">
              Expression evaluated against response. Empty = any 2xx response.
            </p>
          </div>

          <div className="space-y-1">
            <Label htmlFor="sensor-interval" className="text-xs">Interval (seconds)</Label>
            <Input
              id="sensor-interval"
              type="number"
              className="h-7 text-xs"
              value={config.interval ?? 10}
              min={1}
              onChange={(e) => setConfig('interval', Number(e.target.value))}
            />
          </div>
        </>
      )}

      <div className="space-y-1">
        <Label htmlFor="sensor-timeout" className="text-xs">Timeout (seconds)</Label>
        <Input
          id="sensor-timeout"
          type="number"
          className="h-7 text-xs"
          value={config.timeout ?? 300}
          min={1}
          onChange={(e) => setConfig('timeout', Number(e.target.value))}
        />
      </div>
    </div>
  )
}
