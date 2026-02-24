import { cn } from '@/shared/lib/utils'
import { Mailbox } from 'lucide-react'
import type { ContentSession } from '@/entities/content-session'

interface InboxSidebarProps {
    sessions: ContentSession[]
    selectedId: string | null
    onSelect: (id: string) => void
}

export function InboxSidebar({ sessions, selectedId, onSelect }: InboxSidebarProps) {
    if (sessions.length === 0) {
        return (
            <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-6 gap-3 text-center">
                <Mailbox className="w-10 h-10 opacity-20" />
                <div>
                    <p className="font-medium text-foreground">All caught up!</p>
                    <p className="text-xs mt-1">No sessions pending review at the moment.</p>
                </div>
            </div>
        )
    }

    return (
        <div className="flex flex-col h-full animate-in fade-in duration-300">
            <div className="p-4 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md flex items-center justify-between shadow-sm z-10">
                <h2 className="font-semibold flex items-center gap-2">
                    Pending Review
                    <span className="flex items-center justify-center bg-primary/20 text-primary text-[10px] h-5 px-2 rounded-full font-bold">
                        {sessions.length}
                    </span>
                </h2>
            </div>

            <div className="flex-1 overflow-y-auto p-3 space-y-2 relative">
                {sessions.map((session) => {
                    const isSelected = selectedId === session.id
                    const score = session.analysis?.score ?? 0
                    const date = new Date(session.created_at)
                    const timeAgo = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })

                    return (
                        <button
                            key={session.id}
                            onClick={() => onSelect(session.id)}
                            className={cn(
                                "w-full text-left p-3.5 rounded-xl border transition-all duration-200 cursor-pointer flex flex-col gap-2.5",
                                isSelected
                                    ? "bg-primary/5 border-primary/40 shadow-sm ring-1 ring-primary/20"
                                    : "bg-card border-border/60 hover:border-primary/40 hover:bg-muted/50"
                            )}
                        >
                            <div className="flex items-start justify-between gap-3">
                                <div className="flex justify-center items-center w-8 h-8 rounded-lg bg-primary/10 text-primary shrink-0 mt-0.5">
                                    <span className="text-xs font-bold leading-none">{score}</span>
                                </div>
                                <div className="flex flex-col min-w-0 flex-1">
                                    <span className="text-[10px] text-muted-foreground uppercase tracking-wider font-bold truncate">
                                        {session.pipeline_name || 'Unknown Pipeline'}
                                    </span>
                                    <span className="font-medium text-sm truncate mt-0.5 text-foreground leading-tight">
                                        Session {session.session_number ? `#${session.session_number}` : session.id.slice(0, 8)}
                                    </span>
                                </div>
                                <span className="text-[10px] text-muted-foreground shrink-0 mt-0.5 font-medium bg-muted/50 px-1.5 py-0.5 rounded">
                                    {timeAgo}
                                </span>
                            </div>

                            <div className="flex items-center gap-2 px-1">
                                <div className="flex-1 h-1 rounded-full bg-muted overflow-hidden">
                                    <div
                                        className={cn("h-full rounded-full transition-all duration-500", score >= 80 ? "bg-success" : score >= 50 ? "bg-warning" : "bg-destructive")}
                                        style={{ width: `${Math.min(100, Math.max(0, score))}%` }}
                                    />
                                </div>
                            </div>
                        </button>
                    )
                })}
            </div>
        </div>
    )
}
