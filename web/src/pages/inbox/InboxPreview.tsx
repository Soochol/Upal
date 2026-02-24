import { useState, useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { fetchContentSession } from '@/entities/content-session/api'
import { fetchPipeline } from '@/entities/pipeline/api'
import { useContentSessionStore } from '@/entities/content-session/store'
import { AnalyzeStage } from '@/pages/pipelines/session/stages/AnalyzeStage'

interface InboxPreviewProps {
    sessionId: string
}

export function InboxPreview({ sessionId }: InboxPreviewProps) {
    const queryClient = useQueryClient()
    const { approveSession, rejectSession } = useContentSessionStore()

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

    const [isApproving, setIsApproving] = useState(false)
    const [isRejecting, setIsRejecting] = useState(false)

    const handleApprove = useCallback(
        async (selectedWorkflows: string[]) => {
            if (!sessionId || !pipelineId) return
            setIsApproving(true)
            try {
                // Build workflow→channel mapping from pipeline config
                const channelMap: Record<string, string> = {}
                for (const pw of pipeline?.workflows ?? []) {
                    if (pw.channel_id) channelMap[pw.workflow_name] = pw.channel_id
                }
                await approveSession(sessionId, selectedWorkflows, channelMap)
                await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
                await queryClient.invalidateQueries({ queryKey: ['inbox-sessions'] }) // Refresh inbox list immediately
            } finally {
                setIsApproving(false)
            }
        },
        [sessionId, pipelineId, pipeline, approveSession, queryClient],
    )

    const handleReject = useCallback(async () => {
        if (!sessionId || !pipelineId) return
        setIsRejecting(true)
        try {
            await rejectSession(sessionId)
            await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
            await queryClient.invalidateQueries({ queryKey: ['inbox-sessions'] }) // Refresh inbox list immediately
        } finally {
            setIsRejecting(false)
        }
    }, [sessionId, pipelineId, rejectSession, queryClient])

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
            <div className="px-8 py-5 border-b border-border/50 bg-background/80 backdrop-blur-sm shrink-0 shadow-sm z-10 flex flex-col gap-1.5">
                <div className="text-xs font-bold uppercase tracking-widest text-primary/80">
                    {pipeline.name}
                </div>
                <h1 className="text-2xl font-bold tracking-tight text-foreground">
                    Session {session.session_number ? `#${session.session_number}` : session.id.slice(0, 8)}
                </h1>
            </div>
            <div className="flex-1 overflow-y-auto px-8 py-8">
                <div className="max-w-[1200px] mx-auto">
                    <AnalyzeStage
                        session={session}
                        onApprove={handleApprove}
                        onReject={handleReject}
                        isApproving={isApproving}
                        isRejecting={isRejecting}
                    />
                </div>
            </div>
        </div>
    )
}
