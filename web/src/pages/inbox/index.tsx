import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Loader2, ArrowLeft } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { InboxSidebar } from './InboxSidebar'
import { InboxPreview } from './InboxPreview'
import { fetchContentSessions } from '@/entities/content-session/api'
import type { SessionFilter } from '@/entities/content-session/constants'

export default function InboxPage() {
    const [searchParams, setSearchParams] = useSearchParams()
    const selectedSessionId = searchParams.get('s')
    const [activeFilter, setActiveFilter] = useState<SessionFilter>('all')

    const setSelectedSessionId = useCallback(
        (id: string | null) => {
            setSearchParams(id ? { s: id } : {}, { replace: true })
        },
        [setSearchParams],
    )

    const { data: sessions = [], isLoading } = useQuery({
        queryKey: ['inbox-sessions'],
        queryFn: () => fetchContentSessions({ detail: true }),
        refetchInterval: 10000,
    })

    // Auto-select first item if nothing selected or selected was removed
    useEffect(() => {
        const selectedExists = sessions.some(s => s.id === selectedSessionId)
        if (sessions.length > 0 && !selectedExists) {
            setSelectedSessionId(sessions[0].id)
        } else if (sessions.length === 0 && selectedSessionId) {
            setSelectedSessionId(null)
        }
    }, [sessions, activeFilter, selectedSessionId, setSelectedSessionId])

    // Mobile: show preview or list
    const showMobilePreview = !!selectedSessionId

    return (
        <MainLayout headerContent={<span className="font-semibold tracking-tight">Inbox</span>}>
            <div className="flex h-full w-full overflow-hidden bg-background">
                {/* Left List -- full-width on mobile, fixed-width on desktop */}
                <div className={cn(
                    'w-full md:w-[340px] 2xl:w-[400px] shrink-0 md:border-r border-border',
                    'bg-sidebar/30 backdrop-blur-xl z-20 flex flex-col md:shadow-[4px_0_24px_-12px_rgba(0,0,0,0.5)]',
                    showMobilePreview ? 'hidden md:flex' : 'flex',
                )}>
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
                            activeFilter={activeFilter}
                            onFilterChange={setActiveFilter}
                        />
                    )}
                </div>

                {/* Right Preview -- full-width on mobile, flex on desktop */}
                <div className={cn(
                    'flex-1 min-w-0 flex flex-col bg-grid-pattern relative',
                    showMobilePreview ? 'flex' : 'hidden md:flex',
                )}>
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
                        <InboxPreview sessionId={selectedSessionId} />
                    ) : (
                        <div className="flex-1 flex items-center justify-center text-muted-foreground flex-col gap-3">
                            <div className="size-14 rounded-full bg-muted/30 flex items-center justify-center shrink-0 border border-border/50">
                                <span className="text-2xl">📥</span>
                            </div>
                            <div className="text-center">
                                <p className="font-medium text-foreground">Select a Session</p>
                                <p className="text-sm">Choose a session from the list to view details.</p>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </MainLayout>
    )
}
