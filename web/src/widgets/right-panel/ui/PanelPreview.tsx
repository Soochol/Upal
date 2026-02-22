import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useWorkflowStore } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { useExecuteRun } from '@/features/execute-workflow'
import { Button } from '@/shared/ui/button'
import { Separator } from '@/shared/ui/separator'
import { Play, Loader2, Eye, ExternalLink } from 'lucide-react'
import { getInputNodesInOrder } from './preview/getInputNodesInOrder'
import { InputWizard } from './preview/InputWizard'
import { ResultsDisplay } from './preview/ResultsDisplay'

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

      {hasResults && <Separator />}

      {hasResults ? (
        <>
          <ResultsDisplay
            sessionState={sessionState}
            doneEvent={doneEvent}
            workflowName={workflowName}
          />
          {runSessionId && (
            <div className="px-3 pb-3 shrink-0">
              <Link
                to={`/runs/${runSessionId}`}
                className="flex items-center justify-center gap-1.5 w-full py-1.5 rounded-md text-xs text-primary hover:text-primary/80 border border-border hover:bg-muted/30 transition-colors"
              >
                <ExternalLink className="h-3 w-3" />
                View in Runs
              </Link>
            </div>
          )}
        </>
      ) : !isRunning ? (
        <div className="flex-1 flex items-center justify-center text-muted-foreground p-6">
          <div className="text-center">
            <Eye className="h-6 w-6 mb-2 opacity-50 mx-auto" />
            <p className="text-xs">Results will appear here after running.</p>
          </div>
        </div>
      ) : null}

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
