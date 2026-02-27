import { useState, useMemo } from 'react'
import { X } from 'lucide-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import type { Run } from '@/entities/session-run'
import { DEFAULT_RUN_CONTEXT } from '@/entities/session-run/constants'
import type { SessionSource, SessionWorkflow, SessionContext } from '@/entities/session/types'
import { fetchRun, updateRunConfig } from '@/entities/session-run/api'
import { useAutoSave } from '@/shared/hooks/useAutoSave'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { RunConfigFields } from './RunConfigFields'

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

  const fp = JSON.stringify([run.name, run.run_sources, run.run_workflows, run.context, run.schedule])
  return <RunConfigForm key={fp} run={run} onClose={onClose} />
}

/** Inner: pure form, initialized from run props. Remounted on server data change via key. */
function RunConfigForm({ run, onClose }: { run: Run; onClose: () => void }) {
  const qc = useQueryClient()

  const [name, setName] = useState(run.name ?? '')
  const [sources, setSources] = useState<SessionSource[]>(run.run_sources ?? [])
  const [workflows, setWorkflows] = useState<SessionWorkflow[]>(run.run_workflows ?? [])
  const [context, setContext] = useState<SessionContext>(run.context ?? DEFAULT_RUN_CONTEXT)
  const [schedule, setSchedule] = useState(run.schedule ?? '')

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

  return (
    <div className="flex flex-col h-full border-l border-border/50 bg-background">
      <div className="flex items-center justify-between px-4 py-3 border-b border-border/50 shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm font-semibold truncate">{run.name || `Run #${run.run_number ?? ''}`}</span>
          <StatusBadge status={run.status} />
        </div>
        <button onClick={onClose} className="p-1 rounded-lg hover:bg-muted/50 text-muted-foreground cursor-pointer">
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-5">
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
        />
      </div>
    </div>
  )
}
