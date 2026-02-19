import { useState } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useExecutionStore } from '@/stores/executionStore'
import { useExecuteRun } from '@/hooks/useExecuteRun'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Play, Loader2, Eye, ArrowLeft, ExternalLink, Download } from 'lucide-react'

function isHtmlContent(text: string): boolean {
  const trimmed = text.trimStart().toLowerCase()
  return trimmed.startsWith('<!doctype') || trimmed.startsWith('<html')
}

function HtmlPreview({ html }: { html: string }) {
  return (
    <iframe
      srcDoc={html}
      sandbox="allow-scripts allow-same-origin"
      className="w-full h-full border-0 rounded-lg bg-white"
      title="Auto-layout preview"
    />
  )
}

type Phase = 'idle' | 'collecting' | 'running'

export function PanelPreview() {
  const nodes = useWorkflowStore((s) => s.nodes)
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const runEvents = useExecutionStore((s) => s.runEvents)
  const sessionState = useExecutionStore((s) => s.sessionState)
  const { executeRun, isRunning } = useExecuteRun()

  const [phase, setPhase] = useState<Phase>('idle')
  const [inputs, setInputs] = useState<Record<string, string>>({})

  // Find input nodes
  const inputNodes = nodes
    .filter((n) => n.data.nodeType === 'input')
    .map((n) => ({ id: n.id, label: n.data.label }))

  const handleRunClick = () => {
    if (!workflowName || isRunning) return

    // If there are input nodes, show collection form first
    if (inputNodes.length > 0 && phase !== 'collecting') {
      setPhase('collecting')
      return
    }

    // No inputs needed or inputs already collected — execute
    setPhase('running')
    executeRun(inputs)
  }

  const doneEvent = runEvents.find((e) => e.type === 'done')

  // Extract results from session state (populated by backend done event)
  const stateEntries = Object.entries(sessionState).filter(
    ([, v]) => v != null && v !== '',
  )
  const htmlOutput = stateEntries.find(
    ([, v]) => typeof v === 'string' && isHtmlContent(v),
  )?.[1] as string | undefined
  const textOutputs = stateEntries.filter(
    ([, v]) => typeof v === 'string' && !isHtmlContent(v as string),
  ) as [string, string][]

  const hasResults = doneEvent || stateEntries.length > 0

  return (
    <div className="flex flex-col h-full">
      {/* Phase: collecting inputs */}
      {phase === 'collecting' && (
        <div className="p-3 space-y-2.5 shrink-0">
          <div className="flex items-center gap-1.5 mb-1">
            <button
              onClick={() => setPhase('idle')}
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              <ArrowLeft className="h-3.5 w-3.5" />
            </button>
            <p className="text-xs font-medium">Provide inputs</p>
          </div>
          {inputNodes.map((node) => (
            <div key={node.id} className="space-y-1">
              <Label htmlFor={`preview-${node.id}`} className="text-xs">
                {node.label}
              </Label>
              <Textarea
                id={`preview-${node.id}`}
                value={inputs[node.id] ?? ''}
                onChange={(e) =>
                  setInputs((prev) => ({ ...prev, [node.id]: e.target.value }))
                }
                placeholder={`Enter value for ${node.label}...`}
                className="min-h-[48px] resize-y text-xs"
              />
            </div>
          ))}
          <Button
            size="sm"
            className="w-full"
            onClick={handleRunClick}
          >
            <Play className="h-3.5 w-3.5 mr-1.5" />
            Run
          </Button>
        </div>
      )}

      {/* Phase: idle — just the Run button */}
      {phase !== 'collecting' && (
        <div className="p-3 shrink-0">
          <Button
            size="sm"
            className="w-full"
            onClick={handleRunClick}
            disabled={isRunning || !workflowName}
          >
            {isRunning ? (
              <>
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                Running...
              </>
            ) : (
              <>
                <Play className="h-3.5 w-3.5 mr-1.5" />
                Run
              </>
            )}
          </Button>
        </div>
      )}

      {/* Results area */}
      {hasResults && <Separator />}

      {htmlOutput ? (
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
      ) : hasResults ? (
        <ScrollArea className="flex-1 min-h-0">
          <div className="p-3 space-y-3">
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
            {doneEvent && textOutputs.length === 0 && (
              <div className="space-y-1">
                <p className="text-[10px] font-medium text-node-output">
                  Completed
                </p>
                <div className="rounded-lg border border-node-output/30 bg-node-output/5 p-2.5 text-xs whitespace-pre-wrap">
                  {doneEvent.data.status === 'completed'
                    ? 'Workflow completed successfully.'
                    : JSON.stringify(doneEvent.data, null, 2)}
                </div>
              </div>
            )}
          </div>
        </ScrollArea>
      ) : (
        !isRunning && (
          <div className="flex-1 flex items-center justify-center text-muted-foreground p-6">
            <div className="text-center">
              <Eye className="h-6 w-6 mb-2 opacity-50 mx-auto" />
              <p className="text-xs">Results will appear here after running.</p>
            </div>
          </div>
        )
      )}

      {/* Loading state without results yet */}
      {isRunning && !hasResults && (
        <div className="flex-1 flex items-center justify-center text-muted-foreground">
          <div className="text-center">
            <Loader2 className="h-6 w-6 mb-2 animate-spin mx-auto" />
            <p className="text-xs">Running workflow...</p>
          </div>
        </div>
      )}
    </div>
  )
}
