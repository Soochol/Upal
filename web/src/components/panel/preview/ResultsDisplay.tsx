import { Suspense, lazy, useMemo } from 'react'
import type { RunEvent } from '@/lib/api'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useWorkflowStore } from '@/stores/workflowStore'
import { resolveFormat, type OutputFormatDef } from '@/lib/outputFormats'
import { ImagePreview } from './ImagePreview'

function isImageDataURI(text: string): boolean {
  return text.startsWith('data:image/')
}

function containsImageDataURI(text: string): boolean {
  return text.split('\n').some((line) => isImageDataURI(line.trim()))
}

/** Read output_format from the first output node's config. */
function useOutputFormat(): string | undefined {
  return useWorkflowStore((s) => {
    const outputNode = s.nodes.find((n) => n.data.nodeType === 'output')
    return outputNode?.data.config.output_format as string | undefined
  })
}

/** Lazily resolve the ResultView component from the format registry. */
function useLazyResultView(format: OutputFormatDef) {
  return useMemo(
    () => lazy(format.ResultView),
    [format],
  )
}

type ResultsDisplayProps = {
  sessionState: Record<string, unknown>
  doneEvent: RunEvent | undefined
  workflowName: string
}

export function ResultsDisplay({ sessionState, doneEvent, workflowName }: ResultsDisplayProps) {
  const outputFormat = useOutputFormat()

  const stateEntries = Object.entries(sessionState).filter(
    ([, v]) => v != null && v !== '',
  )

  // Separate image outputs (always rendered with ImagePreview, format-independent)
  const imageOutputs = stateEntries.filter(
    ([, v]) => typeof v === 'string' && containsImageDataURI(v as string),
  ) as [string, string][]

  // Find the primary output content (non-image string from output node)
  const primaryOutput = stateEntries.find(
    ([, v]) => typeof v === 'string' && !containsImageDataURI(v as string),
  )

  // Resolve format from config or auto-detect from content
  const format = primaryOutput
    ? resolveFormat(outputFormat, primaryOutput[1] as string)
    : null

  return (
    <>
      {/* Format-specific primary output */}
      {format && primaryOutput && (
        <FormatResultView
          format={format}
          content={primaryOutput[1] as string}
          workflowName={workflowName}
        />
      )}

      {/* Image outputs (format-independent) */}
      {imageOutputs.length > 0 && (
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
          </div>
        </ScrollArea>
      )}

      {/* Fallback: done event with no other output */}
      {!primaryOutput && imageOutputs.length === 0 && doneEvent && doneEvent.type === 'done' && (
        <ScrollArea className="flex-1 min-h-0">
          <div className="p-3 space-y-3">
            <div className="space-y-1">
              <p className="text-[10px] font-medium text-node-output">Completed</p>
              <div className="rounded-lg border border-node-output/30 bg-node-output/5 p-2.5 text-xs whitespace-pre-wrap">
                {doneEvent.status === 'completed'
                  ? 'Workflow completed successfully.'
                  : `status=${doneEvent.status}`}
              </div>
            </div>
          </div>
        </ScrollArea>
      )}
    </>
  )
}

/** Renders the lazy-loaded format-specific ResultView with Suspense. */
function FormatResultView({ format, content, workflowName }: {
  format: OutputFormatDef
  content: string
  workflowName: string
}) {
  const LazyView = useLazyResultView(format)
  return (
    <Suspense fallback={<div className="p-3 text-xs text-muted-foreground">Loading...</div>}>
      <LazyView content={content} workflowName={workflowName} />
    </Suspense>
  )
}
