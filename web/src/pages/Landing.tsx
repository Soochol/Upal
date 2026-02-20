import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Search, GitBranch, History, CalendarClock } from 'lucide-react'
import { listWorkflows, deleteWorkflow, fetchSchedules, type Schedule } from '@/lib/api'
import { deserializeWorkflow, type WorkflowDefinition } from '@/lib/serializer'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useUIStore } from '@/stores/uiStore'
import { useTheme } from '@/components/ThemeProvider'
import { WorkflowCard } from './landing/WorkflowCard'

/* ── Main Dashboard ── */
export default function Landing() {
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()
  const prevThemeRef = useRef(theme)
  const [workflows, setWorkflows] = useState<WorkflowDefinition[]>([])
  const [schedules, setSchedules] = useState<Schedule[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')

  // Force dark theme on landing page so body background is also dark
  useEffect(() => {
    prevThemeRef.current = theme
    if (theme !== 'dark') setTheme('dark')
    return () => {
      if (prevThemeRef.current !== 'dark') setTheme(prevThemeRef.current)
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const addToast = useUIStore((s) => s.addToast)

  useEffect(() => {
    listWorkflows()
      .then(setWorkflows)
      .catch((e) => { setWorkflows([]); addToast(`Failed to load workflows: ${e.message}`) })
      .finally(() => setLoading(false))
    fetchSchedules()
      .then(setSchedules)
      .catch(() => setSchedules([]))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openWorkflow = (wf: WorkflowDefinition) => {
    const { nodes, edges } = deserializeWorkflow(wf)
    useWorkflowStore.setState({ nodes, edges })
    useWorkflowStore.getState().setWorkflowName(wf.name)
    useWorkflowStore.getState().setOriginalName(wf.name)
    navigate('/editor')
  }

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete workflow "${name}"?`)) return
    // Optimistic: remove from UI immediately, restore on failure
    setWorkflows((prev) => prev.filter((w) => w.name !== name))
    try {
      await deleteWorkflow(name)
    } catch {
      // Restore on failure — re-fetch the full list
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

  // Build schedule lookup: workflow name → { count, nearest next_run_at }
  const scheduleMap = new Map<string, { count: number; nextRun?: string }>()
  for (const s of schedules) {
    const entry = scheduleMap.get(s.workflow_name)
    if (entry) {
      entry.count++
      if (s.next_run_at && (!entry.nextRun || s.next_run_at < entry.nextRun)) {
        entry.nextRun = s.next_run_at
      }
    } else {
      scheduleMap.set(s.workflow_name, { count: 1, nextRun: s.next_run_at })
    }
  }

  // Filter by search
  const filteredWorkflows = workflows.filter((w) =>
    w.name.toLowerCase().includes(search.toLowerCase()),
  )
  return (
    <div className="bg-background text-foreground min-h-screen landing-body">
        {/* ═══ NAV ═══ */}
        <nav className="landing-nav sticky top-0 z-50 border-b border-border">
          <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
            {/* Logo */}
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-foreground flex items-center justify-center">
                <span className="text-background text-sm font-bold landing-display">U</span>
              </div>
              <span className="text-lg font-bold landing-display tracking-tight">Upal</span>
            </div>

            {/* Search */}
            <div className="flex-1 max-w-md mx-6">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
                <input
                  type="text"
                  placeholder="Search workflows..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full pl-9 pr-4 py-2 rounded-xl bg-card/60 border border-border text-sm text-foreground placeholder:text-muted-foreground/60 landing-body
                    focus:outline-none focus:ring-1 focus:ring-ring focus:border-ring transition-all"
                />
              </div>
            </div>

            {/* Schedules */}
            <button
              onClick={() => navigate('/schedules')}
              className="flex items-center gap-2 px-3 py-2 rounded-xl text-sm text-muted-foreground hover:text-foreground hover:bg-card/60 transition-all cursor-pointer landing-body"
            >
              <CalendarClock className="w-4 h-4" />
              <span className="hidden sm:inline">Schedules</span>
            </button>

            {/* Run History */}
            <button
              onClick={() => navigate('/runs')}
              className="flex items-center gap-2 px-3 py-2 rounded-xl text-sm text-muted-foreground hover:text-foreground hover:bg-card/60 transition-all cursor-pointer landing-body"
            >
              <History className="w-4 h-4" />
              <span className="hidden sm:inline">Runs</span>
            </button>

          </div>
        </nav>

        {/* ═══ CONTENT ═══ */}
        <main className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
          {/* ─── My Workflows ─── */}
          <section className="mb-14">
            <div className="flex items-center justify-between mb-6">
              <h2 className="landing-display text-xl font-bold tracking-tight">
                My Workflows
              </h2>
              {workflows.length > 0 && (
                <span className="text-xs text-muted-foreground landing-body">
                  {workflows.length} workflow{workflows.length !== 1 ? 's' : ''}
                </span>
              )}
            </div>

            {loading ? (
              /* Skeleton loader */
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {[1, 2, 3].map((i) => (
                  <div
                    key={i}
                    className="rounded-2xl p-5 bg-card/30 border border-border animate-pulse"
                  >
                    <div className="flex gap-1.5 mb-4">
                      {[1, 2, 3].map((j) => (
                        <div key={j} className="w-6 h-6 rounded-lg bg-muted/40" />
                      ))}
                    </div>
                    <div className="h-5 w-32 bg-muted/40 rounded mb-2" />
                    <div className="h-4 w-48 bg-muted/20 rounded" />
                  </div>
                ))}
              </div>
            ) : filteredWorkflows.length > 0 ? (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {/* New workflow card */}
                <button
                  onClick={openNew}
                  className="group rounded-2xl p-5 border-2 border-dashed border-border
                    hover:border-foreground/30 hover:bg-card/40 transition-all duration-200
                    flex flex-col items-center justify-center min-h-[160px] cursor-pointer"
                >
                  <div className="w-12 h-12 rounded-2xl bg-muted/30 flex items-center justify-center mb-3
                    group-hover:bg-muted/50 transition-colors">
                    <Plus className="w-6 h-6 text-muted-foreground group-hover:text-foreground transition-colors" />
                  </div>
                  <span className="text-sm text-muted-foreground group-hover:text-foreground transition-colors landing-body">
                    Create new workflow
                  </span>
                </button>

                {filteredWorkflows.map((wf) => {
                  const sched = scheduleMap.get(wf.name)
                  return (
                    <WorkflowCard
                      key={wf.name}
                      workflow={wf}
                      onClick={() => openWorkflow(wf)}
                      onDelete={() => handleDelete(wf.name)}
                      scheduleCount={sched?.count}
                      nextRun={sched?.nextRun}
                    />
                  )
                })}
              </div>
            ) : workflows.length > 0 && filteredWorkflows.length === 0 ? (
              /* Search returned no results from user workflows */
              <div className="text-center py-12">
                <p className="text-muted-foreground text-sm landing-body">
                  No workflows matching "{search}"
                </p>
              </div>
            ) : (
              /* Empty state — no workflows at all */
              <div className="text-center py-16">
                <div className="w-16 h-16 rounded-2xl bg-muted/20 flex items-center justify-center mx-auto mb-4">
                  <GitBranch className="w-7 h-7 text-muted-foreground/50" />
                </div>
                <h3 className="landing-display text-lg font-semibold text-foreground/80 mb-2">
                  No workflows yet
                </h3>
                <p className="text-sm text-muted-foreground mb-6 max-w-sm mx-auto">
                  Create your first AI workflow to get started.
                </p>
                <button
                  onClick={openNew}
                  className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-foreground text-background text-sm font-medium landing-body hover:opacity-90 transition-opacity cursor-pointer"
                >
                  <Plus className="w-4 h-4" />
                  Create Workflow
                </button>
              </div>
            )}
          </section>

        </main>
    </div>
  )
}
