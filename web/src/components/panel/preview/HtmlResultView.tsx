import { OUTPUT_FORMATS, type ResultViewProps } from '@/lib/outputFormats'
import { FormatActions } from './FormatActions'
import { HtmlPreview } from './HtmlPreview'

export default function HtmlResultView({ content, workflowName }: ResultViewProps) {
  return (
    <div className="flex-1 min-h-0 p-2 flex flex-col gap-1.5">
      <FormatActions
        actions={OUTPUT_FORMATS.html.actions}
        content={content}
        workflowName={workflowName}
      />
      <div className="flex-1 min-h-0">
        <HtmlPreview html={content} />
      </div>
    </div>
  )
}
