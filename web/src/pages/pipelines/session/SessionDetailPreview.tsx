import { useState, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2, Trash2 } from 'lucide-react'

import { fetchPipeline } from '@/entities/pipeline/api'
import { fetchContentSession, publishSession, produceSession, deleteSession } from '@/entities/content-session/api'
import { sessionPollingInterval } from '@/entities/content-session/constants'
import { useContentSessionStore } from '@/entities/content-session/store'
import type { Pipeline } from '@/entities/pipeline'
import type { ContentSession } from '@/entities/content-session'

import { StickyProgressBar, getActiveStage, isStageCompleted } from './StickyProgressBar'
import type { Stage } from './StickyProgressBar'
import { StageSection } from './StageSection'
import type { StageState } from './StageSection'
import { CollectStage, collectSummary } from './stages/CollectStage'
import { AnalyzeStage } from './stages/AnalyzeStage'
import { ProduceStage, produceSummary } from './stages/ProduceStage'
import { PublishStage } from './stages/PublishStage'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getStageState(stage: Stage, session: ContentSession): StageState {
    const activeStage = getActiveStage(session)

    if (isStageCompleted(stage, activeStage)) return 'completed'
    if (stage === activeStage) return 'active'
    if (session.status === 'published') return 'completed'
    return 'locked'
}

type Props = {
    pipelineId: string
    sessionId: string
    showDelete?: boolean
    onMutate?: () => void
}

export function SessionDetailPreview({ pipelineId, sessionId, showDelete = false, onMutate }: Props) {
    const queryClient = useQueryClient()
    const { approveSession, rejectSession } = useContentSessionStore()

    // ----- Data fetching -----
    const { data: pipeline } = useQuery<Pipeline>({
        queryKey: ['pipeline', pipelineId],
        queryFn: () => fetchPipeline(pipelineId),
        enabled: !!pipelineId,
    })

    // We fetch the session individually to ensure it is up-to-date and we have all details
    const { data: session } = useQuery<ContentSession>({
        queryKey: ['content-session', sessionId],
        queryFn: () => fetchContentSession(sessionId),
        enabled: !!sessionId,
        refetchInterval: (query) => sessionPollingInterval(query.state.data),
    })

    // ----- Local state -----
    const [isApproving, setIsApproving] = useState(false)
    const [isRejecting, setIsRejecting] = useState(false)
    const [isPublishing, setIsPublishing] = useState(false)

    // ----- Actions -----
    const handleApprove = useCallback(
        async (selectedWorkflows: string[]) => {
            if (!sessionId || !pipelineId) return
            setIsApproving(true)
            try {
                // Build workflow→channel mapping from session config
                const channelMap: Record<string, string> = {}
                for (const pw of session?.session_workflows ?? []) {
                    if (pw.channel_id) channelMap[pw.workflow_name] = pw.channel_id
                }
                await approveSession(sessionId, selectedWorkflows, channelMap)
                await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
                await queryClient.invalidateQueries({ queryKey: ['content-sessions'] })
                onMutate?.()
            } finally {
                setIsApproving(false)
            }
        },
        [sessionId, pipelineId, session, approveSession, queryClient, onMutate],
    )

    const handleReject = useCallback(async () => {
        if (!sessionId || !pipelineId) return
        setIsRejecting(true)
        try {
            await rejectSession(sessionId)
            await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
            await queryClient.invalidateQueries({ queryKey: ['content-sessions'] })
            onMutate?.()
        } finally {
            setIsRejecting(false)
        }
    }, [sessionId, pipelineId, rejectSession, queryClient, onMutate])

    const handlePublish = useCallback(
        async (approvedRunIds: string[]) => {
            if (!sessionId || !pipelineId) return
            setIsPublishing(true)
            try {
                await publishSession(sessionId, approvedRunIds)
                await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
                await queryClient.invalidateQueries({ queryKey: ['content-sessions'] })
                onMutate?.()
            } finally {
                setIsPublishing(false)
            }
        },
        [sessionId, pipelineId, queryClient, onMutate],
    )

    const handleRetryProduce = useCallback(async () => {
        if (!session || !pipeline) return
        const angles = session.analysis?.angles?.filter(a => a.selected && a.workflow_name) ?? []
        if (angles.length === 0) return
        const sessionWfs = session.session_workflows ?? []
        const workflows = angles.map(a => {
            const channelId = sessionWfs.find(pw => pw.workflow_name === a.workflow_name)?.channel_id
            return channelId
                ? { name: a.workflow_name!, channel_id: channelId }
                : { name: a.workflow_name! }
        })
        await produceSession(sessionId, workflows)
        await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
    }, [session, pipeline, sessionId, queryClient])

    const deleteMutation = useMutation({
        mutationFn: () => deleteSession(sessionId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
            queryClient.invalidateQueries({ queryKey: ['content-sessions'] })
            queryClient.invalidateQueries({ queryKey: ['inbox-sessions'] })
            onMutate?.()
        },
    })

    // ----- Loading state -----
    if (!session || !pipeline) {
        return (
            <div className="flex items-center justify-center h-full">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
        )
    }

    // ----- Derived data -----
    const itemCount = session.sources?.reduce((sum, src) => sum + src.count, 0) ?? 0

    return (
        <div className="flex flex-col h-full overflow-hidden bg-background">
            {/* Sticky progress stepper + actions */}
            <div className="relative">
                <StickyProgressBar session={session} />
                {showDelete && (
                    <button
                        onClick={() => { if (confirm('이 세션을 영구 삭제합니다. 되돌릴 수 없습니다.')) deleteMutation.mutate() }}
                        disabled={deleteMutation.isPending}
                        className="absolute right-4 top-1/2 -translate-y-1/2 p-1.5 rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                        title="Delete session"
                    >
                        <Trash2 className="h-4 w-4" />
                    </button>
                )}
            </div>

            {/* Scrollable stage content */}
            <div className="flex-1 overflow-y-auto w-full">
                <div className="max-w-4xl mx-auto px-6 py-6 space-y-2">
                    {/* 1. Collect */}
                    <StageSection
                        id="stage-collect"
                        number={1}
                        title="Collect"
                        state={getStageState('collect', session)}
                        summary={collectSummary(session)}
                        statusText={`${itemCount} items`}
                    >
                        <CollectStage session={session} />
                    </StageSection>

                    {/* 2. Analyze */}
                    <StageSection
                        id="stage-analyze"
                        number={2}
                        title="Analyze"
                        state={getStageState('analyze', session)}
                        summary={session.analysis?.summary?.slice(0, 80)}
                        statusText={session.analysis?.score != null ? `Score ${session.analysis.score}` : undefined}
                    >
                        <AnalyzeStage
                            session={session}
                            onApprove={handleApprove}
                            onReject={handleReject}
                            isApproving={isApproving}
                            isRejecting={isRejecting}
                        />
                    </StageSection>

                    {/* 3. Produce */}
                    <StageSection
                        id="stage-produce"
                        number={3}
                        title="Produce"
                        state={getStageState('produce', session)}
                        summary={produceSummary(session)}
                    >
                        <ProduceStage session={session} onRetry={handleRetryProduce} />
                    </StageSection>

                    {/* 4. Publish */}
                    <StageSection
                        id="stage-publish"
                        number={4}
                        title="Publish"
                        state={getStageState('publish', session)}
                    >
                        <PublishStage
                            session={session}
                            onPublish={handlePublish}
                            onRejectAll={handleReject}
                            isPublishing={isPublishing}
                        />
                    </StageSection>
                </div>
            </div>
        </div>
    )
}
