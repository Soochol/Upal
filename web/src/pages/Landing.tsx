import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Search, GitBranch, Sparkles } from 'lucide-react'
import { listWorkflows, deleteWorkflow, generateWorkflowThumbnail, deserializeWorkflow, useWorkflowStore } from '@/entities/workflow'
import type { WorkflowDefinition } from '@/entities/workflow'
import { fetchRuns } from '@/entities/run'
import { useUIStore } from '@/entities/ui'
import { Header } from '@/components/Header'
import { WorkflowCard } from './landing/WorkflowCard'

/* ── Main Dashboard ── */
export default function Landing() {
  const navigate = useNavigate()
  const [workflows, setWorkflows] = useState<WorkflowDefinition[]>([])
  const [runningWorkflows, setRunningWorkflows] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')

  const addToast = useUIStore((s) => s.addToast)
  const thumbnailRequested = useRef<Set<string>>(new Set())

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
      .catch(() => {})
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // After list loads, generate thumbnails for any workflows that don't have one yet.
  useEffect(() => {
    if (loading) return
    const missing = workflows.filter(
      (wf) => !wf.thumbnail_svg && !thumbnailRequested.current.has(wf.name),
    )
    if (missing.length === 0) return

    let cancelled = false
    const runNext = async (i: number) => {
      if (cancelled || i >= missing.length) return
      const wf = missing[i]
      thumbnailRequested.current.add(wf.name)
      try {
        const svg = await generateWorkflowThumbnail(wf.name)
        if (!cancelled) {
          setWorkflows((prev) =>
            prev.map((w) => (w.name === wf.name ? { ...w, thumbnail_svg: svg } : w)),
          )
        }
      } catch { /* skip — thumbnail is optional */ }
      runNext(i + 1)
    }
    runNext(0)
    return () => { cancelled = true }
  }, [loading]) // eslint-disable-line react-hooks/exhaustive-deps

  const openWorkflow = (wf: WorkflowDefinition) => {
    const { nodes, edges } = deserializeWorkflow(wf)
    useWorkflowStore.setState({ nodes, edges })
    useWorkflowStore.getState().setWorkflowName(wf.name)
    useWorkflowStore.getState().setOriginalName(wf.name)
    navigate('/editor')
  }

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete workflow "${name}"?`)) return
    setWorkflows((prev) => prev.filter((w) => w.name !== name))
    try {
      await deleteWorkflow(name)
    } catch {
      listWorkflows().then(setWorkflows).catch(() => {})
    }
  }

  const openNew = () => {
    useWorkflowStore.setState({ nodes: [], edges: [] })
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    useWorkflowStore.getState().setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
    navigate('/editor')
  }

  const filteredWorkflows = workflows.filter((w) =>
    w.name.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <div className="flex flex-col h-screen bg-background text-foreground">
      <Header />

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
                className="w-44 pl-8 pr-4 py-1.5 rounded-xl bg-card border border-border text-sm
                  text-foreground placeholder:text-muted-foreground/50
                  focus:outline-none focus:ring-1 focus:ring-ring focus:border-ring transition-all"
              />
            </div>
          </div>

          {/* ─── Grid ─── */}
          {loading ? (
            /* Skeleton */
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="rounded-2xl bg-card border border-border animate-pulse overflow-hidden">
                  <div className="h-[68px] bg-muted/20 border-b border-border" />
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
                className="group rounded-2xl border-2 border-dashed border-border
                  hover:border-foreground/25 hover:bg-card/60 transition-all duration-200
                  flex flex-col items-center justify-center min-h-[164px] cursor-pointer"
              >
                <div className="w-9 h-9 rounded-xl bg-muted/30 flex items-center justify-center mb-2.5
                  group-hover:bg-muted/50 group-hover:scale-110 transition-all duration-200">
                  <Plus className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                </div>
                <span className="text-sm font-medium text-muted-foreground group-hover:text-foreground transition-colors">
                  New workflow
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
    </div>
  )
}
