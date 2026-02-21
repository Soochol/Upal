// web/src/components/pipelines/PipelineEditor.tsx
import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, ArrowLeft, Check, Loader2 } from 'lucide-react'
import { StageCard } from './StageCard'
import { listWorkflows, loadWorkflow, listConnections } from '@/lib/api'
import { deserializeWorkflow } from '@/lib/serializer'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { Pipeline, Stage, StageConfig, Connection } from '@/lib/api/types'

type Props = {
  pipeline: Pipeline
  onSave: (pipeline: Pipeline) => Promise<void>
  onBack: () => void
}

const newStageDefaults: Record<string, Partial<Stage>> = {
  workflow:  { type: 'workflow',  config: {} },
  approval:  { type: 'approval',  config: { timeout: 3600 } },
  schedule:  { type: 'schedule',  config: {} },
  trigger:   { type: 'trigger',   config: {} },
  transform: { type: 'transform', config: {} },
}

export function PipelineEditor({ pipeline, onSave, onBack }: Props) {
  const [draft, setDraft] = useState<Pipeline>({ ...pipeline })
  const [workflowNames, setWorkflowNames] = useState<string[]>([])
  const [connections, setConnections] = useState<Connection[]>([])
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [dropIndex, setDropIndex] = useState<number | null>(null)
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved'>('idle')
  const navigate = useNavigate()

  const draftRef = useRef(draft)
  const onSaveRef = useRef(onSave)
  const initialRef = useRef(JSON.stringify(pipeline))
  const isFirstRender = useRef(true)

  draftRef.current = draft
  onSaveRef.current = onSave

  useEffect(() => {
    listWorkflows()
      .then((wfs) => setWorkflowNames(wfs.map((w) => w.name)))
      .catch(() => {})
    listConnections()
      .then((cs) => setConnections(cs ?? []))
      .catch(() => {})
  }, [])

  // Auto-save on change (debounced)
  useEffect(() => {
    if (isFirstRender.current) { isFirstRender.current = false; return }
    if (JSON.stringify(draft) === initialRef.current) return

    setSaveStatus('saving')
    const timer = setTimeout(() => {
      onSaveRef.current(draft).then(() => {
        initialRef.current = JSON.stringify(draft)
        setSaveStatus('saved')
      }).catch(() => setSaveStatus('idle'))
    }, 800)
    return () => clearTimeout(timer)
  }, [draft])

  // Save on unmount (page leave)
  useEffect(() => {
    return () => {
      if (JSON.stringify(draftRef.current) !== initialRef.current) {
        void onSaveRef.current(draftRef.current)
      }
    }
  }, [])

  const handleOpenWorkflow = async (name: string) => {
    try {
      const wf = await loadWorkflow(name)
      const { nodes, edges } = deserializeWorkflow(wf)
      useWorkflowStore.setState({ nodes, edges })
      useWorkflowStore.getState().setWorkflowName(wf.name)
      useWorkflowStore.getState().setOriginalName(wf.name)
      navigate('/editor')
    } catch {
      // silent — workflow may not exist yet
    }
  }

  const updateStage = (index: number, stage: Stage) => {
    const stages = [...draft.stages]
    stages[index] = stage
    setDraft({ ...draft, stages })
  }

  const deleteStage = (index: number) => {
    const stages = draft.stages.filter((_, i) => i !== index)
    setDraft({ ...draft, stages })
  }

  const handleDragStart = (index: number) => setDragIndex(index)

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault()
    if (index !== dropIndex) setDropIndex(index)
  }

  const handleDrop = (index: number) => {
    if (dragIndex !== null && dragIndex !== index) {
      const stages = [...draft.stages]
      const [moved] = stages.splice(dragIndex, 1)
      stages.splice(index, 0, moved)
      setDraft({ ...draft, stages })
    }
    setDragIndex(null)
    setDropIndex(null)
  }

  const handleDragEnd = () => {
    setDragIndex(null)
    setDropIndex(null)
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
        <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
          {saveStatus === 'saving' && <><Loader2 className="h-3 w-3 animate-spin" />Saving…</>}
          {saveStatus === 'saved'  && <><Check className="h-3 w-3 text-success" />Saved</>}
        </span>
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
              pipelineId={pipeline.id}
              workflowNames={workflowNames}
              connections={connections}
              onChange={(s) => updateStage(i, s)}
              onDelete={() => deleteStage(i)}
              onOpenWorkflow={handleOpenWorkflow}
              isDragging={dragIndex === i}
              isDragOver={dropIndex === i && dragIndex !== i}
              onDragStart={() => handleDragStart(i)}
              onDragOver={(e) => handleDragOver(e, i)}
              onDrop={() => handleDrop(i)}
              onDragEnd={handleDragEnd}
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
