import { useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'

import { MainLayout } from '@/app/layout'
import { fetchPipeline } from '@/entities/pipeline/api'
import { fetchContentSession } from '@/entities/content-session/api'
import { useContentSessionStore } from '@/entities/content-session/store'
import type { Pipeline } from '@/shared/types'
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

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function SessionDetailPage() {
  const { id: pipelineId, sessionId } = useParams<{ id: string; sessionId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { approveSession, rejectSession } = useContentSessionStore()

  // ----- Data fetching -----
  const { data: pipeline } = useQuery<Pipeline>({
    queryKey: ['pipeline', pipelineId],
    queryFn: () => fetchPipeline(pipelineId!),
    enabled: !!pipelineId,
  })

  const { data: session } = useQuery<ContentSession>({
    queryKey: ['content-session', sessionId],
    queryFn: () => fetchContentSession(sessionId!),
    enabled: !!sessionId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'collecting' || status === 'producing' ? 3000 : false
    },
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
        await approveSession(sessionId, selectedWorkflows)
        await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
        await queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      } finally {
        setIsApproving(false)
      }
    },
    [sessionId, pipelineId, approveSession, queryClient],
  )

  const handleReject = useCallback(async () => {
    if (!sessionId || !pipelineId) return
    setIsRejecting(true)
    try {
      await rejectSession(sessionId)
      await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
      await queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      navigate(`/pipelines/${pipelineId}`)
    } finally {
      setIsRejecting(false)
    }
  }, [sessionId, pipelineId, rejectSession, queryClient, navigate])

  const handlePublish = useCallback(
    async (_approvedRunIds: string[]) => {
      if (!sessionId || !pipelineId) return
      setIsPublishing(true)
      try {
        // TODO: call publish API endpoint
        await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
        await queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      } finally {
        setIsPublishing(false)
      }
    },
    [sessionId, pipelineId, queryClient],
  )

  // ----- Loading state -----
  if (!session || !pipeline) {
    return (
      <MainLayout
        headerContent={<span className="text-muted-foreground">Loading session...</span>}
      >
        <div className="flex items-center justify-center flex-1">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      </MainLayout>
    )
  }

  // ----- Derived data -----
  const pipelineWorkflows = pipeline.workflows ?? []
  const sessionLabel = session.session_number
    ? `Session #${session.session_number}`
    : `Session ${session.id.slice(0, 8)}`

  const itemCount = session.sources?.reduce((s, src) => s + src.count, 0) ?? 0

  return (
    <MainLayout
      headerContent={
        <div className="flex items-center gap-1.5 text-sm">
          <button
            onClick={() => navigate('/pipelines')}
            className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            Pipelines
          </button>
          <span className="text-muted-foreground">/</span>
          <button
            onClick={() => navigate(`/pipelines/${pipelineId}`)}
            className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            {pipeline.name}
          </button>
          <span className="text-muted-foreground">/</span>
          <span className="text-foreground font-medium">{sessionLabel}</span>
        </div>
      }
    >
      {/* Sticky progress stepper */}
      <StickyProgressBar session={session} />

      {/* Scrollable stage content */}
      <div className="flex-1 overflow-y-auto">
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
              pipelineWorkflows={pipelineWorkflows}
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
            <ProduceStage session={session} />
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
    </MainLayout>
  )
}
