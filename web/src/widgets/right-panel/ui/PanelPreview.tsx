import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useWorkflowStore } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { useExecuteRun } from '@/features/execute-workflow'
import { Play, Loader2, Zap, ExternalLink } from 'lucide-react'
import { getInputNodesInOrder } from './preview/getInputNodesInOrder'
import { InputWizard } from './preview/InputWizard'
import { ResultsDisplay } from './preview/ResultsDisplay'
import { PreviewBackground } from './preview/PreviewBackground'

type Phase = 'idle' | 'collecting' | 'running'

export function PanelPreview() {
  const nodes = useWorkflowStore((s) => s.nodes)
  const edges = useWorkflowStore((s) => s.edges)
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const runEvents = useExecutionStore((s) => s.runEvents)
  const sessionState = useExecutionStore((s) => s.sessionState)
  const setNodeStatus = useExecutionStore((s) => s.setNodeStatus)
  const clearNodeStatuses = useExecutionStore((s) => s.clearNodeStatuses)
  const { executeRun, isRunning } = useExecuteRun()

  const [phase, setPhase] = useState<Phase>('idle')
  const [inputs, setInputs] = useState<Record<string, string>>({})
  const [stepIndex, setStepIndex] = useState(0)

  const sortedInputs = useMemo(
    () => getInputNodesInOrder(nodes, edges),
    [nodes, edges],
  )

  const currentInput = sortedInputs[stepIndex]

  const startCollecting = () => {
    if (!workflowName || isRunning || sortedInputs.length === 0) return
    clearNodeStatuses()
    setInputs({})
    setStepIndex(0)
    setPhase('collecting')
    setNodeStatus(sortedInputs[0].id, 'running')
  }

  const handleNext = () => {
    if (!currentInput) return
    setNodeStatus(currentInput.id, 'completed')

    if (stepIndex < sortedInputs.length - 1) {
      const nextIndex = stepIndex + 1
      setStepIndex(nextIndex)
      setNodeStatus(sortedInputs[nextIndex].id, 'running')
    } else {
      setPhase('running')
      executeRun(inputs)
    }
  }

  const handleBack = () => {
    if (stepIndex > 0) {
      setNodeStatus(currentInput.id, 'idle')
      const prevIndex = stepIndex - 1
      setStepIndex(prevIndex)
      setNodeStatus(sortedInputs[prevIndex].id, 'running')
    } else {
      setPhase('idle')
      clearNodeStatuses()
    }
  }

  const handleRunClick = () => {
    if (!workflowName || isRunning) return
    if (sortedInputs.length > 0 && phase !== 'collecting') {
      startCollecting()
      return
    }
    setPhase('running')
    executeRun(inputs)
  }

  const doneEvent = runEvents.find((e) => e.type === 'done')
  const hasResults = doneEvent || Object.entries(sessionState).some(([, v]) => v != null && v !== '')
  const runSessionId = doneEvent && doneEvent.type === 'done' ? doneEvent.sessionId : null

  return (
    <div className="flex flex-col h-full">

      {/* ── App chrome bar ── */}
      <div className="shrink-0 flex items-center px-3 py-2.5 border-b border-border/40 bg-muted/20">
        <p className="flex-1 text-center text-[11px] text-muted-foreground font-medium truncate">
          {workflowName || 'Untitled'}
        </p>
        {runSessionId ? (
          <Link
            to={`/runs/${runSessionId}`}
            className="text-muted-foreground hover:text-foreground transition-colors shrink-0"
          >
            <ExternalLink className="h-3 w-3" />
          </Link>
        ) : (
          <div className="w-3 shrink-0" />
        )}
      </div>

      {/* ── App content area ── */}
      <div className="flex-1 min-h-0 overflow-auto">

        {phase === 'collecting' && (
          <InputWizard
            sortedInputs={sortedInputs}
            stepIndex={stepIndex}
            inputs={inputs}
            onInputChange={(nodeId, value) =>
              setInputs((prev) => ({ ...prev, [nodeId]: value }))
            }
            onNext={handleNext}
            onBack={handleBack}
          />
        )}

        {isRunning && !hasResults && (
          <div className="flex flex-col items-center justify-center h-full gap-3 p-6">
            <div className="relative h-12 w-12">
              <div className="absolute inset-0 rounded-full border-2 border-primary/15 border-t-primary animate-spin" />
              <div className="absolute inset-0 flex items-center justify-center">
                <Zap className="h-5 w-5 text-primary/60" />
              </div>
            </div>
            <p className="text-xs text-muted-foreground">Processing...</p>
          </div>
        )}

        {hasResults && phase !== 'collecting' && (
          <ResultsDisplay
            sessionState={sessionState}
            doneEvent={doneEvent}
            workflowName={workflowName}
          />
        )}

        {!hasResults && !isRunning && phase === 'idle' && (
          <div className="relative flex flex-col items-center justify-center h-full gap-4 p-6 text-center overflow-hidden">
            <PreviewBackground />
            {/* Foreground content — centered above the SVG */}
            <div className="relative z-10 flex flex-col items-center gap-3">
              <div className="h-14 w-14 rounded-2xl bg-background/80 backdrop-blur-sm border border-border/60 flex items-center justify-center shadow-md">
                <Zap className="h-6 w-6 text-primary/70" />
              </div>
              <div>
                <p className="text-sm font-semibold text-foreground/80">
                  {workflowName || 'Workflow'}
                </p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Run to see results here
                </p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* ── Bottom Run button ── */}
      {phase !== 'collecting' && (
        <div className="shrink-0 p-3 border-t border-border/40">
          <button
            onClick={handleRunClick}
            disabled={isRunning || !workflowName}
            className="w-full h-10 rounded-lg bg-primary text-primary-foreground text-sm font-semibold
                       flex items-center justify-center gap-2
                       shadow-md shadow-primary/20
                       hover:shadow-lg hover:shadow-primary/35 hover:brightness-105
                       disabled:opacity-40 disabled:shadow-none disabled:cursor-not-allowed
                       transition-all duration-200 active:scale-[0.98]"
          >
            {isRunning ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                Running...
              </>
            ) : (
              <>
                <Play className="h-3.5 w-3.5" fill="currentColor" />
                Run Workflow
              </>
            )}
          </button>
        </div>
      )}
    </div>
  )
}
