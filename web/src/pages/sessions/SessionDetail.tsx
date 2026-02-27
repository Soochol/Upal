import { useState } from 'react'
import { RunHistoryPanel } from './RunHistoryPanel'
import { RunConfigPanel } from './RunConfigPanel'
import type { Run } from '@/entities/session-run'

interface SessionDetailProps {
  sessionId: string
}

export function SessionDetail({ sessionId }: SessionDetailProps) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null)

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
          <RunConfigPanel
            runId={selectedRunId}
            onClose={() => setSelectedRunId(null)}
          />
        </div>
      )}
    </div>
  )
}
