import type { Run } from './types'

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
  const { status } = run
  if (status === 'collecting' || status === 'analyzing' || status === 'producing') return 3000
  if (status === 'approved' && (!run.workflow_runs || run.workflow_runs.length === 0)) return 3000
  return false
}

export function computeRunFilterCounts(
  runs: Run[],
): Record<RunFilter, number> {
  const counts: Record<RunFilter, number> = {
    all: runs.length,
    pending: 0,
    in_progress: 0,
    producing: 0,
    published: 0,
    rejected: 0,
  }
  for (const r of runs) {
    if (matchesRunFilter(r.status, 'pending')) counts.pending++
    if (matchesRunFilter(r.status, 'in_progress')) counts.in_progress++
    if (matchesRunFilter(r.status, 'producing')) counts.producing++
    if (matchesRunFilter(r.status, 'published')) counts.published++
    if (matchesRunFilter(r.status, 'rejected')) counts.rejected++
  }
  return counts
}
