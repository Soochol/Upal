import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Search, GitBranch } from 'lucide-react'
import { listWorkflows, deleteWorkflow, fetchSchedules, type Schedule } from '@/lib/api'
import { deserializeWorkflow, type WorkflowDefinition } from '@/lib/serializer'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useUIStore } from '@/stores/uiStore'
import { Header } from '@/components/Header'
import { WorkflowCard } from './landing/WorkflowCard'

/* ── Main Dashboard ── */
export default function Landing() {
  const navigate = useNavigate()
  const [workflows, setWorkflows] = useState<WorkflowDefinition[]>([])
  const [schedules, setSchedules] = useState<Schedule[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')

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
    <div className="flex flex-col h-screen bg-background text-foreground">
      <Header />

      <main className="flex-1 overflow-y-auto">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
          {/* ─── My Workflows ─── */}
          <section className="mb-14">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold tracking-tight">
                My Workflows
              </h2>
              <div className="flex items-center gap-4">
                {workflows.length > 0 && (
                  <span className="text-xs text-muted-foreground">
                    {workflows.length} workflow{workflows.length !== 1 ? 's' : ''}
                  </span>
                )}
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
                  <input
                    type="text"
                    placeholder="Search workflows..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="w-56 pl-9 pr-4 py-1.5 rounded-lg bg-muted/50 border border-border text-sm text-foreground placeholder:text-muted-foreground/60
                      focus:outline-none focus:ring-1 focus:ring-ring focus:border-ring transition-all"
                  />
                </div>
              </div>
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
                  <span className="text-sm text-muted-foreground group-hover:text-foreground transition-colors">
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
                <p className="text-muted-foreground text-sm">
                  No workflows matching "{search}"
                </p>
              </div>
            ) : (
              /* Empty state — no workflows at all */
              <div className="text-center py-16">
                <div className="w-16 h-16 rounded-2xl bg-muted/20 flex items-center justify-center mx-auto mb-4">
                  <GitBranch className="w-7 h-7 text-muted-foreground/50" />
                </div>
                <h3 className="text-lg font-semibold text-foreground/80 mb-2">
                  No workflows yet
                </h3>
                <p className="text-sm text-muted-foreground mb-6 max-w-sm mx-auto">
                  Create your first AI workflow to get started.
                </p>
                <button
                  onClick={openNew}
                  className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-foreground text-background text-sm font-medium hover:opacity-90 transition-opacity cursor-pointer"
                >
                  <Plus className="w-4 h-4" />
                  Create Workflow
                </button>
              </div>
            )}
          </section>

        </div>
      </main>
    </div>
  )
}
