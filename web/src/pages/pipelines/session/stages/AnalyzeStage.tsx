import { useState, useRef, useCallback, useEffect } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Loader2, CheckCircle, XCircle, RotateCcw } from 'lucide-react'
import { updateSessionAnalysis, retryAnalyze } from '@/entities/content-session/api'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import type { ContentSession } from '@/entities/content-session'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface AnalyzeStageProps {
  session: ContentSession
  onApprove: () => void
  onReject: () => void
  isApproving: boolean
  isRejecting: boolean
}

// ---------------------------------------------------------------------------
// Auto-save status indicator
// ---------------------------------------------------------------------------

type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'

function SaveIndicator({ status }: { status: SaveStatus }) {
  if (status === 'saving') {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
        <Loader2 className="h-3 w-3 animate-spin" />
        Saving...
      </span>
    )
  }
  if (status === 'saved') {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-success">
        <CheckCircle className="h-3 w-3" />
        Saved
      </span>
    )
  }
  if (status === 'error') {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-destructive">
        <XCircle className="h-3 w-3" />
        Save failed
      </span>
    )
  }
  return null
}

// ---------------------------------------------------------------------------
// Score bar
// ---------------------------------------------------------------------------

function ScoreBar({ score, collected, selected }: { score: number; collected: number; selected: number }) {
  const pct = Math.min(100, Math.max(0, score))

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">Analysis Score</span>
          <ScoreIndicator score={score} />
        </div>
        <span className="text-sm text-muted-foreground">
          {selected} selected / {collected} collected
        </span>
      </div>
      <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
        <div
          className="h-full rounded-full bg-success transition-all duration-500"
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function AnalyzeStage({
  session,
  onApprove,
  onReject,
  isApproving,
  isRejecting,
}: AnalyzeStageProps) {
  const analysis = session.analysis
  const sources = session.sources ?? []
  const isPendingReview = session.status === 'pending_review'

  // ---- Editable state ----
  const [editedSummary, setEditedSummary] = useState(analysis?.summary ?? '')
  const [editedInsights, setEditedInsights] = useState<string[]>(analysis?.insights ?? [])

  // Sync with prop changes (e.g. SSE updates)
  useEffect(() => {
    if (analysis) {
      setEditedSummary(analysis.summary)
      setEditedInsights([...(analysis.insights ?? [])])
    }
  }, [analysis])

  // ---- Auto-save with debounce ----
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const savedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const saveMutation = useMutation({
    mutationFn: (data: { summary: string; insights: string[] }) =>
      updateSessionAnalysis(session.id, data),
    onSuccess: () => {
      setSaveStatus('saved')
      savedTimerRef.current = setTimeout(() => setSaveStatus('idle'), 2000)
    },
    onError: () => {
      setSaveStatus('error')
    },
  })

  // Cleanup timers
  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      if (savedTimerRef.current) clearTimeout(savedTimerRef.current)
    }
  }, [])

  const debouncedSave = useCallback(
    (summary: string, insights: string[]) => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      if (savedTimerRef.current) clearTimeout(savedTimerRef.current)
      setSaveStatus('idle')
      debounceRef.current = setTimeout(() => {
        setSaveStatus('saving')
        saveMutation.mutate({ summary, insights })
      }, 1500)
    },
    [saveMutation],
  )

  // ---- Handlers ----
  const handleSummaryInput = useCallback(
    (e: React.FormEvent<HTMLDivElement>) => {
      const text = e.currentTarget.textContent ?? ''
      setEditedSummary(text)
      debouncedSave(text, editedInsights)
    },
    [debouncedSave, editedInsights],
  )

  const handleInsightInput = useCallback(
    (index: number, e: React.FormEvent<HTMLDivElement>) => {
      const text = e.currentTarget.textContent ?? ''
      setEditedInsights((prev) => {
        const next = [...prev]
        next[index] = text
        debouncedSave(editedSummary, next)
        return next
      })
    },
    [debouncedSave, editedSummary],
  )

  // ---- Approve handler ----
  const handleApprove = useCallback(() => {
    onApprove()
  }, [onApprove])

  // ---- Retry analysis for stuck sessions ----
  const [isRetrying, setIsRetrying] = useState(false)
  const handleRetryAnalyze = useCallback(async () => {
    setIsRetrying(true)
    try {
      await retryAnalyze(session.id)
    } catch (err) {
      console.error('Failed to retry analysis:', err)
    } finally {
      setIsRetrying(false)
    }
  }, [session.id])

  // ---- Early return if no analysis ----
  if (!analysis) {
    const isAnalyzing = session.status === 'analyzing'
    return (
      <div className="flex flex-col gap-3">
        {isAnalyzing ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Analyzing sources...
          </div>
        ) : (
          <div className="flex items-center gap-2 text-sm text-destructive">
            <XCircle className="h-3.5 w-3.5" />
            Analysis failed. Please retry.
          </div>
        )}
        <button
          onClick={handleRetryAnalyze}
          disabled={isRetrying}
          className="inline-flex items-center gap-1.5 self-start px-3 py-1.5 rounded-lg text-xs font-medium
            border border-border bg-muted/50 hover:bg-muted transition-colors cursor-pointer
            disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isRetrying ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RotateCcw className="h-3 w-3" />
          )}
          Retry Analysis
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Section 1: Score bar */}
      <ScoreBar
        score={analysis.score}
        collected={analysis.total_collected}
        selected={analysis.total_selected}
      />

      {/* Section 2: Two-column layout */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Left: Raw sources (read-only) */}
        <div className="space-y-3 flex flex-col">
          <h4 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide shrink-0">
            Input Data / Sources
          </h4>
          <div className="rounded-xl border border-border/50 bg-black/5 overflow-hidden h-[300px] md:h-[500px] overflow-y-auto">
            {sources.length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground">No sources.</p>
            ) : (
              sources.map((src) => (
                <div key={src.id} className="border-b border-border/50 last:border-b-0">
                  <div className="px-4 py-2.5 bg-muted/20 text-sm font-semibold text-foreground border-b border-border/50">
                    {src.label} &middot; <span className="text-muted-foreground">{src.count} items</span>
                  </div>
                  {(src.items ?? []).slice(0, 15).map((item, i) => (
                    <div
                      key={i}
                      className="px-4 py-2.5 border-b border-border/30 last:border-b-0"
                    >
                      <p className="text-sm text-muted-foreground line-clamp-2">{item.title}</p>
                    </div>
                  ))}
                  {(src.items ?? []).length > 15 && (
                    <div className="px-4 py-2 text-sm font-medium text-primary">
                      +{(src.items ?? []).length - 15} more
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        {/* Right: AI summary + insights (editable) */}
        <div className="space-y-3 flex flex-col">
          <div className="flex items-center justify-between shrink-0">
            <h4 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
              AI Output / Results
            </h4>
            <SaveIndicator status={saveStatus} />
          </div>

          <div className="flex-1 overflow-y-auto space-y-5 rounded-xl border border-border/50 bg-black/5 p-4 h-[300px] md:h-[500px]">
            {/* Summary */}
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-muted-foreground">AI Summary</label>
              <div
                contentEditable={isPendingReview}
                suppressContentEditableWarning
                onInput={handleSummaryInput}
                className="rounded-lg border border-transparent hover:border-border/50 transition-colors px-2 py-1.5 text-sm min-h-[60px] focus:outline-none focus:ring-1 focus:ring-ring focus:bg-background leading-relaxed"
              >
                {analysis.summary}
              </div>
            </div>

            {/* Insights */}
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-muted-foreground">Key Insights</label>
              <div className="space-y-1.5">
                {editedInsights.map((insight, i) => (
                  <div key={i} className="flex items-start gap-2">
                    <span className="text-xs text-muted-foreground mt-2 shrink-0 w-4 text-right">
                      {i + 1}.
                    </span>
                    <div
                      contentEditable={isPendingReview}
                      suppressContentEditableWarning
                      onInput={(e) => handleInsightInput(i, e)}
                      className="flex-1 rounded-lg border border-transparent hover:border-border/50 transition-colors px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring focus:bg-background leading-relaxed"
                    >
                      {insight}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Section 3: Inline Approval Actions */}
      {isPendingReview && (
        <div className="flex items-center justify-end gap-4 rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onReject}
              disabled={isRejecting || isApproving}
              className="inline-flex items-center gap-2 rounded-xl border border-destructive/20 bg-destructive/10 px-5 py-2.5 text-sm font-medium text-destructive hover:bg-destructive/20 hover:border-destructive/40 transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            >
              {isRejecting ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <XCircle className="h-4 w-4" />
              )}
              Reject
            </button>
            <button
              type="button"
              onClick={handleApprove}
              disabled={isApproving || isRejecting}
              className="inline-flex items-center gap-2 rounded-xl bg-success px-6 py-2.5 text-sm font-medium text-white shadow-lg shadow-success/20 hover:bg-success/90 hover:shadow-xl hover:shadow-success/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            >
              {isApproving ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <CheckCircle className="h-4 w-4" />
              )}
              Approve
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
