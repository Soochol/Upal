import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Search, Plus, Loader2,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'

import { fetchContentSessions } from '@/entities/content-session/api'

// ─── Component ───────────────────────────────────────────────────────────────

interface SessionListPanelProps {
  pipelineId: string
  selectedSessionId: string | null
  onSelectSession: (id: string) => void
  onNewSession?: () => void
  className?: string
}

export function SessionListPanel({
  pipelineId,
  selectedSessionId,
  onSelectSession,
  onNewSession,
  className,
}: SessionListPanelProps) {
  const [search, setSearch] = useState('')

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
      <div className="flex-1 overflow-y-auto w-full p-2 space-y-1">
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
                    <span className={cn('text-sm font-semibold truncate', isSelected ? 'text-primary' : 'text-foreground')}>
                      {s.name || `Session #${s.session_number}`}
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
                  <p className="text-xs text-muted-foreground/50 italic">No description</p>
                )}
              </button>
            )
          })
        )}
      </div>
    </div>
  )
}
