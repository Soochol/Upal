import { useState } from 'react'
import { X } from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { Run } from '@/entities/session-run'
import { DEFAULT_RUN_CONTEXT } from '@/entities/session-run/constants'
import type { SessionSource, SessionWorkflow, SessionContext } from '@/entities/session/types'
import { createRunWithConfig, updateRunConfig } from '@/entities/session-run/api'
import { RunConfigFields } from './RunConfigFields'

interface RunConfigPopupProps {
  sessionId: string
  run?: Run
  onSave: (run: Run) => void
  onClose: () => void
}

export function RunConfigPopup({ sessionId, run, onClose, onSave }: RunConfigPopupProps) {
  const isEdit = !!run
  const qc = useQueryClient()

  const [name, setName] = useState(run?.name ?? '')
  const [sources, setSources] = useState<SessionSource[]>(run?.run_sources ?? [])
  const [workflows, setWorkflows] = useState<SessionWorkflow[]>(run?.run_workflows ?? [])
  const [context, setContext] = useState<SessionContext>(run?.context ?? DEFAULT_RUN_CONTEXT)
  const [schedule, setSchedule] = useState(run?.schedule ?? '')
  const [error, setError] = useState<string | null>(null)
  const [showRequired, setShowRequired] = useState(false)

  const hasContent = sources.length > 0 || (context.prompt?.trim() ?? '') !== ''

  function buildConfig() {
    return { name: name || undefined, sources, workflows, context, schedule: schedule || undefined }
  }

  const saveMutation = useMutation({
    mutationFn: () =>
      isEdit ? updateRunConfig(run!.id, buildConfig()) : createRunWithConfig(sessionId, buildConfig()),
    onSuccess: (savedRun) => {
      qc.invalidateQueries({ queryKey: ['session-runs'] })
      onSave(savedRun)
    },
    onError: (err) => setError(err instanceof Error ? err.message : 'Failed to save'),
  })

  function handleCreate() {
    if (!isEdit && !hasContent) {
      setShowRequired(true)
      return
    }
    saveMutation.mutate()
  }

  // Clear validation highlight when user adds content
  if (showRequired && hasContent) setShowRequired(false)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-2xl mx-4 max-h-[85vh] flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-border/50">
          <h2 className="text-lg font-semibold">{isEdit ? 'Edit Run' : 'New Run'}</h2>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-muted/50 text-muted-foreground cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-6">
          <RunConfigFields
            name={name}
            onNameChange={setName}
            sources={sources}
            onSourcesChange={setSources}
            workflows={workflows}
            onWorkflowsChange={setWorkflows}
            context={context}
            onContextChange={setContext}
            schedule={schedule}
            onScheduleChange={setSchedule}
            namePlaceholder="Run name (optional)"
            showRequired={showRequired}
          />
        </div>

        <div className="flex items-center justify-between gap-2 px-6 py-4 border-t border-border/50">
          <div className="text-xs">
            {error && <p className="text-destructive">{error}</p>}
          </div>
          <div className="flex items-center gap-2 ml-auto shrink-0">
            <button
              onClick={onClose}
              className="px-4 py-2 rounded-lg text-sm font-medium text-muted-foreground hover:bg-muted/50 cursor-pointer"
            >
              Cancel
            </button>
            <button
              onClick={handleCreate}
              disabled={saveMutation.isPending}
              className="px-4 py-2 rounded-lg text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 cursor-pointer"
            >
              {saveMutation.isPending && 'Saving...'}
              {!saveMutation.isPending && (isEdit ? 'Save' : 'Create')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
