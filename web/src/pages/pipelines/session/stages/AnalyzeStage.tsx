import { useState, useRef, useCallback, useEffect } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Loader2, CheckCircle, XCircle, CheckSquare, Square } from 'lucide-react'
import { updateSessionAnalysis } from '@/entities/content-session/api'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import type { ContentSession, ContentAngle } from '@/entities/content-session'
import type { PipelineWorkflow } from '@/shared/types'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface AnalyzeStageProps {
  session: ContentSession
  pipelineWorkflows: PipelineWorkflow[]
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
        <span className="text-xs text-muted-foreground">
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
  pipelineWorkflows,
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

  // ---- Approve handler: map selected angles to workflow names ----
  const handleApprove = useCallback(() => {
    const angles = analysis?.angles ?? []
    const selected = angles.filter((a) => selectedAngles[a.id])
    const workflowNames = selected.map((angle) => {
      const match = pipelineWorkflows.find(
        (pw) =>
          pw.label?.toLowerCase() === angle.format.toLowerCase() ||
          pw.workflow_name.toLowerCase().includes(angle.format.toLowerCase()),
      )
      return match?.workflow_name ?? angle.format
    })
    onApprove(workflowNames)
  }, [analysis, selectedAngles, pipelineWorkflows, onApprove])

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
  const anySelected = angles.some((a) => selectedAngles[a.id])

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
        <div className="space-y-3">
          <h4 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            Raw Sources
          </h4>
          <div className="rounded-xl border border-border overflow-hidden max-h-80 overflow-y-auto">
            {sources.length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground">No sources.</p>
            ) : (
              sources.map((src) => (
                <div key={src.id} className="border-b border-border last:border-b-0">
                  <div className="px-4 py-2 bg-muted/30 text-xs font-medium text-muted-foreground">
                    {src.label} &middot; {src.count} items
                  </div>
                  {src.items.slice(0, 5).map((item, i) => (
                    <div
                      key={i}
                      className="px-4 py-2 text-sm border-b border-border/50 last:border-b-0 truncate"
                    >
                      {item.title}
                    </div>
                  ))}
                  {src.items.length > 5 && (
                    <div className="px-4 py-1.5 text-xs text-muted-foreground">
                      +{src.items.length - 5} more
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        {/* Right: AI summary + insights (editable) */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
              AI Analysis
            </h4>
            <SaveIndicator status={saveStatus} />
          </div>

          {/* Summary */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Summary</label>
            <div
              contentEditable={isPendingReview}
              suppressContentEditableWarning
              onInput={handleSummaryInput}
              className="rounded-lg border border-border px-3 py-2 text-sm min-h-[60px] focus:outline-none focus:ring-1 focus:ring-ring"
            >
              {analysis.summary}
            </div>
          </div>

          {/* Insights */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Key Insights</label>
            <div className="space-y-1.5">
              {editedInsights.map((insight, i) => (
                <div key={i} className="flex items-start gap-2">
                  <span className="text-xs text-muted-foreground mt-1.5 shrink-0">
                    {i + 1}.
                  </span>
                  <div
                    contentEditable={isPendingReview}
                    suppressContentEditableWarning
                    onInput={(e) => handleInsightInput(i, e)}
                    className="flex-1 rounded-lg border border-border px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
                  >
                    {insight}
                  </div>
                </div>
              ))}
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
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            {angles.map((angle) => {
              const checked = !!selectedAngles[angle.id]
              return (
                <button
                  key={angle.id}
                  type="button"
                  onClick={() => isPendingReview && toggleAngle(angle.id)}
                  disabled={!isPendingReview}
                  className={`flex items-start gap-3 rounded-xl border px-4 py-3 text-left transition-colors cursor-pointer
                    ${checked
                      ? 'border-success/40 bg-success/5 hover:bg-success/10'
                      : 'border-border hover:bg-muted/30'}
                    ${!isPendingReview ? 'opacity-60 cursor-not-allowed' : ''}`}
                >
                  <div className="mt-0.5 shrink-0">
                    {checked
                      ? <CheckSquare className="h-4 w-4 text-success" />
                      : <Square className="h-4 w-4 text-muted-foreground" />}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium truncate">{angle.title}</div>
                    <div className="text-xs text-muted-foreground mt-0.5">{angle.format}</div>
                  </div>
                </button>
              )
            })}
          </div>
        </div>
      )}

      {/* Section 4: Approval actions */}
      {isPendingReview && (
        <div className="flex items-center gap-3 pt-2 border-t border-border">
          <button
            type="button"
            onClick={onReject}
            disabled={isRejecting || isApproving}
            className="inline-flex items-center gap-2 rounded-lg border border-destructive/30 px-4 py-2 text-sm font-medium text-destructive hover:bg-destructive/5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
          >
            {isRejecting ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <XCircle className="h-3.5 w-3.5" />
            )}
            Reject
          </button>
          <button
            type="button"
            onClick={handleApprove}
            disabled={isApproving || isRejecting || !anySelected}
            className="inline-flex items-center gap-2 rounded-lg bg-success px-4 py-2 text-sm font-medium text-white hover:bg-success/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
          >
            {isApproving ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <CheckCircle className="h-3.5 w-3.5" />
            )}
            Approve{anySelected ? ` (${angles.filter((a) => selectedAngles[a.id]).length})` : ''}
          </button>
          {!anySelected && (
            <span className="text-xs text-muted-foreground">
              Select at least one workflow to approve
            </span>
          )}
        </div>
      )}
    </div>
  )
}
