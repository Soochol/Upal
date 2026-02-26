import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { Link } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { fetchContentSession } from '@/entities/content-session/api'
import { fetchPipeline } from '@/entities/pipeline/api'
import { SessionDetailPreview } from '@/pages/pipelines/session/SessionDetailPreview'

interface InboxPreviewProps {
    sessionId: string
}

export function InboxPreview({ sessionId }: InboxPreviewProps) {
    const queryClient = useQueryClient()

    const { data: session, isLoading: sessionLoading } = useQuery({
        queryKey: ['content-session', sessionId],
        queryFn: () => fetchContentSession(sessionId),
        enabled: !!sessionId,
    })

    const pipelineId = session?.pipeline_id

    const { data: pipeline, isLoading: pipelineLoading } = useQuery({
        queryKey: ['pipeline', pipelineId],
        queryFn: () => fetchPipeline(pipelineId!),
        enabled: !!pipelineId,
    })

    const handleMutate = useCallback(() => {
        queryClient.invalidateQueries({ queryKey: ['inbox-sessions'] })
        queryClient.invalidateQueries({ queryKey: ['inbox-sessions-archived'] })
    }, [queryClient])

    if (sessionLoading || pipelineLoading || !session || !pipeline) {
        return (
            <div className="flex-1 flex flex-col items-center justify-center p-6 text-muted-foreground gap-3">
                <Loader2 className="w-6 h-6 animate-spin text-primary/50" />
                <p className="text-sm font-medium">Loading session details...</p>
            </div>
        )
    }

    return (
        <div className="flex-1 flex flex-col h-full animate-in slide-in-from-right-4 duration-300">
            <div className="px-4 md:px-8 py-4 md:py-5 border-b border-border/50 bg-background/80 backdrop-blur-sm shrink-0 shadow-sm z-10 flex flex-col gap-1.5">
                <Link
                    to={`/pipelines?p=${pipeline.id}`}
                    className="text-xs font-bold uppercase tracking-widest text-primary/80 hover:text-primary hover:underline transition-colors w-fit"
                >
                    {pipeline.name}
                </Link>
                <h1 className="text-2xl font-bold tracking-tight text-foreground">
                    {session.name || (session.session_number ? `Session #${session.session_number}` : `Session ${session.id.slice(-6)}`)}
                </h1>
            </div>
            <div className="flex-1 overflow-hidden">
                <SessionDetailPreview
                    pipelineId={pipeline.id}
                    sessionId={sessionId}
                    onMutate={handleMutate}
                />
            </div>
        </div>
    )
}
