// web/src/components/pipelines/StageCard.tsx
import { useState, useRef, useEffect } from 'react'
import { Trash2, GripVertical, ExternalLink, Play, PauseCircle, Clock, Zap, RefreshCw, GitBranch, Copy, Check } from 'lucide-react'
import type { Stage, Connection } from '@/lib/api/types'
import { createTrigger, deleteTrigger } from '@/lib/api'
import { loadWorkflow } from '@/lib/api/workflows'
import type { WorkflowDefinition } from '@/lib/serializer'

type Props = {
  stage: Stage
  index: number
  isActive?: boolean
  pipelineId?: string
  workflowNames?: string[]
  connections?: Connection[]
  onChange: (stage: Stage) => void
  onDelete: () => void
  onOpenWorkflow?: (name: string) => void
  // drag-reorder
  isDragging?: boolean
  isDragOver?: boolean
  onDragStart?: () => void
  onDragOver?: (e: React.DragEvent) => void
  onDrop?: () => void
  onDragEnd?: () => void
}

const stageTypeConfig: Record<string, { label: string; icon: typeof GitBranch; color: string }> = {
  workflow:  { label: 'Workflow',  icon: Play,        color: 'var(--info)' },
  approval:  { label: 'Approval',  icon: PauseCircle, color: 'var(--warning)' },
  schedule:  { label: 'Schedule',  icon: Clock,       color: 'var(--success)' },
  trigger:   { label: 'Trigger',   icon: Zap,         color: 'var(--node-agent)' },
  transform: { label: 'Transform', icon: RefreshCw,   color: 'var(--muted-foreground)' },
}

export function StageCard({
  stage, pipelineId, isActive, workflowNames = [], connections = [],
  onChange, onDelete, onOpenWorkflow,
  isDragging, isDragOver, onDragStart, onDragOver, onDrop, onDragEnd,
}: Props) {
  const cfg = stageTypeConfig[stage.type] ?? { label: stage.type, icon: GitBranch, color: 'var(--muted-foreground)' }
  const Icon = cfg.icon
  const [creating, setCreating] = useState(false)
  const [copied, setCopied] = useState(false)
  // draggable is enabled only while the grip handle is held — prevents
  // accidental drags when clicking inputs/selects inside the card.
  const draggableRef = useRef(false)
  const [draggable, setDraggable] = useState(false)
  const [workflowDef, setWorkflowDef] = useState<WorkflowDefinition | null>(null)

  // Fetch selected workflow to extract input nodes
  useEffect(() => {
    const name = stage.config.workflow_name
    if (!name) { setWorkflowDef(null); return }
    let cancelled = false
    loadWorkflow(name).then((wf) => { if (!cancelled) setWorkflowDef(wf) }).catch(() => { if (!cancelled) setWorkflowDef(null) })
    return () => { cancelled = true }
  }, [stage.config.workflow_name])

  const inputNodes = workflowDef?.nodes.filter((n) => n.type === 'input') ?? []

  const webhookPath = stage.config.trigger_id ? `/api/hooks/${stage.config.trigger_id}` : null

  const handleCreateTrigger = async () => {
    if (!pipelineId) return
    setCreating(true)
    try {
      const { trigger } = await createTrigger({ pipeline_id: pipelineId })
      onChange({ ...stage, config: { ...stage.config, trigger_id: trigger.id } })
    } catch {
      // silent
    } finally {
      setCreating(false)
    }
  }

  const handleDeleteTrigger = async () => {
    if (!stage.config.trigger_id) return
    try {
      await deleteTrigger(stage.config.trigger_id)
    } catch {
      // silent — still clear from stage config
    }
    onChange({ ...stage, config: { ...stage.config, trigger_id: '' } })
  }

  const handleCopy = () => {
    if (!webhookPath) return
    const full = `${window.location.origin}${webhookPath}`
    navigator.clipboard.writeText(full).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div
      draggable={draggable}
      onDragStart={(e) => {
        if (!draggableRef.current) { e.preventDefault(); return }
        onDragStart?.()
      }}
      onDragOver={onDragOver}
      onDrop={(e) => { e.preventDefault(); onDrop?.() }}
      onDragEnd={() => { setDraggable(false); draggableRef.current = false; onDragEnd?.() }}
      className={[
        'rounded-xl border bg-card overflow-hidden transition-all duration-150',
        isActive ? 'ring-2 ring-ring' : '',
        isDragging ? 'opacity-40 scale-[0.98]' : '',
        isDragOver ? 'ring-2 ring-primary border-primary' : 'border-border',
      ].join(' ')}
    >
      {/* Header row */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-border/60 bg-muted/20">
        <div className="flex items-center gap-2">
          <GripVertical
            className="h-3.5 w-3.5 text-muted-foreground/40 hover:text-muted-foreground cursor-grab active:cursor-grabbing shrink-0 transition-colors"
            onMouseDown={() => { draggableRef.current = true; setDraggable(true) }}
            onMouseUp={() => { if (!isDragging) { draggableRef.current = false; setDraggable(false) } }}
          />
          {/* Type badge */}
          <span
            className="inline-flex items-center gap-1 text-[10px] font-semibold px-1.5 py-0.5 rounded-md landing-body"
            style={{
              background: `color-mix(in oklch, ${cfg.color}, transparent 88%)`,
              color: cfg.color,
            }}
          >
            <Icon className="h-2.5 w-2.5" />
            {cfg.label}
          </span>
        </div>
        <button
          onClick={onDelete}
          className="p-1 rounded-lg hover:bg-destructive/10 text-muted-foreground hover:text-destructive
            transition-colors cursor-pointer"
        >
          <Trash2 className="h-3 w-3" />
        </button>
      </div>

      {/* Body */}
      <div className="px-3 pb-3 pt-2.5 space-y-2">
        <input
          type="text"
          value={stage.name}
          onChange={(e) => onChange({ ...stage, name: e.target.value })}
          placeholder="Stage name"
          className="w-full text-sm font-medium bg-transparent border-none outline-none
            placeholder:text-muted-foreground/40"
        />

        {stage.type === 'workflow' && (
          <>
            <div className="flex items-center gap-1.5">
              <select
                value={stage.config.workflow_name || ''}
                onChange={(e) => onChange({ ...stage, config: { ...stage.config, workflow_name: e.target.value, input_mapping: {} } })}
                className="flex-1 text-xs bg-muted/40 rounded-lg px-2 py-1.5 outline-none
                  focus:ring-1 focus:ring-ring border border-transparent focus:border-border transition-all"
              >
                <option value="">Select workflow…</option>
                {workflowNames.map((name) => (
                  <option key={name} value={name}>{name}</option>
                ))}
              </select>
              {stage.config.workflow_name && onOpenWorkflow && (
                <button
                  onClick={() => onOpenWorkflow(stage.config.workflow_name!)}
                  title="Open in editor"
                  className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground
                    transition-colors shrink-0 cursor-pointer"
                >
                  <ExternalLink className="h-3 w-3" />
                </button>
              )}
            </div>

            {inputNodes.length > 0 && (
              <div className="space-y-1.5 pt-0.5">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">Inputs</p>
                {inputNodes.map((node) => {
                  const label = (node.config.label as string) || node.id
                  const placeholder = (node.config.placeholder as string) || ''
                  const value = stage.config.input_mapping?.[node.id] ?? ''
                  return (
                    <div key={node.id} className="space-y-0.5">
                      <label className="text-[10px] text-muted-foreground">{label}</label>
                      <textarea
                        rows={2}
                        value={value}
                        placeholder={placeholder || `Enter ${label}…`}
                        onChange={(e) => {
                          const mapping = { ...(stage.config.input_mapping ?? {}), [node.id]: e.target.value }
                          onChange({ ...stage, config: { ...stage.config, input_mapping: mapping } })
                        }}
                        className="w-full text-xs bg-muted/40 rounded-lg px-2 py-1.5 outline-none resize-none
                          border border-transparent focus:border-border focus:ring-1 focus:ring-ring transition-all"
                      />
                    </div>
                  )
                })}
              </div>
            )}
          </>
        )}

        {stage.type === 'approval' && (
          <>
            <textarea
              value={stage.config.message || ''}
              onChange={(e) => onChange({ ...stage, config: { ...stage.config, message: e.target.value } })}
              placeholder="Approval message"
              rows={2}
              className="w-full text-xs bg-muted/40 rounded-lg px-2 py-1.5 outline-none resize-none
                border border-transparent focus:border-border focus:ring-1 focus:ring-ring transition-all"
            />
            <select
              value={stage.config.connection_id || ''}
              onChange={(e) => onChange({ ...stage, config: { ...stage.config, connection_id: e.target.value } })}
              className="w-full text-xs bg-muted/40 rounded-lg px-2 py-1.5 outline-none
                border border-transparent focus:border-border focus:ring-1 focus:ring-ring transition-all"
            >
              <option value="">No notification</option>
              {connections.map((c) => (
                <option key={c.id} value={c.id}>{c.name} ({c.type})</option>
              ))}
            </select>
          </>
        )}

        {stage.type === 'schedule' && (
          <input
            type="text"
            value={stage.config.cron || ''}
            onChange={(e) => onChange({ ...stage, config: { ...stage.config, cron: e.target.value } })}
            placeholder="Cron expression (e.g. 0 9 * * *)"
            className="w-full text-xs bg-muted/40 rounded-lg px-2 py-1.5 outline-none font-mono
              border border-transparent focus:border-border focus:ring-1 focus:ring-ring transition-all"
          />
        )}

        {stage.type === 'trigger' && (
          <>
            {webhookPath ? (
              <div className="space-y-1.5">
                <div className="flex items-center gap-1.5">
                  <code className="flex-1 text-[10px] bg-muted/40 rounded px-2 py-1.5 font-mono truncate text-muted-foreground">
                    {webhookPath}
                  </code>
                  <button
                    onClick={handleCopy}
                    title="Copy webhook URL"
                    className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground
                      transition-colors shrink-0 cursor-pointer"
                  >
                    {copied ? <Check className="h-3 w-3 text-success" /> : <Copy className="h-3 w-3" />}
                  </button>
                </div>
                <button
                  onClick={handleDeleteTrigger}
                  className="text-[10px] text-destructive/70 hover:text-destructive transition-colors cursor-pointer"
                >
                  Remove webhook
                </button>
              </div>
            ) : (
              <button
                onClick={handleCreateTrigger}
                disabled={creating || !pipelineId}
                className="flex items-center gap-1.5 w-full px-2 py-1.5 text-xs rounded-lg
                  border border-dashed border-border hover:border-ring hover:bg-muted/40
                  text-muted-foreground hover:text-foreground transition-all disabled:opacity-50
                  disabled:cursor-not-allowed cursor-pointer"
              >
                <Zap className="h-3 w-3" />
                {creating ? 'Generating…' : 'Generate webhook URL'}
              </button>
            )}
          </>
        )}
      </div>
    </div>
  )
}
