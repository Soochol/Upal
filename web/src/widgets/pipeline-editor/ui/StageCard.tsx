// web/src/widgets/pipeline-editor/ui/StageCard.tsx
import { useState, useRef, useEffect } from 'react'
import { Trash2, GripVertical, ExternalLink, Play, PauseCircle, Clock, Zap, RefreshCw, GitBranch, Copy, Check, Bell, Download, Plus } from 'lucide-react'
import type { Stage, Connection, CollectSource } from '@/shared/types'
import { createTrigger, deleteTrigger } from '@/shared/api'
import { loadWorkflow } from '@/entities/workflow'
import type { WorkflowDefinition } from '@/entities/workflow'

type Props = {
  stage: Stage
  index: number
  isActive?: boolean
  pipelineId?: string
  workflowNames?: string[]
  connections?: Connection[]
  prevStage?: Stage
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

// ─── Notification field config ───────────────────────────────────────────────
// To support a new connection type, add an entry here — no other code changes needed.

type NotifField = {
  key: 'message' | 'subject'
  label: string
  placeholder: string
  multiline: boolean
}

const connTypeFields: Record<string, NotifField[]> = {
  telegram: [
    { key: 'message', label: '메시지',      placeholder: '전송할 메시지를 입력하세요…', multiline: true },
  ],
  slack: [
    { key: 'message', label: '메시지',      placeholder: '전송할 메시지를 입력하세요…', multiline: true },
  ],
  smtp: [
    { key: 'subject', label: '이메일 제목', placeholder: '제목을 입력하세요…',          multiline: false },
    { key: 'message', label: '이메일 본문', placeholder: '이메일 내용을 입력하세요…',  multiline: true  },
  ],
  http: [
    { key: 'message', label: '내용',        placeholder: '전송할 내용을 입력하세요…',  multiline: true  },
  ],
}

const defaultNotifFields: NotifField[] = [
  { key: 'message', label: '메시지',        placeholder: '알림 메시지를 입력하세요…',  multiline: true  },
]

function NotificationFields({
  stage, connections, onChange,
}: {
  stage: Stage
  connections: Connection[]
  onChange: (stage: Stage) => void
}) {
  const selectedConn = connections.find(c => c.id === stage.config.connection_id)
  const fields = selectedConn
    ? (connTypeFields[selectedConn.type] ?? defaultNotifFields)
    : []

  const fieldClass = 'w-full text-xs bg-muted/40 rounded-lg px-2 py-1.5 outline-none border border-transparent focus:border-border focus:ring-1 focus:ring-ring transition-all'

  return (
    <>
      <select
        value={stage.config.connection_id || ''}
        onChange={(e) => onChange({ ...stage, config: { ...stage.config, connection_id: e.target.value } })}
        className={fieldClass}
      >
        <option value="">연결 선택…</option>
        {connections.map((c) => (
          <option key={c.id} value={c.id}>{c.name} ({c.type})</option>
        ))}
      </select>
      {!stage.config.connection_id && (
        <p className="text-[10px] text-warning/80">연결을 선택해야 알림을 전송할 수 있습니다.</p>
      )}
      {fields.map((f) => (
        <div key={f.key} className="space-y-0.5">
          <label className="text-[10px] text-muted-foreground">{f.label}</label>
          {f.multiline ? (
            <textarea
              rows={2}
              value={(stage.config[f.key] as string) || ''}
              placeholder={f.placeholder}
              onChange={(e) => onChange({ ...stage, config: { ...stage.config, [f.key]: e.target.value } })}
              className={`${fieldClass} resize-none`}
            />
          ) : (
            <input
              type="text"
              value={(stage.config[f.key] as string) || ''}
              placeholder={f.placeholder}
              onChange={(e) => onChange({ ...stage, config: { ...stage.config, [f.key]: e.target.value } })}
              className={fieldClass}
            />
          )}
        </div>
      ))}
    </>
  )
}

// ─── Input mapping — reference picker ────────────────────────────────────────

// Known output fields per stage type, shown as selectable references.
const stageOutputFields: Record<string, { field: string; label: string }[]> = {
  collect:      [{ field: 'text', label: '수집 텍스트' }, { field: 'sources', label: '소스 데이터' }],
  workflow:     [{ field: 'output', label: '워크플로우 출력' }],
  transform:    [{ field: 'output', label: '변환 결과' }],
  notification: [{ field: 'sent', label: '전송 여부' }, { field: 'channel', label: '채널 이름' }],
}

function InputMappingField({
  label, placeholder, value, prevStage, onChange,
}: {
  label: string
  placeholder: string
  value: string
  prevStage?: Stage
  onChange: (v: string) => void
}) {
  const refFields = prevStage ? (stageOutputFields[prevStage.type] ?? []) : []
  const isRef    = /^\{\{[^}]+\}\}$/.test(value)
  const isManual = value !== '' && !isRef
  const selectValue = isRef ? value : isManual ? '__manual__' : ''

  const cls = 'w-full text-xs bg-muted/40 rounded-lg px-2 py-1.5 outline-none border border-transparent focus:border-border focus:ring-1 focus:ring-ring transition-all'

  const handleSelect = (v: string) => {
    if (v === '__manual__') { if (isRef) onChange('') }
    else onChange(v)
  }

  return (
    <div className="space-y-0.5">
      <label className="text-[10px] text-muted-foreground">{label}</label>
      {refFields.length > 0 ? (
        <div className="space-y-1">
          <select value={selectValue} onChange={(e) => handleSelect(e.target.value)} className={cls}>
            <option value="">연결 안 함</option>
            {refFields.map(f => (
              <option key={f.field} value={`{{${f.field}}}`}>
                이전 단계 · {f.label}
              </option>
            ))}
            <option value="__manual__">직접 입력…</option>
          </select>
          {(selectValue === '__manual__' || isManual) && (
            <textarea
              rows={2}
              value={value}
              placeholder={placeholder}
              onChange={(e) => onChange(e.target.value)}
              className={`${cls} resize-none`}
            />
          )}
        </div>
      ) : (
        <textarea
          rows={2}
          value={value}
          placeholder={placeholder}
          onChange={(e) => onChange(e.target.value)}
          className={`${cls} resize-none`}
        />
      )}
    </div>
  )
}

// ─── Collect field config ─────────────────────────────────────────────────────

function CollectFields({
  stage,
  onChange,
}: {
  stage: Stage
  onChange: (stage: Stage) => void
}) {
  const sources = stage.config.sources ?? []
  const fieldClass = 'w-full text-xs bg-muted/40 rounded-lg px-2 py-1.5 outline-none border border-transparent focus:border-border focus:ring-1 focus:ring-ring transition-all'

  const addSource = () => {
    const newSource: CollectSource = { id: crypto.randomUUID(), type: 'rss', url: '' }
    onChange({ ...stage, config: { ...stage.config, sources: [...sources, newSource] } })
  }

  const removeSource = (id: string) => {
    onChange({ ...stage, config: { ...stage.config, sources: sources.filter(s => s.id !== id) } })
  }

  const updateSource = (id: string, patch: Partial<CollectSource>) => {
    onChange({
      ...stage,
      config: {
        ...stage.config,
        sources: sources.map(s => s.id === id ? { ...s, ...patch } : s),
      },
    })
  }

  return (
    <div className="space-y-2">
      {sources.map((src) => (
        <div key={src.id} className="rounded-lg border border-border/60 bg-muted/20 p-2 space-y-1.5">
          <div className="flex items-center gap-1.5">
            <select
              value={src.type}
              onChange={(e) => updateSource(src.id, { type: e.target.value as CollectSource['type'] })}
              className="text-xs bg-muted/40 rounded-md px-1.5 py-1 outline-none border border-transparent focus:border-border transition-all"
            >
              <option value="rss">RSS</option>
              <option value="http">HTTP</option>
              <option value="scrape">Scrape</option>
            </select>
            <input
              type="text"
              value={src.url}
              placeholder="URL"
              onChange={(e) => updateSource(src.id, { url: e.target.value })}
              className="flex-1 text-xs bg-muted/40 rounded-md px-2 py-1 outline-none border border-transparent focus:border-border transition-all font-mono"
            />
            <button
              onClick={() => removeSource(src.id)}
              className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors shrink-0 cursor-pointer"
            >
              <Trash2 className="h-3 w-3" />
            </button>
          </div>

          {src.type === 'rss' && (
            <div className="flex items-center gap-1.5">
              <label className="text-[10px] text-muted-foreground shrink-0">최대 항목</label>
              <input
                type="number"
                min={1}
                max={100}
                value={src.limit ?? 20}
                onChange={(e) => updateSource(src.id, { limit: Number(e.target.value) })}
                className="w-16 text-xs bg-muted/40 rounded-md px-2 py-1 outline-none border border-transparent focus:border-border transition-all"
              />
            </div>
          )}

          {src.type === 'http' && (
            <div className="space-y-1">
              <select
                value={src.method ?? 'GET'}
                onChange={(e) => updateSource(src.id, { method: e.target.value })}
                className="text-xs bg-muted/40 rounded-md px-1.5 py-1 outline-none border border-transparent focus:border-border transition-all"
              >
                <option value="GET">GET</option>
                <option value="POST">POST</option>
              </select>
              <textarea
                rows={2}
                value={src.body ?? ''}
                placeholder="요청 본문 (선택)"
                onChange={(e) => updateSource(src.id, { body: e.target.value })}
                className={`${fieldClass} resize-none`}
              />
            </div>
          )}

          {src.type === 'scrape' && (
            <div className="space-y-1">
              <input
                type="text"
                value={src.selector ?? ''}
                placeholder="CSS 셀렉터 (예: .article-title)"
                onChange={(e) => updateSource(src.id, { selector: e.target.value })}
                className={`${fieldClass} font-mono`}
              />
              <div className="flex items-center gap-1.5">
                <label className="text-[10px] text-muted-foreground shrink-0">최대 항목</label>
                <input
                  type="number"
                  min={1}
                  max={200}
                  value={src.scrape_limit ?? 30}
                  onChange={(e) => updateSource(src.id, { scrape_limit: Number(e.target.value) })}
                  className="w-16 text-xs bg-muted/40 rounded-md px-2 py-1 outline-none border border-transparent focus:border-border transition-all"
                />
              </div>
            </div>
          )}
        </div>
      ))}

      <button
        onClick={addSource}
        className="flex items-center gap-1.5 w-full px-2 py-1.5 text-xs rounded-lg
          border border-dashed border-border hover:border-ring hover:bg-muted/40
          text-muted-foreground hover:text-foreground transition-all cursor-pointer"
      >
        <Plus className="h-3 w-3" />
        소스 추가
      </button>
    </div>
  )
}

// ─────────────────────────────────────────────────────────────────────────────

const stageTypeConfig: Record<string, { label: string; icon: typeof GitBranch; color: string }> = {
  workflow:     { label: 'Workflow',     icon: Play,        color: 'var(--info)' },
  approval:     { label: 'Approval',     icon: PauseCircle, color: 'var(--warning)' },
  notification: { label: 'Notification', icon: Bell,        color: 'var(--node-agent)' },
  schedule:     { label: 'Schedule',     icon: Clock,       color: 'var(--success)' },
  trigger:      { label: 'Trigger',      icon: Zap,         color: 'var(--node-agent)' },
  transform:    { label: 'Transform',    icon: RefreshCw,   color: 'var(--muted-foreground)' },
  collect:      { label: 'Collect',      icon: Download,    color: 'var(--info)' },
}

export function StageCard({
  stage, pipelineId, isActive, workflowNames = [], connections = [],
  prevStage, onChange, onDelete, onOpenWorkflow,
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
                  const placeholder = (node.config.placeholder as string) || `Enter ${label}…`
                  const value = stage.config.input_mapping?.[node.id] ?? ''
                  return (
                    <InputMappingField
                      key={node.id}
                      label={label}
                      placeholder={placeholder}
                      value={value}
                      prevStage={prevStage}
                      onChange={(v) => {
                        const mapping = { ...(stage.config.input_mapping ?? {}), [node.id]: v }
                        onChange({ ...stage, config: { ...stage.config, input_mapping: mapping } })
                      }}
                    />
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

        {stage.type === 'notification' && (
          <NotificationFields
            stage={stage}
            connections={connections}
            onChange={onChange}
          />
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

        {stage.type === 'collect' && (
          <CollectFields
            stage={stage}
            onChange={onChange}
          />
        )}
      </div>
    </div>
  )
}
