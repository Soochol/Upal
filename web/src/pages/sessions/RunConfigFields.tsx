import { useState } from 'react'
import { Search, Sparkles, Plus, Trash2 } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import type { SessionSource, SessionWorkflow, SessionContext } from '@/entities/session/types'
import { SCHEDULE_PRESETS } from '@/entities/session/constants'
import { WorkflowPicker } from '@/features/workflow-picker'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'

interface RunConfigFieldsProps {
  name: string
  onNameChange: (name: string) => void
  sources: SessionSource[]
  onSourcesChange: (sources: SessionSource[]) => void
  workflows: SessionWorkflow[]
  onWorkflowsChange: (workflows: SessionWorkflow[]) => void
  context: SessionContext
  onContextChange: (context: SessionContext) => void
  schedule: string
  onScheduleChange: (schedule: string) => void
  namePlaceholder?: string
}

export function RunConfigFields({
  name,
  onNameChange,
  sources,
  onSourcesChange,
  workflows,
  onWorkflowsChange,
  context,
  onContextChange,
  schedule,
  onScheduleChange,
  namePlaceholder = 'Run name',
}: RunConfigFieldsProps) {
  const [showSourceModal, setShowSourceModal] = useState(false)
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)

  const isPreset = SCHEDULE_PRESETS.some(p => p.cron === schedule)

  const scheduleValue = isPreset ? schedule : schedule ? '__custom__' : ''

  return (
    <>
      <div className="space-y-1">
        <label className="text-xs font-medium text-muted-foreground">Name</label>
        <input
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          placeholder={namePlaceholder}
          className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
        />
      </div>

      <div className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground flex items-center gap-1.5">
          <Search className="h-3.5 w-3.5" />
          Research
        </h3>
        <div className="flex gap-2">
          {(['light', 'deep'] as const).map((mode) => (
            <button
              key={mode}
              onClick={() => onContextChange({ ...context, research_depth: mode })}
              className={cn(
                'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer',
                context.research_depth === mode
                  ? 'bg-foreground text-background'
                  : 'text-muted-foreground hover:bg-muted/50',
              )}
            >
              {mode === 'light' ? 'Search' : 'Deep Research'}
            </button>
          ))}
        </div>
        {sources.length > 0 && (
          <div className="space-y-1">
            {sources.map((s, i) => (
              <div key={s.id || i} className="flex items-center justify-between px-3 py-1.5 rounded-lg bg-muted/30 text-xs">
                <span>{s.label || s.type}</span>
                <button
                  onClick={() => onSourcesChange(sources.filter((_, idx) => idx !== i))}
                  className="p-0.5 text-muted-foreground hover:text-destructive cursor-pointer"
                >
                  <Trash2 className="h-3 w-3" />
                </button>
              </div>
            ))}
          </div>
        )}
        <button
          onClick={() => setShowSourceModal(true)}
          className="text-xs text-primary hover:underline cursor-pointer"
        >
          + Add source
        </button>
      </div>

      <div className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground flex items-center gap-1.5">
          <Sparkles className="h-3.5 w-3.5" />
          Task
        </h3>
        <textarea
          value={context.prompt ?? ''}
          onChange={(e) => onContextChange({ ...context, prompt: e.target.value })}
          placeholder="What should this run focus on?"
          className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring resize-none"
          rows={3}
        />
      </div>

      <div className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground">Processing</h3>
        {workflows.map((wf, i) => (
          <div key={wf.workflow_name} className="flex items-center justify-between px-3 py-1.5 rounded-lg bg-muted/30 text-xs">
            <span>{wf.label || wf.workflow_name}</span>
            <button
              onClick={() => onWorkflowsChange(workflows.filter((_, idx) => idx !== i))}
              className="p-0.5 text-muted-foreground hover:text-destructive cursor-pointer"
            >
              <Trash2 className="h-3 w-3" />
            </button>
          </div>
        ))}
        <button
          onClick={() => setShowWorkflowPicker(true)}
          className="flex items-center gap-1 text-xs text-primary hover:underline cursor-pointer"
        >
          <Plus className="h-3 w-3" /> Add workflow
        </button>
      </div>

      <div className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground">Schedule</h3>
        <select
          value={scheduleValue}
          onChange={(e) => onScheduleChange(e.target.value === '__custom__' ? '' : e.target.value)}
          className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
        >
          <option value="">No schedule</option>
          {SCHEDULE_PRESETS.map((p) => (
            <option key={p.cron} value={p.cron}>{p.label}</option>
          ))}
          <option value="__custom__">Custom</option>
        </select>
        {(!isPreset && schedule !== '') && (
          <input
            value={schedule}
            onChange={(e) => onScheduleChange(e.target.value)}
            placeholder="Custom cron: 0 */6 * * *"
            className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm font-mono outline-none focus:ring-1 focus:ring-ring"
          />
        )}
      </div>

      {showSourceModal && (
        <AddSourceModal
          onAdd={(source) => {
            onSourcesChange([...sources, { ...source, id: source.id || crypto.randomUUID() }])
            setShowSourceModal(false)
          }}
          onClose={() => setShowSourceModal(false)}
        />
      )}
      {showWorkflowPicker && (
        <WorkflowPicker
          existingWorkflows={workflows}
          onAdd={(wf) => {
            onWorkflowsChange([...workflows, wf])
            setShowWorkflowPicker(false)
          }}
          onClose={() => setShowWorkflowPicker(false)}
        />
      )}
    </>
  )
}
