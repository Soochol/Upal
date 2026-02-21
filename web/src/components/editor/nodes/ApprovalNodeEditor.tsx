import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { ApprovalNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function ApprovalNodeEditor({ config, setConfig }: NodeEditorFieldProps<ApprovalNodeConfig>) {
  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label htmlFor="approval-conn" className="text-xs">Connection ID</Label>
        <Input
          id="approval-conn"
          className="h-7 text-xs"
          value={config.connection_id ?? ''}
          placeholder="e.g. conn-telegram-1"
          onChange={(e) => setConfig('connection_id', e.target.value)}
        />
        <p className="text-[10px] text-muted-foreground">
          Connection for sending the approval request notification.
        </p>
      </div>

      <div className="space-y-1">
        <Label htmlFor="approval-channel" className="text-xs">Channel Override</Label>
        <Input
          id="approval-channel"
          className="h-7 text-xs"
          value={config.channel ?? ''}
          placeholder="Optional: chat ID, channel name, or email"
          onChange={(e) => setConfig('channel', e.target.value)}
        />
      </div>

      <div className="space-y-1">
        <Label htmlFor="approval-msg" className="text-xs">Message Template</Label>
        <Textarea
          id="approval-msg"
          className="text-xs min-h-20"
          value={config.message ?? ''}
          placeholder="Please approve deployment of {{agent1}}"
          onChange={(e) => setConfig('message', e.target.value)}
        />
        <p className="text-[10px] text-muted-foreground">
          Use {'{{node_id}}'} to include upstream values.
        </p>
      </div>

      <div className="space-y-1">
        <Label htmlFor="approval-timeout" className="text-xs">Timeout (seconds)</Label>
        <Input
          id="approval-timeout"
          type="number"
          className="h-7 text-xs"
          value={config.timeout ?? 3600}
          min={1}
          onChange={(e) => setConfig('timeout', Number(e.target.value))}
        />
      </div>
    </div>
  )
}
