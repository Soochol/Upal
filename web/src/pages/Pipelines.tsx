// web/src/pages/Pipelines.tsx — Pipeline dashboard page
import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import {
  Plus, GitBranch, Search, Clock, Rss, CalendarCheck, Sparkles,
} from 'lucide-react'
import { MainLayout } from '@/app/layout'
import { fetchPipelines, deletePipeline, collectPipeline, startPipeline } from '@/entities/pipeline'
import { useContentSessionStore } from '@/entities/content-session/store'
import { PipelineCard } from '@/widgets/pipeline-editor'
import type { Pipeline } from '@/shared/types'

type PipelineTab = 'all' | 'content'

function isContentPipeline(p: Pipeline) {
  return (p.sources && p.sources.length > 0) || !!p.schedule || !!p.context
}

export default function PipelinesPage() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<PipelineTab>('all')
  const [search, setSearch] = useState('')

  const { data: pipelines = [], isLoading } = useQuery({
    queryKey: ['pipelines'],
    queryFn: fetchPipelines,
  })

  // Fetch pending sessions for triage section (same pattern as Landing.tsx)
  const pendingSessions = useContentSessionStore((s) => s.sessions)
  useEffect(() => {
    useContentSessionStore.getState().setFilters({ status: 'pending_review' })
    useContentSessionStore.getState().fetchSessions()
  }, [])

  const deleteMutation = useMutation({
    mutationFn: deletePipeline,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pipelines'] }),
  })

  const collectMutation = useMutation({
    mutationFn: (id: string) => collectPipeline(id),
    onSuccess: ({ session_id }, pipelineId) => navigate(`/pipelines/${pipelineId}/sessions/${session_id}`),
  })

  const startMutation = useMutation({
    mutationFn: (id: string) => startPipeline(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pipelines'] }),
  })

  const handleDelete = (pid: string) => {
    if (!confirm('Delete this pipeline?')) return
    deleteMutation.mutate(pid)
  }

  // Derived data
  const contentPipelines = pipelines.filter(isContentPipeline)
  const totalPendingReview = pipelines.reduce((sum, p) => sum + (p.pending_session_count ?? 0), 0)
  const scheduledCount = pipelines.filter((p) => !!p.schedule).length
  const displayPipelines = activeTab === 'content' ? contentPipelines : pipelines
  const filteredPipelines = displayPipelines.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()),
  )

  // Stats
  const stats = [
    {
      label: 'Total Pipelines',
      value: pipelines.length.toString(),
      icon: GitBranch,
      color: 'text-blue-400',
      bg: 'bg-blue-400/10',
    },
    {
      label: 'Pending Review',
      value: totalPendingReview.toString(),
      icon: Clock,
      color: 'text-warning',
      bg: 'bg-warning/10',
      pulse: totalPendingReview > 0,
    },
    {
      label: 'Content Pipelines',
      value: contentPipelines.length.toString(),
      icon: Rss,
      color: 'text-purple-400',
      bg: 'bg-purple-400/10',
    },
    {
      label: 'Scheduled',
      value: scheduledCount.toString(),
      icon: CalendarCheck,
      color: 'text-green-400',
      bg: 'bg-green-400/10',
    },
  ]

  return (
    <MainLayout headerContent={<span className="font-semibold">Pipelines</span>}>
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 py-8">

          {/* ─── Page header ─── */}
          <div className="flex items-start justify-between mb-8 gap-4">
            <div>
              <h1 className="landing-display text-2xl font-bold tracking-tight">Pipelines</h1>
              {!isLoading && (
                <div className="flex items-center gap-5 mt-1.5">
                  <span className="text-sm text-muted-foreground">
                    <span className="text-foreground font-semibold tabular-nums">{pipelines.length}</span>
                    {' '}total
                  </span>
                  {totalPendingReview > 0 && (
                    <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
                      <span className="run-pulse w-1.5 h-1.5 rounded-full bg-warning inline-block shrink-0" />
                      <span className="text-warning font-semibold tabular-nums">{totalPendingReview}</span>
                      {' '}pending review
                    </span>
                  )}
                </div>
              )}
            </div>

            {/* Search */}
            <div className="relative shrink-0">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
              <input
                type="text"
                placeholder="Search…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-44 pl-8 pr-4 py-1.5 rounded-xl bg-background/50 backdrop-blur-md border border-border/60 text-sm shadow-sm
                  text-foreground placeholder:text-muted-foreground/50
                  focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-border transition-all duration-200"
              />
            </div>
          </div>

          {/* ─── Action Required Triage ─── */}
          {pendingSessions.length > 0 && (
            <div className="mb-10 animate-in fade-in slide-in-from-top-4 duration-500">
              <div className="flex items-center gap-2 mb-4">
                <div className="size-2 rounded-full bg-warning animate-pulse" />
                <h2 className="text-lg font-semibold tracking-tight text-warning">Action Required</h2>
                <span className="px-2 py-0.5 rounded-full bg-warning/10 text-warning text-xs font-bold tabular-nums ml-1">
                  {pendingSessions.length}
                </span>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {pendingSessions.slice(0, 4).map((session) => (
                  <div
                    key={session.id}
                    className="glass-panel border border-warning/20 bg-warning/5 rounded-2xl p-4 flex gap-4
                      hover:border-warning/40 hover:bg-warning/10 transition-all cursor-pointer"
                    onClick={() => navigate(`/pipelines/${session.pipeline_id}/sessions/${session.id}`)}
                  >
                    <div className="w-10 h-10 rounded-xl bg-warning/10 flex items-center justify-center shrink-0">
                      <Clock className="w-5 h-5 text-warning" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <h3 className="font-medium text-sm text-foreground truncate">
                        {session.pipeline_name || 'Content Pipeline'}
                      </h3>
                      <p className="text-xs text-muted-foreground mt-1 line-clamp-1">
                        Session #{session.session_number ?? session.id.split('-')[0]} &middot; {session.sources?.length || 0} sources collected
                      </p>
                    </div>
                    <div className="shrink-0 flex items-center">
                      <span className="px-3 py-1.5 text-xs font-medium rounded-lg bg-warning text-warning-foreground hover:bg-warning/90 transition-colors">
                        Review
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* ─── Dashboard Stats ─── */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
            {stats.map((stat, i) => {
              const Icon = stat.icon
              return (
                <div key={i} className="glass-panel p-5 rounded-2xl border border-white/5 flex items-center gap-4 hover:-translate-y-0.5 transition-transform duration-300">
                  <div className={`size-12 rounded-xl flex items-center justify-center shrink-0 ${stat.bg}`}>
                    <Icon className={`size-5 ${stat.color} ${stat.pulse ? 'animate-pulse' : ''}`} />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">{stat.label}</p>
                    <p className="text-2xl font-bold tracking-tight text-foreground">{isLoading ? '-' : stat.value}</p>
                  </div>
                </div>
              )
            })}
          </div>

          {/* ─── Tabs ─── */}
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-1 p-1 rounded-xl bg-muted/30 w-fit">
              <button
                onClick={() => setActiveTab('all')}
                className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer ${
                  activeTab === 'all' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                All Pipelines
              </button>
              <button
                onClick={() => setActiveTab('content')}
                className={`flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer ${
                  activeTab === 'content' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                Content Pipelines
                {contentPipelines.length > 0 && (
                  <span className={`text-[10px] font-semibold rounded-full min-w-[16px] px-1 text-center ${
                    activeTab === 'content' ? 'bg-foreground/10' : 'bg-muted-foreground/20 text-muted-foreground'
                  }`}>
                    {contentPipelines.length}
                  </span>
                )}
              </button>
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

          {/* ─── Content ─── */}
          {isLoading ? (
            <>
              {/* Stats skeleton */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-10 hidden">
                {[1, 2, 3, 4].map((i) => (
                  <div key={i} className="glass-panel p-5 rounded-2xl border border-white/5 flex items-center gap-4 animate-pulse">
                    <div className="size-12 rounded-xl bg-muted/20" />
                    <div className="space-y-2 flex-1">
                      <div className="h-3 w-20 bg-muted/30 rounded" />
                      <div className="h-6 w-12 bg-muted/40 rounded" />
                    </div>
                  </div>
                ))}
              </div>
              {/* Card skeleton */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="rounded-2xl glass-panel border border-border/60 animate-pulse overflow-hidden">
                    <div className="h-[68px] bg-muted/20 border-b border-border/40" />
                    <div className="p-4 space-y-2.5">
                      <div className="h-3.5 w-28 bg-muted/40 rounded" />
                      <div className="h-3 w-40 bg-muted/25 rounded" />
                      <div className="flex gap-1.5">
                        <div className="h-4 w-12 bg-muted/25 rounded-full" />
                        <div className="h-4 w-16 bg-muted/25 rounded-full" />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </>
          ) : filteredPipelines.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {/* New pipeline card */}
              <button
                onClick={() => navigate('/pipelines/new')}
                className="group rounded-2xl border-2 border-dashed border-white/10
                  hover:border-primary/50 hover:bg-primary/5 transition-all duration-300
                  flex flex-col items-center justify-center min-h-[164px] cursor-pointer"
              >
                <div className="w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center mb-3
                  group-hover:bg-primary/10 group-hover:scale-110 transition-all duration-300">
                  <Plus className="w-5 h-5 text-muted-foreground group-hover:text-primary transition-colors" />
                </div>
                <span className="text-sm font-medium text-muted-foreground group-hover:text-primary transition-colors">
                  New Pipeline
                </span>
              </button>

              {filteredPipelines.map((p, i) => (
                <div
                  key={p.id}
                  className="animate-in fade-in slide-in-from-bottom-2 duration-300"
                  style={{ animationDelay: `${i * 35}ms` }}
                >
                  <PipelineCard
                    pipeline={p}
                    onClick={() => navigate(`/pipelines/${p.id}`)}
                    onStart={() => startMutation.mutate(p.id)}
                    onCollect={isContentPipeline(p) ? () => collectMutation.mutate(p.id) : undefined}
                    onDelete={() => handleDelete(p.id)}
                    isCollecting={collectMutation.isPending && collectMutation.variables === p.id}
                  />
                </div>
              ))}
            </div>
          ) : pipelines.length > 0 && search ? (
            /* Search no results */
            <div className="text-center py-14">
              <p className="text-sm text-muted-foreground">
                No pipelines matching &ldquo;<span className="text-foreground">{search}</span>&rdquo;
              </p>
            </div>
          ) : (
            /* Empty state */
            <div className="text-center py-20">
              <div className="w-14 h-14 rounded-2xl bg-muted/20 flex items-center justify-center mx-auto mb-5">
                <GitBranch className="w-6 h-6 text-muted-foreground/40" />
              </div>
              <h3 className="landing-display text-lg font-semibold mb-2">
                {activeTab === 'content' ? 'No content pipelines' : 'No pipelines yet'}
              </h3>
              <p className="text-sm text-muted-foreground mb-8 max-w-xs mx-auto leading-relaxed">
                {activeTab === 'content'
                  ? 'Pipelines with sources or a schedule appear here.'
                  : 'Create your first pipeline to orchestrate multi-stage AI workflows.'
                }
              </p>
              <div className="flex items-center justify-center gap-3">
                <button
                  onClick={() => navigate('/pipelines/new')}
                  className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-foreground text-background
                    text-sm font-medium hover:opacity-90 transition-opacity cursor-pointer"
                >
                  <Plus className="w-4 h-4" />
                  Create Pipeline
                </button>
                <button
                  onClick={() => navigate('/pipelines/new?generate=true')}
                  className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl border border-border
                    text-sm font-medium text-foreground hover:bg-card/60 transition-all cursor-pointer"
                >
                  <Sparkles className="w-4 h-4" />
                  Generate with AI
                </button>
              </div>
            </div>
          )}

        </div>
      </main>
    </MainLayout>
  )
}
