import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft, Play, Loader2, Trash2, Plus, ExternalLink,
  Clock, Settings,
} from 'lucide-react'
import { Header } from '@/shared/ui/Header'
import { BreadcrumbNav } from '@/shared/ui/BreadcrumbNav'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { EditorialBriefForm } from '@/features/define-editorial-brief/EditorialBriefForm'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import {
  fetchPipeline, updatePipeline, collectPipeline,
} from '@/entities/pipeline'
import { fetchContentSessions } from '@/entities/content-session/api'
import type { Pipeline, PipelineSource, PipelineContext } from '@/shared/types'

// ─── Sub-panels ──────────────────────────────────────────────────────────────

function SessionHistoryTab({
  pipelineId,
  onViewSession,
}: {
  pipelineId: string
  onViewSession: (sessionId: string) => void
}) {
  const { data: sessions = [], isLoading } = useQuery({
    queryKey: ['content-sessions', { pipelineId }],
    queryFn: () => fetchContentSessions({ pipelineId }),
    enabled: !!pipelineId,
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (sessions.length === 0) {
    return (
      <div className="text-center py-12 text-sm text-muted-foreground">
        아직 세션이 없습니다. "지금 수집하기"로 첫 세션을 시작하세요.
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {sessions.map((session) => {
        const createdAt = new Date(session.created_at).toLocaleString('ko-KR', {
          month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
        })
        return (
          <div
            key={session.id}
            className="rounded-xl border border-border bg-card px-4 py-3 flex items-center justify-between gap-3"
          >
            <div className="flex items-center gap-3 flex-wrap min-w-0">
              <span className="text-sm font-medium shrink-0">Session {session.session_number}</span>
              <span className="text-xs text-muted-foreground shrink-0">{createdAt}</span>
              {session.analysis && (
                <div className="flex items-center gap-1 shrink-0">
                  <span className="text-[11px] text-muted-foreground">Score:</span>
                  <ScoreIndicator score={session.analysis.score} />
                </div>
              )}
              <StatusBadge status={session.status} />
            </div>
            <button
              onClick={() => onViewSession(session.id)}
              className="flex items-center gap-1 text-xs text-muted-foreground
                hover:text-foreground transition-colors shrink-0 cursor-pointer"
            >
              <ExternalLink className="h-3.5 w-3.5" />
              보기
            </button>
          </div>
        )
      })}
    </div>
  )
}

function SourceConfigTab({
  sources,
  schedule,
  onSourcesChange,
  onScheduleChange,
  onSave,
}: {
  sources: PipelineSource[]
  schedule: string
  onSourcesChange: (sources: PipelineSource[]) => void
  onScheduleChange: (cron: string) => void
  onSave: () => Promise<void>
}) {
  const [showAddModal, setShowAddModal] = useState(false)
  const [saving, setSaving] = useState(false)

  const handleSave = async () => {
    setSaving(true)
    try { await onSave() } finally { setSaving(false) }
  }

  return (
    <div className="space-y-5">
      {/* Source list */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">수집 소스</h3>
        <button
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-1 text-xs text-muted-foreground
            hover:text-foreground transition-colors cursor-pointer"
        >
          <Plus className="h-3.5 w-3.5" />
          소스 추가
        </button>
      </div>

      {sources.length === 0 ? (
        <p className="text-sm text-muted-foreground py-4 text-center rounded-xl border border-dashed border-border">
          소스가 없습니다. "소스 추가"로 수집 소스를 설정하세요.
        </p>
      ) : (
        <div className="rounded-xl border border-border overflow-hidden">
          {sources.map((src, i) => (
            <div
              key={src.id}
              className="flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0"
            >
              <SourceTypeBadge type={src.source_type} />
              <span className="text-sm font-medium flex-1">{src.label}</span>
              <span className="text-xs text-muted-foreground truncate max-w-[180px]">
                {src.url ?? src.subreddit ?? (src.keywords?.join(', ') ?? '')}
              </span>
              <button
                onClick={() => onSourcesChange(sources.filter((_, j) => j !== i))}
                className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Schedule */}
      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">
          <Clock className="inline h-3.5 w-3.5 mr-1" />
          수집 스케줄 (cron)
        </label>
        <input
          type="text"
          value={schedule}
          onChange={(e) => onScheduleChange(e.target.value)}
          placeholder="0 */6 * * *  (6시간마다)"
          className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm
            font-mono outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground placeholder:font-sans"
        />
        <p className="text-xs text-muted-foreground mt-1">
          cron 표현식. 예: <code className="font-mono">0 */6 * * *</code> = 6시간마다
        </p>
      </div>

      <button
        onClick={handleSave}
        disabled={saving}
        className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
          bg-foreground text-background hover:opacity-90 transition-opacity
          disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
      >
        {saving ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />저장 중…</> : '저장'}
      </button>

      {showAddModal && (
        <AddSourceModal
          onAdd={(src) => onSourcesChange([...sources, src])}
          onClose={() => setShowAddModal(false)}
        />
      )}
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

type Tab = 'sessions' | 'sources' | 'brief' | 'workflows'

export default function PipelineDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<Tab>('sessions')

  const { data: pipeline, isLoading } = useQuery({
    queryKey: ['pipeline', id],
    queryFn: () => fetchPipeline(id!),
    enabled: !!id,
  })

  const [localSources, setLocalSources] = useState<PipelineSource[]>([])
  const [localSchedule, setLocalSchedule] = useState('')

  // Sync local state when a different pipeline loads
  useEffect(() => {
    if (pipeline) {
      setLocalSources(pipeline.sources ?? [])
      setLocalSchedule(pipeline.schedule ?? '')
    }
  }, [pipeline?.id])

  const collectMutation = useMutation({
    mutationFn: () => collectPipeline(id!),
    onSuccess: ({ session_id }) => {
      navigate(`/inbox/${session_id}`)
    },
  })

  const updateContextMutation = useMutation({
    mutationFn: (ctx: PipelineContext) => updatePipeline(id!, { ...pipeline!, context: ctx }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline', id] })
    },
  })

  const handleSaveSourcesAndSchedule = async () => {
    if (!pipeline) return
    await updatePipeline(id!, { ...pipeline, sources: localSources, schedule: localSchedule })
    queryClient.invalidateQueries({ queryKey: ['pipeline', id] })
  }

  if (isLoading || !pipeline) {
    return (
      <div className="flex flex-col h-screen bg-background">
        <Header />
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      </div>
    )
  }

  const tabs: { value: Tab; label: string }[] = [
    { value: 'sessions', label: '세션 이력' },
    { value: 'sources', label: '소스 설정' },
    { value: 'brief', label: 'Editorial Brief' },
    { value: 'workflows', label: '워크플로우' },
  ]

  const lastCollectedLabel = pipeline.last_collected_at
    ? new Date(pipeline.last_collected_at).toLocaleString('ko-KR', {
        month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
      })
    : '없음'

  return (
    <div className="flex flex-col h-screen bg-background">
      <Header />
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8">

          {/* Breadcrumb */}
          <BreadcrumbNav
            className="mb-5"
            items={[
              { label: 'Pipelines', to: '/pipelines' },
              { label: pipeline.name },
            ]}
          />

          {/* Pipeline header */}
          <div className="flex items-start justify-between gap-4 mb-2">
            <div className="flex items-center gap-3">
              <button
                onClick={() => navigate('/pipelines')}
                className="p-1.5 rounded-md hover:bg-muted transition-colors cursor-pointer"
              >
                <ArrowLeft className="h-4 w-4" />
              </button>
              <h1 className="text-xl font-bold">{pipeline.name}</h1>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <button
                onClick={() => collectMutation.mutate()}
                disabled={collectMutation.isPending}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium
                  bg-primary text-primary-foreground hover:opacity-90 transition-opacity
                  disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
              >
                {collectMutation.isPending
                  ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />수집 중…</>
                  : <><Play className="h-3.5 w-3.5" />지금 수집하기</>
                }
              </button>
              <button
                onClick={() => setActiveTab('sources')}
                className="p-1.5 rounded-xl border border-border hover:bg-muted transition-colors cursor-pointer"
                title="설정"
              >
                <Settings className="h-4 w-4 text-muted-foreground" />
              </button>
            </div>
          </div>

          {/* Meta info */}
          {pipeline.description && (
            <p className="text-sm text-muted-foreground ml-10 mb-1">{pipeline.description}</p>
          )}
          <div className="flex items-center gap-4 text-xs text-muted-foreground ml-10 mb-6">
            {pipeline.schedule && (
              <span className="flex items-center gap-1">
                <Clock className="h-3 w-3" />
                스케줄: <code className="font-mono">{pipeline.schedule}</code>
              </span>
            )}
            <span>소스: {(pipeline.sources ?? []).length}개</span>
            <span>최근 수집: {lastCollectedLabel}</span>
            {(pipeline.pending_session_count ?? 0) > 0 && (
              <span className="text-warning font-medium">
                ⏳ 리뷰 대기: {pipeline.pending_session_count}
              </span>
            )}
          </div>

          {/* Tabs */}
          <div className="flex items-center gap-1 p-1 rounded-xl bg-muted/30 w-fit mb-6 flex-wrap">
            {tabs.map(({ value, label }) => (
              <button
                key={value}
                onClick={() => setActiveTab(value)}
                className={`px-4 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer ${
                  activeTab === value
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {label}
              </button>
            ))}
          </div>

          {/* Tab content */}
          {activeTab === 'sessions' && (
            <SessionHistoryTab
              pipelineId={pipeline.id}
              onViewSession={(sid) => navigate(`/inbox/${sid}`)}
            />
          )}

          {activeTab === 'sources' && (
            <SourceConfigTab
              sources={localSources}
              schedule={localSchedule}
              onSourcesChange={setLocalSources}
              onScheduleChange={setLocalSchedule}
              onSave={handleSaveSourcesAndSchedule}
            />
          )}

          {activeTab === 'brief' && (
            <EditorialBriefForm
              initialContext={pipeline.context}
              onSave={async (ctx) => { await updateContextMutation.mutateAsync(ctx) }}
            />
          )}

          {activeTab === 'workflows' && (
            <WorkflowTemplatesTab pipeline={pipeline} />
          )}

        </div>
      </main>
    </div>
  )
}

// ─── Workflow templates tab ───────────────────────────────────────────────────

function WorkflowTemplatesTab({ pipeline }: { pipeline: Pipeline }) {
  const navigate = useNavigate()
  const workflowStages = pipeline.stages.filter((s) => s.type === 'workflow')

  if (workflowStages.length === 0) {
    return (
      <div className="text-center py-12 text-sm text-muted-foreground">
        이 파이프라인에 연결된 워크플로우 스테이지가 없습니다.
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {workflowStages.map((stage) => (
        <div
          key={stage.id}
          className="rounded-xl border border-border bg-card px-4 py-3 flex items-center justify-between"
        >
          <div>
            <p className="text-sm font-medium">{stage.name || stage.config.workflow_name || '워크플로우'}</p>
            {stage.config.workflow_name && (
              <p className="text-xs text-muted-foreground mt-0.5">
                workflow: {stage.config.workflow_name}
              </p>
            )}
          </div>
          <button
            onClick={() => navigate(`/editor?workflow=${stage.config.workflow_name}`)}
            className="flex items-center gap-1 text-xs text-muted-foreground
              hover:text-foreground transition-colors cursor-pointer"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            에디터에서 열기
          </button>
        </div>
      ))}
    </div>
  )
}
