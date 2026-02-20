import { Fragment } from 'react'
import { Inbox, Bot, ArrowRightFromLine } from 'lucide-react'
import type { ComponentType } from 'react'

type FlowNode = {
  type: 'input' | 'agent' | 'output'
  label: string
  icon: ComponentType<{ className?: string }>
  delay: string
  glowDelay: string
}

const NODES: FlowNode[] = [
  { type: 'input', label: 'User Query', icon: Inbox, delay: '0.3s', glowDelay: '0.9s' },
  { type: 'agent', label: 'AI Agent', icon: Bot, delay: '0.5s', glowDelay: '1.4s' },
  { type: 'output', label: 'Result', icon: ArrowRightFromLine, delay: '0.7s', glowDelay: '1.9s' },
]

const colorMap = {
  input: { bg: 'bg-node-input/15', accent: 'bg-node-input', fg: 'text-node-input-foreground', glow: 'oklch(0.795 0.184 86.047 / 25%)' },
  agent: { bg: 'bg-node-agent/15', accent: 'bg-node-agent', fg: 'text-node-agent-foreground', glow: 'oklch(0.588 0.158 241.966 / 25%)' },
  output: { bg: 'bg-node-output/15', accent: 'bg-node-output', fg: 'text-node-output-foreground', glow: 'oklch(0.648 0.2 142.495 / 25%)' },
}

export function HeroFlow() {
  return (
    <div className="flex flex-col sm:flex-row items-center justify-center mt-16 sm:mt-20 gap-0">
      {NODES.map((node, i) => {
        const colors = colorMap[node.type]
        const Icon = node.icon
        return (
          <Fragment key={node.type}>
            {/* Node card */}
            <div
              className="hero-node hero-glow rounded-xl bg-card border border-border w-[160px] sm:w-[180px] overflow-hidden"
              style={{
                animationDelay: `${node.delay}, ${node.glowDelay}`,
                '--hero-glow': colors.glow,
              } as React.CSSProperties}
            >
              {/* Colored header */}
              <div className={`flex items-center gap-2 px-3 py-2.5 ${colors.bg}`}>
                <div className={`h-6 w-6 rounded-md flex items-center justify-center ${colors.accent} ${colors.fg}`}>
                  <Icon className="h-3 w-3" />
                </div>
                <span className="text-xs font-semibold text-card-foreground landing-display">
                  {node.label}
                </span>
              </div>
              {/* Body placeholder lines */}
              <div className="px-3 py-3 space-y-1.5">
                <div className="h-2 w-full rounded bg-muted/30" />
                <div className="h-2 w-2/3 rounded bg-muted/20" />
              </div>
            </div>

            {/* Edge connector */}
            {i < NODES.length - 1 && (
              <>
                {/* Horizontal edge (sm+) */}
                <div
                  className="hero-edge hidden sm:block h-[2px] w-[60px]"
                  style={{ animationDelay: `${0.4 + i * 0.2}s` }}
                />
                {/* Vertical edge (mobile) */}
                <div
                  className="hero-edge-v block sm:hidden w-[2px] h-[40px]"
                  style={{ animationDelay: `${0.4 + i * 0.2}s` }}
                />
              </>
            )}
          </Fragment>
        )
      })}
    </div>
  )
}
