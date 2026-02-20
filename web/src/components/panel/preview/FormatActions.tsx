import { Button } from '@/components/ui/button'
import type { FormatAction } from '@/lib/outputFormats'

type FormatActionsProps = {
  actions: FormatAction[]
  content: string
  workflowName: string
}

export function FormatActions({ actions, content, workflowName }: FormatActionsProps) {
  return (
    <div className="flex items-center gap-1 shrink-0">
      {actions.map((action) => (
        <Button
          key={action.id}
          variant="ghost"
          size="sm"
          className="h-6 px-2 text-[10px]"
          onClick={() => action.handler(content, workflowName)}
        >
          <action.icon className="h-3 w-3 mr-1" />
          {action.label}
        </Button>
      ))}
    </div>
  )
}
