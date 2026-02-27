import { useState, useMemo } from 'react'
import { X, Search, Sparkles, Plus, Trash2 } from 'lucide-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { cn } from '@/shared/lib/utils'
import type { Run } from '@/entities/session-run'
import type { SessionSource, SessionWorkflow, SessionContext } from '@/entities/session/types'
import { fetchRun, updateRunConfig } from '@/entities/session-run/api'
import { useAutoSave } from '@/shared/hooks/useAutoSave'
import { WorkflowPicker } from '@/features/workflow-picker'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { StatusBadge } from '@/shared/ui/StatusBadge'

const SCHEDULE_PRESETS = [
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Daily', value: '0 8 * * *' },
  { label: 'Weekly', value: '0 8 * * MON' },
]

interface RunConfigPanelProps {
  runId: string
  onClose: () => void
}

/** Outer: fetches data, renders form with key to force remount on data change */
export function RunConfigPanel({ runId, onClose }: RunConfigPanelProps) {
  const { data: run } = useQuery({
    queryKey: ['session-run', runId],
    queryFn: () => fetchRun(runId),
  })

  if (!run) return null

  // key = server fingerprint → remount form when server data changes
  const fp = JSON.stringify([run.name, run.run_sources, run.run_workflows, run.context, run.schedule])
  return <RunConfigForm key={fp} run={run} onClose={onClose} />
}

/** Inner: pure form, initialized from run props. Remounted on server data change via key. */
function RunConfigForm({ run, onClose }: { run: Run; onClose: () => void }) {
  const qc = useQueryClient()

  const [name, setName] = useState(run.name ?? '')
  const [sources, setSources] = useState<SessionSource[]>(run.run_sources ?? [])
  const [workflows, setWorkflows] = useState<SessionWorkflow[]>(run.run_workflows ?? [])
  const [context, setContext] = useState<SessionContext>(run.context ?? { research_depth: 'deep' })
  const [schedule, setSchedule] = useState(run.schedule ?? '')
  const [showSourceModal, setShowSourceModal] = useState(false)
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)

  const settingsData = useMemo(() => ({
    name: name || undefined,
    sources,
    workflows,
    context,
    schedule: schedule || undefined,
  }), [name, sources, workflows, context, schedule])

  useAutoSave({
    data: settingsData,
    onSave: async (data) => {
      await updateRunConfig(run.id, data)
      qc.invalidateQueries({ queryKey: ['session-runs'] })
    },
    delay: 2000,
    saveOnUnmount: true,
    onError: (err) => console.error('Auto-save failed:', err),
  })

  const isPreset = SCHEDULE_PRESETS.some(p => p.value === schedule)

  return (
    <div className="flex flex-col h-full border-l border-border/50 bg-background">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border/50 shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm font-semibold truncate">{run.name || `Run #${run.run_number ?? ''}`}</span>
          <StatusBadge status={run.status} />
        </div>
        <button onClick={onClose} className="p-1 rounded-lg hover:bg-muted/50 text-muted-foreground cursor-pointer">
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-5">
        {/* Name */}
        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Name</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Run name"
            className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
          />
        </div>

        {/* Research */}
        <div className="space-y-2">
          <h3 className="text-xs font-semibold text-muted-foreground flex items-center gap-1.5">
            <Search className="h-3.5 w-3.5" />
            Research
          </h3>
          <div className="flex gap-2">
            {(['light', 'deep'] as const).map((mode) => (
              <button
                key={mode}
                onClick={() => setContext({ ...context, research_depth: mode })}
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
                    onClick={() => setSources(sources.filter((_, idx) => idx !== i))}
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

        {/* Task */}
        <div className="space-y-2">
          <h3 className="text-xs font-semibold text-muted-foreground flex items-center gap-1.5">
            <Sparkles className="h-3.5 w-3.5" />
            Task
          </h3>
          <textarea
            value={context.prompt ?? ''}
            onChange={(e) => setContext({ ...context, prompt: e.target.value })}
            placeholder="What should this run focus on?"
            className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring resize-none"
            rows={3}
          />
        </div>

        {/* Processing */}
        <div className="space-y-2">
          <h3 className="text-xs font-semibold text-muted-foreground">Processing</h3>
          {workflows.map((wf, i) => (
            <div key={wf.workflow_name} className="flex items-center justify-between px-3 py-1.5 rounded-lg bg-muted/30 text-xs">
              <span>{wf.label || wf.workflow_name}</span>
              <button
                onClick={() => setWorkflows(workflows.filter((_, idx) => idx !== i))}
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

        {/* Schedule */}
        <div className="space-y-2">
          <h3 className="text-xs font-semibold text-muted-foreground">Schedule</h3>
          <select
            value={isPreset ? schedule : schedule ? '__custom__' : ''}
            onChange={(e) => setSchedule(e.target.value === '__custom__' ? '' : e.target.value)}
            className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="">No schedule</option>
            {SCHEDULE_PRESETS.map((p) => (
              <option key={p.value} value={p.value}>{p.label}</option>
            ))}
            <option value="__custom__">Custom</option>
          </select>
          {(!isPreset && schedule !== '') && (
            <input
              value={schedule}
              onChange={(e) => setSchedule(e.target.value)}
              placeholder="Custom cron: 0 */6 * * *"
              className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm font-mono outline-none focus:ring-1 focus:ring-ring"
            />
          )}
        </div>
      </div>

      {showSourceModal && (
        <AddSourceModal
          onAdd={(source) => {
            setSources([...sources, { ...source, id: source.id || crypto.randomUUID() }])
            setShowSourceModal(false)
          }}
          onClose={() => setShowSourceModal(false)}
        />
      )}
      {showWorkflowPicker && (
        <WorkflowPicker
          existingWorkflows={workflows}
          onAdd={(wf) => {
            setWorkflows([...workflows, wf])
            setShowWorkflowPicker(false)
          }}
          onClose={() => setShowWorkflowPicker(false)}
        />
      )}
    </div>
  )
}
