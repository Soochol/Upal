import { useEffect, useRef } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Plus, Sparkles, GitBranch, Play, CalendarClock, ArrowRight } from 'lucide-react'
import { useTheme } from '@/components/ThemeProvider'
import { useWorkflowStore } from '@/stores/workflowStore'
import { HeroFlow } from './product-landing/HeroFlow'

export default function ProductLanding() {
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()
  const prevThemeRef = useRef(theme)

  // Force dark theme on landing page
  useEffect(() => {
    prevThemeRef.current = theme
    if (theme !== 'dark') setTheme('dark')
    return () => {
      if (prevThemeRef.current !== 'dark') setTheme(prevThemeRef.current)
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openNew = () => {
    useWorkflowStore.setState({ nodes: [], edges: [] })
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    useWorkflowStore.getState().setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
    navigate('/editor')
  }

  return (
    <div className="bg-background text-foreground min-h-screen landing-body">
      {/* ═══ NAV ═══ */}
      <nav className="landing-nav sticky top-0 z-50 border-b border-border">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-foreground flex items-center justify-center">
              <span className="text-background text-sm font-bold landing-display">U</span>
            </div>
            <span className="text-lg font-bold landing-display tracking-tight">Upal</span>
          </div>

          <div className="flex items-center gap-1">
            <Link
              to="/workflows"
              className="px-3 py-2 rounded-xl text-sm text-muted-foreground hover:text-foreground hover:bg-card/60 transition-all landing-body"
            >
              Workflows
            </Link>
            <Link
              to="/runs"
              className="px-3 py-2 rounded-xl text-sm text-muted-foreground hover:text-foreground hover:bg-card/60 transition-all landing-body hidden sm:block"
            >
              Runs
            </Link>
            <Link
              to="/schedules"
              className="px-3 py-2 rounded-xl text-sm text-muted-foreground hover:text-foreground hover:bg-card/60 transition-all landing-body hidden sm:block"
            >
              Schedules
            </Link>
          </div>
        </div>
      </nav>

      {/* ═══ HERO ═══ */}
      <section className="max-w-5xl mx-auto px-4 sm:px-6 pt-24 sm:pt-32 pb-20 sm:pb-28 text-center">
        <h1 className="landing-display text-5xl sm:text-6xl lg:text-7xl font-extrabold tracking-tight leading-[1.05]">
          Build AI workflows{' '}
          <span className="hero-gradient-text">visually.</span>
        </h1>

        <p className="landing-body text-lg sm:text-xl text-muted-foreground max-w-xl mx-auto mt-6 leading-relaxed">
          Connect models, tools, and data in a visual DAG editor.
          Ship multi-step AI pipelines without code.
        </p>

        {/* CTAs */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-3 mt-10">
          <button
            onClick={openNew}
            className="flex items-center gap-2 px-6 py-3 rounded-xl bg-foreground text-background text-sm font-medium landing-body hover:opacity-90 transition-opacity cursor-pointer"
          >
            <Plus className="w-4 h-4" />
            Create Workflow
          </button>
          <button
            onClick={() => navigate('/editor?generate=true')}
            className="flex items-center gap-2 px-6 py-3 rounded-xl border border-border text-sm font-medium landing-body text-foreground hover:bg-card/60 transition-all cursor-pointer"
          >
            <Sparkles className="w-4 h-4" />
            Generate with AI
          </button>
        </div>

        {/* Animated Flow Visualization */}
        <HeroFlow />
      </section>

      {/* ═══ FEATURE CARDS ═══ */}
      <section className="max-w-5xl mx-auto px-4 sm:px-6 pb-20 sm:pb-28">
        <h2 className="landing-display text-2xl sm:text-3xl font-bold tracking-tight text-center mb-12">
          Everything you need
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {/* Workflows */}
          <Link
            to="/workflows"
            className="group rounded-2xl p-6 bg-card/60 border border-border
              hover:border-node-agent/40 hover:bg-card hover:shadow-lg hover:shadow-node-agent/5
              hover:-translate-y-1 transition-all duration-200"
          >
            <div className="w-10 h-10 rounded-xl bg-node-agent/15 flex items-center justify-center mb-4">
              <GitBranch className="w-5 h-5 text-node-agent" />
            </div>
            <h3 className="landing-display text-lg font-semibold mb-2">Workflows</h3>
            <p className="text-sm text-muted-foreground leading-relaxed mb-4">
              Design multi-step AI pipelines with a drag-and-drop visual editor. Connect inputs, agents, and outputs.
            </p>
            <span className="inline-flex items-center gap-1 text-sm text-node-agent font-medium landing-body">
              Browse Workflows
              <ArrowRight className="w-3.5 h-3.5 group-hover:translate-x-0.5 transition-transform" />
            </span>
          </Link>

          {/* Runs */}
          <Link
            to="/runs"
            className="group rounded-2xl p-6 bg-card/60 border border-border
              hover:border-node-output/40 hover:bg-card hover:shadow-lg hover:shadow-node-output/5
              hover:-translate-y-1 transition-all duration-200"
          >
            <div className="w-10 h-10 rounded-xl bg-node-output/15 flex items-center justify-center mb-4">
              <Play className="w-5 h-5 text-node-output" />
            </div>
            <h3 className="landing-display text-lg font-semibold mb-2">Runs</h3>
            <p className="text-sm text-muted-foreground leading-relaxed mb-4">
              Execute workflows with real-time streaming. Watch each node process live with SSE events.
            </p>
            <span className="inline-flex items-center gap-1 text-sm text-node-output font-medium landing-body">
              View Run History
              <ArrowRight className="w-3.5 h-3.5 group-hover:translate-x-0.5 transition-transform" />
            </span>
          </Link>

          {/* Schedules */}
          <Link
            to="/schedules"
            className="group rounded-2xl p-6 bg-card/60 border border-border
              hover:border-node-input/40 hover:bg-card hover:shadow-lg hover:shadow-node-input/5
              hover:-translate-y-1 transition-all duration-200"
          >
            <div className="w-10 h-10 rounded-xl bg-node-input/15 flex items-center justify-center mb-4">
              <CalendarClock className="w-5 h-5 text-node-input" />
            </div>
            <h3 className="landing-display text-lg font-semibold mb-2">Schedules</h3>
            <p className="text-sm text-muted-foreground leading-relaxed mb-4">
              Automate with cron schedules and webhook triggers. Let your AI pipelines run autonomously.
            </p>
            <span className="inline-flex items-center gap-1 text-sm text-node-input font-medium landing-body">
              Manage Schedules
              <ArrowRight className="w-3.5 h-3.5 group-hover:translate-x-0.5 transition-transform" />
            </span>
          </Link>
        </div>
      </section>

      {/* ═══ FOOTER ═══ */}
      <footer className="border-t border-border py-8">
        <p className="text-center text-xs text-muted-foreground/60 landing-body">
          Built with Upal
        </p>
      </footer>
    </div>
  )
}
