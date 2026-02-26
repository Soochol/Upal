import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { Link } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { fetchContentSession } from '@/entities/content-session/api'
import { sessionDisplayName } from '@/entities/content-session/constants'
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

    const handleMutate = useCallback(() => {
        queryClient.invalidateQueries({ queryKey: ['inbox-sessions'] })
    }, [queryClient])

    if (sessionLoading || !session) {
        return (
            <div className="flex-1 flex flex-col items-center justify-center p-6 text-muted-foreground gap-3">
                <Loader2 className="w-6 h-6 animate-spin text-primary/50" />
                <p className="text-sm font-medium">Loading session details...</p>
            </div>
        )
    }

    return (
        <div className="flex-1 h-full overflow-y-auto animate-in slide-in-from-right-4 duration-300">
            <div className="px-4 md:px-8 py-4 md:py-5 bg-background/80 backdrop-blur-sm z-10 flex flex-col gap-1.5">
                {session.pipeline_name ? (
                    <Link
                        to={`/pipelines?p=${session.pipeline_id}`}
                        className="text-xs font-bold uppercase tracking-widest text-primary/80 hover:text-primary hover:underline transition-colors w-fit"
                    >
                        {session.pipeline_name}
                    </Link>
                ) : (
                    <span className="text-xs font-bold uppercase tracking-widest text-muted-foreground w-fit">
                        Pipeline
                    </span>
                )}
                <h1 className="text-2xl font-bold tracking-tight text-foreground">
                    {sessionDisplayName(session)}
                </h1>
            </div>
            <SessionDetailPreview
                pipelineId={session.pipeline_id}
                sessionId={sessionId}
                showDelete
                onMutate={handleMutate}
            />
        </div>
    )
}
