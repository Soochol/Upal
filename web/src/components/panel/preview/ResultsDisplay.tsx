import type { RunEvent } from '@/lib/api'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { ExternalLink, Download } from 'lucide-react'
import { ImagePreview } from './ImagePreview'
import { HtmlPreview } from './HtmlPreview'

function isHtmlContent(text: string): boolean {
  const trimmed = text.trimStart().toLowerCase()
  return trimmed.startsWith('<!doctype') || trimmed.startsWith('<html')
}

function isImageDataURI(text: string): boolean {
  return text.startsWith('data:image/')
}

function containsImageDataURI(text: string): boolean {
  return text.split('\n').some((line) => isImageDataURI(line.trim()))
}

type ResultsDisplayProps = {
  sessionState: Record<string, unknown>
  doneEvent: RunEvent | undefined
  workflowName: string
}

export function ResultsDisplay({ sessionState, doneEvent, workflowName }: ResultsDisplayProps) {
  const stateEntries = Object.entries(sessionState).filter(
    ([, v]) => v != null && v !== '',
  )
  const htmlOutput = stateEntries.find(
    ([, v]) => typeof v === 'string' && isHtmlContent(v),
  )?.[1] as string | undefined
  const imageOutputs = stateEntries.filter(
    ([, v]) => typeof v === 'string' && containsImageDataURI(v as string),
  ) as [string, string][]
  const textOutputs = stateEntries.filter(
    ([, v]) => typeof v === 'string' && !isHtmlContent(v as string) && !containsImageDataURI(v as string),
  ) as [string, string][]

  if (htmlOutput) {
    return (
      <div className="flex-1 min-h-0 p-2 flex flex-col gap-1.5">
        <div className="flex items-center gap-1 shrink-0">
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-[10px]"
            onClick={() => {
              const blob = new Blob([htmlOutput], { type: 'text/html' })
              const url = URL.createObjectURL(blob)
              window.open(url, '_blank')
              setTimeout(() => URL.revokeObjectURL(url), 1000)
            }}
          >
            <ExternalLink className="h-3 w-3 mr-1" />
            Open in tab
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-[10px]"
            onClick={() => {
              const blob = new Blob([htmlOutput], { type: 'text/html' })
              const url = URL.createObjectURL(blob)
              const a = document.createElement('a')
              a.href = url
              a.download = `${workflowName || 'output'}.html`
              a.click()
              URL.revokeObjectURL(url)
            }}
          >
            <Download className="h-3 w-3 mr-1" />
            Save
          </Button>
        </div>
        <div className="flex-1 min-h-0">
          <HtmlPreview html={htmlOutput} />
        </div>
      </div>
    )
  }

  return (
    <ScrollArea className="flex-1 min-h-0">
      <div className="p-3 space-y-3">
        {imageOutputs.map(([key, value]) => {
          const lines = value.split('\n')
          return (
            <div key={key} className="space-y-1">
              <p className="text-[10px] font-medium text-muted-foreground">{key}</p>
              {lines.map((line, i) => {
                const trimmed = line.trim()
                if (isImageDataURI(trimmed)) {
                  return <ImagePreview key={i} dataURI={trimmed} workflowName={workflowName} />
                }
                if (trimmed) {
                  return (
                    <div key={i} className="rounded-lg border border-border bg-card p-2.5 text-xs whitespace-pre-wrap">
                      {trimmed}
                    </div>
                  )
                }
                return null
              })}
            </div>
          )
        })}
        {textOutputs.map(([key, value]) => (
          <div key={key} className="space-y-1">
            <p className="text-[10px] font-medium text-muted-foreground">
              {key}
            </p>
            <div className="rounded-lg border border-border bg-card p-2.5 text-xs whitespace-pre-wrap">
              {value}
            </div>
          </div>
        ))}
        {doneEvent && doneEvent.type === 'done' && textOutputs.length === 0 && imageOutputs.length === 0 && (
          <div className="space-y-1">
            <p className="text-[10px] font-medium text-node-output">
              Completed
            </p>
            <div className="rounded-lg border border-node-output/30 bg-node-output/5 p-2.5 text-xs whitespace-pre-wrap">
              {doneEvent.status === 'completed'
                ? 'Workflow completed successfully.'
                : `status=${doneEvent.status}`}
            </div>
          </div>
        )}
      </div>
    </ScrollArea>
  )
}
