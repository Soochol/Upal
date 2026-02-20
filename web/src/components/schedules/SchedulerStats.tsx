import { Activity, CalendarClock, AlertTriangle } from 'lucide-react'
import type { SchedulerStats as StatsType } from '@/lib/api'

type Props = {
  stats: StatsType | null
  totalSchedules: number
  failedRuns: number
}

export function SchedulerStats({ stats, totalSchedules, failedRuns }: Props) {
  const cards = [
    {
      label: 'Active Runs',
      value: stats?.concurrency?.active_runs ?? 0,
      icon: Activity,
      color: 'text-blue-500',
    },
    {
      label: 'Total Schedules',
      value: totalSchedules,
      icon: CalendarClock,
      color: 'text-foreground',
    },
    {
      label: 'Recent Failures',
      value: failedRuns,
      icon: AlertTriangle,
      color: failedRuns > 0 ? 'text-red-500' : 'text-muted-foreground',
    },
  ]

  return (
    <div className="grid grid-cols-3 gap-4">
      {cards.map((c) => {
        const Icon = c.icon
        return (
          <div
            key={c.label}
            className="border border-border rounded-lg p-4 bg-card"
          >
            <div className="flex items-center gap-2 mb-1">
              <Icon className={`h-4 w-4 ${c.color}`} />
              <span className="text-xs text-muted-foreground">{c.label}</span>
            </div>
            <p className={`text-2xl font-semibold ${c.color}`}>{c.value}</p>
          </div>
        )
      })}
    </div>
  )
}
