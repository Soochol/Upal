import { cn } from '@/shared/lib/utils'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import type { ContentSession } from '@/entities/content-session'

// ---------------------------------------------------------------------------
// Types & constants
// ---------------------------------------------------------------------------

export type Stage = 'collect' | 'analyze' | 'produce' | 'publish'

const STAGES: { value: Stage; label: string; sectionId: string }[] = [
  { value: 'collect', label: 'Collect', sectionId: 'stage-collect' },
  { value: 'analyze', label: 'Analyze', sectionId: 'stage-analyze' },
  { value: 'produce', label: 'Produce', sectionId: 'stage-produce' },
  { value: 'publish', label: 'Publish', sectionId: 'stage-publish' },
]

// ---------------------------------------------------------------------------
// Helpers (exported for reuse)
// ---------------------------------------------------------------------------

export function getActiveStage(session: ContentSession): Stage {
  switch (session.status) {
    case 'collecting':
      return 'collect'
    case 'analyzing':
    case 'pending_review':
      return 'analyze'
    case 'approved':
    case 'producing':
      return 'produce'
    case 'published':
      return 'publish'
    case 'rejected':
      return 'analyze'
    default:
      return 'collect'
  }
}

export function isStageCompleted(stage: Stage, activeStage: Stage): boolean {
  const order: Stage[] = ['collect', 'analyze', 'produce', 'publish']
  return order.indexOf(stage) < order.indexOf(activeStage)
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface StickyProgressBarProps {
  session: ContentSession
  className?: string
}

function scrollToSection(sectionId: string) {
  const el = document.getElementById(sectionId)
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}

function triggerLabel(type: ContentSession['trigger_type']): string {
  switch (type) {
    case 'schedule':
      return 'Scheduled'
    case 'manual':
      return 'Manual'
    case 'surge':
      return 'Surge'
    default:
      return type
  }
}

export function StickyProgressBar({ session, className }: StickyProgressBarProps) {
  const activeStage = getActiveStage(session)

  return (
    <div
      className={cn(
        'sticky top-0 z-30 bg-background/80 backdrop-blur-md px-6 py-3',
        className,
      )}
    >
      {/* Stepper row */}
      <div className="flex items-center justify-center gap-0">
        {STAGES.map((stage, idx) => {
          const completed = isStageCompleted(stage.value, activeStage)
          const active = stage.value === activeStage
          const future = !completed && !active

          return (
            <div key={stage.value} className="flex items-center">
              {/* Connecting line before (except first) */}
              {idx > 0 && (
                <div
                  className={cn(
                    'h-px w-8 shrink-0',
                    completed || active ? 'bg-success' : 'bg-border',
                  )}
                />
              )}

              {/* Stage pill */}
              <button
                onClick={() => scrollToSection(stage.sectionId)}
                className={cn(
                  'flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors cursor-pointer',
                  active && 'bg-foreground text-background',
                  completed && 'bg-success/10 text-success hover:bg-success/20',
                  future && 'bg-muted text-muted-foreground hover:bg-muted/80',
                )}
              >
                {/* Dot indicator */}
                <span
                  className={cn(
                    'inline-block h-1.5 w-1.5 rounded-full shrink-0',
                    active && 'bg-background',
                    completed && 'bg-success',
                    future && 'bg-muted-foreground/50',
                  )}
                />
                {stage.label}
              </button>
            </div>
          )
        })}
      </div>

      {/* Meta line */}
      <div className="mt-2 flex items-center justify-center gap-2 text-xs text-muted-foreground">
        <span className="font-medium text-foreground">
          {session.name || `Session #${session.session_number ?? '—'}`}
        </span>
        <span className="text-border">|</span>
        <StatusBadge status={session.status} />
        {session.analysis?.score != null && (
          <>
            <span className="text-border">|</span>
            <ScoreIndicator score={session.analysis.score} />
          </>
        )}
        <span className="text-border">|</span>
        <span>{formatDate(session.created_at)}</span>
        <span className="text-border">|</span>
        <span>{triggerLabel(session.trigger_type)}</span>
      </div>
    </div>
  )
}
