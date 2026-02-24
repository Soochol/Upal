import { Loader2, ExternalLink } from 'lucide-react'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import type { ContentSession } from '@/entities/content-session'

/**
 * Build a collapsed summary string for the collect stage.
 * Example: "35 items from 3 sources (HN 12, Reddit 8, RSS 15)"
 */
export function collectSummary(session: ContentSession): string {
  const sources = session.sources ?? []
  if (sources.length === 0) return 'No sources'
  const totalItems = sources.reduce((sum, s) => sum + s.count, 0)
  const breakdown = sources.map((s) => `${s.label} ${s.count}`).join(', ')
  return `${totalItems} items from ${sources.length} source${sources.length === 1 ? '' : 's'} (${breakdown})`
}

interface CollectStageProps {
  session: ContentSession
}

export function CollectStage({ session }: CollectStageProps) {
  const isCollecting = session.status === 'collecting'
  const sources = session.sources ?? []

  const createdAt = new Date(session.created_at).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })

  return (
    <div className="space-y-6">
      {/* Loading state */}
      {isCollecting && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Collecting sources...
        </div>
      )}

      {/* Source groups */}
      {sources.map((src) => (
        <div key={src.id}>
          <div className="flex items-center gap-2 mb-2">
            <SourceTypeBadge type={src.source_type} />
            <span className="text-sm font-medium">{src.label}</span>
            <span className="text-xs text-muted-foreground">
              &middot; {src.count} items
            </span>
          </div>
          <div className="rounded-xl border border-border overflow-hidden">
            {(src.items ?? []).map((item, i) => (
              <div
                key={i}
                className="flex items-center gap-3 px-4 py-2.5 border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors"
              >
                {item.score != null && (
                  <span className="text-xs text-muted-foreground tabular-nums w-16 shrink-0">
                    {item.extra ?? item.score}
                  </span>
                )}
                <span className="text-sm flex-1 truncate">{item.title}</span>
                {item.url && (
                  <a
                    href={item.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-muted-foreground hover:text-foreground transition-colors shrink-0"
                  >
                    <ExternalLink className="h-3.5 w-3.5" />
                  </a>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}

      {/* Empty state */}
      {sources.length === 0 && !isCollecting && (
        <p className="text-sm text-muted-foreground">No sources collected.</p>
      )}

      {/* Metadata footer */}
      {(sources.length > 0 || isCollecting) && (
        <p className="text-xs text-muted-foreground pt-2 border-t border-border">
          Collected {createdAt} &middot; Trigger: {session.trigger_type}
        </p>
      )}
    </div>
  )
}
