import { useState, type ReactNode } from 'react'
import { X, Rss, Flame, MessageCircle, TrendingUp, Globe, Search } from 'lucide-react'
import { KeywordTagInput } from '@/shared/ui/KeywordTagInput'
import type { SessionSource, SessionSourceType } from '@/entities/session'

type SourceTypeDef = {
  type: SessionSourceType
  source_type: 'static' | 'signal' | 'research'
  label: string
  description: string
  icon: ReactNode
  accent: string          // tailwind color class for icon bg
  accentText: string      // tailwind color class for icon fg
}

export const STATIC_SOURCES: SourceTypeDef[] = [
  { type: 'rss',  source_type: 'static', label: 'RSS Feed',      description: 'RSS/Atom feed URLs',       icon: <Rss className="h-4 w-4" />,   accent: 'bg-warning/12', accentText: 'text-warning' },
  { type: 'hn',   source_type: 'static', label: 'Hacker News',   description: 'Top HN stories',           icon: <Flame className="h-4 w-4" />, accent: 'bg-[oklch(0.75_0.15_50)]/12', accentText: 'text-[oklch(0.65_0.15_50)]' },
  { type: 'http', source_type: 'static', label: 'HTTP Endpoint',  description: 'Custom HTTP API',          icon: <Globe className="h-4 w-4" />, accent: 'bg-muted', accentText: 'text-muted-foreground' },
]

export const SIGNAL_SOURCES: SourceTypeDef[] = [
  { type: 'reddit',        source_type: 'signal', label: 'Reddit',         description: 'Subreddit hot posts',        icon: <MessageCircle className="h-4 w-4" />, accent: 'bg-[oklch(0.7_0.15_25)]/12', accentText: 'text-[oklch(0.6_0.15_25)]' },
  { type: 'google_trends', source_type: 'signal', label: 'Google Trends',  description: 'Trending topics by region',  icon: <TrendingUp className="h-4 w-4" />,    accent: 'bg-info/12', accentText: 'text-info' },
  { type: 'social',        source_type: 'signal', label: 'Social Trends',  description: 'Bluesky & Mastodon trends',  icon: <TrendingUp className="h-4 w-4" />, accent: 'bg-primary/12', accentText: 'text-primary' },
]

export const RESEARCH_SOURCES: SourceTypeDef[] = [
  { type: 'research', source_type: 'research', label: 'Web Research', description: 'LLM-powered topic investigation', icon: <Search className="h-4 w-4" />, accent: 'bg-success/12', accentText: 'text-success' },
]

type Props = {
  editSource?: SessionSource
  onAdd: (source: SessionSource) => void
  onClose: () => void
}

function generateId() {
  return `src-${crypto.randomUUID()}`
}

const ALL_SOURCES = [...STATIC_SOURCES, ...SIGNAL_SOURCES, ...RESEARCH_SOURCES]

type SourceGroupProps = {
  label: string
  sources: SourceTypeDef[]
  onSelect: (typeDef: SourceTypeDef) => void
}

function SourceGroup({ label, sources, onSelect }: SourceGroupProps) {
  return (
    <div>
      <div className="flex items-center gap-2 mb-2.5">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">{label}</span>
        <div className="flex-1 h-px bg-border" />
      </div>
      <div className="space-y-1.5">
        {sources.map((typeDef) => (
          <button
            key={typeDef.type}
            onClick={() => onSelect(typeDef)}
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
  )
}

export function AddSourceModal({ editSource, onAdd, onClose }: Props) {
  const editTypeDef = editSource
    ? ALL_SOURCES.find(s => s.type === editSource.type) ?? null
    : null
  const [step, setStep] = useState<'select' | 'config'>(editSource ? 'config' : 'select')
  const [selectedType, setSelectedType] = useState<SourceTypeDef | null>(editTypeDef)
  const [draft, setDraft] = useState<Partial<SessionSource>>(editSource ?? {})

  const handleSelectType = (typeDef: SourceTypeDef) => {
    setSelectedType(typeDef)
    setDraft({
      id: generateId(),
      type: typeDef.type,
      source_type: typeDef.source_type,
      label: typeDef.label,
      limit: 20,
      keywords: [],
      accounts: [],
    })
    setStep('config')
  }

  const handleAdd = () => {
    if (!draft.id || !draft.type) return
    onAdd(draft as SessionSource)
    onClose()
  }

  const modalTitle = step === 'select'
    ? 'Add Source'
    : `${editSource ? 'Edit' : 'Configure'} ${selectedType?.label}`

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-md mx-4 overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="text-sm font-semibold">
            {modalTitle}
          </h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="p-5">
          {step === 'select' ? (
            <div className="space-y-5">
              <SourceGroup label="Data Sources" sources={STATIC_SOURCES} onSelect={handleSelectType} />
              <SourceGroup label="Signals" sources={SIGNAL_SOURCES} onSelect={handleSelectType} />
              <SourceGroup label="Research" sources={RESEARCH_SOURCES} onSelect={handleSelectType} />
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

              {draft.type === 'social' && (
                <>
                  <div>
                    <label className="block text-xs font-medium text-muted-foreground mb-1.5">Keywords to monitor</label>
                    <KeywordTagInput
                      keywords={draft.keywords ?? []}
                      onChange={(kws) => setDraft({ ...draft, keywords: kws })}
                      placeholder="AI, LLM, GPT..."
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-muted-foreground mb-1.5">Follow accounts</label>
                    <KeywordTagInput
                      keywords={draft.accounts ?? []}
                      onChange={(accts) => setDraft({ ...draft, accounts: accts })}
                      placeholder="alice.bsky.social, user@mastodon.social"
                    />
                    <p className="text-[10px] text-muted-foreground mt-1">
                      Bluesky: handle (e.g. alice.bsky.social) · Mastodon: user@instance
                    </p>
                  </div>
                </>
              )}

              {draft.type === 'google_trends' && (
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">Region</label>
                  <select
                    value={draft.geo ?? 'US'}
                    onChange={(e) => setDraft({ ...draft, geo: e.target.value })}
                    className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
                  >
                    <option value="US">United States</option>
                    <option value="KR">South Korea</option>
                    <option value="JP">Japan</option>
                    <option value="GB">United Kingdom</option>
                    <option value="DE">Germany</option>
                    <option value="FR">France</option>
                    <option value="IN">India</option>
                    <option value="BR">Brazil</option>
                    <option value="CA">Canada</option>
                    <option value="AU">Australia</option>
                  </select>
                </div>
              )}

              {draft.type === 'research' && (
                <>
                  <div>
                    <label className="block text-xs font-medium text-muted-foreground mb-1.5">Topic</label>
                    <input
                      type="text"
                      value={draft.topic ?? ''}
                      onChange={(e) => setDraft({ ...draft, topic: e.target.value })}
                      placeholder="EV battery technology trends 2026"
                      className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-muted-foreground mb-1.5">Depth</label>
                    <select
                      value={draft.depth ?? 'deep'}
                      onChange={(e) => setDraft({ ...draft, depth: e.target.value as 'light' | 'deep' })}
                      className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
                    >
                      <option value="light">Light — single search pass</option>
                      <option value="deep">Deep — iterative sub-question decomposition</option>
                    </select>
                  </div>
                </>
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

              {draft.type !== 'research' && (
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Max items</label>
                <input
                  type="number"
                  value={draft.limit ?? 20}
                  onChange={(e) => setDraft({ ...draft, limit: Number(e.target.value) })}
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
                />
              </div>
              )}
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
              {editSource ? 'Save' : 'Add source'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
