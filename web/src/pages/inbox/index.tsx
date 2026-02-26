import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Loader2, ArrowLeft } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { InboxSidebar } from './InboxSidebar'
import { InboxPreview } from './InboxPreview'
import { fetchRuns } from '@/entities/session-run'
import type { RunFilter } from '@/entities/session-run/constants'

export default function InboxPage() {
    const [searchParams, setSearchParams] = useSearchParams()
    const selectedRunId = searchParams.get('r')
    const [activeFilter, setActiveFilter] = useState<RunFilter>('all')

    const setSelectedRunId = useCallback(
        (id: string | null) => {
            setSearchParams(id ? { r: id } : {}, { replace: true })
        },
        [setSearchParams],
    )

    const { data: runs = [], isLoading } = useQuery({
        queryKey: ['inbox-runs'],
        queryFn: () => fetchRuns(),
        refetchInterval: 10000,
    })

    useEffect(() => {
        const selectedExists = runs.some(r => r.id === selectedRunId)
        if (runs.length > 0 && !selectedExists) {
            setSelectedRunId(runs[0].id)
        } else if (runs.length === 0 && selectedRunId) {
            setSelectedRunId(null)
        }
    }, [runs, activeFilter, selectedRunId, setSelectedRunId])

    const showMobilePreview = !!selectedRunId

    return (
        <MainLayout headerContent={<span className="font-semibold tracking-tight">Inbox</span>}>
            <div className="flex h-full w-full overflow-hidden bg-background">
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
                            runs={runs}
                            selectedId={selectedRunId}
                            onSelect={setSelectedRunId}
                            activeFilter={activeFilter}
                            onFilterChange={setActiveFilter}
                        />
                    )}
                </div>

                <div className={cn(
                    'flex-1 min-w-0 flex flex-col bg-grid-pattern relative',
                    showMobilePreview ? 'flex' : 'hidden md:flex',
                )}>
                    {selectedRunId && (
                        <button
                            onClick={() => setSelectedRunId(null)}
                            className="md:hidden flex items-center gap-2 px-4 py-3 border-b border-border text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
                        >
                            <ArrowLeft className="w-4 h-4" />
                            Back to list
                        </button>
                    )}
                    {selectedRunId ? (
                        <InboxPreview runId={selectedRunId} />
                    ) : (
                        <div className="flex-1 flex items-center justify-center text-muted-foreground flex-col gap-3">
                            <div className="size-14 rounded-full bg-muted/30 flex items-center justify-center shrink-0 border border-border/50">
                                <span className="text-2xl">&#128229;</span>
                            </div>
                            <div className="text-center">
                                <p className="font-medium text-foreground">Select a Run</p>
                                <p className="text-sm">Choose a run from the list to view details.</p>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </MainLayout>
    )
}
