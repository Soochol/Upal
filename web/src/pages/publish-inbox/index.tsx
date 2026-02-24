import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { MainLayout } from '@/app/layout'
import { PublishInboxSidebar } from './PublishInboxSidebar'
import { PublishInboxPreview } from './PublishInboxPreview'
import { fetchContentSessions } from '@/entities/content-session/api'
import { useSettingsStore } from '@/entities/settings/store'
import { Loader2 } from 'lucide-react'

export default function PublishInboxPage() {
    const [userSelectedId, setUserSelectedId] = useState<string | null>(null)
    const showArchived = useSettingsStore((s) => s.showArchived)

    const { data: sessions = [], isLoading } = useQuery({
        queryKey: ['publish-inbox-sessions', showArchived],
        queryFn: async () => {
            const [approved, producing, errored] = await Promise.all([
                fetchContentSessions({ status: 'approved', includeArchived: showArchived }),
                fetchContentSessions({ status: 'producing', includeArchived: showArchived }),
                fetchContentSessions({ status: 'error', includeArchived: showArchived }),
            ])
            return [...approved, ...producing, ...errored].sort(
                (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
            )
        },
        staleTime: 2000,
        refetchInterval: 5000,
    })

    // Derive effective selection: use user choice if still valid, else first session
    const selectedSessionId = sessions.length === 0
        ? null
        : (userSelectedId && sessions.find(s => s.id === userSelectedId))
            ? userSelectedId
            : sessions[0].id

    return (
        <MainLayout headerContent={<span className="font-semibold tracking-tight">Publish Inbox</span>}>
            <div className="flex h-full w-full overflow-hidden bg-background">
                {/* Left List */}
                <div className="w-[340px] 2xl:w-[400px] shrink-0 border-r border-border bg-sidebar/30 backdrop-blur-xl z-20 flex flex-col shadow-[4px_0_24px_-12px_rgba(0,0,0,0.5)]">
                    {isLoading ? (
                        <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-4 gap-3">
                            <Loader2 className="w-6 h-6 animate-spin text-primary/50" />
                            <span className="text-sm font-medium">Loading...</span>
                        </div>
                    ) : (
                        <PublishInboxSidebar
                            sessions={sessions}
                            selectedId={selectedSessionId}
                            onSelect={setUserSelectedId}
                        />
                    )}
                </div>

                {/* Right Preview */}
                <div className="flex-1 min-w-0 flex flex-col bg-grid-pattern relative">
                    {selectedSessionId ? (
                        <PublishInboxPreview sessionId={selectedSessionId} />
                    ) : (
                        <div className="flex-1 flex items-center justify-center text-muted-foreground flex-col gap-3">
                            <div className="size-14 rounded-full bg-muted/30 flex items-center justify-center shrink-0 border border-border/50">
                                <span className="text-2xl">📤</span>
                            </div>
                            <div className="text-center">
                                <p className="font-medium text-foreground">No Content to Review</p>
                                <p className="text-sm">Completed workflows will appear here for review.</p>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </MainLayout>
    )
}
