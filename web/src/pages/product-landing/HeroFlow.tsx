import { Fragment } from 'react'
import { Inbox, Bot, ArrowRightFromLine } from 'lucide-react'
import type { ComponentType } from 'react'

type FlowNode = {
  type: 'input' | 'agent' | 'output'
  label: string
  icon: ComponentType<{ className?: string }>
  delay: string
  glowDelay: string
  bodyContent: React.ReactNode
}

const NODES: FlowNode[] = [
  {
    type: 'input',
    label: 'User Query',
    icon: Inbox,
    delay: '0.3s',
    glowDelay: '0.9s',
    bodyContent: (
      <p className="text-[11px] text-muted-foreground leading-relaxed landing-body">
        "Summarize this article and extract key action items…"
      </p>
    ),
  },
  {
    type: 'agent',
    label: 'AI Agent',
    icon: Bot,
    delay: '0.5s',
    glowDelay: '1.4s',
    bodyContent: (
      <div className="space-y-1.5">
        <div className="flex items-center gap-1.5">
          <span className="run-pulse w-1.5 h-1.5 rounded-full bg-node-agent inline-block" />
          <span className="text-[10px] font-medium text-node-agent landing-body">Running</span>
        </div>
        <div className="text-[10px] text-muted-foreground landing-body">claude-sonnet-4-6</div>
      </div>
    ),
  },
  {
    type: 'output',
    label: 'Result',
    icon: ArrowRightFromLine,
    delay: '0.7s',
    glowDelay: '1.9s',
    bodyContent: (
      <div className="space-y-1.5">
        <div className="h-2 w-full rounded bg-muted/40" />
        <div className="h-2 w-4/5 rounded bg-muted/30" />
        <div className="h-2 w-3/5 rounded bg-muted/20" />
      </div>
    ),
  },
]

const colorMap = {
  input:  { bg: 'bg-node-input/15',  accent: 'bg-node-input',  fg: 'text-node-input-foreground',  glow: 'oklch(0.795 0.184 86.047 / 25%)' },
  agent:  { bg: 'bg-node-agent/15',  accent: 'bg-node-agent',  fg: 'text-node-agent-foreground',  glow: 'oklch(0.588 0.158 241.966 / 25%)' },
  output: { bg: 'bg-node-output/15', accent: 'bg-node-output', fg: 'text-node-output-foreground', glow: 'oklch(0.648 0.2 142.495 / 25%)' },
}

export function HeroFlow() {
  return (
    <div className="flex flex-col sm:flex-row items-center justify-center mt-16 sm:mt-20 gap-0 w-full">
      {NODES.map((node, i) => {
        const colors = colorMap[node.type]
        const Icon = node.icon
        return (
          <Fragment key={node.type}>
            {/* Node card */}
            <div
              className="hero-node hero-glow rounded-xl bg-card border border-border w-[172px] sm:w-[196px] overflow-hidden"
              style={{
                animationDelay: `${node.delay}, ${node.glowDelay}`,
                '--hero-glow': colors.glow,
              } as React.CSSProperties}
            >
              {/* Colored header */}
              <div className={`flex items-center gap-2 px-3 py-2.5 ${colors.bg}`}>
                <div className={`h-6 w-6 rounded-md flex items-center justify-center shrink-0 ${colors.accent} ${colors.fg}`}>
                  <Icon className="h-3 w-3" />
                </div>
                <span className="text-xs font-semibold text-card-foreground landing-display truncate">
                  {node.label}
                </span>
              </div>
              {/* Rich body */}
              <div className="px-3 py-3">
                {node.bodyContent}
              </div>
            </div>

            {/* Edge connector */}
            {i < NODES.length - 1 && (
              <>
                <div
                  className="hero-edge hidden sm:block h-[2px] w-[56px] shrink-0"
                  style={{ animationDelay: `${0.4 + i * 0.2}s` }}
                />
                <div
                  className="hero-edge-v block sm:hidden w-[2px] h-[40px] shrink-0"
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
