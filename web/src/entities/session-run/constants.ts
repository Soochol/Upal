import type { Run } from './types'
import type { SessionContext } from '@/entities/session/types'

// --- Defaults ----------------------------------------------------------------

export const DEFAULT_RUN_CONTEXT: SessionContext = { research_depth: 'deep' }

// --- Status dot color mapping ------------------------------------------------

export const RUN_STATUS_DOT: Record<string, string> = {
  collecting: 'bg-primary',
  analyzing: 'bg-primary',
  pending_review: 'bg-warning',
  approved: 'bg-success',
  producing: 'bg-info',
  published: 'bg-success/70',
  rejected: 'bg-muted-foreground/40',
  error: 'bg-destructive',
}

// --- Run display name --------------------------------------------------------

export function runDisplayName(r: Run): string {
  if (r.run_number) return `Run ${r.run_number}`
  return `Run ${r.id.slice(-6)}`
}

// --- Run filter --------------------------------------------------------------

export type RunFilter = 'all' | 'pending' | 'in_progress' | 'producing' | 'published' | 'rejected'

export const RUN_FILTER_TABS: { value: RunFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'producing', label: 'Producing' },
  { value: 'published', label: 'Published' },
  { value: 'rejected', label: 'Rejected' },
]

export function matchesRunFilter(status: string, filter: RunFilter): boolean {
  switch (filter) {
    case 'all': return true
    case 'pending': return status === 'pending_review'
    case 'in_progress': return status === 'collecting' || status === 'analyzing'
    case 'producing': return status === 'approved' || status === 'producing' || status === 'error'
    case 'published': return status === 'published'
    case 'rejected': return status === 'rejected'
    default: return true
  }
}

export function runPollingInterval(run: Run | undefined): number | false {
  if (!run) return false
  switch (run.status) {
    case 'collecting':
    case 'analyzing':
    case 'producing':
      return 3000
    case 'approved':
      return (!run.workflow_runs || run.workflow_runs.length === 0) ? 3000 : false
    default:
      return false
  }
}

export function computeRunFilterCounts(runs: Run[]): Record<RunFilter, number> {
  const counts: Record<RunFilter, number> = {
    all: runs.length,
    pending: 0,
    in_progress: 0,
    producing: 0,
    published: 0,
    rejected: 0,
  }
  for (const r of runs) {
    switch (r.status) {
      case 'pending_review': counts.pending++; break
      case 'collecting':
      case 'analyzing': counts.in_progress++; break
      case 'approved':
      case 'producing':
      case 'error': counts.producing++; break
      case 'published': counts.published++; break
      case 'rejected': counts.rejected++; break
    }
  }
  return counts
}
