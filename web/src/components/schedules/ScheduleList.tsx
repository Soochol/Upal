import { CalendarClock } from 'lucide-react'
import type { Schedule } from '@/lib/api'
import { ScheduleRow } from './ScheduleRow'
import { AddScheduleForm } from './AddScheduleForm'

type Props = {
  schedules: Schedule[]
  workflowNames: string[]
  onPause: (id: string) => void
  onResume: (id: string) => void
  onEdit: (schedule: Schedule) => void
  onDelete: (id: string) => void
  onCreated: () => void
}

export function ScheduleList({
  schedules,
  workflowNames,
  onPause,
  onResume,
  onEdit,
  onDelete,
  onCreated,
}: Props) {
  return (
    <div className="space-y-3">
      {schedules.length === 0 ? (
        <div className="text-center py-10">
          <CalendarClock className="h-8 w-8 text-muted-foreground/40 mx-auto mb-2" />
          <p className="text-sm text-muted-foreground">No schedules configured.</p>
        </div>
      ) : (
        <div className="border border-border rounded-lg overflow-hidden">
          {schedules.map((s) => (
            <ScheduleRow
              key={s.id}
              schedule={s}
              onPause={onPause}
              onResume={onResume}
              onEdit={onEdit}
              onDelete={onDelete}
            />
          ))}
        </div>
      )}
      <AddScheduleForm workflowNames={workflowNames} onCreated={onCreated} />
    </div>
  )
}
