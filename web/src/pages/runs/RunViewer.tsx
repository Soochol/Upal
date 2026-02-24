import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { ReactFlowProvider } from '@xyflow/react'
import type { Node } from '@xyflow/react'
import { fetchRun, connectToRunEvents } from '@/entities/run'
import { useExecutionStore } from '@/entities/run'
import { useWorkflowStore, deserializeWorkflow } from '@/entities/workflow'
import type { NodeData } from '@/entities/workflow'
import { useUIStore } from '@/entities/ui/model/store'
import type { RunRecord, RunEvent } from '@/shared/types'
import { Header } from '@/shared/ui/Header'
import { Canvas } from '@/widgets/workflow-canvas/ui/Canvas'
import { RunNodePanel } from './RunNodePanel'
import { useResizeDrag } from '@/shared/lib/useResizeDrag'
import {
  ArrowLeft, CheckCircle2, XCircle, Clock, Loader2, Timer,
  ExternalLink, PanelRightClose, PanelRightOpen,
} from 'lucide-react'

const statusConfig: Record<string, { icon: typeof Clock; color: string; label: string }> = {
  pending:   { icon: Clock,        color: 'text-warning',          label: 'Pending' },
  running:   { icon: Loader2,      color: 'text-info',             label: 'Running' },
  success:   { icon: CheckCircle2, color: 'text-success',          label: 'Success' },
  failed:    { icon: XCircle,      color: 'text-destructive',      label: 'Failed' },
  cancelled: { icon: XCircle,      color: 'text-muted-foreground', label: 'Cancelled' },
  retrying:  { icon: Timer,        color: 'text-warning',          label: 'Retrying' },
}

function formatDuration(run: RunRecord): string {
  if (!run.started_at) return '-'
  const end = run.completed_at ? new Date(run.completed_at) : new Date()
  const ms = end.getTime() - new Date(run.started_at).getTime()
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

const noop = () => {}

export function RunViewer() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [run, setRun] = useState<RunRecord | null>(null)
  const [loading, setLoading] = useState(true)
  const [isPanelOpen, setIsPanelOpen] = useState(false)

  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const selectNode = useUIStore((s) => s.selectNode)
  const nodes = useWorkflowStore((s) => s.nodes)

  const setNodeStatus = useExecutionStore((s) => s.setNodeStatus)
  const clearNodeStatuses = useExecutionStore((s) => s.clearNodeStatuses)
  const addRunEvent = useExecutionStore((s) => s.addRunEvent)

  const { size: panelWidth, handleMouseDown: onPanelDrag } = useResizeDrag({
    direction: 'horizontal',
    min: 280,
    max: 700,
    initial: 400,
  })

  // Keep a ref to the abort controller for SSE cleanup
  const abortRef = useRef<AbortController | null>(null)

  // Auto-open panel on node selection
  useEffect(() => {
    if (selectedNodeId) {
      setIsPanelOpen(true)
    }
  }, [selectedNodeId])

  // Fetch run and load workflow into canvas
  useEffect(() => {
    if (!id) return

    let cancelled = false

    clearNodeStatuses()
    selectNode(null)

    fetchRun(id)
      .then((r) => {
        if (cancelled) return
        setRun(r)

        // Deserialize the workflow definition into the canvas store
        if (r.workflow_definition) {
          const { nodes: flowNodes, edges: flowEdges } = deserializeWorkflow(r.workflow_definition as import('@/entities/workflow/lib/serializer').WorkflowDefinition)
          useWorkflowStore.setState({ nodes: flowNodes, edges: flowEdges })
        }

        // Map existing node_runs statuses to the execution store
        if (r.node_runs) {
          for (const nr of r.node_runs) {
            const status = nr.status === 'success' || nr.status === 'completed'
              ? 'completed'
              : nr.status === 'failed'
              ? 'error'
              : (nr.status as 'idle' | 'running' | 'completed' | 'error' | 'waiting' | 'skipped')
            setNodeStatus(nr.node_id, status)
          }
        }

        // Connect to SSE for live runs
        if (r.status === 'running' || r.status === 'pending') {
          connectSSE(r.id)
        }
      })
      .catch(() => {
        if (!cancelled) setRun(null)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
      abortRef.current?.abort()
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  function connectSSE(runId: string) {
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    connectToRunEvents(
      runId,
      (event: RunEvent) => {
        addRunEvent(event)

        if (event.type === 'node_started' || event.type === 'tool_call' || event.type === 'tool_result') {
          const nodeId = 'nodeId' in event && event.nodeId ? event.nodeId : 'system'
          setNodeStatus(nodeId, 'running', event.type === 'node_started' ? event.startedAt : undefined)
        } else if (event.type === 'node_completed') {
          const nodeId = 'nodeId' in event && event.nodeId ? event.nodeId : 'system'
          setNodeStatus(nodeId, 'completed')
          // Update run outputs in local state
          setRun((prev) => {
            if (!prev) return prev
            return {
              ...prev,
              outputs: { ...prev.outputs, [nodeId]: event.output },
            }
          })
        } else if (event.type === 'node_skipped') {
          setNodeStatus(event.nodeId, 'skipped')
        } else if (event.type === 'node_waiting') {
          setNodeStatus(event.nodeId, 'waiting')
        }
      },
      (_result) => {
        // Run completed — re-fetch for final state
        fetchRun(runId).then((r) => setRun(r)).catch(() => {})
      },
      (error) => {
        console.error('SSE error:', error)
      },
      { signal: controller.signal },
    )
  }

  // Loading state
  if (loading) {
    return (
      <div className="h-screen flex flex-col bg-background">
        <Header />
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="animate-spin text-muted-foreground" size={32} />
        </div>
      </div>
    )
  }

  // Not found
  if (!run) {
    return (
      <div className="h-screen flex flex-col bg-background">
        <Header />
        <div className="flex-1 flex items-center justify-center">
          <p className="text-muted-foreground">Run not found</p>
        </div>
      </div>
    )
  }

  const cfg = statusConfig[run.status] ?? statusConfig.pending
  const StatusIcon = cfg.icon
  const isLive = run.status === 'running' || run.status === 'pending'
  const hasWorkflow = !!run.workflow_definition

  const selectedNode: Node<NodeData> | undefined = selectedNodeId
    ? nodes.find((n) => n.id === selectedNodeId)
    : undefined

  return (
    <div className="h-screen flex flex-col bg-background text-foreground">
      <Header
        actions={
          <div className="flex items-center gap-2">
            {/* Back button */}
            <button
              onClick={() => navigate('/runs')}
              className="flex items-center gap-1 px-2.5 py-1.5 rounded-md text-xs text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            >
              <ArrowLeft size={14} />
              Back
            </button>

            {/* Run ID */}
            <code className="text-xs text-muted-foreground bg-muted px-2 py-1 rounded font-mono">
              {run.id.length > 12 ? `${run.id.slice(0, 12)}...` : run.id}
            </code>

            {/* Status badge */}
            <div className="flex items-center gap-1.5">
              <StatusIcon
                size={16}
                className={`${cfg.color} ${run.status === 'running' ? 'animate-spin' : ''}`}
              />
              <span className={`text-xs font-semibold ${cfg.color}`}>{cfg.label}</span>
            </div>

            {isLive && (
              <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-info/10 text-info animate-pulse">
                live
              </span>
            )}

            {/* Workflow name */}
            <span className="text-xs text-muted-foreground">
              {run.workflow_name}
            </span>

            {/* Duration */}
            <span className="text-xs text-muted-foreground font-mono">
              {formatDuration(run)}
            </span>

            {/* Open Workflow */}
            <Link
              to={`/workflows?w=${encodeURIComponent(run.workflow_name)}`}
              className="flex items-center gap-1 px-2.5 py-1.5 rounded-md text-xs border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            >
              <ExternalLink size={12} />
              Open Workflow
            </Link>

            {/* Panel toggle */}
            <button
              onClick={() => setIsPanelOpen((v) => !v)}
              className="p-2 rounded-xl border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
              title="Toggle Panel"
            >
              {isPanelOpen ? <PanelRightClose className="w-4 h-4" /> : <PanelRightOpen className="w-4 h-4" />}
            </button>
          </div>
        }
      />

      {/* Canvas + Panel */}
      <div className="flex-1 min-h-0 overflow-hidden flex">
        {/* Canvas area */}
        <div className="flex-1 min-w-0 h-full">
          {hasWorkflow ? (
            <ReactFlowProvider>
              <Canvas
                onAddFirstNode={noop}
                onDropNode={noop}
                onPromptSubmit={noop}
                isGenerating={false}
                onAddNode={noop}
                readOnly
              />
            </ReactFlowProvider>
          ) : (
            <div className="h-full flex items-center justify-center text-muted-foreground text-sm">
              No workflow definition available for this run
            </div>
          )}
        </div>

        {/* Right panel */}
        {isPanelOpen && selectedNode && (
          <div className="hidden md:contents">
            <div
              onMouseDown={onPanelDrag}
              className="w-1 shrink-0 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 transition-colors z-30 relative"
            />
            <div
              className="shrink-0 h-full overflow-hidden"
              style={{ width: panelWidth }}
            >
              <RunNodePanel
                selectedNode={selectedNode}
                run={run}
                onClose={() => {
                  selectNode(null)
                  setIsPanelOpen(false)
                }}
                onCollapse={() => setIsPanelOpen(false)}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
