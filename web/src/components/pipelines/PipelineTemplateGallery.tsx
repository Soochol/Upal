import { useState } from 'react'
import { Play, PauseCircle, Clock, Zap, RefreshCw, GitBranch, Loader2 } from 'lucide-react'
import { createPipeline, saveWorkflow } from '@/lib/api'
import { ApiError } from '@/lib/api/client'
import type { Pipeline } from '@/lib/api/types'
import { PIPELINE_TEMPLATES } from './pipelineTemplates'

const stageTypeIcons: Record<string, typeof GitBranch> = {
  workflow:  Play,
  approval:  PauseCircle,
  schedule:  Clock,
  trigger:   Zap,
  transform: RefreshCw,
}

const stageTypeColors: Record<string, string> = {
  workflow:  'var(--info)',
  approval:  'var(--warning)',
  schedule:  'var(--success)',
  trigger:   'var(--node-agent)',
  transform: 'var(--muted-foreground)',
}

type Props = {
  onCreated: (pipeline: Pipeline) => void
}

export function PipelineTemplateGallery({ onCreated }: Props) {
  const [loadingIndex, setLoadingIndex] = useState<number | null>(null)

  const handleUse = async (index: number) => {
    const tpl = PIPELINE_TEMPLATES[index]
    setLoadingIndex(index)
    try {
      await Promise.all(
        tpl.workflows.map(async (wf) => {
          try {
            await saveWorkflow(wf)
          } catch (err) {
            if (err instanceof ApiError && err.status === 409) return
            throw err
          }
        }),
      )
      const stages = tpl.stages.map((s, i) => ({ ...s, id: `stage-${i + 1}` }))
      const pipeline = await createPipeline({
        name: tpl.name,
        description: tpl.description,
        stages,
      })
      onCreated(pipeline)
    } finally {
      setLoadingIndex(null)
    }
  }

  return (
    <div className="space-y-4">
      <p className="text-xs font-semibold text-muted-foreground uppercase tracking-widest landing-display">
        Start from a template
      </p>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {PIPELINE_TEMPLATES.map((tpl, i) => {
          const isLoading = loadingIndex === i
          return (
            <button
              key={tpl.name}
              onClick={() => handleUse(i)}
              disabled={loadingIndex !== null}
              className="group text-left rounded-2xl border border-border bg-card p-4
                hover:border-foreground/20 hover:shadow-md hover:shadow-black/5 hover:-translate-y-0.5
                transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0"
            >
              {/* Emoji + use badge */}
              <div className="flex items-start justify-between mb-3">
                <span className="text-2xl leading-none">{tpl.emoji}</span>
                {isLoading ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground mt-0.5" />
                ) : (
                  <span className="text-[10px] font-medium text-muted-foreground border border-border
                    rounded px-1.5 py-0.5 opacity-0 group-hover:opacity-100 transition-opacity landing-body">
                    Use
                  </span>
                )}
              </div>

              {/* Name + description */}
              <p className="landing-display text-sm font-semibold mb-1">{tpl.name}</p>
              <p className="text-xs text-muted-foreground mb-3 leading-relaxed line-clamp-2">{tpl.description}</p>

              {/* Stage flow */}
              <div className="flex flex-wrap items-center gap-1">
                {tpl.stages.map((stage, j) => {
                  const Icon = stageTypeIcons[stage.type] || GitBranch
                  const color = stageTypeColors[stage.type] || 'var(--muted-foreground)'
                  return (
                    <span key={j} className="flex items-center gap-1">
                      {j > 0 && <span className="text-border text-[10px]">→</span>}
                      <span
                        className="flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded-md landing-body font-medium"
                        style={{
                          background: `color-mix(in oklch, ${color}, transparent 88%)`,
                          color,
                        }}
                      >
                        <Icon className="h-2.5 w-2.5" />
                        {stage.name}
                      </span>
                    </span>
                  )
                })}
              </div>

              {tpl.workflows.length > 0 && (
                <p className="text-[10px] text-muted-foreground/50 mt-2 landing-body">
                  {tpl.workflows.length} workflow{tpl.workflows.length !== 1 ? 's' : ''} auto-created
                </p>
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}
