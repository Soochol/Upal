import { useState, useEffect, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Search, Loader2, Archive, ArchiveRestore, Trash2,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { fetchContentSessions, archiveSession, unarchiveSession, deleteSession } from '@/entities/content-session/api'

// ─── Status dot ──────────────────────────────────────────────────────────────

const STATUS_DOT: Record<string, string> = {
  pending_review: 'bg-warning',
  approved: 'bg-success',
  producing: 'bg-info',
  published: 'bg-success/70',
  rejected: 'bg-muted-foreground/40',
  collecting: 'bg-primary',
}

// ─── Filter type ─────────────────────────────────────────────────────────────

type SessionFilter = 'all' | 'pending_review' | 'producing' | 'published' | 'rejected' | 'archived'

const FILTER_TABS: { value: SessionFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'pending_review', label: 'Pending' },
  { value: 'producing', label: 'Producing' },
  { value: 'published', label: 'Published' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'archived', label: 'Archived' },
]

// ─── Component ───────────────────────────────────────────────────────────────

interface SessionListPanelProps {
  pipelineId: string
  selectedSessionId: string | null
  onSelectSession: (id: string) => void
  className?: string
}

export function SessionListPanel({
  pipelineId,
  selectedSessionId,
  onSelectSession,
  className,
}: SessionListPanelProps) {
  const queryClient = useQueryClient()
  const [activeFilter, setActiveFilter] = useState<SessionFilter>('all')
  const [search, setSearch] = useState('')

  // ─── Data fetching ───────────────────────────────────────────────────────

  const { data: sessions = [], isLoading: sessionsLoading } = useQuery({
    queryKey: ['content-sessions', { pipelineId }],
    queryFn: () => fetchContentSessions({ pipelineId }),
    enabled: !!pipelineId,
  })

  const { data: archivedSessions = [], isLoading: archivedLoading } = useQuery({
    queryKey: ['content-sessions', { pipelineId, archived: true }],
    queryFn: () => fetchContentSessions({ pipelineId, archivedOnly: true }),
    enabled: !!pipelineId && activeFilter === 'archived',
  })

  // ─── Mutations ───────────────────────────────────────────────────────────

  const archiveMutation = useMutation({
    mutationFn: (sessionId: string) => archiveSession(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId, archived: true }] })
    },
  })

  const unarchiveMutation = useMutation({
    mutationFn: (sessionId: string) => unarchiveSession(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId, archived: true }] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (sessionId: string) => deleteSession(sessionId),
    onSuccess: () => {
      onSelectSession('')
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId, archived: true }] })
    },
  })

  // ─── Derived data ────────────────────────────────────────────────────────

  const filterCounts = useMemo(() => {
    const counts: Record<SessionFilter, number> = {
      all: sessions.length, pending_review: 0, producing: 0,
      published: 0, rejected: 0, archived: archivedSessions.length,
    }
    for (const s of sessions) {
      if (s.status in counts) counts[s.status as SessionFilter]++
    }
    return counts
  }, [sessions, archivedSessions])

  const filteredSessions = (activeFilter === 'archived' ? archivedSessions : sessions)
    .filter(s => activeFilter === 'all' || activeFilter === 'archived' || s.status === activeFilter)
    .filter(s => {
      if (!search) return true
      const q = search.toLowerCase()
      return `session ${s.session_number}`.includes(q) ||
        s.analysis?.summary?.toLowerCase().includes(q) ||
        s.status.includes(q)
    })

  // Auto-select first session
  useEffect(() => {
    if (!selectedSessionId && filteredSessions.length > 0) {
      const pending = filteredSessions.find(s => s.status === 'pending_review')
      onSelectSession((pending ?? filteredSessions[0]).id)
    }
  }, [filteredSessions, selectedSessionId, onSelectSession])

  // ─── Render ──────────────────────────────────────────────────────────────

  return (
    <div className={cn('flex flex-col', className)}>
      {/* Header: Search + Filters */}
      <div className="p-4 border-b border-border/50 bg-background/50 sticky top-0 z-10">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="search"
            placeholder="Search sessions..."
            className="w-full h-9 pl-9 pr-4 rounded-lg bg-background border border-input text-sm outline-none focus:ring-1 focus:ring-ring transition-shadow"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="flex items-center gap-1 mt-3 overflow-x-auto pb-1 scrollbar-none">
          {FILTER_TABS.map(tab => {
            const count = filterCounts[tab.value]
            const isActive = activeFilter === tab.value
            return (
              <button
                key={tab.value}
                onClick={() => setActiveFilter(tab.value)}
                className={cn(
                  'flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer whitespace-nowrap shrink-0',
                  isActive
                    ? 'bg-foreground text-background'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted/50',
                )}
              >
                {tab.label}
                {count > 0 && (
                  <span className={cn(
                    'text-[10px] font-bold tabular-nums px-1 rounded-full',
                    tab.value === 'pending_review' ? 'bg-warning/20 text-warning' : 'bg-muted-foreground/20',
                  )}>
                    {count}
                  </span>
                )}
              </button>
            )
          })}
        </div>
      </div>

      {/* Session list */}
      <div className="flex-1 overflow-y-auto w-full p-2 space-y-1">
        {(activeFilter === 'archived' ? archivedLoading : sessionsLoading) ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : filteredSessions.length === 0 ? (
          <div className="text-center py-12 px-4">
            <p className="text-sm text-muted-foreground">No sessions found.</p>
          </div>
        ) : (
          filteredSessions.map((s) => {
            const isSelected = selectedSessionId === s.id
            return (
              <button
                key={s.id}
                onClick={() => onSelectSession(s.id)}
                className={cn(
                  'group w-full text-left p-3 rounded-xl transition-all duration-200 cursor-pointer border',
                  isSelected
                    ? 'bg-primary/5 border-primary/20 shadow-sm'
                    : 'bg-transparent border-transparent hover:bg-muted/50',
                )}
              >
                <div className="flex items-start justify-between gap-2 mb-1">
                  <div className="flex items-center gap-1.5 min-w-0">
                    <span className={cn('w-1.5 h-1.5 rounded-full shrink-0', STATUS_DOT[s.status] ?? 'bg-muted')} />
                    <span className={cn('text-sm font-semibold truncate', isSelected ? 'text-primary' : 'text-foreground')}>
                      Session {s.session_number}
                    </span>
                  </div>
                  <span className="text-xs text-muted-foreground/60 whitespace-nowrap shrink-0">
                    {new Date(s.created_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
                  </span>
                </div>
                {s.analysis?.summary ? (
                  <p className={cn('text-sm line-clamp-2', isSelected ? 'text-foreground/80' : 'text-muted-foreground')}>
                    {s.analysis.summary}
                  </p>
                ) : (
                  <p className="text-xs text-muted-foreground/50 italic">Processing...</p>
                )}
                <div className="flex items-center gap-2 mt-2">
                  <StatusBadge status={s.status} />
                  {s.status === 'pending_review' && (
                    <span className="flex h-2 w-2 rounded-full bg-warning animate-pulse ml-auto" title="Needs Review" />
                  )}
                  {activeFilter === 'archived' ? (
                    <div className="flex items-center gap-1 ml-auto">
                      <button
                        type="button"
                        onClick={(e) => { e.stopPropagation(); unarchiveMutation.mutate(s.id) }}
                        className="p-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
                        title="Unarchive"
                      >
                        <ArchiveRestore className="h-3.5 w-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={(e) => { e.stopPropagation(); if (confirm('Permanently delete this session? This cannot be undone.')) deleteMutation.mutate(s.id) }}
                        className="p-1 rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                        title="Delete permanently"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  ) : (
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); archiveMutation.mutate(s.id) }}
                      className="p-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors opacity-0 group-hover:opacity-100 ml-auto cursor-pointer"
                      title="Archive"
                    >
                      <Archive className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              </button>
            )
          })
        )}
      </div>
    </div>
  )
}
