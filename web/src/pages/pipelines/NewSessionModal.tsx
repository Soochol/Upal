import { useState } from 'react'
import { X, ChevronDown } from 'lucide-react'
import { SCHEDULE_PRESETS } from '@/entities/pipeline'
import { ModelSelector } from '@/shared/ui/ModelSelector'
import { WorkflowPicker } from './WorkflowPicker'

export type NewSessionData = {
  name: string
  description: string
  prompt: string
  model: string
  workflow: string
  schedule: string
}

type Props = {
  onSave: (data: NewSessionData) => void
  onClose: () => void
}

export function NewSessionModal({ onSave, onClose }: Props) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [prompt, setPrompt] = useState('')
  const [model, setModel] = useState('')
  const [workflow, setWorkflow] = useState('')
  const [schedule, setSchedule] = useState('')
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)

  const isValid = name.trim() && description.trim() && prompt.trim()

  const handleSave = () => {
    if (!isValid) return
    onSave({
      name: name.trim(),
      description: description.trim(),
      prompt: prompt.trim(),
      model,
      workflow,
      schedule,
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-lg mx-4 overflow-hidden">

        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <h2 className="text-sm font-semibold">Create Scheduled Task</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-4 max-h-[70vh] overflow-y-auto">

          {/* Name */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">
              Name <span className="text-destructive">*</span>
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="daily-briefing"
              autoFocus
              className="w-full px-3 py-2 rounded-lg border border-input bg-background text-sm outline-none
                focus:ring-1 focus:ring-ring placeholder:text-muted-foreground/40"
            />
          </div>

          {/* Description */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">
              Description <span className="text-destructive">*</span>
            </label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What does this task do?"
              className="w-full px-3 py-2 rounded-lg border border-input bg-background text-sm outline-none
                focus:ring-1 focus:ring-ring placeholder:text-muted-foreground/40"
            />
          </div>

          {/* Prompt */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">
              Prompt <span className="text-destructive">*</span>
            </label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Tell the AI what to do..."
              rows={4}
              className="w-full px-3 py-2 rounded-lg border border-input bg-background text-sm outline-none
                focus:ring-1 focus:ring-ring placeholder:text-muted-foreground/40 resize-none"
            />
          </div>

          {/* Model */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">Model</label>
            <ModelSelector
              value={model}
              onChange={setModel}
              placeholder="Default"
            />
          </div>

          {/* Workflow */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">Workflow</label>
            <button
              onClick={() => setShowWorkflowPicker(true)}
              className="w-full flex items-center justify-between px-3 py-2 rounded-lg border border-input
                bg-background text-sm hover:bg-muted/30 transition-colors cursor-pointer"
            >
              <span className={workflow ? 'text-foreground' : 'text-muted-foreground/40'}>
                {workflow || 'Select workflow...'}
              </span>
              <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/40" />
            </button>
          </div>

          {/* Schedule */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1.5">Schedule</label>
            <div className="relative">
              <select
                value={schedule}
                onChange={(e) => setSchedule(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-input bg-background text-sm outline-none
                  cursor-pointer appearance-none [-webkit-appearance:none] pr-8"
              >
                <option value="">Select...</option>
                {SCHEDULE_PRESETS.map(p => (
                  <option key={p.cron} value={p.cron}>{p.label}</option>
                ))}
              </select>
              <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/40 pointer-events-none" />
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-border">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg text-sm font-medium text-muted-foreground
              hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={!isValid}
            className="px-4 py-2 rounded-lg text-sm font-medium bg-foreground text-background
              hover:opacity-90 transition-opacity cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Save
          </button>
        </div>
      </div>

      {/* Workflow picker sub-modal */}
      {showWorkflowPicker && (
        <WorkflowPicker
          mode="select"
          currentWorkflow={workflow}
          onSelect={setWorkflow}
          onClose={() => setShowWorkflowPicker(false)}
        />
      )}
    </div>
  )
}
