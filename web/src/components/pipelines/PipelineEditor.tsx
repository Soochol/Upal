// web/src/components/pipelines/PipelineEditor.tsx
import { useState } from 'react'
import { Plus, ArrowLeft, Save } from 'lucide-react'
import { StageCard } from './StageCard'
import type { Pipeline, Stage, StageConfig } from '@/lib/api/types'

type Props = {
  pipeline: Pipeline
  onSave: (pipeline: Pipeline) => void
  onBack: () => void
}

const newStageDefaults: Record<string, Partial<Stage>> = {
  workflow:  { type: 'workflow',  config: {} },
  approval:  { type: 'approval',  config: { timeout: 3600 } },
  schedule:  { type: 'schedule',  config: {} },
  transform: { type: 'transform', config: {} },
}

export function PipelineEditor({ pipeline, onSave, onBack }: Props) {
  const [draft, setDraft] = useState<Pipeline>({ ...pipeline })

  const updateStage = (index: number, stage: Stage) => {
    const stages = [...draft.stages]
    stages[index] = stage
    setDraft({ ...draft, stages })
  }

  const deleteStage = (index: number) => {
    const stages = draft.stages.filter((_, i) => i !== index)
    setDraft({ ...draft, stages })
  }

  const addStage = (type: string) => {
    const defaults = newStageDefaults[type] || { type, config: {} }
    const stage: Stage = {
      id:     `stage-${draft.stages.length + 1}`,
      name:   '',
      type:   type as Stage['type'],
      config: (defaults.config ?? {}) as StageConfig,
    }
    setDraft({ ...draft, stages: [...draft.stages, stage] })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={onBack} className="p-1.5 rounded-md hover:bg-muted transition-colors">
            <ArrowLeft className="h-4 w-4" />
          </button>
          <input
            type="text"
            value={draft.name}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            className="text-lg font-semibold bg-transparent border-none outline-none"
            placeholder="Pipeline name"
          />
        </div>
        <button
          onClick={() => onSave(draft)}
          className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Save className="h-3.5 w-3.5" />
          Save
        </button>
      </div>

      <input
        type="text"
        value={draft.description || ''}
        onChange={(e) => setDraft({ ...draft, description: e.target.value })}
        placeholder="Pipeline description (optional)"
        className="w-full text-sm text-muted-foreground bg-transparent border-none outline-none"
      />

      <div className="space-y-2">
        {draft.stages.map((stage, i) => (
          <div key={stage.id}>
            {i > 0 && (
              <div className="flex justify-center py-1">
                <div className="w-px h-4 bg-border" />
              </div>
            )}
            <StageCard
              stage={stage}
              index={i}
              onChange={(s) => updateStage(i, s)}
              onDelete={() => deleteStage(i)}
            />
          </div>
        ))}
      </div>

      <div className="flex items-center gap-2 pt-2">
        <span className="text-xs text-muted-foreground">Add stage:</span>
        {Object.keys(newStageDefaults).map((type) => (
          <button
            key={type}
            onClick={() => addStage(type)}
            className="flex items-center gap-1 px-2 py-1 text-xs rounded-md border hover:bg-muted transition-colors"
          >
            <Plus className="h-3 w-3" />
            {type}
          </button>
        ))}
      </div>
    </div>
  )
}
