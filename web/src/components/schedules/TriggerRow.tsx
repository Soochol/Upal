import { Copy, Check, Trash2, Webhook } from 'lucide-react'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import type { Trigger } from '@/lib/api'

type Props = {
  trigger: Trigger
  onDelete: (id: string) => void
}

export function TriggerRow({ trigger, onDelete }: Props) {
  const webhookUrl = `${window.location.origin}/api/hooks/${trigger.id}`
  const { copied: copiedUrl, copyToClipboard: copyUrl } = useCopyToClipboard()
  const { copied: copiedSecret, copyToClipboard: copySecret } = useCopyToClipboard()

  return (
    <div className="flex items-center justify-between py-3 px-4 border-b border-border last:border-0 hover:bg-muted/30 transition-colors">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <Webhook className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
          <span className="text-sm font-medium text-foreground truncate">
            {trigger.workflow_name}
          </span>
        </div>
        <div className="space-y-1">
          <div className="flex items-center gap-1.5">
            <code className="text-xs text-muted-foreground font-mono truncate max-w-[400px]">
              {webhookUrl}
            </code>
            <button
              onClick={() => copyUrl(webhookUrl)}
              className="p-0.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors shrink-0"
              title="Copy URL"
            >
              {copiedUrl ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
            </button>
          </div>
          {trigger.config.secret && (
            <div className="flex items-center gap-1.5">
              <code className="text-xs text-muted-foreground/60 font-mono">
                {trigger.config.secret.slice(0, 12)}...
              </code>
              <button
                onClick={() => copySecret(trigger.config.secret!)}
                className="p-0.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors shrink-0"
                title="Copy Secret"
              >
                {copiedSecret ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
              </button>
            </div>
          )}
        </div>
      </div>

      <button
        onClick={() => onDelete(trigger.id)}
        className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-destructive transition-colors ml-3"
        title="Delete"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}
