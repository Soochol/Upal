import { useState } from 'react'
import { ExternalLink, ArrowRight, Loader2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { MainLayout } from '@/app/layout'
import { fetchPublishedContent } from '@/entities/published-content'
import type { PublishedContent, PublishedChannel } from '@/entities/published-content'

// ─── Channel config ───────────────────────────────────────────────────────────

const CHANNEL_LABELS: Record<PublishedChannel, string> = {
  youtube: '📺 YouTube',
  substack: '📧 Substack',
  discord: '💬 Discord',
  telegram: '✈️ Telegram',
  other: '🔗 Other',
}

const CHANNEL_TABS: { label: string; value: PublishedChannel | 'all' }[] = [
  { label: 'All', value: 'all' },
  { label: 'YouTube', value: 'youtube' },
  { label: 'Substack', value: 'substack' },
  { label: 'Discord', value: 'discord' },
]

// ─── PublishedCard ────────────────────────────────────────────────────────────

function PublishedCard({
  item,
  onViewSession,
  onViewRun,
}: {
  item: PublishedContent
  onViewSession: () => void
  onViewRun: () => void
}) {
  const publishedAt = new Date(item.published_at).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })

  return (
    <div className="rounded-xl border border-border bg-card p-4 space-y-2">
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-xs font-medium text-muted-foreground">
          {CHANNEL_LABELS[item.channel]}
        </span>
        <span className="text-muted-foreground/50 text-xs">·</span>
        <span className="text-xs text-muted-foreground">{item.pipeline_name}</span>
        <span className="text-muted-foreground/50 text-xs">·</span>
        <span className="text-xs text-muted-foreground">{publishedAt}</span>
      </div>
      <p className="text-sm font-medium">{item.title}</p>
      <div className="flex items-center justify-between pt-1">
        <div className="flex items-center gap-3">
          {item.url && (
            <a
              href={item.url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              <ExternalLink className="h-3.5 w-3.5" />
              View Original
            </a>
          )}
          <button
            onClick={onViewRun}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            <ArrowRight className="h-3.5 w-3.5" />
            Run Results
          </button>
          <button
            onClick={onViewSession}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            <ArrowRight className="h-3.5 w-3.5" />
            Session {item.session_number}
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function PublishedPage() {
  const navigate = useNavigate()
  const [channelFilter, setChannelFilter] = useState<PublishedChannel | 'all'>('all')

  const { data: items = [], isLoading } = useQuery({
    queryKey: ['published-content', channelFilter],
    queryFn: () =>
      fetchPublishedContent(channelFilter !== 'all' ? { channel: channelFilter } : undefined),
  })

  return (
    <MainLayout headerContent={<span className="font-semibold">Published Content</span>}>
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8">

          <div className="flex items-start justify-between mb-8 gap-4">
            <div>
              <h1 className="landing-display text-2xl font-bold tracking-tight">Published Content</h1>
              {!isLoading && (
                <span className="text-sm text-muted-foreground mt-1.5 block">
                  <span className="text-foreground font-semibold tabular-nums">{items.length}</span>
                  {' '}items
                </span>
              )}
            </div>
          </div>

          <div className="flex items-center gap-1 p-1 rounded-xl bg-muted/30 w-fit mb-6 flex-wrap">
            {CHANNEL_TABS.map(({ label, value }) => (
              <button
                key={value}
                onClick={() => setChannelFilter(value)}
                className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer ${channelFilter === value
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                  }`}
              >
                {label}
              </button>
            ))}
          </div>

          {isLoading ? (
            <div className="flex justify-center py-16">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : items.length === 0 ? (
            <div className="text-center py-16 text-sm text-muted-foreground">
              No published content yet.
            </div>
          ) : (
            <div className="space-y-3">
              {items.map((item) => (
                <PublishedCard
                  key={item.id}
                  item={item}
                  onViewSession={() => navigate(`/pipelines?p=${item.pipeline_id}&s=${item.session_id}`)}
                  onViewRun={() => navigate(`/runs/${item.run_id}`)}
                />
              ))}
            </div>
          )}

        </div>
      </main>
    </MainLayout>
  )
}
