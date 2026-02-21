import { useNavigate, Link } from 'react-router-dom'
import { Plus, Sparkles } from 'lucide-react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { ThemeToggle } from '@/components/ThemeToggle'
import { HeroFlow } from './product-landing/HeroFlow'

export default function ProductLanding() {
  const navigate = useNavigate()

  const openNew = () => {
    useWorkflowStore.setState({ nodes: [], edges: [] })
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    useWorkflowStore.getState().setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
    navigate('/editor')
  }

  return (
    <div className="bg-background text-foreground h-screen flex flex-col landing-body">
      {/* ═══ NAV ═══ */}
      <nav className="landing-nav sticky top-0 z-50 border-b border-border">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-foreground flex items-center justify-center">
              <span className="text-background text-sm font-bold landing-display">U</span>
            </div>
            <span className="text-lg font-bold landing-display tracking-tight">Upal</span>
          </div>

          <div className="flex items-center gap-2">
            <Link
              to="/workflows"
              className="px-3 py-2 rounded-xl text-sm text-muted-foreground hover:text-foreground hover:bg-card/60 transition-all landing-body"
            >
              Workflows
            </Link>
            <ThemeToggle />
          </div>
        </div>
      </nav>

      {/* ═══ HERO ═══ */}
      <section className="flex-1 flex items-stretch px-4 sm:px-6 py-6 sm:py-8">
        <div className="hero-card rounded-3xl border border-border overflow-hidden w-full max-w-5xl mx-auto px-8 sm:px-14 pt-14 sm:pt-20 pb-0 flex flex-col justify-between">
          {/* Text + CTAs */}
          <div className="text-center">
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
          </div>

          {/* Animated Flow Visualization — fills card bottom */}
          <div className="flex-1 flex items-end">
            <HeroFlow />
          </div>
        </div>
      </section>
    </div>
  )
}
