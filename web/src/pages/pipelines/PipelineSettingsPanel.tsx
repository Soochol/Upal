import { useState } from 'react'
import {
  Loader2, Trash2, Plus, Clock,
  ChevronDown, CloudUpload,
} from 'lucide-react'
import { ModelSelector } from '@/shared/ui/ModelSelector'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { EditorialBriefForm } from '@/features/define-editorial-brief/EditorialBriefForm'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { WorkflowPicker } from './WorkflowPicker'
import { PipelineChatEditor } from '@/features/configure-pipeline/ui/PipelineChatEditor'
import type { PipelineSource, PipelineContext, PipelineWorkflow } from '@/shared/types'

const SCHEDULE_PRESETS: { label: string; cron: string }[] = [
  { label: 'Every hour', cron: '0 * * * *' },
  { label: 'Every 6 hours', cron: '0 */6 * * *' },
  { label: 'Every 12 hours', cron: '0 */12 * * *' },
  { label: 'Daily at 09:00', cron: '0 9 * * *' },
  { label: 'Weekdays at 09:00', cron: '0 9 * * 1-5' },
  { label: 'Weekly (Mon 09:00)', cron: '0 9 * * 1' },
  { label: 'Monthly (1st 09:00)', cron: '0 9 1 * *' },
]

export function PipelineSettingsPanel({
  pipelineId,
  sources, schedule, context, workflows, model, channels,
  onSourcesChange, onScheduleChange, onContextSave, onWorkflowsChange, onModelChange, autoSaveStatus,
}: {
  pipelineId: string
  sources: PipelineSource[]
  schedule: string
  context: PipelineContext | undefined
  workflows: PipelineWorkflow[]
  model: string
  channels: { id: string; name: string; type: string }[]
  onSourcesChange: (s: PipelineSource[]) => void
  onScheduleChange: (cron: string) => void
  onContextSave: (ctx: PipelineContext) => Promise<void>
  onWorkflowsChange: (w: PipelineWorkflow[]) => void
  onModelChange: (model: string) => void
  autoSaveStatus: 'idle' | 'saving' | 'saved'
}) {
  const [showAddModal, setShowAddModal] = useState(false)
  const [sourcesOpen, setSourcesOpen] = useState(true)
  const [briefOpen, setBriefOpen] = useState(false)
  const [workflowsOpen, setWorkflowsOpen] = useState(false)
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)
  const [modelOpen, setModelOpen] = useState(false)

  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-3 border-b border-border flex items-center justify-between">
        <h2 className="text-sm font-semibold">Settings</h2>
        <div className="h-4">
          {autoSaveStatus === 'saving' && (
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />Saving…
            </span>
          )}
          {autoSaveStatus === 'saved' && (
            <span className="flex items-center gap-1 text-xs text-success">
              <CloudUpload className="h-3 w-3" />Saved
            </span>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        <section>
          <button
            onClick={() => setModelOpen(v => !v)}
            className="w-full flex items-center justify-between mb-2 cursor-pointer"
          >
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              Analysis Model
            </span>
            <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${modelOpen ? 'rotate-180' : ''}`} />
          </button>
          {modelOpen && (
            <div className="space-y-2">
              <p className="text-[11px] text-muted-foreground">
                Model used for content analysis. Leave empty for system default.
              </p>
              <ModelSelector
                key={model || '__default__'}
                value={model}
                onChange={onModelChange}
                placeholder="System Default"
              />
              {model && (
                <button
                  onClick={() => onModelChange('')}
                  className="text-[11px] text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  Reset to default
                </button>
              )}
            </div>
          )}
        </section>

        <section>
          <button
            onClick={() => setSourcesOpen(v => !v)}
            className="w-full flex items-center justify-between mb-2 cursor-pointer"
          >
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              Sources &amp; Schedule
            </span>
            <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${sourcesOpen ? 'rotate-180' : ''}`} />
          </button>

          {sourcesOpen && (
            <div className="space-y-4">
              {sources.length === 0 ? (
                <div className="py-4 text-center rounded-xl border border-dashed border-border">
                  <p className="text-xs text-muted-foreground mb-2">No sources configured.</p>
                  <button
                    onClick={() => setShowAddModal(true)}
                    className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
                  >
                    <Plus className="h-3 w-3" /> Add source
                  </button>
                </div>
              ) : (
                <div className="rounded-xl border border-border overflow-hidden">
                  {sources.map((src, i) => (
                    <div key={src.id} className="flex items-center gap-2 px-3 py-2.5 border-b border-border last:border-b-0">
                      <SourceTypeBadge type={src.source_type} />
                      <span className="text-xs font-medium flex-1 truncate">{src.label}</span>
                      <button
                        onClick={() => onSourcesChange(sources.filter((_, j) => j !== i))}
                        className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
                      >
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </div>
                  ))}
                </div>
              )}

              {sources.length > 0 && (
                <button
                  onClick={() => setShowAddModal(true)}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  <Plus className="h-3 w-3" /> Add source
                </button>
              )}

              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">
                  <Clock className="inline h-3 w-3 mr-1" />Schedule
                </label>
                <select
                  value={SCHEDULE_PRESETS.some(p => p.cron === schedule) ? schedule : '__custom__'}
                  onChange={(e) => onScheduleChange(e.target.value === '__custom__' ? '' : e.target.value)}
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-xs outline-none focus:ring-1 focus:ring-ring cursor-pointer"
                >
                  <option value="" disabled>Select schedule…</option>
                  {SCHEDULE_PRESETS.map(p => <option key={p.cron} value={p.cron}>{p.label}</option>)}
                  <option value="__custom__">Custom cron…</option>
                </select>
                <input
                  type="text"
                  value={schedule}
                  onChange={(e) => onScheduleChange(e.target.value)}
                  placeholder="0 */6 * * *"
                  readOnly={SCHEDULE_PRESETS.some(p => p.cron === schedule)}
                  className={`w-full rounded-xl border border-input bg-background px-3 py-2 text-xs font-mono outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground mt-2 ${SCHEDULE_PRESETS.some(p => p.cron === schedule) ? 'text-muted-foreground' : ''}`}
                />
              </div>

            </div>
          )}
        </section>

        <section>
          <button
            onClick={() => setWorkflowsOpen(v => !v)}
            className="w-full flex items-center justify-between mb-2 cursor-pointer"
          >
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              Workflows
            </span>
            <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${workflowsOpen ? 'rotate-180' : ''}`} />
          </button>

          {workflowsOpen && (
            <div className="space-y-3">
              {workflows.length === 0 ? (
                <div className="py-4 text-center rounded-xl border border-dashed border-border">
                  <p className="text-xs text-muted-foreground mb-2">No workflows configured.</p>
                  <button
                    onClick={() => setShowWorkflowPicker(true)}
                    className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
                  >
                    <Plus className="h-3 w-3" /> Add workflow
                  </button>
                </div>
              ) : (
                <div className="rounded-xl border border-border overflow-hidden">
                  {workflows.map((wf, i) => (
                    <div key={wf.workflow_name} className="flex items-center gap-2 px-3 py-2.5 border-b border-border last:border-b-0">
                      <a href={`/workflows?w=${encodeURIComponent(wf.workflow_name)}`} className="text-xs font-medium flex-1 truncate hover:text-primary hover:underline transition-colors">{wf.label || wf.workflow_name}</a>
                      {channels.length > 0 && (
                        <select
                          value={wf.channel_id || ''}
                          onChange={(e) => {
                            const updated = [...workflows]
                            updated[i] = { ...updated[i], channel_id: e.target.value || undefined }
                            onWorkflowsChange(updated)
                          }}
                          className="w-28 rounded-md border border-input bg-background px-1.5 py-1 text-xs outline-none focus:ring-2 focus:ring-ring cursor-pointer"
                        >
                          <option value="">No channel</option>
                          {channels.map(ch => (
                            <option key={ch.id} value={ch.id}>{ch.name}</option>
                          ))}
                        </select>
                      )}
                      <button
                        onClick={() => onWorkflowsChange(workflows.filter((_, j) => j !== i))}
                        className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
                      >
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </div>
                  ))}
                </div>
              )}

              {workflows.length > 0 && (
                <button
                  onClick={() => setShowWorkflowPicker(true)}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  <Plus className="h-3 w-3" /> Add workflow
                </button>
              )}
            </div>
          )}

          {showWorkflowPicker && (
            <WorkflowPicker
              existingWorkflows={workflows}
              onAdd={(wf) => onWorkflowsChange([...workflows, wf])}
              onClose={() => setShowWorkflowPicker(false)}
            />
          )}
        </section>

        <section>
          <button
            onClick={() => setBriefOpen(v => !v)}
            className="w-full flex items-center justify-between mb-2 cursor-pointer"
          >
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              Editorial Brief
            </span>
            <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${briefOpen ? 'rotate-180' : ''}`} />
          </button>
          {briefOpen && (
            <EditorialBriefForm initialContext={context} onSave={onContextSave} autoSave />
          )}
        </section>
      </div>

      {/* AI Assistant — pinned to bottom */}
      <div className="border-t border-border bg-background/80 backdrop-blur-sm">
        <PipelineChatEditor
          pipelineId={pipelineId}
          currentSources={sources}
          currentSchedule={schedule}
          currentWorkflows={workflows}
          currentModel={model}
          currentContext={context}
          onSourcesChange={onSourcesChange}
          onScheduleChange={onScheduleChange}
          onWorkflowsChange={onWorkflowsChange}
          onModelChange={onModelChange}
          onContextSave={onContextSave}
        />
      </div>

      {showAddModal && (
        <AddSourceModal
          onAdd={(src) => onSourcesChange([...sources, src])}
          onClose={() => setShowAddModal(false)}
        />
      )}
    </div>
  )
}
