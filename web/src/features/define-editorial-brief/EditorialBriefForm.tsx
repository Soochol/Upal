import { useState } from 'react'
import { Loader2, Check, AlertCircle } from 'lucide-react'
import { Textarea } from '@/shared/ui/textarea'
import { KeywordTagInput } from '@/shared/ui/KeywordTagInput'
import { useAutoSave } from '@/shared/hooks/useAutoSave'
import type { PipelineContext } from '@/entities/pipeline'

const LANGUAGE_OPTIONS = ['Korean', 'English', 'Japanese', 'Chinese']

const DEFAULT_CONTEXT: PipelineContext = {
  purpose: '',
  target_audience: '',
  tone_style: '',
  focus_keywords: [],
  exclude_keywords: [],
  language: 'Korean',
}

type Props = {
  initialContext?: PipelineContext
  onSave: (context: PipelineContext) => Promise<void>
  onBack?: () => void
  submitLabel?: string
  skipLabel?: string
  onSkip?: () => void
  autoSave?: boolean
}

export function EditorialBriefForm({ initialContext, onSave, onBack, submitLabel = 'Save', skipLabel, onSkip, autoSave }: Props) {
  const [draft, setDraft] = useState<PipelineContext>(initialContext ?? DEFAULT_CONTEXT)

  const { saveStatus, saveNow } = useAutoSave({
    data: draft,
    onSave,
    delay: 2000,
    enabled: autoSave ?? false,
    onError: (err) => console.error('Failed to save editorial brief:', err),
  })

  const handleSave = () => { void saveNow() }

  const field = (key: keyof PipelineContext) => ({
    value: (draft[key] as string) ?? '',
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setDraft({ ...draft, [key]: e.target.value }),
  })

  return (
    <div className="space-y-5">
      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">Topic / Purpose</label>
        <Textarea
          rows={3}
          placeholder="Describe the topic and goal of this pipeline…"
          className="resize-none text-sm"
          {...field('purpose')}
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">Target audience</label>
        <input
          type="text"
          placeholder="e.g. Tech professionals, AI enthusiasts"
          className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none
            focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
          {...field('target_audience')}
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">Tone / Style</label>
        <input
          type="text"
          placeholder="e.g. Technical but approachable"
          className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none
            focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
          {...field('tone_style')}
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">
          Focus keywords
          <span className="ml-1 font-normal text-muted-foreground/70">(prioritize items matching these)</span>
        </label>
        <KeywordTagInput
          keywords={draft.focus_keywords ?? []}
          onChange={(kws) => setDraft({ ...draft, focus_keywords: kws })}
          placeholder="LLM, AI models, big tech…"
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">
          Exclude keywords
          <span className="ml-1 font-normal text-muted-foreground/70">(skip items matching these)</span>
        </label>
        <KeywordTagInput
          keywords={draft.exclude_keywords ?? []}
          onChange={(kws) => setDraft({ ...draft, exclude_keywords: kws })}
          placeholder="gaming, sports…"
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">Output language</label>
        <select
          value={draft.language}
          onChange={(e) => setDraft({ ...draft, language: e.target.value })}
          className="rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none
            focus:ring-1 focus:ring-ring"
        >
          {LANGUAGE_OPTIONS.map((lang) => (
            <option key={lang} value={lang}>{lang}</option>
          ))}
        </select>
      </div>

      {autoSave ? (
        <div className="flex justify-end pt-1 h-6">
          {saveStatus === 'saving' && (
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />Saving…
            </span>
          )}
          {saveStatus === 'saved' && (
            <span className="flex items-center gap-1 text-xs text-success">
              <Check className="h-3 w-3" />Saved
            </span>
          )}
          {saveStatus === 'error' && (
            <span className="flex items-center gap-1 text-xs text-destructive">
              <AlertCircle className="h-3 w-3" />Failed to save
            </span>
          )}
        </div>
      ) : (
        <div className="flex items-center justify-between pt-1">
          <div className="flex items-center gap-3">
            {onBack && (
              <button
                type="button"
                onClick={onBack}
                className="px-4 py-2 rounded-xl text-sm font-medium text-muted-foreground
                  hover:text-foreground transition-colors cursor-pointer"
              >
                Back
              </button>
            )}
            {onSkip && (
              <button
                type="button"
                onClick={onSkip}
                className="px-4 py-2 rounded-xl text-sm font-medium text-muted-foreground
                  hover:text-foreground transition-colors cursor-pointer"
              >
                {skipLabel ?? 'Skip'}
              </button>
            )}
          </div>
          <button
            onClick={handleSave}
            disabled={saveStatus === 'saving'}
            className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
              bg-foreground text-background hover:opacity-90 transition-opacity
              disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
          >
            {(() => {
              switch (saveStatus) {
                case 'saving': return <><Loader2 className="h-3.5 w-3.5 animate-spin" />Saving…</>
                case 'saved': return <><Check className="h-3.5 w-3.5" />Saved</>
                case 'error': return <><AlertCircle className="h-3.5 w-3.5" />Failed</>
                default: return submitLabel
              }
            })()}
          </button>
        </div>
      )}
    </div>
  )
}
