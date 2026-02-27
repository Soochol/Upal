import { useState } from 'react'
import { X, Search, Sparkles, Plus, Trash2 } from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { cn } from '@/shared/lib/utils'
import type { Run } from '@/entities/session-run'
import type { SessionSource, SessionWorkflow, SessionContext } from '@/entities/session/types'
import { createRunWithConfig, updateRunConfig } from '@/entities/session-run/api'
import { WorkflowPicker } from '@/features/workflow-picker'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'

const SCHEDULE_PRESETS = [
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Daily', value: '0 8 * * *' },
  { label: 'Weekly', value: '0 8 * * MON' },
]

interface RunConfigPopupProps {
  sessionId: string
  run?: Run                    // undefined = create mode
  onSave: (run: Run) => void
  onClose: () => void
}

export function RunConfigPopup({ sessionId, run, onClose, onSave }: RunConfigPopupProps) {
  const isEdit = !!run
  const qc = useQueryClient()

  // Form state
  const [name, setName] = useState(run?.name ?? '')
  const [sources, setSources] = useState<SessionSource[]>(run?.run_sources ?? [])
  const [workflows, setWorkflows] = useState<SessionWorkflow[]>(run?.run_workflows ?? [])
  const [context, setContext] = useState<SessionContext>(run?.context ?? { research_depth: 'deep' })
  const [schedule, setSchedule] = useState(run?.schedule ?? '')

  // Modal state
  const [showSourceModal, setShowSourceModal] = useState(false)
  const [editingSourceIndex, setEditingSourceIndex] = useState<number | null>(null)
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)

  const isPreset = SCHEDULE_PRESETS.some(p => p.value === schedule && p.value !== '')

  const saveMutation = useMutation({
    mutationFn: () => {
      const config = { name: name || undefined, sources, workflows, context, schedule: schedule || undefined }
      return isEdit
        ? updateRunConfig(run!.id, config)
        : createRunWithConfig(sessionId, config)
    },
    onSuccess: (savedRun) => {
      qc.invalidateQueries({ queryKey: ['session-runs'] })
      onSave(savedRun)
    },
  })

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-2xl mx-4 max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border/50">
          <h2 className="text-lg font-semibold">{isEdit ? 'Edit Run' : 'New Run'}</h2>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-muted/50 text-muted-foreground cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-6">
          {/* Name */}
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Run name (optional)"
              className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring"
            />
          </div>

          {/* Research */}
          <div className="space-y-3">
            <h3 className="text-sm font-semibold flex items-center gap-2">
              <Search className="h-4 w-4 text-muted-foreground" />
              Research
            </h3>
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground">Mode</label>
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
            </div>
            {/* Additional Sources */}
            {sources.length > 0 && (
              <div className="space-y-1">
                {sources.map((s, i) => (
                  <div key={s.id || i} className="flex items-center justify-between px-3 py-2 rounded-lg bg-muted/30 text-sm">
                    <span>{s.label || s.type}</span>
                    <button
                      onClick={() => setSources(sources.filter((_, idx) => idx !== i))}
                      className="p-1 text-muted-foreground hover:text-destructive cursor-pointer"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                ))}
              </div>
            )}
            <button
              onClick={() => { setEditingSourceIndex(null); setShowSourceModal(true) }}
              className="text-xs text-primary hover:underline cursor-pointer"
            >
              + Add source
            </button>
          </div>

          {/* Task */}
          <div className="space-y-2">
            <h3 className="text-sm font-semibold flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-muted-foreground" />
              Task
            </h3>
            <textarea
              value={context.prompt ?? ''}
              onChange={(e) => setContext({ ...context, prompt: e.target.value })}
              placeholder="What should this run focus on? e.g., ai news, tech trends..."
              className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring resize-none"
              rows={3}
            />
          </div>

          {/* Processing */}
          <div className="space-y-3">
            <h3 className="text-sm font-semibold">Processing</h3>
            <div className="space-y-1">
              {workflows.map((wf, i) => (
                <div key={wf.workflow_name} className="flex items-center justify-between px-3 py-2 rounded-lg bg-muted/30 text-sm">
                  <span>{wf.label || wf.workflow_name}</span>
                  <button
                    onClick={() => setWorkflows(workflows.filter((_, idx) => idx !== i))}
                    className="p-1 text-muted-foreground hover:text-destructive cursor-pointer"
                  >
                    <Trash2 className="h-3 w-3" />
                  </button>
                </div>
              ))}
            </div>
            <button
              onClick={() => setShowWorkflowPicker(true)}
              className="flex items-center gap-1 text-xs text-primary hover:underline cursor-pointer"
            >
              <Plus className="h-3 w-3" /> Add workflow
            </button>
          </div>

          {/* Schedule */}
          <div className="space-y-2">
            <h3 className="text-sm font-semibold">Schedule</h3>
            <div className="space-y-2">
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
              {!isPreset && schedule !== '' && (
                <input
                  value={schedule}
                  onChange={(e) => setSchedule(e.target.value)}
                  placeholder="0 */6 * * *"
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm font-mono outline-none focus:ring-1 focus:ring-ring"
                />
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-border/50">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg text-sm font-medium text-muted-foreground hover:bg-muted/50 cursor-pointer"
          >
            Cancel
          </button>
          <button
            onClick={() => saveMutation.mutate()}
            disabled={saveMutation.isPending}
            className="px-4 py-2 rounded-lg text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 cursor-pointer"
          >
            {saveMutation.isPending ? 'Saving...' : isEdit ? 'Save' : 'Create'}
          </button>
        </div>
      </div>

      {/* Sub-modals */}
      {showSourceModal && (
        <AddSourceModal
          editSource={editingSourceIndex != null ? sources[editingSourceIndex] : undefined}
          onAdd={(source) => {
            if (editingSourceIndex != null) {
              const updated = [...sources]
              updated[editingSourceIndex] = source
              setSources(updated)
            } else {
              setSources([...sources, { ...source, id: source.id || crypto.randomUUID() }])
            }
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
