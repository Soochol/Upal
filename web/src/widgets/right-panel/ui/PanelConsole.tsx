// web/src/widgets/right-panel/ui/PanelConsole.tsx
import { useMemo, useState, useCallback, useEffect, useRef } from 'react'
import type { Node, Edge } from '@xyflow/react'
import { useWorkflowStore } from '@/entities/workflow'
import type { NodeData } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { useExecuteRun } from '@/features/execute-workflow'
import { getNodeDefinition } from '@/entities/node'
import { Button } from '@/shared/ui/button'
import {
  Trash2, ChevronDown, Play, Loader2,
  Check, X, Pause, SkipForward,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { NodeOutputViewer } from './console/NodeOutputViewer'
import type { NodeCompletedEvent } from '@/entities/run'
import type { NodeRunStatus } from '@/entities/run'

// ── Topological sort of ALL non-group nodes ──────────────────────────────────

type SortedEntry = { node: Node<NodeData>; inDegree: number }

function sortAllNodesTopologically(nodes: Node<NodeData>[], edges: Edge[]): SortedEntry[] {
  const regularNodes = nodes.filter((n) => n.type !== 'groupNode')
  const inDegreeMap = new Map<string, number>()
  const adj = new Map<string, string[]>()

  for (const node of regularNodes) {
    inDegreeMap.set(node.id, 0)
    adj.set(node.id, [])
  }
  for (const edge of edges) {
    if (adj.has(edge.source)) {
      adj.get(edge.source)!.push(edge.target)
      inDegreeMap.set(edge.target, (inDegreeMap.get(edge.target) ?? 0) + 1)
    }
  }

  const tempDeg = new Map(inDegreeMap)
  const queue: string[] = []
  for (const [id, deg] of tempDeg) {
    if (deg === 0) queue.push(id)
  }
  const sorted: string[] = []
  while (queue.length > 0) {
    const id = queue.shift()!
    sorted.push(id)
    for (const next of adj.get(id) ?? []) {
      const d = (tempDeg.get(next) ?? 1) - 1
      tempDeg.set(next, d)
      if (d === 0) queue.push(next)
    }
  }

  const nodeMap = new Map(regularNodes.map((n) => [n.id, n]))
  return sorted
    .filter((id) => nodeMap.has(id))
    .map((id) => ({ node: nodeMap.get(id)!, inDegree: inDegreeMap.get(id) ?? 0 }))
}

// ── NodeStepCard ──────────────────────────────────────────────────────────────

type NodeStepCardProps = {
  node: Node<NodeData>
  inDegree: number
  status: NodeRunStatus
  output: string | null
  isRunning: boolean
  onPlay: () => void
}

function StatusIcon({ status }: { status: NodeRunStatus }) {
  if (status === 'running') return <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
  if (status === 'error')   return <X className="h-3 w-3 text-destructive" />
  if (status === 'skipped') return <SkipForward className="h-3 w-3 text-muted-foreground/40" />
  if (status === 'waiting') return <Pause className="h-3 w-3 text-warning" />
  return null
}

function NodeStepCard({ node, inDegree, status, output, isRunning, onPlay }: NodeStepCardProps) {
  const [expanded, setExpanded] = useState(false)
  const autoExpandedRef = useRef<string | null>(null)

  const nodeType = node.data.nodeType as 'input' | 'agent' | 'output' | 'asset'
  let def
  try { def = getNodeDefinition(nodeType) } catch { return null }
  const Icon = def.icon
  const cssVar = def.cssVar

  // Auto-expand only once per unique completion (output value + completed status)
  useEffect(() => {
    if (status === 'completed' && output) {
      const key = output  // unique per completion result
      if (autoExpandedRef.current !== key) {
        autoExpandedRef.current = key
        setExpanded(true)
      }
    }
    // Reset when node goes back to idle (new run cleared state)
    if (status === 'idle') {
      autoExpandedRef.current = null
      setExpanded(false)
    }
  }, [status, output])

  const canPlay = inDegree === 0 && !isRunning
  const isCompleted = status === 'completed'

  return (
    <div
      className={cn(
        'rounded-lg border border-border/50 overflow-hidden transition-all',
        status === 'running' && 'shadow-sm',
      )}
      style={{
        borderLeftWidth: '3px',
        borderLeftColor: cssVar,
      }}
    >
      <div
        className="flex items-center gap-2 px-2.5 py-2 cursor-pointer select-none"
        style={{ background: `color-mix(in oklch, ${cssVar} 8%, transparent)` }}
        onClick={() => setExpanded((p) => !p)}
      >
        <ChevronDown
          className={cn(
            'h-3 w-3 text-muted-foreground/40 transition-transform shrink-0',
            expanded && 'rotate-180',
          )}
        />

        <span style={{ color: cssVar }} className="shrink-0 flex">
          <Icon className="h-3.5 w-3.5" />
        </span>

        <span className="text-xs font-medium flex-1 truncate leading-tight">
          {node.data.label}
        </span>

        {isCompleted ? (
          <Check className="h-3 w-3 shrink-0" style={{ color: cssVar }} />
        ) : (
          <StatusIcon status={status} />
        )}

        <button
          onClick={(e) => { e.stopPropagation(); if (canPlay) onPlay() }}
          disabled={!canPlay}
          className={cn(
            'h-5 w-5 rounded-full flex items-center justify-center transition-all shrink-0',
            canPlay
              ? 'hover:opacity-80 cursor-pointer'
              : 'opacity-20 cursor-not-allowed',
          )}
          style={{ background: canPlay ? cssVar : undefined }}
          title={canPlay ? 'Run workflow' : 'Has upstream dependencies'}
        >
          <Play
            className="h-2.5 w-2.5 ml-px"
            style={{ color: canPlay ? 'white' : undefined }}
          />
        </button>
      </div>

      {expanded && (
        <div className="border-t border-border/30">
          {output ? (
            <NodeOutputViewer output={output} />
          ) : status === 'running' ? (
            <div className="px-3 py-2 flex items-center gap-2 text-xs text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />
              Running…
            </div>
          ) : (
            <p className="px-3 py-2 text-xs text-muted-foreground/50">No output yet.</p>
          )}
        </div>
      )}
    </div>
  )
}

// ── PanelConsole ──────────────────────────────────────────────────────────────

export function PanelConsole() {
  const nodes    = useWorkflowStore((s) => s.nodes)
  const edges    = useWorkflowStore((s) => s.edges)
  const runEvents      = useExecutionStore((s) => s.runEvents)
  const nodeStatuses   = useExecutionStore((s) => s.nodeStatuses)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const clearNodeStatuses = useExecutionStore((s) => s.clearNodeStatuses)
  const { executeRun, isRunning } = useExecuteRun()

  const sortedNodes = useMemo(() => sortAllNodesTopologically(nodes, edges), [nodes, edges])

  const renderableNodes = useMemo(
    () => sortedNodes.filter(({ node }) => {
      try { getNodeDefinition(node.data.nodeType as 'input' | 'agent' | 'output' | 'asset'); return true }
      catch { return false }
    }),
    [sortedNodes],
  )

  const nodeOutputs = useMemo(() => {
    const map: Record<string, string> = {}
    for (const event of runEvents) {
      if (event.type === 'node_completed') {
        const c = event as NodeCompletedEvent
        if (c.output) map[c.nodeId] = c.output
      }
    }
    return map
  }, [runEvents])

  const handlePlay = useCallback(() => {
    if (isRunning) return
    const inputs: Record<string, string> = {}
    for (const n of nodes) {
      if (n.data.nodeType === 'input') {
        inputs[n.id] = (n.data.config.value as string) ?? ''
      }
    }
    executeRun(inputs)
  }, [nodes, isRunning, executeRun])

  const handleClear = useCallback(() => {
    clearRunEvents()
    clearNodeStatuses()
  }, [clearRunEvents, clearNodeStatuses])

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-border shrink-0">
        <span className="text-xs font-medium text-muted-foreground">Steps</span>
        {runEvents.length > 0 && (
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={handleClear} title="Clear">
            <Trash2 className="h-3 w-3" />
          </Button>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-1.5">
        {renderableNodes.length === 0 ? (
          <p className="text-xs text-muted-foreground px-1 py-2">
            Add nodes to the canvas to see them here.
          </p>
        ) : (
          renderableNodes.map(({ node, inDegree }) => (
            <NodeStepCard
              key={node.id}
              node={node}
              inDegree={inDegree}
              status={(nodeStatuses[node.id] ?? 'idle') as NodeRunStatus}
              output={nodeOutputs[node.id] ?? null}
              isRunning={isRunning}
              onPlay={handlePlay}
            />
          ))
        )}
      </div>
    </div>
  )
}
