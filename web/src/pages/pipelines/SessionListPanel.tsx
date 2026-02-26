import { useMemo, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Search, Loader2, FileText,
  Plus, ArrowLeft, GitBranch,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { EditableName } from '@/shared/ui/EditableName'
import { fetchContentSessions, updateSessionSettings } from '@/entities/content-session/api'
import { sessionDisplayName } from '@/entities/content-session/constants'

// ─── Component ───────────────────────────────────────────────────────────────

interface SessionListPanelProps {
  pipelineId: string
  pipelineName?: string
  isContentPipeline?: boolean
  selectedSessionId: string | null
  onSelectSession: (id: string) => void
  onNewSession?: () => void
  onBack?: () => void
  className?: string
}

export function SessionListPanel({
  pipelineId,
  pipelineName,
  isContentPipeline = true,
  selectedSessionId,
  onSelectSession,
  onNewSession,
  onBack,
  className,
}: SessionListPanelProps) {
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')

  // ─── Data fetching ───────────────────────────────────────────────────────

  const { data: sessions = [], isLoading: sessionsLoading } = useQuery({
    queryKey: ['content-sessions', { pipelineId, templateOnly: true }],
    queryFn: () => fetchContentSessions({ pipelineId, templateOnly: true }),
    enabled: !!pipelineId,
  })


  // ─── Mutations ───────────────────────────────────────────────────────────

  const invalidateSessions = () => {
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId, templateOnly: true }] })
  }

  const renameMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => updateSessionSettings(id, { name }),
    onSuccess: invalidateSessions,
  })

  // ─── Derived data ────────────────────────────────────────────────────────

  const filteredSessions = useMemo(() => {
    if (!search) return sessions
    const q = search.toLowerCase()
    return sessions.filter(s =>
      sessionDisplayName(s).toLowerCase().includes(q) ||
      s.schedule?.toLowerCase().includes(q),
    )
  }, [sessions, search])

  const showSessionList = isContentPipeline || sessions.length > 0 || sessionsLoading

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
        {onNewSession && (
          <button
            onClick={onNewSession}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer shrink-0"
          >
            <Plus className="h-3 w-3" />
            <span className="hidden sm:inline">New</span>
          </button>
        )}
      </div>

      {/* Non-content pipeline: show guidance instead of session list */}
      {!showSessionList && (
        <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-6 gap-3 text-center">
          <GitBranch className="w-10 h-10 opacity-20" />
          <div>
            <p className="font-medium text-foreground">Workflow Pipeline</p>
            <p className="text-xs mt-1">This pipeline runs workflows directly. Add a collect stage to enable content sessions.</p>
          </div>
        </div>
      )}

      {/* Search + Filters */}
      {showSessionList && (
        <div className="p-4 border-b border-border/50 bg-background/50 sticky top-0 z-10">
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
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
        </div>
      )}

      {/* Session list */}
      {showSessionList && (
        <div className="flex-1 overflow-y-auto w-full p-2 space-y-1">
          {/* Regular sessions */}
          {sessionsLoading ? (
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
                      <span className={cn(
                        'w-1.5 h-1.5 rounded-full shrink-0',
                        s.status === 'active' ? 'bg-success' : 'bg-muted-foreground/30',
                      )} />
                      <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground/60" />
                      <EditableName
                        value={sessionDisplayName(s)}
                        placeholder={sessionDisplayName(s)}
                        onSave={(name) => renameMutation.mutate({ id: s.id, name })}
                        className={cn('text-sm font-semibold', isSelected ? 'text-primary' : 'text-foreground')}
                      />
                    </div>
                    <span className="text-xs text-muted-foreground/60 whitespace-nowrap shrink-0">
                      {new Date(s.created_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
                    </span>
                  </div>
                  {s.schedule ? (
                    <p className="text-xs text-muted-foreground/70 truncate">{s.schedule}</p>
                  ) : (
                    <p className="text-xs text-muted-foreground/50 italic">No schedule set</p>
                  )}
                </button>
              )
            })
          )}
        </div>
      )}
    </div>
  )
}
