import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { Header } from '@/components/Header'
import { SchedulerStats } from '@/components/schedules/SchedulerStats'
import { ScheduleList } from '@/components/schedules/ScheduleList'
import { TriggerList } from '@/components/schedules/TriggerList'
import {
  fetchSchedules,
  fetchSchedulerStats,
  pauseSchedule,
  resumeSchedule,
  deleteSchedule,
  deleteTrigger,
  fetchRuns,
  listWorkflows,
  type Schedule,
  type Trigger,
  type SchedulerStats as StatsType,
  type RunRecord,
} from '@/lib/api'

export default function Schedules() {
  const navigate = useNavigate()
  const [schedules, setSchedules] = useState<Schedule[]>([])
  const [triggers, setTriggers] = useState<Trigger[]>([])
  const [stats, setStats] = useState<StatsType | null>(null)
  const [recentRuns, setRecentRuns] = useState<RunRecord[]>([])
  const [workflowNames, setWorkflowNames] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'cron' | 'webhooks'>('cron')

  const reload = async () => {
    try {
      const [sched, st, runsResp, wfs] = await Promise.all([
        fetchSchedules(),
        fetchSchedulerStats().catch(() => null),
        fetchRuns(10, 0),
        listWorkflows(),
      ])
      setSchedules(sched ?? [])
      setStats(st)
      setRecentRuns((runsResp.runs ?? []).filter(r => r.trigger_type !== 'manual'))
      setWorkflowNames(wfs.map(w => w.name))

      // Collect triggers for all workflows
      const allTriggers: Trigger[] = []
      const uniqueNames = [...new Set(wfs.map(w => w.name))]
      await Promise.all(
        uniqueNames.map(async (name) => {
          try {
            const { fetchTriggers } = await import('@/lib/api')
            const t = await fetchTriggers(name)
            allTriggers.push(...t)
          } catch { /* skip */ }
        })
      )
      setTriggers(allTriggers)
    } catch {
      // silently handle
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { reload() }, [])

  const handlePause = async (id: string) => {
    await pauseSchedule(id)
    reload()
  }

  const handleResume = async (id: string) => {
    await resumeSchedule(id)
    reload()
  }

  const handleDeleteSchedule = async (id: string) => {
    if (!confirm('Delete this schedule?')) return
    await deleteSchedule(id)
    reload()
  }

  const handleDeleteTrigger = async (id: string) => {
    if (!confirm('Delete this trigger?')) return
    await deleteTrigger(id)
    reload()
  }

  const handleEditSchedule = (_schedule: Schedule) => {
    // TODO: open edit dialog
  }

  const failedRuns = recentRuns.filter(r => r.status === 'failed').length

  const statusIcon = (status: string) => {
    switch (status) {
      case 'success': return <span className="text-success">&#10003;</span>
      case 'failed': return <span className="text-destructive">&#10007;</span>
      case 'running': return <Loader2 className="h-3 w-3 animate-spin text-info inline" />
      default: return <span className="text-muted-foreground">&#8226;</span>
    }
  }

  return (
    <div className="h-screen flex flex-col bg-background text-foreground">
      <Header />

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto px-6 py-6 space-y-6">
          <h1 className="text-xl font-semibold">Schedules</h1>

          {loading ? (
            <div className="flex items-center justify-center py-20">
              <Loader2 className="animate-spin text-muted-foreground" size={32} />
            </div>
          ) : (
            <>
              <SchedulerStats
                stats={stats}
                totalSchedules={schedules.length}
                failedRuns={failedRuns}
              />

              {/* Tab switcher */}
              <div className="flex gap-1 border-b border-border">
                {(['cron', 'webhooks'] as const).map((tab) => (
                  <button
                    key={tab}
                    onClick={() => setActiveTab(tab)}
                    className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                      activeTab === tab
                        ? 'border-primary text-foreground'
                        : 'border-transparent text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    {tab === 'cron' ? 'Cron Schedules' : 'Webhook Triggers'}
                  </button>
                ))}
              </div>

              {activeTab === 'cron' ? (
                <ScheduleList
                  schedules={schedules}
                  workflowNames={workflowNames}
                  onPause={handlePause}
                  onResume={handleResume}
                  onEdit={handleEditSchedule}
                  onDelete={handleDeleteSchedule}
                  onCreated={reload}
                />
              ) : (
                <TriggerList
                  triggers={triggers}
                  workflowNames={workflowNames}
                  onDelete={handleDeleteTrigger}
                  onCreated={reload}
                />
              )}

              {/* Recent scheduled runs */}
              {recentRuns.length > 0 && (
                <div>
                  <h2 className="text-sm font-medium text-foreground mb-3">Recent Scheduled Runs</h2>
                  <div className="border border-border rounded-lg overflow-hidden">
                    <table className="w-full">
                      <thead>
                        <tr className="bg-muted/50 text-left text-xs text-muted-foreground">
                          <th className="px-4 py-2 font-medium">Status</th>
                          <th className="px-4 py-2 font-medium">Workflow</th>
                          <th className="px-4 py-2 font-medium">Trigger</th>
                          <th className="px-4 py-2 font-medium">Duration</th>
                          <th className="px-4 py-2 font-medium">Time</th>
                        </tr>
                      </thead>
                      <tbody>
                        {recentRuns.map((run) => (
                          <tr
                            key={run.id}
                            className="border-t border-border hover:bg-muted/30 cursor-pointer transition-colors"
                            onClick={() => navigate(`/runs/${run.id}`)}
                          >
                            <td className="px-4 py-2 text-sm">
                              {statusIcon(run.status)}
                              <span className="ml-1.5 text-xs capitalize">{run.status}</span>
                            </td>
                            <td className="px-4 py-2 text-sm text-foreground">{run.workflow_name}</td>
                            <td className="px-4 py-2 text-xs text-muted-foreground capitalize">{run.trigger_type}</td>
                            <td className="px-4 py-2 text-xs text-muted-foreground font-mono">
                              {run.started_at && run.completed_at
                                ? `${((new Date(run.completed_at).getTime() - new Date(run.started_at).getTime()) / 1000).toFixed(1)}s`
                                : '-'}
                            </td>
                            <td className="px-4 py-2 text-xs text-muted-foreground">
                              {new Date(run.created_at).toLocaleString(undefined, {
                                month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
                              })}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
