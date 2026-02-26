import { useState, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Search, Loader2, Archive, ArchiveRestore, Trash2, FileText,
  Plus, ArrowLeft, Settings, GitBranch,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { EditableName } from '@/shared/ui/EditableName'
import { fetchContentSessions, archiveSession, unarchiveSession, deleteSession, updateSessionSettings } from '@/entities/content-session/api'
import { SESSION_STATUS_DOT, SESSION_FILTER_TABS, matchesSessionFilter } from '@/entities/content-session/constants'
import type { SessionFilter } from '@/entities/content-session/constants'

// ─── Component ───────────────────────────────────────────────────────────────

interface SessionListPanelProps {
  pipelineId: string
  pipelineName?: string
  isContentPipeline?: boolean
  selectedSessionId: string | null
  onSelectSession: (id: string) => void
  onStartSession?: () => void
  onBack?: () => void
  className?: string
}

export function SessionListPanel({
  pipelineId,
  pipelineName,
  isContentPipeline = true,
  selectedSessionId,
  onSelectSession,
  onStartSession,
  onBack,
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

  const { data: templateSessions = [] } = useQuery({
    queryKey: ['content-sessions', { pipelineId, templateOnly: true }],
    queryFn: () => fetchContentSessions({ pipelineId, templateOnly: true }),
    enabled: !!pipelineId,
  })
  const templateSession = templateSessions[0] ?? null

  // ─── Mutations ───────────────────────────────────────────────────────────

  const invalidateSessions = () => {
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId, archived: true }] })
  }

  const archiveMutation = useMutation({
    mutationFn: (sessionId: string) => archiveSession(sessionId),
    onSuccess: invalidateSessions,
  })

  const unarchiveMutation = useMutation({
    mutationFn: (sessionId: string) => unarchiveSession(sessionId),
    onSuccess: invalidateSessions,
  })

  const deleteMutation = useMutation({
    mutationFn: (sessionId: string) => deleteSession(sessionId),
    onSuccess: () => {
      onSelectSession('')
      invalidateSessions()
    },
  })

  const renameMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => updateSessionSettings(id, { name }),
    onSuccess: invalidateSessions,
  })

  // ─── Derived data ────────────────────────────────────────────────────────

  const filterCounts = useMemo(() => {
    const counts: Record<SessionFilter, number> = {
      all: sessions.length, pending: 0, in_progress: 0, producing: 0,
      published: 0, rejected: 0, archived: archivedSessions.length,
    }
    for (const s of sessions) {
      if (matchesSessionFilter(s.status, 'pending')) counts.pending++
      if (matchesSessionFilter(s.status, 'in_progress')) counts.in_progress++
      if (matchesSessionFilter(s.status, 'producing')) counts.producing++
      if (matchesSessionFilter(s.status, 'published')) counts.published++
      if (matchesSessionFilter(s.status, 'rejected')) counts.rejected++
    }
    return counts
  }, [sessions, archivedSessions])

  const filteredSessions = (activeFilter === 'archived' ? archivedSessions : sessions)
    .filter(s => matchesSessionFilter(s.status, activeFilter))
    .filter(s => {
      if (!search) return true
      const q = search.toLowerCase()
      const displayName = s.name || `Session ${s.session_number}`
      return displayName.toLowerCase().includes(q) ||
        s.analysis?.summary?.toLowerCase().includes(q) ||
        s.status.includes(q)
    })

  // ─── Render ──────────────────────────────────────────────────────────────

  return (
    <div className={cn('flex flex-col', className)}>
      {/* Pipeline header */}
      <div className="px-4 py-3 border-b border-border/50 bg-background/50 backdrop-blur-md shrink-0 flex items-center justify-between gap-2">
        {onBack && (
          <button onClick={onBack} className="md:hidden text-muted-foreground hover:text-foreground transition-colors shrink-0">
            <ArrowLeft className="w-4 h-4" />
          </button>
        )}
        <h2 className="text-sm font-semibold truncate flex-1 min-w-0">
          {pipelineName ?? 'Sessions'}
        </h2>
        {onStartSession && (
          <button
            onClick={onStartSession}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer shrink-0"
          >
            <Plus className="h-3 w-3" />
            <span className="hidden sm:inline">Start</span>
          </button>
        )}
      </div>

      {/* Non-content pipeline: show guidance instead of session list */}
      {!isContentPipeline && sessions.length === 0 && !sessionsLoading && (
        <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-6 gap-3 text-center">
          <GitBranch className="w-10 h-10 opacity-20" />
          <div>
            <p className="font-medium text-foreground">Workflow Pipeline</p>
            <p className="text-xs mt-1">This pipeline runs workflows directly. Add a collect stage to enable content sessions.</p>
          </div>
        </div>
      )}

      {/* Search + Filters + Session list */}
      {(isContentPipeline || sessions.length > 0 || sessionsLoading) && <><div className="p-4 border-b border-border/50 bg-background/50 sticky top-0 z-10">
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
          {SESSION_FILTER_TABS.map(tab => {
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
                    tab.value === 'pending' ? 'bg-warning/20 text-warning' : 'bg-muted-foreground/20',
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
        {/* Template session (pinned "Defaults") */}
        {templateSession && activeFilter !== 'archived' && (
          <>
            <button
              onClick={() => onSelectSession(templateSession.id)}
              className={cn(
                'group w-full text-left p-3 rounded-xl transition-all duration-200 cursor-pointer border',
                selectedSessionId === templateSession.id
                  ? 'bg-primary/5 border-primary/20 shadow-sm'
                  : 'bg-transparent border-transparent hover:bg-muted/50',
              )}
            >
              <div className="flex items-center gap-2">
                <Settings className="h-3.5 w-3.5 text-muted-foreground/60 shrink-0" />
                <span className={cn(
                  'text-sm font-semibold',
                  selectedSessionId === templateSession.id ? 'text-primary' : 'text-foreground',
                )}>
                  Pipeline Defaults
                </span>
              </div>
              <p className="text-xs text-muted-foreground/50 mt-1 ml-6">
                New sessions inherit these settings
              </p>
            </button>
            {filteredSessions.length > 0 && (
              <div className="border-b border-border/30 mx-2 my-1" />
            )}
          </>
        )}

        {/* Regular sessions */}
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
                    <span className={cn('w-1.5 h-1.5 rounded-full shrink-0', SESSION_STATUS_DOT[s.status] ?? 'bg-muted')} />
                    <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground/60" />
                    <EditableName
                      value={s.name || `Session ${s.session_number}`}
                      placeholder={`Session ${s.session_number}`}
                      onSave={(name) => renameMutation.mutate({ id: s.id, name })}
                      className={cn('text-sm font-semibold', isSelected ? 'text-primary' : 'text-foreground')}
                    />
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
      </div></>}
    </div>
  )
}
