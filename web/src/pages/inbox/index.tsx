import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { MainLayout } from '@/app/layout'
import { InboxSidebar } from './InboxSidebar'
import { InboxPreview } from './InboxPreview'
import { fetchContentSessions } from '@/entities/content-session/api'
import { useSettingsStore } from '@/entities/settings/store'
import { Loader2, ArrowLeft } from 'lucide-react'

export default function ReviewInboxPage() {
    const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null)
    const showArchived = useSettingsStore((s) => s.showArchived)

    const { data: sessions = [], isLoading } = useQuery({
        queryKey: ['inbox-sessions', showArchived],
        queryFn: () => fetchContentSessions({ status: 'pending_review', includeArchived: showArchived }),
        refetchInterval: 10000,
    })

    // Auto-select first item if nothing selected
    useEffect(() => {
        if (!selectedSessionId && sessions.length > 0) {
            setSelectedSessionId(sessions[0].id)
        } else if (sessions.length > 0 && !sessions.find(s => s.id === selectedSessionId)) {
            // If selected session was removed (e.g. approved), select the new first one
            setSelectedSessionId(sessions[0].id)
        } else if (sessions.length === 0 && selectedSessionId) {
            setSelectedSessionId(null)
        }
    }, [sessions, selectedSessionId])

    // Mobile: show preview or list
    const showMobilePreview = !!selectedSessionId

    return (
        <MainLayout headerContent={<span className="font-semibold tracking-tight">Review Inbox</span>}>
            <div className="flex h-full w-full overflow-hidden bg-background">
                {/* Left List — full-width on mobile, fixed-width on desktop */}
                <div className={`w-full md:w-[340px] 2xl:w-[400px] shrink-0 md:border-r border-border bg-sidebar/30 backdrop-blur-xl z-20 flex flex-col md:shadow-[4px_0_24px_-12px_rgba(0,0,0,0.5)] ${showMobilePreview ? 'hidden md:flex' : 'flex'}`}>
                    {isLoading ? (
                        <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-4 gap-3">
                            <Loader2 className="w-6 h-6 animate-spin text-primary/50" />
                            <span className="text-sm font-medium">Loading inbox...</span>
                        </div>
                    ) : (
                        <InboxSidebar
                            sessions={sessions}
                            selectedId={selectedSessionId}
                            onSelect={setSelectedSessionId}
                        />
                    )}
                </div>

                {/* Right Preview — full-width on mobile, flex on desktop */}
                <div className={`flex-1 min-w-0 flex flex-col bg-grid-pattern relative ${showMobilePreview ? 'flex' : 'hidden md:flex'}`}>
                    {/* Mobile back button */}
                    {selectedSessionId && (
                        <button
                            onClick={() => setSelectedSessionId(null)}
                            className="md:hidden flex items-center gap-2 px-4 py-3 border-b border-border text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
                        >
                            <ArrowLeft className="w-4 h-4" />
                            Back to list
                        </button>
                    )}
                    {selectedSessionId ? (
                        <InboxPreview
                            sessionId={selectedSessionId}
                        />
                    ) : (
                        <div className="flex-1 flex items-center justify-center text-muted-foreground flex-col gap-3">
                            <div className="size-14 rounded-full bg-muted/30 flex items-center justify-center shrink-0 border border-border/50">
                                <span className="text-2xl">📥</span>
                            </div>
                            <div className="text-center">
                                <p className="font-medium text-foreground">Inbox Empty</p>
                                <p className="text-sm">Select a session to begin reviewing.</p>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </MainLayout>
    )
}
