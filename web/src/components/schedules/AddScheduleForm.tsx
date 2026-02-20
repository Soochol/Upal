import { useState } from 'react'
import { Plus, ChevronDown, ChevronUp } from 'lucide-react'
import { createSchedule } from '@/lib/api'

const CRON_PRESETS = [
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Daily at midnight', value: '0 0 * * *' },
  { label: 'Weekly Mon 9am', value: '0 9 * * 1' },
]

const TIMEZONES = [
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Berlin',
  'Asia/Tokyo',
  'Asia/Seoul',
  'Asia/Shanghai',
  'Australia/Sydney',
]

type Props = {
  workflowNames: string[]
  onCreated: () => void
}

export function AddScheduleForm({ workflowNames, onCreated }: Props) {
  const [open, setOpen] = useState(false)
  const [workflowName, setWorkflowName] = useState('')
  const [cronExpr, setCronExpr] = useState('0 * * * *')
  const [timezone, setTimezone] = useState('UTC')
  const [showRetry, setShowRetry] = useState(false)
  const [maxRetries, setMaxRetries] = useState(3)
  const [creating, setCreating] = useState(false)

  const handleCreate = async () => {
    if (!workflowName || !cronExpr) return
    setCreating(true)
    try {
      await createSchedule({
        workflow_name: workflowName,
        cron_expr: cronExpr,
        timezone,
        enabled: true,
        ...(showRetry && {
          retry_policy: {
            max_retries: maxRetries,
            initial_delay: 5,
            max_delay: 300,
            backoff_factor: 2,
          },
        }),
      })
      setOpen(false)
      setCronExpr('0 * * * *')
      setWorkflowName('')
      setTimezone('UTC')
      setShowRetry(false)
      onCreated()
    } catch {
      // error handled by caller
    } finally {
      setCreating(false)
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="flex items-center gap-1.5 px-3 py-2 text-sm text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded-md transition-colors w-full"
      >
        <Plus className="h-3.5 w-3.5" />
        Add Schedule
      </button>
    )
  }

  return (
    <div className="border border-border rounded-lg p-4 bg-card space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-foreground">New Schedule</h3>
        <button
          onClick={() => setOpen(false)}
          className="text-xs text-muted-foreground hover:text-foreground"
        >
          Cancel
        </button>
      </div>

      {/* Workflow select */}
      <div>
        <label className="text-xs text-muted-foreground mb-1 block">Workflow</label>
        <select
          value={workflowName}
          onChange={(e) => setWorkflowName(e.target.value)}
          data-testid="workflow-select"
          className="w-full h-8 px-2 rounded-md border border-border bg-background text-sm text-foreground"
        >
          <option value="">Select workflow...</option>
          {workflowNames.map((n) => (
            <option key={n} value={n}>{n}</option>
          ))}
        </select>
      </div>

      {/* Cron expression */}
      <div>
        <label className="text-xs text-muted-foreground mb-1 block">Cron Expression</label>
        <div className="flex gap-2 mb-2">
          {CRON_PRESETS.map((p) => (
            <button
              key={p.value}
              onClick={() => setCronExpr(p.value)}
              className={`px-2 py-1 rounded text-xs transition-colors ${
                cronExpr === p.value
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:text-foreground'
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
        <input
          type="text"
          value={cronExpr}
          onChange={(e) => setCronExpr(e.target.value)}
          data-testid="cron-input"
          className="w-full h-8 px-2 rounded-md border border-border bg-background text-sm font-mono text-foreground"
          placeholder="0 * * * *"
        />
      </div>

      {/* Timezone */}
      <div>
        <label className="text-xs text-muted-foreground mb-1 block">Timezone</label>
        <select
          value={timezone}
          onChange={(e) => setTimezone(e.target.value)}
          data-testid="timezone-select"
          className="w-full h-8 px-2 rounded-md border border-border bg-background text-sm text-foreground"
        >
          {TIMEZONES.map((tz) => (
            <option key={tz} value={tz}>{tz}</option>
          ))}
        </select>
      </div>

      {/* Retry policy toggle */}
      <button
        onClick={() => setShowRetry(!showRetry)}
        className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
      >
        {showRetry ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
        Retry Policy
      </button>
      {showRetry && (
        <div>
          <label className="text-xs text-muted-foreground mb-1 block">Max Retries</label>
          <input
            type="number"
            min={0}
            max={10}
            value={maxRetries}
            onChange={(e) => setMaxRetries(Number(e.target.value))}
            data-testid="max-retries-input"
            className="w-20 h-8 px-2 rounded-md border border-border bg-background text-sm text-foreground"
          />
        </div>
      )}

      {/* Create button */}
      <button
        onClick={handleCreate}
        disabled={!workflowName || !cronExpr || creating}
        className="w-full h-8 rounded-md bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50 transition-opacity"
      >
        {creating ? 'Creating...' : 'Create Schedule'}
      </button>
    </div>
  )
}
