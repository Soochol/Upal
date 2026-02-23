import { useState } from 'react'
import { X } from 'lucide-react'
import { KeywordTagInput } from '@/shared/ui/KeywordTagInput'
import type { PipelineSource, PipelineSourceType } from '@/shared/types'

type SourceTypeDef = {
  type: PipelineSourceType
  source_type: 'static' | 'signal'
  label: string
  description: string
}

const SOURCE_TYPES: SourceTypeDef[] = [
  { type: 'rss',           source_type: 'static', label: 'RSS Feed',         description: 'Collect from RSS/Atom feed URLs' },
  { type: 'hn',            source_type: 'static', label: 'Hacker News',      description: 'Collect top HN stories' },
  { type: 'reddit',        source_type: 'signal', label: 'Reddit',           description: 'Signal from subreddit hot posts' },
  { type: 'google_trends', source_type: 'signal', label: 'Google Trends',    description: 'Signal from keyword search spikes' },
  { type: 'twitter',       source_type: 'signal', label: 'X / Twitter',      description: 'Signal from trending keywords' },
  { type: 'http',          source_type: 'static', label: 'HTTP Endpoint',    description: 'Collect from custom HTTP API' },
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
            {step === 'select' ? 'Select source type' : `Configure ${selectedType?.label}`}
          </h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="p-5">
          {step === 'select' ? (
            <div className="grid grid-cols-2 gap-2">
              {SOURCE_TYPES.map((typeDef) => (
                <button
                  key={typeDef.type}
                  onClick={() => handleSelectType(typeDef)}
                  className="flex flex-col items-start gap-1 rounded-xl border border-border p-3
                    hover:border-foreground/20 hover:bg-muted/30 transition-all text-left cursor-pointer"
                >
                  <div className="flex items-center gap-1.5">
                    <span className="text-sm font-medium">{typeDef.label}</span>
                    <span className={`text-[10px] px-1.5 py-0.5 rounded border font-medium
                      ${typeDef.source_type === 'static'
                        ? 'border-border text-muted-foreground bg-muted/30'
                        : 'border-primary/20 text-primary bg-primary/10'}`}>
                      {typeDef.source_type === 'static' ? 'static' : 'signal'}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground">{typeDef.description}</p>
                </button>
              ))}
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
