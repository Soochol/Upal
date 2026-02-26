import { useState, useMemo } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
    Search, Mailbox, Loader2, Archive, ArchiveRestore, Trash2, FileText,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { EditableName } from '@/shared/ui/EditableName'
import {
    archiveSession, unarchiveSession, deleteSession, updateSessionSettings,
} from '@/entities/content-session/api'
import {
    SESSION_STATUS_DOT, SESSION_FILTER_TABS, matchesSessionFilter,
    sessionDisplayName, computeFilterCounts,
} from '@/entities/content-session/constants'
import type { SessionFilter } from '@/entities/content-session/constants'
import type { ContentSession } from '@/entities/content-session'

// ─── Component ──────────────────────────────────────────────────────────────

interface InboxSidebarProps {
    sessions: ContentSession[]
    archivedSessions: ContentSession[]
    archivedLoading: boolean
    selectedId: string | null
    onSelect: (id: string) => void
    activeFilter: SessionFilter
    onFilterChange: (filter: SessionFilter) => void
}

export function InboxSidebar({
    sessions,
    archivedSessions,
    archivedLoading,
    selectedId,
    onSelect,
    activeFilter,
    onFilterChange,
}: InboxSidebarProps) {
    const queryClient = useQueryClient()
    const [search, setSearch] = useState('')

    // ─── Mutations ───────────────────────────────────────────────────────────

    const invalidateInbox = () => {
        queryClient.invalidateQueries({ queryKey: ['inbox-sessions'] })
        queryClient.invalidateQueries({ queryKey: ['inbox-sessions-archived'] })
    }

    const archiveMutation = useMutation({
        mutationFn: (sessionId: string) => archiveSession(sessionId),
        onSuccess: invalidateInbox,
    })

    const unarchiveMutation = useMutation({
        mutationFn: (sessionId: string) => unarchiveSession(sessionId),
        onSuccess: invalidateInbox,
    })

    const deleteMutation = useMutation({
        mutationFn: (sessionId: string) => deleteSession(sessionId),
        onSuccess: invalidateInbox,
    })

    const renameMutation = useMutation({
        mutationFn: ({ id, name }: { id: string; name: string }) => updateSessionSettings(id, { name }),
        onSuccess: invalidateInbox,
    })

    // ─── Derived data ────────────────────────────────────────────────────────

    const filterCounts = useMemo(
        () => computeFilterCounts(sessions, archivedSessions),
        [sessions, archivedSessions],
    )

    const filteredSessions = useMemo(() => {
        const pool = activeFilter === 'archived' ? archivedSessions : sessions
        return pool
            .filter(s => matchesSessionFilter(s.status, activeFilter))
            .filter(s => {
                if (!search) return true
                const q = search.toLowerCase()
                return (
                    sessionDisplayName(s).toLowerCase().includes(q) ||
                    s.pipeline_name?.toLowerCase().includes(q) ||
                    s.analysis?.summary?.toLowerCase().includes(q) ||
                    s.status.includes(q)
                )
            })
    }, [sessions, archivedSessions, activeFilter, search])

    // ─── Empty state ─────────────────────────────────────────────────────────

    if (sessions.length === 0 && activeFilter !== 'archived') {
        return (
            <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-6 gap-3 text-center">
                <Mailbox className="w-10 h-10 opacity-20" />
                <div>
                    <p className="font-medium text-foreground">No sessions yet</p>
                    <p className="text-xs mt-1">Sessions from your pipelines will appear here.</p>
                </div>
            </div>
        )
    }

    // ─── Render ──────────────────────────────────────────────────────────────

    return (
        <div className="flex flex-col h-full animate-in fade-in duration-300">
            {/* Header: Search + Filter tabs */}
            <div className="p-4 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md shadow-sm z-10">
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
                                onClick={() => onFilterChange(tab.value)}
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
                {(activeFilter === 'archived' && archivedLoading) ? (
                    <div className="flex items-center justify-center py-12">
                        <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
                    </div>
                ) : filteredSessions.length === 0 ? (
                    <div className="text-center py-12 px-4">
                        <p className="text-sm text-muted-foreground">No sessions found.</p>
                    </div>
                ) : (
                    filteredSessions.map((s) => {
                        const isSelected = selectedId === s.id
                        return (
                            <button
                                key={s.id}
                                onClick={() => onSelect(s.id)}
                                className={cn(
                                    'group w-full text-left p-3 rounded-xl transition-all duration-200 cursor-pointer border',
                                    isSelected
                                        ? 'bg-primary/5 border-primary/20 shadow-sm'
                                        : 'bg-transparent border-transparent hover:bg-muted/50',
                                )}
                            >
                                <div className="flex items-start justify-between gap-2 mb-0.5">
                                    <div className="flex items-center gap-1.5 min-w-0 flex-1">
                                        <span className={cn('w-1.5 h-1.5 rounded-full shrink-0', SESSION_STATUS_DOT[s.status] ?? 'bg-muted')} />
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

                                {/* Pipeline name */}
                                {s.pipeline_name && (
                                    <p className="text-[10px] text-muted-foreground/60 uppercase tracking-wider font-bold truncate pl-6 mb-1">
                                        {s.pipeline_name}
                                    </p>
                                )}

                                {/* Analysis summary */}
                                {s.analysis?.summary ? (
                                    <p className={cn('text-sm line-clamp-2 pl-6', isSelected ? 'text-foreground/80' : 'text-muted-foreground')}>
                                        {s.analysis.summary}
                                    </p>
                                ) : (
                                    <p className="text-xs text-muted-foreground/50 italic pl-6">Processing...</p>
                                )}

                                {/* Status + actions */}
                                <div className="flex items-center gap-2 mt-2 pl-6">
                                    <StatusBadge status={s.status} />
                                    {s.status === 'pending_review' && (
                                        <span className="flex h-2 w-2 rounded-full bg-warning animate-pulse" title="Needs Review" />
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
