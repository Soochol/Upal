import { useState, useMemo } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import type { Run } from '@/entities/session-run'
import { DEFAULT_RUN_CONTEXT } from '@/entities/session-run/constants'
import type { SessionSource, SessionWorkflow, SessionContext } from '@/entities/session/types'
import { fetchRun, updateRunConfig } from '@/entities/session-run/api'
import { useAutoSave } from '@/shared/hooks/useAutoSave'
import { RunConfigFields } from './RunConfigFields'

interface RunConfigPanelProps {
  runId: string
}

export function RunConfigPanel({ runId }: RunConfigPanelProps) {
  const { data: run } = useQuery({
    queryKey: ['session-run', runId],
    queryFn: () => fetchRun(runId),
  })

  if (!run) return null

  // Remount the form when server data changes so local state resets.
  const configKey = JSON.stringify([run.name, run.run_sources, run.run_workflows, run.context, run.schedule])
  return <RunConfigForm key={configKey} run={run} />
}

function RunConfigForm({ run }: { run: Run }) {
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
      qc.invalidateQueries({ queryKey: ['inbox-runs'] })
    },
    delay: 2000,
    saveOnUnmount: true,
    onError: (err) => console.error('Auto-save failed:', err),
  })

  return (
    <div className="flex flex-col h-full border-l border-border/50 bg-background">
      <div className="flex items-center px-4 py-3 border-b border-border/50 shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm font-semibold truncate">{run.name || `Run #${run.run_number ?? ''}`}</span>
        </div>
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
