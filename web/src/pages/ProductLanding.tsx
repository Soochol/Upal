import { useNavigate, Link } from 'react-router-dom'
import { Plus, Sparkles, Network, Layers, Zap } from 'lucide-react'
import { useWorkflowStore } from '@/entities/workflow'
import { ThemeToggle } from '@/components/ThemeToggle'
import { HeroFlow } from './product-landing/HeroFlow'

const HOW_IT_WORKS = [
  {
    step: '01',
    title: 'Add nodes',
    desc: 'Pick from inputs, agents, branches, and outputs. Each node type has a clear purpose in the flow.',
  },
  {
    step: '02',
    title: 'Connect them',
    desc: 'Draw edges between nodes to define how data moves. Branch for parallel paths, merge to combine.',
  },
  {
    step: '03',
    title: 'Run and watch',
    desc: 'Execute the workflow and watch each node run in real-time. Results stream back as they arrive.',
  },
]

const FEATURES = [
  {
    icon: Network,
    color: 'text-node-agent',
    bg: 'bg-node-agent/10',
    title: 'Multi-model support',
    desc: 'Mix Claude, Gemini, GPT-4, and Ollama in a single graph. Each node picks its own model.',
  },
  {
    icon: Layers,
    color: 'text-node-input',
    bg: 'bg-node-input/10',
    title: 'Visual DAG editor',
    desc: 'Drag, drop, and connect — no YAML, no code. The graph is the source of truth.',
  },
  {
    icon: Zap,
    color: 'text-node-output',
    bg: 'bg-node-output/10',
    title: 'Streaming execution',
    desc: 'Watch nodes execute in real-time via SSE. See token-by-token output as agents think.',
  },
]

export default function ProductLanding() {
  const navigate = useNavigate()

  const openNew = () => {
    useWorkflowStore.setState({ nodes: [], edges: [] })
    const name = `Untitled-${Date.now().toString(36).slice(-4)}`
    useWorkflowStore.getState().setWorkflowName(name)
    useWorkflowStore.getState().setOriginalName('')
    navigate('/editor')
  }

  const CTAButtons = ({ inverted = false }: { inverted?: boolean }) => (
    <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
      <button
        onClick={openNew}
        className={`flex items-center gap-2 px-6 py-3 rounded-xl text-sm font-medium landing-body hover:opacity-90 transition-opacity cursor-pointer ${
          inverted
            ? 'bg-background text-foreground'
            : 'bg-foreground text-background'
        }`}
      >
        <Plus className="w-4 h-4" />
        Create Workflow
      </button>
      <button
        onClick={() => navigate('/editor?generate=true')}
        className={`flex items-center gap-2 px-6 py-3 rounded-xl text-sm font-medium landing-body transition-all cursor-pointer ${
          inverted
            ? 'border border-background/30 text-background hover:bg-background/10'
            : 'border border-border text-foreground hover:bg-card/60'
        }`}
      >
        <Sparkles className="w-4 h-4" />
        Generate with AI
      </button>
    </div>
  )

  return (
    <div className="bg-background text-foreground min-h-screen flex flex-col landing-body">

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
      <section className="px-4 sm:px-6 py-6 sm:py-8">
        <div className="hero-card relative rounded-3xl border border-border overflow-hidden w-full max-w-5xl mx-auto px-8 sm:px-14 pt-14 sm:pt-20 pb-0">
          {/* Dot-grid overlay */}
          <div className="hero-dot-grid absolute inset-0 pointer-events-none" />

          {/* Text + CTAs */}
          <div className="relative text-center">
            <h1 className="landing-display font-extrabold tracking-tight leading-[1.05]">
              <span className="block text-4xl sm:text-5xl lg:text-6xl text-foreground/80">
                Build AI workflows
              </span>
              <span className="block text-5xl sm:text-7xl lg:text-8xl hero-gradient-text mt-1">
                visually.
              </span>
            </h1>

            <p className="landing-body text-base sm:text-lg text-muted-foreground max-w-lg mx-auto mt-6 leading-relaxed">
              Connect models, tools, and data in a visual DAG editor.
              Ship multi-step AI pipelines without code.
            </p>

            <div className="mt-10">
              <CTAButtons />
            </div>
          </div>

          {/* Animated Flow Visualization */}
          <div className="relative flex items-end mt-4">
            <HeroFlow />
          </div>
        </div>
      </section>

      {/* ═══ HOW IT WORKS ═══ */}
      <section className="border-y border-border bg-card/40 px-4 sm:px-6 py-16 sm:py-20">
        <div className="max-w-5xl mx-auto">
          <h2 className="landing-display text-2xl sm:text-3xl font-bold tracking-tight text-center mb-12">
            How it works
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-8 sm:gap-12">
            {HOW_IT_WORKS.map(({ step, title, desc }) => (
              <div key={step} className="flex flex-col gap-4">
                <div className="landing-step-number">{step}</div>
                <div>
                  <h3 className="landing-display text-lg font-bold mb-2">{title}</h3>
                  <p className="text-sm text-muted-foreground leading-relaxed landing-body">{desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ═══ FEATURES ═══ */}
      <section className="px-4 sm:px-6 py-16 sm:py-20">
        <div className="max-w-5xl mx-auto">
          <h2 className="landing-display text-2xl sm:text-3xl font-bold tracking-tight text-center mb-12">
            Everything you need
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-5">
            {FEATURES.map(({ icon: Icon, color, bg, title, desc }) => (
              <div
                key={title}
                className="landing-feature-card rounded-2xl border border-border bg-card p-6"
              >
                <div className={`w-10 h-10 rounded-xl ${bg} flex items-center justify-center mb-4`}>
                  <Icon className={`w-5 h-5 ${color}`} />
                </div>
                <h3 className="landing-display text-base font-bold mb-2">{title}</h3>
                <p className="text-sm text-muted-foreground leading-relaxed landing-body">{desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ═══ CTA FOOTER ═══ */}
      <section className="mt-auto border-t border-border bg-foreground text-background px-4 sm:px-6 py-16 sm:py-20">
        <div className="max-w-5xl mx-auto text-center">
          <h2 className="landing-display text-3xl sm:text-4xl font-extrabold tracking-tight mb-4">
            Start building in seconds.
          </h2>
          <p className="landing-body text-sm opacity-60 mb-10 max-w-sm mx-auto">
            No sign-up required. Open the editor and create your first workflow immediately.
          </p>
          <CTAButtons inverted />
        </div>
      </section>

    </div>
  )
}
