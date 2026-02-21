// web/src/components/pipelines/StageCard.tsx
import { Trash2, GripVertical } from 'lucide-react'
import type { Stage } from '@/lib/api/types'

type Props = {
  stage: Stage
  index: number
  isActive?: boolean
  onChange: (stage: Stage) => void
  onDelete: () => void
}

const stageTypeLabels: Record<string, string> = {
  workflow: 'Workflow',
  approval: 'Approval',
  schedule: 'Schedule',
  trigger: 'Trigger',
  transform: 'Transform',
}

const stageTypeBg: Record<string, string> = {
  workflow:  'border-l-info',
  approval:  'border-l-warning',
  schedule:  'border-l-success',
  trigger:   'border-l-[oklch(0.7_0.15_30)]',
  transform: 'border-l-muted-foreground',
}

export function StageCard({ stage, isActive, onChange, onDelete }: Props) {
  return (
    <div className={`border rounded-lg border-l-4 ${stageTypeBg[stage.type] || ''} ${isActive ? 'ring-2 ring-primary' : ''}`}>
      <div className="flex items-center justify-between px-3 py-2">
        <div className="flex items-center gap-2">
          <GripVertical className="h-3.5 w-3.5 text-muted-foreground/50 cursor-grab" />
          <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
            {stageTypeLabels[stage.type] || stage.type}
          </span>
        </div>
        <button
          onClick={onDelete}
          className="p-1 rounded hover:bg-muted transition-colors text-muted-foreground hover:text-destructive"
        >
          <Trash2 className="h-3 w-3" />
        </button>
      </div>

      <div className="px-3 pb-3 space-y-2">
        <input
          type="text"
          value={stage.name}
          onChange={(e) => onChange({ ...stage, name: e.target.value })}
          placeholder="Stage name"
          className="w-full text-sm font-medium bg-transparent border-none outline-none placeholder:text-muted-foreground/50"
        />

        {stage.type === 'workflow' && (
          <input
            type="text"
            value={stage.config.workflow_name || ''}
            onChange={(e) => onChange({ ...stage, config: { ...stage.config, workflow_name: e.target.value } })}
            placeholder="Workflow name"
            className="w-full text-xs bg-muted/50 rounded px-2 py-1.5 outline-none"
          />
        )}

        {stage.type === 'approval' && (
          <textarea
            value={stage.config.message || ''}
            onChange={(e) => onChange({ ...stage, config: { ...stage.config, message: e.target.value } })}
            placeholder="Approval message"
            rows={2}
            className="w-full text-xs bg-muted/50 rounded px-2 py-1.5 outline-none resize-none"
          />
        )}

        {stage.type === 'schedule' && (
          <input
            type="text"
            value={stage.config.cron || ''}
            onChange={(e) => onChange({ ...stage, config: { ...stage.config, cron: e.target.value } })}
            placeholder="Cron expression (e.g. 0 9 * * *)"
            className="w-full text-xs bg-muted/50 rounded px-2 py-1.5 outline-none font-mono"
          />
        )}
      </div>
    </div>
  )
}
