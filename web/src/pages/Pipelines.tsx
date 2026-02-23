// web/src/pages/Pipelines.tsx — Pipeline list page
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Plus, GitBranch, Play, Clock, Loader2 } from 'lucide-react'
import { Header } from '@/shared/ui/Header'
import { fetchPipelines, deletePipeline, collectPipeline } from '@/entities/pipeline'
import { PipelineCard } from '@/widgets/pipeline-editor'
import type { Pipeline } from '@/shared/types'

type PipelineTab = 'all' | 'content'

function isContentPipeline(p: Pipeline) {
  return (p.sources && p.sources.length > 0) || !!p.schedule || !!p.context
}

function ContentPipelineCard({
  pipeline,
  onClick,
  onCollect,
  isCollecting,
}: {
  pipeline: Pipeline
  onClick: () => void
  onCollect: () => void
  isCollecting?: boolean
}) {
  const lastCollected = pipeline.last_collected_at
    ? new Date(pipeline.last_collected_at).toLocaleString('ko-KR', {
        month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
      })
    : '없음'

  return (
    <div
      className="relative group rounded-2xl border border-border bg-card hover:border-foreground/20
        hover:shadow-lg hover:shadow-black/8 hover:-translate-y-0.5 transition-all duration-200 cursor-pointer overflow-hidden"
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => { if (e.key === 'Enter') onClick() }}
    >
      {/* Action buttons */}
      <div className="absolute top-2.5 right-2.5 z-10 flex items-center gap-0.5
        opacity-0 group-hover:opacity-100 transition-opacity duration-150">
        <button
          onClick={(e) => { e.stopPropagation(); onCollect() }}
          disabled={isCollecting}
          className="p-1.5 rounded-lg hover:bg-primary/10 text-muted-foreground hover:text-primary transition-colors disabled:opacity-60"
          title="지금 수집하기"
        >
          {isCollecting
            ? <Loader2 className="w-3.5 h-3.5 animate-spin" />
            : <Play className="w-3.5 h-3.5" />
          }
        </button>
      </div>

      <div className="relative px-4 pt-4 pb-3.5">
        <h3 className="font-semibold text-sm text-foreground truncate pr-10 leading-snug mb-1">
          {pipeline.name}
        </h3>
        {pipeline.description && (
          <p className="text-xs text-muted-foreground line-clamp-1 mb-2">{pipeline.description}</p>
        )}

        {/* Stats row */}
        <div className="flex items-center gap-3 text-xs text-muted-foreground flex-wrap">
          {pipeline.schedule && (
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {pipeline.schedule}
            </span>
          )}
          <span>소스: {(pipeline.sources ?? []).length}개</span>
          <span>최근 수집: {lastCollected}</span>
        </div>

        {/* Pending badge */}
        {(pipeline.pending_session_count ?? 0) > 0 && (
          <div className="mt-2 flex items-center gap-1.5">
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full
              bg-warning/15 text-warning text-[11px] font-medium border border-warning/20">
              ⏳ 리뷰 대기 {pipeline.pending_session_count}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}

export default function PipelinesPage() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<PipelineTab>('all')

  const { data: pipelines = [], isLoading } = useQuery({
    queryKey: ['pipelines'],
    queryFn: fetchPipelines,
  })

  const deleteMutation = useMutation({
    mutationFn: deletePipeline,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pipelines'] }),
  })

  const collectMutation = useMutation({
    mutationFn: (id: string) => collectPipeline(id),
    onSuccess: ({ session_id }) => navigate(`/inbox/${session_id}`),
  })

  const handleDelete = (pid: string) => {
    if (!confirm('이 파이프라인을 삭제할까요?')) return
    deleteMutation.mutate(pid)
  }

  const contentPipelines = pipelines.filter(isContentPipeline)
  const displayPipelines = activeTab === 'content' ? contentPipelines : pipelines

  return (
    <div className="flex flex-col h-screen bg-background">
      <Header />
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 py-8">

          {/* Page header */}
          <div className="flex items-start justify-between mb-6 gap-4">
            <div>
              <h1 className="landing-display text-2xl font-bold tracking-tight">Pipelines</h1>
              {!isLoading && (
                <span className="text-sm text-muted-foreground mt-1.5 block">
                  <span className="text-foreground font-semibold tabular-nums">{pipelines.length}</span>
                  {' '}total
                </span>
              )}
            </div>
            <button
              onClick={() => navigate('/pipelines/new')}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-xl
                bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer shrink-0"
            >
              <Plus className="h-3.5 w-3.5" />
              New Pipeline
            </button>
          </div>

          {/* Tabs */}
          <div className="flex items-center gap-1 p-1 rounded-xl bg-muted/30 w-fit mb-6">
            <button
              onClick={() => setActiveTab('all')}
              className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer ${
                activeTab === 'all' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              일반 파이프라인
            </button>
            <button
              onClick={() => setActiveTab('content')}
              className={`flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer ${
                activeTab === 'content' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              콘텐츠 파이프라인
              {contentPipelines.length > 0 && (
                <span className="text-[10px] font-semibold">{contentPipelines.length}</span>
              )}
            </button>
          </div>

          {/* Content */}
          {isLoading ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="rounded-2xl bg-card border border-border animate-pulse overflow-hidden">
                  <div className="h-[68px] bg-muted/20 border-b border-border" />
                  <div className="p-4 space-y-2.5">
                    <div className="h-3.5 w-28 bg-muted/40 rounded" />
                    <div className="h-3 w-16 bg-muted/25 rounded" />
                  </div>
                </div>
              ))}
            </div>
          ) : displayPipelines.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {/* New pipeline card */}
              <button
                onClick={() => navigate('/pipelines/new')}
                className="group rounded-2xl border-2 border-dashed border-border
                  hover:border-foreground/25 hover:bg-card/60 transition-all duration-200
                  flex flex-col items-center justify-center min-h-[120px] cursor-pointer"
              >
                <div className="w-9 h-9 rounded-xl bg-muted/30 flex items-center justify-center mb-2
                  group-hover:bg-muted/50 group-hover:scale-110 transition-all duration-200">
                  <Plus className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                </div>
                <span className="text-sm font-medium text-muted-foreground group-hover:text-foreground transition-colors">
                  New pipeline
                </span>
              </button>

              {displayPipelines.map((p, i) => (
                <div
                  key={p.id}
                  className="relative animate-in fade-in slide-in-from-bottom-2 duration-300"
                  style={{ animationDelay: `${i * 35}ms` }}
                >
                  {activeTab === 'content' || isContentPipeline(p) ? (
                    <ContentPipelineCard
                      pipeline={p}
                      onClick={() => navigate(`/pipelines/${p.id}`)}
                      onCollect={() => collectMutation.mutate(p.id)}
                      isCollecting={collectMutation.isPending && collectMutation.variables === p.id}
                    />
                  ) : (
                    <PipelineCard
                      pipeline={p}
                      onClick={() => navigate(`/pipelines/${p.id}`)}
                      onStart={() => {}}
                      onDelete={() => handleDelete(p.id)}
                    />
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-16">
              <div className="w-14 h-14 rounded-2xl bg-muted/20 flex items-center justify-center mx-auto mb-5">
                <GitBranch className="w-6 h-6 text-muted-foreground/40" />
              </div>
              <h3 className="landing-display text-lg font-semibold mb-2">
                {activeTab === 'content' ? '콘텐츠 파이프라인이 없습니다' : '파이프라인이 없습니다'}
              </h3>
              <p className="text-sm text-muted-foreground max-w-xs mx-auto leading-relaxed">
                {activeTab === 'content'
                  ? '소스/스케줄이 설정된 파이프라인이 여기에 표시됩니다.'
                  : '"New Pipeline"으로 파이프라인을 생성하세요.'
                }
              </p>
            </div>
          )}

        </div>
      </main>
    </div>
  )
}
