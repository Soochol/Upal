import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Loader2, Plus, Play } from 'lucide-react'
import { fetchSessionRuns } from '@/entities/session-run'
import { RunCard } from './RunCard'
import { RunConfigPopup } from './RunConfigPopup'
import type { Run } from '@/entities/session-run'

interface RunHistoryPanelProps {
  sessionId: string
  selectedRunId?: string | null
  onSelectRun?: (run: Run | null) => void
}

export function RunHistoryPanel({ sessionId, selectedRunId, onSelectRun }: RunHistoryPanelProps) {
  const { data: runs = [], isLoading } = useQuery({
    queryKey: ['session-runs', sessionId],
    queryFn: () => fetchSessionRuns(sessionId),
    enabled: !!sessionId,
    refetchInterval: 10000,
  })

  const [showConfigPopup, setShowConfigPopup] = useState(false)

  const openCreate = () => setShowConfigPopup(true)
  const closePopup = () => setShowConfigPopup(false)

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
        <Loader2 className="w-5 h-5 animate-spin text-primary/50" />
        <p className="text-sm">Loading runs...</p>
      </div>
    )
  }

  if (runs.length === 0) {
    return (
      <>
        <div className="flex flex-col items-center justify-center py-16 text-muted-foreground gap-4 text-center">
          <Play className="w-8 h-8 opacity-20" />
          <div>
            <p className="font-medium text-foreground">No runs yet</p>
            <p className="text-xs mt-1">Create a run to start collecting content.</p>
          </div>
          <button
            onClick={openCreate}
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 cursor-pointer"
          >
            <Plus className="h-4 w-4" /> New Run
          </button>
        </div>
        {showConfigPopup && (
          <RunConfigPopup sessionId={sessionId} onSave={closePopup} onClose={closePopup} />
        )}
      </>
    )
  }

  return (
    <>
      <div className="max-w-4xl mx-auto px-6 py-4">
        <div className="flex justify-end mb-3">
          <button
            onClick={openCreate}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary text-primary-foreground text-xs font-medium hover:bg-primary/90 cursor-pointer"
          >
            <Plus className="h-3.5 w-3.5" /> Run
          </button>
        </div>
        <div className="space-y-2">
          {runs.map((run: Run) => (
            <RunCard
              key={run.id}
              run={run}
              isSelected={run.id === selectedRunId}
              onSelect={(r) => onSelectRun?.(r)}
            />
          ))}
        </div>
      </div>
      {showConfigPopup && (
        <RunConfigPopup sessionId={sessionId} onSave={closePopup} onClose={closePopup} />
      )}
    </>
  )
}
