// Shared session UI constants and utilities

// ─── Status dot color mapping ────────────────────────────────────────────────

export const SESSION_STATUS_DOT: Record<string, string> = {
  pending_review: 'bg-warning',
  approved: 'bg-success',
  producing: 'bg-info',
  published: 'bg-success/70',
  rejected: 'bg-muted-foreground/40',
  collecting: 'bg-primary',
  analyzing: 'bg-primary',
  error: 'bg-destructive',
}

// ─── Session filter ──────────────────────────────────────────────────────────

export type SessionFilter = 'all' | 'pending' | 'in_progress' | 'producing' | 'published' | 'rejected' | 'archived'

export const SESSION_FILTER_TABS: { value: SessionFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'producing', label: 'Producing' },
  { value: 'published', label: 'Published' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'archived', label: 'Archived' },
]

/** Map a session status to the appropriate filter tab */
export function matchesSessionFilter(status: string, filter: SessionFilter): boolean {
  switch (filter) {
    case 'all': return true
    case 'pending': return status === 'pending_review'
    case 'in_progress': return status === 'collecting' || status === 'analyzing'
    case 'producing': return status === 'approved' || status === 'producing' || status === 'error'
    case 'published': return status === 'published'
    case 'rejected': return status === 'rejected'
    case 'archived': return true // handled separately via query
    default: return true
  }
}
