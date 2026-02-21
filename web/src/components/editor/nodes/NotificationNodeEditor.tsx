import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { NotificationNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function NotificationNodeEditor({ config, setConfig }: NodeEditorFieldProps<NotificationNodeConfig>) {
  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label htmlFor="notif-conn" className="text-xs">Connection ID</Label>
        <Input
          id="notif-conn"
          className="h-7 text-xs"
          value={config.connection_id ?? ''}
          placeholder="e.g. conn-telegram-1"
          onChange={(e) => setConfig('connection_id', e.target.value)}
        />
        <p className="text-[10px] text-muted-foreground">
          ID of a configured connection (Telegram, Slack, SMTP).
        </p>
      </div>

      <div className="space-y-1">
        <Label htmlFor="notif-channel" className="text-xs">Channel Override</Label>
        <Input
          id="notif-channel"
          className="h-7 text-xs"
          value={config.channel ?? ''}
          placeholder="Optional: chat ID, channel name, or email"
          onChange={(e) => setConfig('channel', e.target.value)}
        />
      </div>

      <div className="space-y-1">
        <Label htmlFor="notif-msg" className="text-xs">Message Template</Label>
        <Textarea
          id="notif-msg"
          className="text-xs min-h-20"
          value={config.message ?? ''}
          placeholder="Workflow completed: {{output1}}"
          onChange={(e) => setConfig('message', e.target.value)}
        />
        <p className="text-[10px] text-muted-foreground">
          Use {'{{node_id}}'} to include upstream values in the message.
        </p>
      </div>
    </div>
  )
}
