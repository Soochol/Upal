import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Search, LayoutGrid, Sparkles, GitBranch, Cpu, Wrench } from 'lucide-react'
import { listWorkflows } from '@/lib/api'
import { deserializeWorkflow, type WorkflowDefinition } from '@/lib/serializer'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useTheme } from '@/components/ThemeProvider'

/* ── Template workflows ── */
const templates: (WorkflowDefinition & { description: string; tags: string[] })[] = [
  {
    name: 'Simple Chat',
    description: 'A basic prompt-response flow with a single AI agent.',
    version: 1,
    tags: ['starter', 'chat'],
    nodes: [
      { id: 'input_1', type: 'input', config: { value: '' } },
      { id: 'agent_1', type: 'agent', config: { model: 'anthropic/claude-sonnet-4-20250514', prompt: '{{input_1}}' } },
      { id: 'output_1', type: 'output', config: {} },
    ],
    edges: [
      { from: 'input_1', to: 'agent_1' },
      { from: 'agent_1', to: 'output_1' },
    ],
  },
  {
    name: 'Multi-Model Compare',
    description: 'Send the same prompt to multiple models and compare outputs side by side.',
    version: 1,
    tags: ['comparison', 'multi-model'],
    nodes: [
      { id: 'input_1', type: 'input', config: { value: '' } },
      { id: 'agent_1', type: 'agent', config: { model: 'anthropic/claude-sonnet-4-20250514', prompt: '{{input_1}}', description: 'Claude' } },
      { id: 'agent_2', type: 'agent', config: { model: 'gemini/gemini-2.0-flash', prompt: '{{input_1}}', description: 'Gemini' } },
      { id: 'output_1', type: 'output', config: {} },
    ],
    edges: [
      { from: 'input_1', to: 'agent_1' },
      { from: 'input_1', to: 'agent_2' },
      { from: 'agent_1', to: 'output_1' },
      { from: 'agent_2', to: 'output_1' },
    ],
  },
  {
    name: 'Content Pipeline',
    description: 'Chain agents to research, draft, and refine content automatically.',
    version: 1,
    tags: ['content', 'chain'],
    nodes: [
      { id: 'input_1', type: 'input', config: { value: '' } },
      { id: 'agent_1', type: 'agent', config: { model: 'anthropic/claude-sonnet-4-20250514', system_prompt: 'You are a researcher. Gather key facts about the topic.', prompt: 'Research this topic: {{input_1}}', description: 'Researcher' } },
      { id: 'agent_2', type: 'agent', config: { model: 'anthropic/claude-sonnet-4-20250514', system_prompt: 'You are a skilled writer. Write a polished article using the research provided.', prompt: 'Write an article based on this research:\n{{agent_1}}', description: 'Writer' } },
      { id: 'output_1', type: 'output', config: {} },
    ],
    edges: [
      { from: 'input_1', to: 'agent_1' },
      { from: 'agent_1', to: 'agent_2' },
      { from: 'agent_2', to: 'output_1' },
    ],
  },
  {
    name: 'Tool-Augmented Agent',
    description: 'An agent with access to external tools for enhanced capabilities.',
    version: 1,
    tags: ['tools', 'agentic'],
    nodes: [
      { id: 'input_1', type: 'input', config: { value: '' } },
      { id: 'agent_1', type: 'agent', config: { model: 'anthropic/claude-sonnet-4-20250514', prompt: '{{input_1}}', tools: ['web_search'], max_turns: 5, description: 'Tool Agent' } },
      { id: 'output_1', type: 'output', config: {} },
    ],
    edges: [
      { from: 'input_1', to: 'agent_1' },
      { from: 'agent_1', to: 'output_1' },
    ],
  },
]

/* ── Node type colors ── */
const nodeColors: Record<string, string> = {
  input: 'var(--node-input)',
  agent: 'var(--node-agent)',
  tool: 'var(--node-tool)',
  output: 'var(--node-output)',
}

const nodeIcons: Record<string, typeof GitBranch> = {
  input: Sparkles,
  agent: Cpu,
  tool: Wrench,
  output: LayoutGrid,
}

/* ── Workflow card component ── */
function WorkflowCard({
  workflow,
  description,
  onClick,
}: {
  workflow: WorkflowDefinition
  description?: string
  onClick: () => void
}) {
  // Count nodes by type
  const typeCounts: Record<string, number> = {}
  for (const n of workflow.nodes) {
    typeCounts[n.type] = (typeCounts[n.type] || 0) + 1
  }

  return (
    <button
      onClick={onClick}
      className="group text-left w-full rounded-2xl p-5 transition-all duration-200
        bg-card/60 border border-border hover:border-foreground/20
        hover:bg-card hover:shadow-lg hover:shadow-black/10
        hover:-translate-y-0.5 cursor-pointer"
    >
      {/* Mini node graph preview */}
      <div className="flex items-center gap-1.5 mb-4">
        {workflow.nodes.slice(0, 6).map((n, i) => (
          <div key={n.id} className="flex items-center gap-1.5">
            <div
              className="w-6 h-6 rounded-lg flex items-center justify-center"
              style={{
                background: `color-mix(in oklch, ${nodeColors[n.type] || 'var(--muted)'}, transparent 80%)`,
              }}
            >
              {(() => {
                const Icon = nodeIcons[n.type]
                return Icon ? (
                  <Icon className="w-3 h-3" style={{ color: nodeColors[n.type] }} />
                ) : null
              })()}
            </div>
            {i < Math.min(workflow.nodes.length, 6) - 1 && (
              <div className="w-3 h-px bg-border" />
            )}
          </div>
        ))}
        {workflow.nodes.length > 6 && (
          <span className="text-[10px] text-muted-foreground ml-1">
            +{workflow.nodes.length - 6}
          </span>
        )}
      </div>

      {/* Name */}
      <h3 className="landing-display font-semibold text-base text-foreground group-hover:text-foreground truncate">
        {workflow.name}
      </h3>

      {/* Description */}
      {description && (
        <p className="text-sm text-muted-foreground mt-1.5 line-clamp-2 leading-relaxed">
          {description}
        </p>
      )}

      {/* Node type badges */}
      <div className="flex items-center gap-2 mt-4 flex-wrap">
        {Object.entries(typeCounts).map(([type, count]) => (
          <span
            key={type}
            className="inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-full landing-body"
            style={{
              background: `color-mix(in oklch, ${nodeColors[type] || 'var(--muted)'}, transparent 88%)`,
              color: nodeColors[type] || 'var(--muted-foreground)',
            }}
          >
            <span
              className="w-1.5 h-1.5 rounded-full"
              style={{ background: nodeColors[type] || 'var(--muted-foreground)' }}
            />
            {count} {type}
          </span>
        ))}
        <span className="text-[11px] text-muted-foreground/60 ml-auto landing-body">
          {workflow.edges.length} edge{workflow.edges.length !== 1 ? 's' : ''}
        </span>
      </div>
    </button>
  )
}

/* ── Main Dashboard ── */
export default function Landing() {
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()
  const prevThemeRef = useRef(theme)
  const [workflows, setWorkflows] = useState<WorkflowDefinition[]>([])
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

  useEffect(() => {
    listWorkflows()
      .then(setWorkflows)
      .catch(() => setWorkflows([]))
      .finally(() => setLoading(false))
  }, [])

  const openWorkflow = (wf: WorkflowDefinition) => {
    const { nodes, edges } = deserializeWorkflow(wf)
    useWorkflowStore.setState({ nodes, edges })
    useWorkflowStore.getState().setWorkflowName(wf.name)
    navigate('/editor')
  }

  const openNew = () => {
    useWorkflowStore.setState({ nodes: [], edges: [] })
    useWorkflowStore.getState().setWorkflowName('')
    navigate('/editor')
  }

  // Filter by search
  const filteredWorkflows = workflows.filter((w) =>
    w.name.toLowerCase().includes(search.toLowerCase()),
  )
  const filteredTemplates = templates.filter(
    (t) =>
      t.name.toLowerCase().includes(search.toLowerCase()) ||
      t.description.toLowerCase().includes(search.toLowerCase()) ||
      t.tags.some((tag) => tag.toLowerCase().includes(search.toLowerCase())),
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

            {/* New workflow */}
            <button
              onClick={openNew}
              className="flex items-center gap-2 px-4 py-2 rounded-xl bg-foreground text-background text-sm font-medium landing-body hover:opacity-90 transition-opacity cursor-pointer"
            >
              <Plus className="w-4 h-4" />
              <span className="hidden sm:inline">New Workflow</span>
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

                {filteredWorkflows.map((wf) => (
                  <WorkflowCard
                    key={wf.name}
                    workflow={wf}
                    onClick={() => openWorkflow(wf)}
                  />
                ))}
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
                  Create your first AI workflow or start from a template below.
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

          {/* ─── Templates ─── */}
          <section>
            <div className="flex items-center justify-between mb-6">
              <h2 className="landing-display text-xl font-bold tracking-tight">
                Templates
              </h2>
              <span className="text-xs text-muted-foreground landing-body">
                {filteredTemplates.length} template{filteredTemplates.length !== 1 ? 's' : ''}
              </span>
            </div>

            {filteredTemplates.length > 0 ? (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {filteredTemplates.map((t) => (
                  <WorkflowCard
                    key={t.name}
                    workflow={t}
                    description={t.description}
                    onClick={() => openWorkflow(t)}
                  />
                ))}
              </div>
            ) : (
              <div className="text-center py-12">
                <p className="text-muted-foreground text-sm landing-body">
                  No templates matching "{search}"
                </p>
              </div>
            )}
          </section>
        </main>
    </div>
  )
}
