import { useState, useRef, useCallback, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2, CheckCircle, XCircle, CheckSquare, Square, AlertTriangle, Sparkles, ExternalLink } from 'lucide-react'
import { updateSessionAnalysis, generateAngleWorkflow } from '@/entities/content-session/api'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import type { ContentSession, ContentAngle } from '@/entities/content-session'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface AnalyzeStageProps {
  session: ContentSession
  onApprove: (selectedWorkflows: string[]) => void
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
// Match type badge
// ---------------------------------------------------------------------------

function MatchBadge({ matchType }: { matchType?: string }) {
  if (matchType === 'matched') {
    return (
      <span className="inline-flex items-center gap-1 text-xs font-medium px-1.5 py-0.5 rounded bg-success/10 text-success border border-success/20 shrink-0 select-none">
        <CheckCircle className="h-2.5 w-2.5" />
        AI Matched
      </span>
    )
  }
  if (matchType === 'generated') {
    return (
      <span className="inline-flex items-center gap-1 text-xs font-medium px-1.5 py-0.5 rounded bg-info/10 text-info border border-info/20 shrink-0 select-none">
        <Sparkles className="h-2.5 w-2.5" />
        Auto-Generated
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-1 text-xs font-medium px-1.5 py-0.5 rounded bg-warning/10 text-warning border border-warning/20 shrink-0 select-none">
      <AlertTriangle className="h-2.5 w-2.5" />
      No Workflow
    </span>
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
  const queryClient = useQueryClient()

  // ---- Editable state ----
  const [editedSummary, setEditedSummary] = useState(analysis?.summary ?? '')
  const [editedInsights, setEditedInsights] = useState<string[]>(analysis?.insights ?? [])
  const [selectedAngles, setSelectedAngles] = useState<Record<string, boolean>>(() => {
    const map: Record<string, boolean> = {}
    for (const angle of analysis?.angles ?? []) {
      map[angle.id] = angle.selected
    }
    return map
  })

  // Sync with prop changes (e.g. SSE updates)
  useEffect(() => {
    if (analysis) {
      setEditedSummary(analysis.summary)
      setEditedInsights([...analysis.insights])
      const map: Record<string, boolean> = {}
      for (const angle of analysis.angles) {
        map[angle.id] = angle.selected
      }
      setSelectedAngles(map)
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

  const toggleAngle = useCallback((angleId: string) => {
    setSelectedAngles((prev) => ({ ...prev, [angleId]: !prev[angleId] }))
  }, [])

  // ---- Generate workflow for unmatched angle ----
  const [generatingAngles, setGeneratingAngles] = useState<Record<string, boolean>>({})

  const handleGenerateWorkflow = useCallback(async (angleId: string) => {
    setGeneratingAngles((prev) => ({ ...prev, [angleId]: true }))
    try {
      await generateAngleWorkflow(session.id, angleId)
      queryClient.invalidateQueries({ queryKey: ['content-session', session.id] })
    } catch (err) {
      console.error('Failed to generate workflow:', err)
    } finally {
      setGeneratingAngles((prev) => ({ ...prev, [angleId]: false }))
    }
  }, [session.id, queryClient])

  // ---- Approve handler: use LLM-matched workflow_name directly ----
  const handleApprove = useCallback(() => {
    const angles = analysis?.angles ?? []
    const selected = angles.filter((a) => selectedAngles[a.id])
    const workflowNames = selected
      .filter((a) => a.workflow_name)
      .map((a) => a.workflow_name!)
    if (workflowNames.length === 0) return
    onApprove(workflowNames)
  }, [analysis, selectedAngles, onApprove])

  // ---- Early return if no analysis ----
  if (!analysis) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        Analyzing sources...
      </div>
    )
  }

  const angles: ContentAngle[] = analysis.angles ?? []
  const anySelected = angles.some((a) => selectedAngles[a.id] && a.workflow_name)

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
                  {src.items.slice(0, 15).map((item, i) => (
                    <div
                      key={i}
                      className="px-4 py-2.5 border-b border-border/30 last:border-b-0"
                    >
                      <p className="text-sm text-muted-foreground line-clamp-2">{item.title}</p>
                    </div>
                  ))}
                  {src.items.length > 15 && (
                    <div className="px-4 py-2 text-sm font-medium text-primary">
                      +{src.items.length - 15} more
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

      {/* Section 3: Workflow selection */}
      {angles.length > 0 && (
        <div className="space-y-3">
          <h4 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            Recommended Workflows
          </h4>
          <div className="grid grid-cols-1 gap-2">
            {angles.map((angle) => {
              const checked = !!selectedAngles[angle.id]
              const matchType = angle.match_type ?? 'none'
              const hasWorkflow = !!angle.workflow_name

              return (
                <div
                  key={angle.id}
                  className={`flex flex-col gap-2 rounded-xl border px-4 py-3 transition-colors
                    ${checked
                      ? 'border-success/40 bg-success/5'
                      : 'border-border bg-background'}`}
                >
                  <div className="flex items-start gap-3">
                    <button
                      type="button"
                      onClick={() => isPendingReview && toggleAngle(angle.id)}
                      disabled={!isPendingReview}
                      className={`mt-0.5 shrink-0 transition-opacity cursor-pointer ${!isPendingReview ? 'opacity-60 cursor-not-allowed' : 'hover:opacity-80'}`}
                    >
                      {checked
                        ? <CheckSquare className="h-4 w-4 text-success" />
                        : <Square className="h-4 w-4 text-muted-foreground" />}
                    </button>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <div className="text-sm font-medium truncate">{angle.title}</div>
                        <MatchBadge matchType={matchType} />
                      </div>
                      <div className="text-sm text-muted-foreground mt-1">
                        {angle.format}
                        {hasWorkflow && (
                          <><span className="mx-1 opacity-50">&middot;</span><span className="text-foreground/70">{angle.workflow_name}</span></>
                        )}
                      </div>
                      {angle.rationale && (
                        <div className="text-sm text-muted-foreground/70 mt-0.5 italic">{angle.rationale}</div>
                      )}
                    </div>
                    {hasWorkflow && (
                      <a
                        href={`/editor?name=${encodeURIComponent(angle.workflow_name!)}`}
                        onClick={(e) => e.stopPropagation()}
                        className="shrink-0 inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-primary transition-colors mt-0.5"
                      >
                        <ExternalLink className="h-3 w-3" />
                        Workflow
                      </a>
                    )}
                  </div>

                  {matchType === 'none' && isPendingReview && (
                    <div className="pl-7 pt-1 animate-in fade-in">
                      <button
                        onClick={() => handleGenerateWorkflow(angle.id)}
                        disabled={generatingAngles[angle.id]}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-semibold
                          bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer
                          disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {generatingAngles[angle.id] ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <Sparkles className="h-3 w-3" />
                        )}
                        Generate Workflow
                      </button>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Section 4: Inline Approval Actions */}
      {isPendingReview && (
        <div className="flex items-center justify-between gap-4 rounded-xl border border-border bg-card p-4">
          <span className="text-sm text-muted-foreground">
            {anySelected ? `${angles.filter((a) => selectedAngles[a.id] && a.workflow_name).length} workflow${angles.filter((a) => selectedAngles[a.id] && a.workflow_name).length > 1 ? 's' : ''} selected` : 'Select workflows above to proceed'}
          </span>
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
              disabled={isApproving || isRejecting || !anySelected}
              className="inline-flex items-center gap-2 rounded-xl bg-success px-6 py-2.5 text-sm font-medium text-white shadow-lg shadow-success/20 hover:bg-success/90 hover:shadow-xl hover:shadow-success/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            >
              {isApproving ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <CheckCircle className="h-4 w-4" />
              )}
              Approve{anySelected ? ` (${angles.filter((a) => selectedAngles[a.id] && a.workflow_name).length})` : ''}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
