import type { ContentSession } from '@/entities/content-session/types'

type Props = {
    sessions: ContentSession[]
    selectedId: string | null
    onSelect: (id: string) => void
}

export function PublishInboxSidebar({ sessions, selectedId, onSelect }: Props) {
    if (sessions.length === 0) {
        return (
            <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-6 gap-3">
                <div className="size-12 rounded-full bg-success/10 flex items-center justify-center">
                    <span className="text-xl">✅</span>
                </div>
                <div className="text-center">
                    <p className="text-sm font-medium text-foreground">All caught up!</p>
                    <p className="text-xs mt-1">No workflows awaiting publish review.</p>
                </div>
            </div>
        )
    }

    return (
        <div className="flex flex-col h-full">
            <div className="px-4 py-3 border-b border-border/50 flex items-center justify-between">
                <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Ready for Review</span>
                <span className="text-xs font-bold bg-primary/10 text-primary px-2.5 py-0.5 rounded-full">
                    {sessions.length}
                </span>
            </div>
            <div className="flex-1 overflow-y-auto">
                {sessions.map((s) => {
                    const results = s.workflow_results ?? []
                    const successCount = results.filter(r => r.status === 'success').length
                    const totalCount = results.length
                    const isSelected = s.id === selectedId

                    return (
                        <button
                            key={s.id}
                            onClick={() => onSelect(s.id)}
                            className={`w-full text-left px-4 py-3.5 border-b border-border/30 transition-all cursor-pointer
                                ${isSelected
                                    ? 'bg-primary/5 border-l-2 border-l-primary/40'
                                    : 'hover:bg-muted/30 border-l-2 border-l-transparent'
                                }`}
                        >
                            <div className="flex items-center justify-between mb-1">
                                <span className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
                                    {s.pipeline_name || 'Pipeline'}
                                </span>
                                <span className="text-[10px] text-muted-foreground">
                                    {s.updated_at ? new Date(s.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : ''}
                                </span>
                            </div>
                            <p className="text-sm font-medium text-foreground truncate">
                                Session #{s.session_number || s.id.slice(0, 8)}
                            </p>
                            <div className="flex items-center gap-2 mt-1.5">
                                <div className="flex-1 h-1.5 rounded-full bg-muted/50 overflow-hidden">
                                    <div
                                        className="h-full rounded-full bg-success transition-all duration-500"
                                        style={{ width: totalCount > 0 ? `${(successCount / totalCount) * 100}%` : '0%' }}
                                    />
                                </div>
                                <span className="text-[10px] font-medium text-muted-foreground">
                                    {successCount}/{totalCount}
                                </span>
                            </div>
                        </button>
                    )
                })}
            </div>
        </div>
    )
}
