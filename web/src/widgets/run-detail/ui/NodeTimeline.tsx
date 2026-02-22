import type { NodeRunRecord } from '@/shared/types'
import { TimelineNode } from './TimelineNode'

type Props = {
  nodeRuns: NodeRunRecord[]
  selectedNodeId: string | null
  onSelectNode: (nodeId: string) => void
}

export function NodeTimeline({ nodeRuns, selectedNodeId, onSelectNode }: Props) {
  if (nodeRuns.length === 0) return null

  // Compute max duration for proportional bar widths
  const durations = nodeRuns.map((nr) => {
    if (!nr.started_at || !nr.completed_at) return 0
    return new Date(nr.completed_at).getTime() - new Date(nr.started_at).getTime()
  })
  const maxDuration = Math.max(...durations, 1)

  return (
    <div className="border border-border rounded-lg p-4 bg-card">
      <h2 className="text-sm font-medium text-foreground mb-3">Node Timeline</h2>
      <div className="flex items-center gap-2 overflow-x-auto pb-2">
        {nodeRuns.map((nr, i) => (
          <div key={nr.node_id} className="flex items-center gap-2 shrink-0">
            <TimelineNode
              nodeRun={nr}
              durationMs={durations[i]}
              maxDurationMs={maxDuration}
              isSelected={selectedNodeId === nr.node_id}
              onClick={() => onSelectNode(nr.node_id)}
            />
            {i < nodeRuns.length - 1 && (
              <svg width="20" height="12" viewBox="0 0 20 12" className="shrink-0 text-muted-foreground/40">
                <line x1="0" y1="6" x2="14" y2="6" stroke="currentColor" strokeWidth="1.5" />
                <polyline points="12,2 18,6 12,10" fill="none" stroke="currentColor" strokeWidth="1.5" />
              </svg>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
