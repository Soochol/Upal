import { useState, useRef, useEffect } from 'react'
import { ChevronDown, Check, Lock } from 'lucide-react'

export type StageState = 'completed' | 'active' | 'locked'

interface StageSectionProps {
  id: string
  number: number
  title: string
  state: StageState
  summary?: string
  statusText?: string
  children: React.ReactNode
}

export function StageSection({ id, number, title, state, summary, statusText, children }: StageSectionProps) {
  const [expanded, setExpanded] = useState(state === 'active')
  const ref = useRef<HTMLElement>(null)

  useEffect(() => {
    if (state === 'active') setExpanded(true)
  }, [state])

  const isVisible = state !== 'locked' && expanded

  return (
    <section id={id} ref={ref} className="mb-6">
      <button
        onClick={() => state !== 'locked' && setExpanded(v => !v)}
        disabled={state === 'locked'}
        className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl border transition-all cursor-pointer
          ${state === 'active' ? 'border-foreground/20 bg-foreground/5'
            : state === 'completed' ? 'border-success/20 bg-success/5 hover:bg-success/10'
            : 'border-border/50 bg-muted/20 opacity-50 cursor-not-allowed'}`}
      >
        <div className={`flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold shrink-0
          ${state === 'completed' ? 'bg-success text-white'
            : state === 'active' ? 'bg-foreground text-background'
            : 'bg-muted text-muted-foreground'}`}>
          {state === 'completed' ? <Check className="h-3.5 w-3.5" /> : number}
        </div>
        <span className={`text-sm font-semibold ${state === 'locked' ? 'text-muted-foreground' : 'text-foreground'}`}>
          {title}
        </span>
        {state === 'completed' && !expanded && summary && (
          <span className="text-sm text-muted-foreground truncate flex-1 text-left ml-1">{summary}</span>
        )}
        <div className="flex-1" />
        {statusText && <span className="text-sm text-muted-foreground shrink-0">{statusText}</span>}
        {state === 'locked'
          ? <Lock className="h-3.5 w-3.5 text-muted-foreground" />
          : <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${isVisible ? 'rotate-180' : ''}`} />}
      </button>
      {isVisible && (
        <div className="mt-3 px-1 animate-in fade-in slide-in-from-top-2 duration-200">
          {children}
        </div>
      )}
    </section>
  )
}
