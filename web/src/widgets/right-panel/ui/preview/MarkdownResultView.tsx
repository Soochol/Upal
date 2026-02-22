import { OUTPUT_FORMATS, type ResultViewProps } from '@/shared/lib/outputFormats'
import { ScrollArea } from '@/shared/ui/scroll-area'
import { FormatActions } from './FormatActions'

export default function MarkdownResultView({ content, workflowName }: ResultViewProps) {
  return (
    <div className="flex-1 min-h-0 flex flex-col gap-1.5">
      <div className="px-2 pt-2">
        <FormatActions
          actions={OUTPUT_FORMATS.md.actions}
          content={content}
          workflowName={workflowName}
        />
      </div>
      <ScrollArea className="flex-1 min-h-0">
        <div className="px-3 pb-3 text-xs whitespace-pre-wrap break-words">
          {content}
        </div>
      </ScrollArea>
    </div>
  )
}
