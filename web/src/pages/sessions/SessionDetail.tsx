import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { RunHistoryPanel } from './RunHistoryPanel'
import { RunConfigPanel } from './RunConfigPanel'
import { fetchSessionRuns } from '@/entities/session-run'
import type { Run } from '@/entities/session-run'

interface SessionDetailProps {
  sessionId: string
}

export function SessionDetail({ sessionId }: SessionDetailProps) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null)

  const { data: runs = [] } = useQuery({
    queryKey: ['session-runs', sessionId],
    queryFn: () => fetchSessionRuns(sessionId),
    enabled: !!sessionId,
  })

  // Auto-select first run
  useEffect(() => {
    if (!selectedRunId && runs.length > 0) {
      setSelectedRunId(runs[0].id)
    }
  }, [runs, selectedRunId])

  return (
    <div className="flex h-full">
      <div className={selectedRunId ? 'flex-1 min-w-0' : 'w-full'}>
        <RunHistoryPanel
          sessionId={sessionId}
          selectedRunId={selectedRunId}
          onSelectRun={(run: Run | null) => setSelectedRunId(run?.id ?? null)}
        />
      </div>
      {selectedRunId && (
        <div className="w-[400px] shrink-0">
          <RunConfigPanel runId={selectedRunId} />
        </div>
      )}
    </div>
  )
}
