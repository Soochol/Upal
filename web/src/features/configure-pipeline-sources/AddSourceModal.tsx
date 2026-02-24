import { useState, type ReactNode } from 'react'
import { X, Rss, Flame, MessageCircle, TrendingUp, Hash, Globe } from 'lucide-react'
import { KeywordTagInput } from '@/shared/ui/KeywordTagInput'
import type { PipelineSource, PipelineSourceType } from '@/shared/types'

type SourceTypeDef = {
  type: PipelineSourceType
  source_type: 'static' | 'signal'
  label: string
  description: string
  icon: ReactNode
  accent: string          // tailwind color class for icon bg
  accentText: string      // tailwind color class for icon fg
}

const STATIC_SOURCES: SourceTypeDef[] = [
  { type: 'rss',  source_type: 'static', label: 'RSS Feed',      description: 'RSS/Atom feed URLs',       icon: <Rss className="h-4 w-4" />,   accent: 'bg-warning/12', accentText: 'text-warning' },
  { type: 'hn',   source_type: 'static', label: 'Hacker News',   description: 'Top HN stories',           icon: <Flame className="h-4 w-4" />, accent: 'bg-[oklch(0.75_0.15_50)]/12', accentText: 'text-[oklch(0.65_0.15_50)]' },
  { type: 'http', source_type: 'static', label: 'HTTP Endpoint',  description: 'Custom HTTP API',          icon: <Globe className="h-4 w-4" />, accent: 'bg-muted', accentText: 'text-muted-foreground' },
]

const SIGNAL_SOURCES: SourceTypeDef[] = [
  { type: 'reddit',        source_type: 'signal', label: 'Reddit',         description: 'Subreddit hot posts',        icon: <MessageCircle className="h-4 w-4" />, accent: 'bg-[oklch(0.7_0.15_25)]/12', accentText: 'text-[oklch(0.6_0.15_25)]' },
  { type: 'google_trends', source_type: 'signal', label: 'Google Trends',  description: 'Keyword search spikes',      icon: <TrendingUp className="h-4 w-4" />,    accent: 'bg-info/12', accentText: 'text-info' },
  { type: 'twitter',       source_type: 'signal', label: 'X / Twitter',    description: 'Trending keywords',          icon: <Hash className="h-4 w-4" />,       accent: 'bg-foreground/8', accentText: 'text-foreground' },
]


type Props = {
  onAdd: (source: PipelineSource) => void
  onClose: () => void
}

function generateId() {
  return `src-${crypto.randomUUID()}`
}

export function AddSourceModal({ onAdd, onClose }: Props) {
  const [step, setStep] = useState<'select' | 'config'>('select')
  const [selectedType, setSelectedType] = useState<SourceTypeDef | null>(null)
  const [draft, setDraft] = useState<Partial<PipelineSource>>({})

  const handleSelectType = (typeDef: SourceTypeDef) => {
    setSelectedType(typeDef)
    setDraft({
      id: generateId(),
      type: typeDef.type,
      source_type: typeDef.source_type,
      label: typeDef.label,
      limit: 20,
      keywords: [],
    })
    setStep('config')
  }

  const handleAdd = () => {
    if (!draft.id || !draft.type) return
    onAdd(draft as PipelineSource)
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-md mx-4 overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="text-sm font-semibold">
            {step === 'select' ? 'Add Source' : `Configure ${selectedType?.label}`}
          </h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="p-5">
          {step === 'select' ? (
            <div className="space-y-5">
              {/* Static data sources */}
              <div>
                <div className="flex items-center gap-2 mb-2.5">
                  <span className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">Data Sources</span>
                  <div className="flex-1 h-px bg-border" />
                </div>
                <div className="space-y-1.5">
                  {STATIC_SOURCES.map((typeDef) => (
                    <button
                      key={typeDef.type}
                      onClick={() => handleSelectType(typeDef)}
                      className="group w-full flex items-center gap-3 rounded-xl border border-border p-3
                        hover:border-foreground/15 hover:bg-muted/40 hover:shadow-sm
                        active:scale-[0.99] transition-all text-left cursor-pointer"
                    >
                      <div className={`w-8 h-8 rounded-lg ${typeDef.accent} ${typeDef.accentText} flex items-center justify-center shrink-0
                        group-hover:scale-110 transition-transform`}>
                        {typeDef.icon}
                      </div>
                      <div className="min-w-0 flex-1">
                        <span className="text-sm font-medium text-foreground">{typeDef.label}</span>
                        <p className="text-[11px] text-muted-foreground leading-tight">{typeDef.description}</p>
                      </div>
                    </button>
                  ))}
                </div>
              </div>

              {/* Signal sources */}
              <div>
                <div className="flex items-center gap-2 mb-2.5">
                  <span className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">Signals</span>
                  <div className="flex-1 h-px bg-border" />
                </div>
                <div className="space-y-1.5">
                  {SIGNAL_SOURCES.map((typeDef) => (
                    <button
                      key={typeDef.type}
                      onClick={() => handleSelectType(typeDef)}
                      className="group w-full flex items-center gap-3 rounded-xl border border-border p-3
                        hover:border-foreground/15 hover:bg-muted/40 hover:shadow-sm
                        active:scale-[0.99] transition-all text-left cursor-pointer"
                    >
                      <div className={`w-8 h-8 rounded-lg ${typeDef.accent} ${typeDef.accentText} flex items-center justify-center shrink-0
                        group-hover:scale-110 transition-transform`}>
                        {typeDef.icon}
                      </div>
                      <div className="min-w-0 flex-1">
                        <span className="text-sm font-medium text-foreground">{typeDef.label}</span>
                        <p className="text-[11px] text-muted-foreground leading-tight">{typeDef.description}</p>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Label */}
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Display name</label>
                <input
                  type="text"
                  value={draft.label ?? ''}
                  onChange={(e) => setDraft({ ...draft, label: e.target.value })}
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
                />
              </div>

              {/* Type-specific fields */}
              {(draft.type === 'rss' || draft.type === 'http') && (
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">URL</label>
                  <input
                    type="url"
                    value={draft.url ?? ''}
                    onChange={(e) => setDraft({ ...draft, url: e.target.value })}
                    placeholder="https://..."
                    className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
                  />
                </div>
              )}

              {draft.type === 'reddit' && (
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">Subreddit</label>
                  <input
                    type="text"
                    value={draft.subreddit ?? ''}
                    onChange={(e) => setDraft({ ...draft, subreddit: e.target.value })}
                    placeholder="MachineLearning"
                    className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
                  />
                </div>
              )}

              {(draft.type === 'google_trends' || draft.type === 'twitter') && (
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">Keywords to monitor</label>
                  <KeywordTagInput
                    keywords={draft.keywords ?? []}
                    onChange={(kws) => setDraft({ ...draft, keywords: kws })}
                    placeholder="AI, LLM, GPT..."
                  />
                </div>
              )}

              {(draft.type === 'reddit' || draft.type === 'hn') && (
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">Min score</label>
                  <input
                    type="number"
                    value={draft.min_score ?? ''}
                    onChange={(e) => setDraft({ ...draft, min_score: Number(e.target.value) })}
                    placeholder="100"
                    className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
                  />
                </div>
              )}

              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Max items</label>
                <input
                  type="number"
                  value={draft.limit ?? 20}
                  onChange={(e) => setDraft({ ...draft, limit: Number(e.target.value) })}
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
                />
              </div>
            </div>
          )}
        </div>

        {step === 'config' && (
          <div className="flex items-center justify-between px-5 py-3 border-t border-border">
            <button
              onClick={() => setStep('select')}
              className="text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            >
              ← Back
            </button>
            <button
              onClick={handleAdd}
              className="px-4 py-2 rounded-xl text-sm font-medium bg-foreground text-background
                hover:opacity-90 transition-opacity cursor-pointer"
            >
              Add source
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
