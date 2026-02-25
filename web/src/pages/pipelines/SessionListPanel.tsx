import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Search, Plus, Loader2, Trash2, FileText, Pencil,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'

import { fetchContentSessions, deleteSession, updateSessionSettings } from '@/entities/content-session/api'
import { ConfirmDialog } from '@/shared/ui/ConfirmDialog'
import { useUIStore } from '@/entities/ui'
import type { ContentSession } from '@/entities/content-session'
import type { ContentSessionStatus } from '@/shared/types'

const STATUS_DOT_COLOR: Record<ContentSessionStatus, string> = {
  draft:          'bg-muted-foreground/50',
  active:         'bg-success animate-active-pulse',
  collecting:     'bg-info',
  analyzing:      'bg-info',
  pending_review: 'bg-warning',
  approved:       'bg-success',
  rejected:       'bg-destructive',
  producing:      'bg-primary',
  published:      'bg-success',
  error:          'bg-destructive',
}

// ─── Component ───────────────────────────────────────────────────────────────

interface SessionListPanelProps {
  pipelineId: string
  selectedSessionId: string | null
  onSelectSession: (id: string) => void
  onNewSession?: () => void
  onDeleteSession?: (id: string) => void
  className?: string
}

export function SessionListPanel({
  pipelineId,
  selectedSessionId,
  onSelectSession,
  onNewSession,
  onDeleteSession,
  className,
}: SessionListPanelProps) {
  const queryClient = useQueryClient()
  const addToast = useUIStore((s) => s.addToast)
  const [search, setSearch] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<ContentSession | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteSession(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      setDeleteTarget(null)
      onDeleteSession?.(id)
    },
    onError: (err) => {
      addToast(`Failed to delete session: ${err instanceof Error ? err.message : 'unknown error'}`)
    },
  })

  const renameMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => updateSessionSettings(id, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      setEditingId(null)
    },
    onError: (err) => {
      addToast(`Failed to rename: ${err instanceof Error ? err.message : 'unknown error'}`)
    },
  })

  const startRename = (s: ContentSession) => {
    setEditingId(s.id)
    setEditName(s.name || `Session #${s.session_number}`)
  }

  const commitRename = (id: string, originalName: string) => {
    const trimmed = editName.trim()
    setEditingId(null)
    if (!trimmed || trimmed === originalName) return
    renameMutation.mutate({ id, name: trimmed })
  }

  // ─── Data fetching ───────────────────────────────────────────────────────

  const { data: sessions = [], isLoading } = useQuery({
    queryKey: ['content-sessions', { pipelineId }],
    queryFn: () => fetchContentSessions({ pipelineId }),
    enabled: !!pipelineId,
  })

  // ─── Derived data ────────────────────────────────────────────────────────

  const filteredSessions = sessions.filter(s => {
    if (!search) return true
    const q = search.toLowerCase()
    const label = s.name || `Session #${s.session_number}`
    return label.toLowerCase().includes(q) ||
      s.analysis?.summary?.toLowerCase().includes(q)
  })

  // Auto-select first session
  useEffect(() => {
    if (!selectedSessionId && filteredSessions.length > 0) {
      onSelectSession(filteredSessions[0].id)
    }
  }, [filteredSessions, selectedSessionId, onSelectSession])

  // ─── Render ──────────────────────────────────────────────────────────────

  return (
    <div className={cn('flex flex-col', className)}>
      {/* Header: Label + Search + Filters */}
      <div className="p-4 border-b border-border/50 bg-background/50 sticky top-0 z-10 space-y-3">
        {/* Label + New */}
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold">Sessions</h2>
          {onNewSession && (
            <button
              onClick={onNewSession}
              className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity shrink-0 cursor-pointer"
            >
              <Plus className="h-3 w-3" />
              New
            </button>
          )}
        </div>

        {/* Search */}
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
      </div>

      {/* Session list */}
      <div className="flex-1 overflow-y-auto w-full p-3 space-y-1.5">
        {isLoading ? (
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
            const isActive = s.status === 'active'
            return (
              <button
                key={s.id}
                onClick={() => onSelectSession(s.id)}
                className={cn(
                  'group w-full text-left p-3 rounded-xl transition-all duration-200 cursor-pointer border min-h-[84px]',
                  isSelected
                    ? 'bg-primary/5 border-primary/40 shadow-sm ring-1 ring-primary/20'
                    : isActive
                      ? 'bg-card border-success/30 shadow-[0_0_8px_oklch(var(--success)/0.15)] hover:border-success/50'
                      : 'bg-card border-border/60 hover:border-primary/40 hover:bg-muted/50',
                )}
              >
                <div className="flex items-start gap-2.5">
                  {/* Icon with status dot */}
                  <div className="relative shrink-0 mt-0.5">
                    <div className="w-7 h-7 rounded-lg bg-card border border-border/50 flex items-center justify-center">
                      <FileText className="w-3.5 h-3.5 text-muted-foreground" />
                    </div>
                    <span className={cn(
                      'absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-card',
                      STATUS_DOT_COLOR[s.status] ?? 'bg-muted-foreground/50',
                    )} />
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2 mb-0.5">
                      {editingId === s.id ? (
                        <input
                          autoFocus
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          onBlur={() => commitRename(s.id, s.name || `Session #${s.session_number}`)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') commitRename(s.id, s.name || `Session #${s.session_number}`)
                            if (e.key === 'Escape') setEditingId(null)
                          }}
                          onClick={(e) => e.stopPropagation()}
                          className={cn('text-sm font-semibold bg-transparent outline-none border-b border-primary/50 min-w-0 flex-1', isSelected ? 'text-primary' : 'text-foreground')}
                        />
                      ) : (
                        <span
                          className={cn('text-sm font-semibold truncate', isSelected ? 'text-primary' : 'text-foreground')}
                          onDoubleClick={(e) => { e.stopPropagation(); startRename(s) }}
                        >
                          {s.name || `Session #${s.session_number}`}
                        </span>
                      )}
                      <div className="flex items-center gap-1 shrink-0">
                        <button
                          onClick={(e) => { e.stopPropagation(); startRename(s) }}
                          className="p-0.5 rounded-md text-muted-foreground/40 hover:text-foreground hover:bg-muted/50 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer shrink-0"
                          title="Rename session"
                        >
                          <Pencil className="h-2.5 w-2.5" />
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); setDeleteTarget(s) }}
                          className="p-0.5 rounded-md text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer shrink-0"
                          title="Delete session"
                        >
                          <Trash2 className="h-3 w-3" />
                        </button>
                      </div>
                    </div>
                    {s.analysis?.summary && (
                      <p className={cn('text-sm line-clamp-2', isSelected ? 'text-foreground/80' : 'text-muted-foreground')}>
                        {s.analysis.summary}
                      </p>
                    )}
                  </div>
                </div>
              </button>
            )
          })
        )}
      </div>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="Delete session"
        description={`"${deleteTarget?.name || `Session #${deleteTarget?.session_number}`}" will be permanently deleted.`}
        isPending={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </div>
  )
}
