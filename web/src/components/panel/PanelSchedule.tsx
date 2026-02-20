import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useWorkflowStore } from '@/stores/workflowStore'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Clock, Plus, Trash2, Pause, Play, Webhook, Copy, Check, ExternalLink, Pencil, X,
} from 'lucide-react'
import {
  fetchSchedules, createSchedule, updateSchedule, deleteSchedule, pauseSchedule, resumeSchedule,
  fetchTriggers, createTrigger, deleteTrigger,
  type Schedule, type Trigger,
} from '@/lib/api'

const cronPresets = [
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Daily at midnight', value: '0 0 * * *' },
  { label: 'Weekly (Mon 9am)', value: '0 9 * * 1' },
]

const timezones = [
  'UTC', 'America/New_York', 'America/Los_Angeles', 'Europe/London',
  'Europe/Berlin', 'Asia/Tokyo', 'Asia/Seoul', 'Asia/Shanghai',
]

export function PanelSchedule() {
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const [schedules, setSchedules] = useState<Schedule[]>([])
  const [triggers, setTriggers] = useState<Trigger[]>([])
  const [loading, setLoading] = useState(true)
  const [cronExpr, setCronExpr] = useState('0 * * * *')
  const [timezone, setTimezone] = useState('UTC')
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editCron, setEditCron] = useState('')
  const [editTz, setEditTz] = useState('UTC')

  const reload = () => {
    if (!workflowName) return
    setLoading(true)
    Promise.all([
      fetchSchedules().then((all) => all.filter((s) => s.workflow_name === workflowName)),
      fetchTriggers(workflowName),
    ])
      .then(([s, t]) => { setSchedules(s); setTriggers(t) })
      .catch(() => { setSchedules([]); setTriggers([]) })
      .finally(() => setLoading(false))
  }

  useEffect(() => { reload() }, [workflowName]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleAddSchedule = async () => {
    if (!workflowName || !cronExpr.trim()) return
    try {
      await createSchedule({ workflow_name: workflowName, cron_expr: cronExpr.trim(), timezone })
      reload()
    } catch (err) {
      console.error('Failed to create schedule', err)
    }
  }

  const handleEditSchedule = async (id: string) => {
    if (!editCron.trim()) return
    try {
      await updateSchedule(id, { cron_expr: editCron.trim(), timezone: editTz })
      setEditingId(null)
      reload()
    } catch (err) {
      console.error('Failed to update schedule', err)
    }
  }

  const startEditing = (s: Schedule) => {
    setEditingId(s.id)
    setEditCron(s.cron_expr)
    setEditTz(s.timezone || 'UTC')
  }

  const handleToggle = async (s: Schedule) => {
    try {
      if (s.enabled) await pauseSchedule(s.id)
      else await resumeSchedule(s.id)
      reload()
    } catch (err) {
      console.error('Failed to toggle schedule', err)
    }
  }

  const handleDeleteSchedule = async (id: string) => {
    try {
      await deleteSchedule(id)
      reload()
    } catch (err) {
      console.error('Failed to delete schedule', err)
    }
  }

  const handleAddTrigger = async () => {
    if (!workflowName) return
    try {
      await createTrigger({ workflow_name: workflowName, type: 'webhook' })
      reload()
    } catch (err) {
      console.error('Failed to create trigger', err)
    }
  }

  const handleDeleteTrigger = async (id: string) => {
    try {
      await deleteTrigger(id)
      reload()
    } catch (err) {
      console.error('Failed to delete trigger', err)
    }
  }

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  if (!workflowName) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-6">
        <Clock className="w-8 h-8 mb-2 opacity-40" />
        <p className="text-xs">Save workflow first to manage schedules.</p>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-32 text-muted-foreground">
        <span className="text-xs animate-pulse">Loading...</span>
      </div>
    )
  }

  return (
    <ScrollArea className="h-full">
      <div className="p-4 space-y-6">
        {/* ─── Cron Schedules ─── */}
        <section>
          <h3 className="text-xs font-semibold text-foreground mb-3 flex items-center gap-1.5">
            <Clock className="w-3.5 h-3.5" />
            Cron Schedules
          </h3>

          {/* Add schedule */}
          <div className="flex gap-2 mb-3">
            <select
              value={cronExpr}
              onChange={(e) => setCronExpr(e.target.value)}
              className="flex-1 text-xs bg-muted/50 border border-border rounded-md px-2 py-1.5 text-foreground"
            >
              {cronPresets.map((p) => (
                <option key={p.value} value={p.value}>{p.label} ({p.value})</option>
              ))}
            </select>
            <button
              onClick={handleAddSchedule}
              className="p-1.5 rounded-md bg-primary text-primary-foreground hover:opacity-90 transition-opacity"
              title="Add schedule"
            >
              <Plus className="w-3.5 h-3.5" />
            </button>
          </div>

          {/* Custom cron input */}
          <input
            type="text"
            value={cronExpr}
            onChange={(e) => setCronExpr(e.target.value)}
            placeholder="0 */6 * * *"
            className="w-full text-xs bg-muted/30 border border-border rounded-md px-2 py-1.5 text-foreground placeholder:text-muted-foreground/50 font-mono mb-2"
          />

          {/* Timezone */}
          <select
            value={timezone}
            onChange={(e) => setTimezone(e.target.value)}
            className="w-full text-xs bg-muted/30 border border-border rounded-md px-2 py-1.5 text-foreground mb-3"
          >
            {timezones.map((tz) => (
              <option key={tz} value={tz}>{tz}</option>
            ))}
          </select>

          {/* Schedule list */}
          {schedules.length === 0 ? (
            <p className="text-xs text-muted-foreground/60">No schedules configured.</p>
          ) : (
            <div className="space-y-2">
              {schedules.map((s) => (
                <div key={s.id} className="p-2 rounded-lg bg-muted/30 border border-border space-y-1.5">
                  {editingId === s.id ? (
                    /* Edit mode */
                    <div className="space-y-1.5">
                      <input
                        type="text"
                        value={editCron}
                        onChange={(e) => setEditCron(e.target.value)}
                        className="w-full text-xs bg-background border border-border rounded-md px-2 py-1 text-foreground font-mono"
                      />
                      <select
                        value={editTz}
                        onChange={(e) => setEditTz(e.target.value)}
                        className="w-full text-xs bg-background border border-border rounded-md px-2 py-1 text-foreground"
                      >
                        {timezones.map((tz) => (
                          <option key={tz} value={tz}>{tz}</option>
                        ))}
                      </select>
                      <div className="flex gap-1">
                        <button
                          onClick={() => handleEditSchedule(s.id)}
                          className="flex-1 text-xs py-1 rounded-md bg-primary text-primary-foreground hover:opacity-90"
                        >
                          Save
                        </button>
                        <button
                          onClick={() => setEditingId(null)}
                          className="p-1 rounded hover:bg-muted text-muted-foreground"
                        >
                          <X className="w-3 h-3" />
                        </button>
                      </div>
                    </div>
                  ) : (
                    /* Display mode */
                    <div className="flex items-center gap-2">
                      <code className="text-xs font-mono text-foreground flex-1 truncate">{s.cron_expr}</code>
                      {s.timezone && s.timezone !== 'UTC' && (
                        <span className="text-[10px] text-muted-foreground">{s.timezone}</span>
                      )}
                      <span className={`text-[10px] px-1.5 py-0.5 rounded-full ${s.enabled ? 'bg-green-500/15 text-green-500' : 'bg-muted text-muted-foreground'}`}>
                        {s.enabled ? 'active' : 'paused'}
                      </span>
                      <button onClick={() => startEditing(s)} className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors" title="Edit">
                        <Pencil className="w-3 h-3" />
                      </button>
                      <button onClick={() => handleToggle(s)} className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors" title={s.enabled ? 'Pause' : 'Resume'}>
                        {s.enabled ? <Pause className="w-3 h-3" /> : <Play className="w-3 h-3" />}
                      </button>
                      <button onClick={() => handleDeleteSchedule(s.id)} className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors" title="Delete">
                        <Trash2 className="w-3 h-3" />
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </section>

        {/* ─── Webhook Triggers ─── */}
        <section>
          <h3 className="text-xs font-semibold text-foreground mb-3 flex items-center gap-1.5">
            <Webhook className="w-3.5 h-3.5" />
            Webhook Triggers
          </h3>

          <button
            onClick={handleAddTrigger}
            className="flex items-center gap-1.5 text-xs text-primary hover:text-primary/80 transition-colors mb-3"
          >
            <Plus className="w-3 h-3" />
            Add Webhook
          </button>

          {triggers.length === 0 ? (
            <p className="text-xs text-muted-foreground/60">No webhook triggers configured.</p>
          ) : (
            <div className="space-y-2">
              {triggers.map((t) => {
                const hookUrl = `${window.location.origin}/api/hooks/${t.id}`
                return (
                  <div key={t.id} className="p-2 rounded-lg bg-muted/30 border border-border space-y-1.5">
                    <div className="flex items-center gap-2">
                      <span className="text-[10px] text-muted-foreground font-mono truncate flex-1">{t.id}</span>
                      <button onClick={() => handleDeleteTrigger(t.id)} className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors" title="Delete">
                        <Trash2 className="w-3 h-3" />
                      </button>
                    </div>
                    <div className="flex items-center gap-1">
                      <code className="text-[10px] font-mono text-foreground/70 bg-muted/50 px-1.5 py-0.5 rounded truncate flex-1">
                        {hookUrl}
                      </code>
                      <button
                        onClick={() => copyToClipboard(hookUrl, `url-${t.id}`)}
                        className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                        title="Copy URL"
                      >
                        {copiedId === `url-${t.id}` ? <Check className="w-3 h-3 text-green-500" /> : <Copy className="w-3 h-3" />}
                      </button>
                    </div>
                    {t.config.secret && (
                      <div className="flex items-center gap-1">
                        <span className="text-[10px] text-muted-foreground">Secret:</span>
                        <code className="text-[10px] font-mono text-foreground/50 truncate flex-1">
                          {t.config.secret.slice(0, 8)}...
                        </code>
                        <button
                          onClick={() => copyToClipboard(t.config.secret!, `secret-${t.id}`)}
                          className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                          title="Copy secret"
                        >
                          {copiedId === `secret-${t.id}` ? <Check className="w-3 h-3 text-green-500" /> : <Copy className="w-3 h-3" />}
                        </button>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </section>
        {/* View all link */}
        <Link
          to="/schedules"
          className="flex items-center gap-1 text-xs text-primary hover:text-primary/80 transition-colors"
        >
          <ExternalLink className="w-3 h-3" />
          View all schedules
        </Link>
      </div>
    </ScrollArea>
  )
}
