import { useWorkflowStore } from '@/stores/workflowStore'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Eye } from 'lucide-react'

export function PanelPreview() {
  const runEvents = useWorkflowStore((s) => s.runEvents)

  const doneEvent = runEvents.find((e) => e.type === 'done')
  const outputEvents = runEvents.filter(
    (e) => e.type === 'node.completed' && e.data.output,
  )

  if (!doneEvent && outputEvents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-6">
        <Eye className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm text-center">
          Run a workflow to see results here.
        </p>
      </div>
    )
  }

  return (
    <ScrollArea className="h-full">
      <div className="p-4 space-y-4">
        {outputEvents.map((event, i) => (
          <div key={i} className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground">
              {(event.data.node_id as string) || `Step ${i + 1}`}
            </p>
            <div className="rounded-lg border border-border bg-card p-3 text-sm whitespace-pre-wrap">
              {typeof event.data.output === 'string'
                ? event.data.output
                : JSON.stringify(event.data.output, null, 2)}
            </div>
          </div>
        ))}
        {doneEvent && (
          <div className="space-y-1">
            <p className="text-xs font-medium text-node-output">Final Result</p>
            <div className="rounded-lg border border-node-output/30 bg-node-output/5 p-3 text-sm whitespace-pre-wrap">
              {typeof doneEvent.data.result === 'string'
                ? doneEvent.data.result
                : JSON.stringify(doneEvent.data, null, 2)}
            </div>
          </div>
        )}
      </div>
    </ScrollArea>
  )
}
