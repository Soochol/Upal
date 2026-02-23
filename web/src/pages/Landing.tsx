import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Search, GitBranch, Sparkles, Activity, CheckCircle2, PlayCircle, Clock } from 'lucide-react'
import { listWorkflows, deleteWorkflow, deserializeWorkflow, useWorkflowStore } from '@/entities/workflow'
import type { WorkflowDefinition } from '@/entities/workflow'
import { fetchRuns, useExecutionStore } from '@/entities/run'
import { useContentSessionStore } from '@/entities/content-session'
import { useUIStore } from '@/entities/ui'

import { templates } from '@/shared/lib/templates'
import { WorkflowMiniGraph } from '@/shared/ui/WorkflowMiniGraph'
import { WorkflowCard } from './landing/WorkflowCard'
import { MainLayout } from '@/app/layout'

/* ── Main Dashboard ── */
export default function Landing() {
  const navigate = useNavigate()
  const [workflows, setWorkflows] = useState<WorkflowDefinition[]>([])
  const [runningWorkflows, setRunningWorkflows] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')

  const addToast = useUIStore((s) => s.addToast)

  useEffect(() => {
    listWorkflows()
      .then(setWorkflows)
      .catch((e) => { setWorkflows([]); addToast(`Failed to load workflows: ${e.message}`) })
      .finally(() => setLoading(false))
    fetchRuns(100, 0)
      .then(({ runs }) => {
        const running = new Set<string>()
        for (const r of runs ?? []) {
          if (r.status === 'running') running.add(r.workflow_name)
        }
        setRunningWorkflows(running)
      })
      .catch(() => { })

    useContentSessionStore.getState().setFilters({ status: 'pending_review' })
    useContentSessionStore.getState().fetchSessions()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openWorkflow = (wf: WorkflowDefinition) => {
    const { nodes, edges } = deserializeWorkflow(wf)
    useWorkflowStore.setState({ nodes, edges, isTemplate: false })
    useWorkflowStore.getState().setWorkflowName(wf.name)
    useWorkflowStore.getState().setOriginalName(wf.name)
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    navigate('/editor')
  }

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete workflow "${name}"?`)) return
    setWorkflows((prev) => prev.filter((w) => w.name !== name))
    try {
      await deleteWorkflow(name)
    } catch {
      listWorkflows().then(setWorkflows).catch(() => { })
    }
  }

  const openNew = () => {
    useWorkflowStore.setState({ nodes: [], edges: [], isTemplate: false })
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    useWorkflowStore.getState().setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    navigate('/editor')
  }

  const openTemplate = (tpl: (typeof templates)[number]) => {
    const { nodes, edges } = deserializeWorkflow(tpl.workflow)
    useWorkflowStore.setState({ nodes, edges, isTemplate: true })
    useWorkflowStore.getState().setWorkflowName(tpl.workflow.name)
    useWorkflowStore.getState().setOriginalName('')
    useExecutionStore.getState().clearNodeStatuses()
    useExecutionStore.getState().clearRunEvents()
    navigate('/editor')
  }

  const filteredWorkflows = workflows.filter((w) =>
    w.name.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <MainLayout headerContent={<span className="font-semibold">Workflows</span>}>
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 py-8">

          {/* ─── Page header ─── */}
          <div className="flex items-start justify-between mb-8 gap-4">
            <div>
              <h1 className="landing-display text-2xl font-bold tracking-tight">Workflows</h1>
              {!loading && (
                <div className="flex items-center gap-5 mt-1.5">
                  <span className="text-sm text-muted-foreground">
                    <span className="text-foreground font-semibold tabular-nums">{workflows.length}</span>
                    {' '}total
                  </span>
                  {runningWorkflows.size > 0 && (
                    <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
                      <span className="run-pulse w-1.5 h-1.5 rounded-full bg-node-agent inline-block shrink-0" />
                      <span className="text-node-agent font-semibold tabular-nums">{runningWorkflows.size}</span>
                      {' '}running
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

          {/* ─── Triage Inbox (Action Required) ─── */}
          {useContentSessionStore(s => s.sessions).length > 0 && (
            <div className="mb-10 animate-in fade-in slide-in-from-top-4 duration-500">
              <div className="flex items-center gap-2 mb-4">
                <div className="size-2 rounded-full bg-warning animate-pulse" />
                <h2 className="text-lg font-semibold tracking-tight text-warning">Action Required</h2>
                <span className="px-2 py-0.5 rounded-full bg-warning/10 text-warning text-xs font-bold tabular-nums ml-1">
                  {useContentSessionStore(s => s.sessions).length}
                </span>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {useContentSessionStore(s => s.sessions).slice(0, 4).map((session) => (
                  <div key={session.id} className="glass-panel border border-warning/20 bg-warning/5 rounded-2xl p-4 flex gap-4 hover:border-warning/40 hover:bg-warning/10 transition-all cursor-pointer">
                    <div className="w-10 h-10 rounded-xl bg-warning/10 flex items-center justify-center shrink-0">
                      <Clock className="w-5 h-5 text-warning" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <h3 className="font-medium text-sm text-foreground truncate">
                        {session.pipeline_name || 'AI Summary Pipeline'}
                      </h3>
                      <p className="text-xs text-muted-foreground mt-1 line-clamp-1">
                        Session #{session.id.split('-')[0]} • {session.sources?.length || 0} sources collected
                      </p>
                    </div>
                    <div className="shrink-0 flex items-center">
                      <button onClick={() => navigate(`/sessions/${session.id}`)} className="px-3 py-1.5 text-xs font-medium rounded-lg bg-warning text-warning-foreground hover:bg-warning/90 transition-colors">
                        Review
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* ─── Dashboard Stats ─── */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
            {[
              { label: 'Total Workflows', value: workflows.length.toString(), icon: GitBranch, color: 'text-blue-400', bg: 'bg-blue-400/10' },
              { label: 'Active Runs', value: runningWorkflows.size.toString(), icon: Activity, color: 'text-amber-400', bg: 'bg-amber-400/10', pulse: runningWorkflows.size > 0 },
              { label: 'Success Rate', value: '98.2%', icon: CheckCircle2, color: 'text-green-400', bg: 'bg-green-400/10' },
              { label: 'Total Executions', value: '1,204', icon: PlayCircle, color: 'text-purple-400', bg: 'bg-purple-400/10' },
            ].map((stat, i) => {
              const Icon = stat.icon
              return (
                <div key={i} className="glass-panel p-5 rounded-2xl border border-white/5 flex items-center gap-4 hover:-translate-y-0.5 transition-transform duration-300">
                  <div className={`size-12 rounded-xl flex items-center justify-center shrink-0 ${stat.bg}`}>
                    <Icon className={`size-5 ${stat.color} ${stat.pulse ? 'animate-pulse' : ''}`} />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">{stat.label}</p>
                    <p className="text-2xl font-bold tracking-tight text-foreground">{loading ? '-' : stat.value}</p>
                  </div>
                </div>
              )
            })}
          </div>

          {/* ─── Quick Start Templates ─── */}
          <div className="mb-10">
            <h2 className="text-lg font-semibold mb-4 tracking-tight">Quick Start Templates</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {templates.map((tpl) => {
                const Icon = tpl.icon
                return (
                  <button
                    key={tpl.id}
                    onClick={() => openTemplate(tpl)}
                    className="group text-left rounded-2xl overflow-hidden glass-panel border border-white/5
                      hover:border-white/10 hover:bg-white/5 hover:shadow-[0_12px_24px_rgba(0,0,0,0.2)]
                      hover:-translate-y-1 transition-all duration-300 cursor-pointer"
                  >
                    {/* Mini graph preview */}
                    <div className="relative h-[68px] border-b border-white/5 overflow-hidden">
                      <WorkflowMiniGraph
                        nodes={tpl.workflow.nodes}
                        edges={tpl.workflow.edges}
                        uid={tpl.id}
                      />
                    </div>

                    {/* Card body */}
                    <div className="px-4 pt-3 pb-3.5">
                      <div className="flex items-center gap-2 mb-1.5">
                        <div className={`size-6 rounded-md flex items-center justify-center ${tpl.color}`}>
                          <Icon className="size-3.5" />
                        </div>
                        <h3 className="font-semibold text-sm text-foreground group-hover:text-primary transition-colors truncate">
                          {tpl.title}
                        </h3>
                      </div>
                      <p className="text-xs text-muted-foreground/80 leading-relaxed line-clamp-1 mb-2">
                        {tpl.description}
                      </p>
                      <div className="flex items-center gap-2">
                        <span className="text-[10px] text-muted-foreground/50 tabular-nums">
                          {tpl.workflow.nodes.length} nodes
                        </span>
                        <span className={`text-[10px] px-1.5 py-0.5 rounded-full ${
                          tpl.difficulty === 'Beginner'
                            ? 'bg-success/10 text-success'
                            : 'bg-warning/10 text-warning'
                        }`}>
                          {tpl.difficulty}
                        </span>
                      </div>
                    </div>
                  </button>
                )
              })}
            </div>
          </div>

          {/* ─── Recent Workflows Grid ─── */}
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold tracking-tight">Recent Workflows</h2>
          </div>

          {loading ? (
            /* Skeleton */
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="rounded-2xl glass-panel border-border/60 animate-pulse overflow-hidden">
                  <div className="h-[68px] bg-muted/20 border-b border-border/40" />
                  <div className="p-4 space-y-2.5">
                    <div className="h-3.5 w-28 bg-muted/40 rounded" />
                    <div className="flex gap-1.5">
                      {[1, 2].map((j) => (
                        <div key={j} className="h-4 w-12 bg-muted/25 rounded-full" />
                      ))}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : filteredWorkflows.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {/* Create new card */}
              <button
                onClick={openNew}
                className="group rounded-2xl border-2 border-dashed border-white/10
                  hover:border-primary/50 hover:bg-primary/5 transition-all duration-300
                  flex flex-col items-center justify-center min-h-[164px] cursor-pointer"
              >
                <div className="w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center mb-3
                  group-hover:bg-primary/10 group-hover:scale-110 transition-all duration-300">
                  <Plus className="w-5 h-5 text-muted-foreground group-hover:text-primary transition-colors" />
                </div>
                <span className="text-sm font-medium text-muted-foreground group-hover:text-primary transition-colors">
                  Blank Workflow
                </span>
              </button>

              {filteredWorkflows.map((wf, i) => (
                <div
                  key={wf.name}
                  className="animate-in fade-in slide-in-from-bottom-2 duration-300"
                  style={{ animationDelay: `${i * 35}ms` }}
                >
                  <WorkflowCard
                    workflow={wf}
                    onClick={() => openWorkflow(wf)}
                    onDelete={() => handleDelete(wf.name)}
                    isRunning={runningWorkflows.has(wf.name)}
                  />
                </div>
              ))}
            </div>
          ) : workflows.length > 0 && filteredWorkflows.length === 0 ? (
            /* Search no results */
            <div className="text-center py-14">
              <p className="text-sm text-muted-foreground">
                No workflows matching "<span className="text-foreground">{search}</span>"
              </p>
            </div>
          ) : (
            /* Empty state */
            <div className="text-center py-20">
              <div className="w-14 h-14 rounded-2xl bg-muted/20 flex items-center justify-center mx-auto mb-5">
                <GitBranch className="w-6 h-6 text-muted-foreground/40" />
              </div>
              <h3 className="landing-display text-lg font-semibold mb-2">No workflows yet</h3>
              <p className="text-sm text-muted-foreground mb-8 max-w-xs mx-auto leading-relaxed">
                Create your first AI workflow or generate one from a description.
              </p>
              <div className="flex items-center justify-center gap-3">
                <button
                  onClick={openNew}
                  className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-foreground text-background
                    text-sm font-medium hover:opacity-90 transition-opacity cursor-pointer"
                >
                  <Plus className="w-4 h-4" />
                  Create Workflow
                </button>
                <button
                  onClick={() => navigate('/editor?generate=true')}
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
