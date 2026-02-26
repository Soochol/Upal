import { useState, useMemo } from 'react'
import { Search, Mailbox, FileText } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import {
    RUN_STATUS_DOT, RUN_FILTER_TABS, matchesRunFilter,
    runDisplayName, computeRunFilterCounts,
} from '@/entities/session-run/constants'
import type { RunFilter } from '@/entities/session-run/constants'
import type { Run } from '@/entities/session-run'

interface InboxSidebarProps {
    runs: Run[]
    selectedId: string | null
    onSelect: (id: string) => void
    activeFilter: RunFilter
    onFilterChange: (filter: RunFilter) => void
}

export function InboxSidebar({
    runs,
    selectedId,
    onSelect,
    activeFilter,
    onFilterChange,
}: InboxSidebarProps) {
    const [search, setSearch] = useState('')

    const filterCounts = useMemo(
        () => computeRunFilterCounts(runs),
        [runs],
    )

    const filteredRuns = useMemo(() => {
        return runs
            .filter(r => matchesRunFilter(r.status, activeFilter))
            .filter(r => {
                if (!search) return true
                const q = search.toLowerCase()
                return (
                    runDisplayName(r).toLowerCase().includes(q) ||
                    r.session_name?.toLowerCase().includes(q) ||
                    r.analysis?.summary?.toLowerCase().includes(q) ||
                    r.status.includes(q)
                )
            })
    }, [runs, activeFilter, search])

    if (runs.length === 0) {
        return (
            <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-6 gap-3 text-center">
                <Mailbox className="w-10 h-10 opacity-20" />
                <div>
                    <p className="font-medium text-foreground">No runs yet</p>
                    <p className="text-xs mt-1">Runs from your sessions will appear here.</p>
                </div>
            </div>
        )
    }

    return (
        <div className="flex flex-col h-full animate-in fade-in duration-300">
            <div className="p-4 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md shadow-sm z-10">
                <div className="relative">
                    <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <input
                        type="search"
                        placeholder="Search runs..."
                        className="w-full h-9 pl-9 pr-4 rounded-lg bg-background border border-input text-sm outline-none focus:ring-1 focus:ring-ring transition-shadow"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                    />
                </div>
                <div className="flex items-center gap-1 mt-3 overflow-x-auto pb-1 scrollbar-none">
                    {RUN_FILTER_TABS.map(tab => {
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

            <div className="flex-1 overflow-y-auto w-full p-2 space-y-1">
                {filteredRuns.length === 0 ? (
                    <div className="text-center py-12 px-4">
                        <p className="text-sm text-muted-foreground">No runs found.</p>
                    </div>
                ) : (
                    filteredRuns.map((r) => {
                        const isSelected = selectedId === r.id
                        return (
                            <button
                                key={r.id}
                                onClick={() => onSelect(r.id)}
                                className={cn(
                                    'group w-full text-left p-3 rounded-xl transition-all duration-200 cursor-pointer border',
                                    isSelected
                                        ? 'bg-primary/5 border-primary/20 shadow-sm'
                                        : 'bg-transparent border-transparent hover:bg-muted/50',
                                )}
                            >
                                <div className="flex items-start justify-between gap-2 mb-0.5">
                                    <div className="flex items-center gap-1.5 min-w-0 flex-1">
                                        <span className={cn('w-1.5 h-1.5 rounded-full shrink-0', RUN_STATUS_DOT[r.status] ?? 'bg-muted')} />
                                        <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground/60" />
                                        <span className={cn('text-sm font-semibold truncate', isSelected ? 'text-primary' : 'text-foreground')}>
                                            {runDisplayName(r)}
                                        </span>
                                    </div>
                                    <span className="text-xs text-muted-foreground/60 whitespace-nowrap shrink-0">
                                        {new Date(r.created_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
                                    </span>
                                </div>

                                {r.session_name && (
                                    <p className="text-[10px] text-muted-foreground/60 uppercase tracking-wider font-bold truncate pl-6 mb-1">
                                        {r.session_name}
                                    </p>
                                )}

                                {r.analysis?.summary ? (
                                    <p className={cn('text-sm line-clamp-2 pl-6', isSelected ? 'text-foreground/80' : 'text-muted-foreground')}>
                                        {r.analysis.summary}
                                    </p>
                                ) : (
                                    <p className="text-xs text-muted-foreground/50 italic pl-6">Processing...</p>
                                )}

                                <div className="flex items-center gap-2 mt-2 pl-6">
                                    <StatusBadge status={r.status} />
                                    {r.status === 'pending_review' && (
                                        <span className="flex h-2 w-2 rounded-full bg-warning animate-pulse" title="Needs Review" />
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
