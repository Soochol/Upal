import type { ContentSession } from './types'

// ─── Status dot color mapping ────────────────────────────────────────────────

export const SESSION_STATUS_DOT: Record<string, string> = {
  draft: 'bg-muted-foreground/30',
  active: 'bg-success',
  pending_review: 'bg-warning',
  approved: 'bg-success',
  producing: 'bg-info',
  published: 'bg-success/70',
  rejected: 'bg-muted-foreground/40',
  collecting: 'bg-primary',
  analyzing: 'bg-primary',
  error: 'bg-destructive',
}

// ─── Session display name ────────────────────────────────────────────────────

export function sessionDisplayName(s: ContentSession): string {
  if (s.name) return s.name
  if (s.session_number) return `Session ${s.session_number}`
  return `Session ${s.id.slice(-6)}`
}

// ─── Session filter ──────────────────────────────────────────────────────────

export type SessionFilter = 'all' | 'pending' | 'in_progress' | 'producing' | 'published' | 'rejected'

export const SESSION_FILTER_TABS: { value: SessionFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'producing', label: 'Producing' },
  { value: 'published', label: 'Published' },
  { value: 'rejected', label: 'Rejected' },
]

/** Map a session status to the appropriate filter tab */
export function matchesSessionFilter(status: string, filter: SessionFilter): boolean {
  switch (filter) {
    case 'all': return true
    case 'pending': return status === 'pending_review'
    case 'in_progress': return status === 'draft' || status === 'collecting' || status === 'analyzing'
    case 'producing': return status === 'approved' || status === 'producing' || status === 'error'
    case 'published': return status === 'published'
    case 'rejected': return status === 'rejected'
    default: return true
  }
}

/** Polling interval for session queries: polls while session is in-flight */
export function sessionPollingInterval(session: ContentSession | undefined): number | false {
  if (!session) return false
  const { status } = session
  if (status === 'collecting' || status === 'analyzing' || status === 'producing') return 3000
  if (status === 'approved' && (!session.workflow_results || session.workflow_results.length === 0)) return 3000
  return false
}

/** Compute per-filter counts from session list */
export function computeFilterCounts(
  sessions: ContentSession[],
): Record<SessionFilter, number> {
  const counts: Record<SessionFilter, number> = {
    all: sessions.length,
    pending: 0,
    in_progress: 0,
    producing: 0,
    published: 0,
    rejected: 0,
  }
  for (const s of sessions) {
    if (matchesSessionFilter(s.status, 'pending')) counts.pending++
    if (matchesSessionFilter(s.status, 'in_progress')) counts.in_progress++
    if (matchesSessionFilter(s.status, 'producing')) counts.producing++
    if (matchesSessionFilter(s.status, 'published')) counts.published++
    if (matchesSessionFilter(s.status, 'rejected')) counts.rejected++
  }
  return counts
}
