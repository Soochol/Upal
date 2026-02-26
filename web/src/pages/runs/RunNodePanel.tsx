import { useState, useEffect, useRef, useMemo, Suspense, lazy } from 'react'
import type { Node, Edge } from '@xyflow/react'
import type { NodeData } from '@/entities/workflow'
import { useWorkflowStore } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import type { RunRecord, NodeRunRecord } from '@/entities/run'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/shared/ui/tabs'
import { getNodeDefinition } from '@/entities/node'
import type { NodeType } from '@/entities/node'
import { TemplateText } from '@/shared/ui/TemplateText'
import { NodeOutputViewer } from '@/widgets/right-panel/ui/console/NodeOutputViewer'
import type { OutputFormatDef } from '@/shared/lib/outputFormats'
import { resolveFormat } from '@/shared/lib/outputFormats'
import {
  Settings2, Terminal, Eye, X,
  CheckCircle2, XCircle, Loader2,
  ChevronDown, SkipForward, Pause,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'

type Props = {
  selectedNode: Node<NodeData> | null
  run: RunRecord
  onClose: () => void
}

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

const STATUS_ICONS: Record<string, { icon: typeof Loader2; className: string }> = {
  running: { icon: Loader2, className: 'h-3 w-3 animate-spin text-muted-foreground' },
  error:   { icon: XCircle, className: 'h-3 w-3 text-destructive' },
  skipped: { icon: SkipForward, className: 'h-3 w-3 text-muted-foreground/40' },
  waiting: { icon: Pause, className: 'h-3 w-3 text-warning' },
}

function StatusIcon({ status }: { status: string }) {
  const cfg = STATUS_ICONS[status]
  if (!cfg) return null
  const Icon = cfg.icon
  return <Icon className={cfg.className} />
}

function formatOutput(value: unknown): string {
  if (value === undefined || value === null) return ''
  if (typeof value === 'string') return value
  return JSON.stringify(value, null, 2)
}

function NodeStepCard({ node, status, output, isSelected }: {
  node: Node<NodeData>
  status: string
  output: string | null
  isSelected: boolean
}) {
  const [expanded, setExpanded] = useState(false)
  const autoExpandedRef = useRef<string | null>(null)

  const nodeType = node.data.nodeType as 'input' | 'agent' | 'output' | 'asset'
  let def
  try { def = getNodeDefinition(nodeType) } catch { return null }
  const Icon = def.icon
  const cssVar = def.cssVar
  const isCompleted = status === 'completed'

  useEffect(() => {
    if (isCompleted && output) {
      const key = output
      if (autoExpandedRef.current !== key) {
        autoExpandedRef.current = key
        setExpanded(true)
      }
    }
  }, [isCompleted, output])

  return (
    <div
      className={cn(
        'rounded-lg border border-border/50 overflow-hidden transition-all',
        status === 'running' && 'shadow-sm',
        isSelected && 'ring-1 ring-primary/60',
      )}
      style={{ borderLeftWidth: '3px', borderLeftColor: cssVar }}
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
          <CheckCircle2 className="h-3 w-3 shrink-0" style={{ color: cssVar }} />
        ) : (
          <StatusIcon status={status} />
        )}
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

function RunConsole({ run, selectedNodeId }: { run: RunRecord; selectedNodeId: string | null }) {
  const nodes = useWorkflowStore((s) => s.nodes)
  const edges = useWorkflowStore((s) => s.edges)
  const liveStatuses = useExecutionStore((s) => s.nodeStatuses)

  const sortedNodes = useMemo(() => sortAllNodesTopologically(nodes, edges), [nodes, edges])

  const nodeRunMap = useMemo(() => {
    const map = new Map<string, NodeRunRecord>()
    for (const nr of run.node_runs ?? []) map.set(nr.node_id, nr)
    return map
  }, [run.node_runs])

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center px-3 py-1.5 border-b border-border shrink-0">
        <span className="text-xs font-medium text-muted-foreground">Steps</span>
      </div>
      <div className="flex-1 overflow-y-auto p-2 space-y-1.5">
        {sortedNodes.length === 0 ? (
          <p className="text-xs text-muted-foreground px-1 py-2">No nodes.</p>
        ) : (
          sortedNodes.map(({ node }) => {
            const nr = nodeRunMap.get(node.id)
            const status = liveStatuses[node.id] ?? nr?.status ?? 'idle'
            const output = run.outputs?.[node.id]
            return (
              <NodeStepCard
                key={node.id}
                node={node}
                status={status}
                output={output != null ? formatOutput(output) : null}
                isSelected={selectedNodeId === node.id}
              />
            )
          })
        )}
      </div>
    </div>
  )
}

function useLazyResultView(format: OutputFormatDef) {
  return useMemo(() => lazy(format.ResultView), [format])
}

function RunPreview({ run }: { run: RunRecord }) {
  const nodes = useWorkflowStore((s) => s.nodes)

  const outputNode = useMemo(
    () => nodes.find((n) => n.data.nodeType === 'output'),
    [nodes],
  )

  const outputFormat = outputNode?.data.config.output_format as string | undefined
  const outputNodeId = outputNode?.id

  // Find primary output content
  const primaryContent = useMemo(() => {
    if (!run.outputs) return null

    // Try output node first
    if (outputNodeId) {
      const v = run.outputs[outputNodeId]
      if (typeof v === 'string' && v.trim()) return v
    }

    // Check __output__ map
    const outputMap = run.outputs['__output__']
    if (outputMap && typeof outputMap === 'object' && outputNodeId) {
      const v = (outputMap as Record<string, unknown>)[outputNodeId]
      if (typeof v === 'string' && v.trim()) return v
    }

    // Fallback: first string output
    for (const [, v] of Object.entries(run.outputs)) {
      if (typeof v === 'string' && v.trim()) return v
    }
    return null
  }, [run.outputs, outputNodeId])

  const format = primaryContent ? resolveFormat(outputFormat, primaryContent) : null
  const isLive = run.status === 'running' || run.status === 'pending'

  if (!primaryContent) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3 p-6 text-center">
        {isLive ? (
          <>
            <div className="relative h-12 w-12">
              <div className="absolute inset-0 rounded-full border-2 border-primary/15 border-t-primary animate-spin" />
            </div>
            <p className="text-xs text-muted-foreground">Processing…</p>
          </>
        ) : (
          <p className="text-sm text-muted-foreground">No output available</p>
        )}
      </div>
    )
  }

  if (!format) return null

  return (
    <div className="flex flex-col h-full min-h-0">
      <FormatResultView format={format} content={primaryContent} workflowName={run.workflow_name} />
    </div>
  )
}

function FormatResultView({ format, content, workflowName }: {
  format: OutputFormatDef
  content: string
  workflowName: string
}) {
  const LazyView = useLazyResultView(format)
  return (
    <Suspense fallback={<div className="p-3 text-xs text-muted-foreground">Loading…</div>}>
      <LazyView content={content} workflowName={workflowName} />
    </Suspense>
  )
}

const fieldBoxClass = "text-xs rounded-md border border-input bg-transparent px-3 py-2 whitespace-pre-wrap break-words select-text"

function str(v: unknown): string | null {
  return typeof v === 'string' && v ? v : null
}

function formatExtraction(extract: { mode?: string; key?: string; tag?: string }): string {
  switch (extract.mode) {
    case 'json': return `JSON key: ${extract.key ?? ''}`
    case 'tagged': return `XML tag: <${extract.tag ?? ''}>`
    default: return 'None'
  }
}

function RunProperties({ node }: { node: Node<NodeData> }) {
  const config = node.data.config
  const nodeType = node.data.nodeType as string

  let def
  try { def = getNodeDefinition(nodeType as NodeType) } catch { /* unknown type */ }
  const Icon = def?.icon

  const model = str(config.model)
  const outputFormat = str(config.output_format)
  const tools = Array.isArray(config.tools) ? (config.tools as string[]) : null
  const prompt = str(config.prompt)
  const systemPrompt = str(config.system_prompt)
  const outputTpl = str(config.output)
  const value = str(config.value)
  const tool = str(config.tool)
  const input = str(config.input)
  const assetId = str(config.asset_id)
  const extract = config.output_extract as { mode?: string; key?: string; tag?: string } | undefined

  return (
    <div className="flex-1 min-h-0 flex flex-col overflow-y-auto">
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-2.5 border-b border-border shrink-0">
        {Icon && (
          <div className={cn('h-6 w-6 rounded-md flex items-center justify-center shrink-0', def?.accent)}>
            <Icon className="h-3.5 w-3.5" />
          </div>
        )}
        <span className="flex-1 min-w-0 text-sm font-semibold truncate">{node.data.label}</span>
        <span className="text-[10px] text-muted-foreground capitalize">{nodeType}</span>
      </div>

      {/* Fields */}
      <div className="p-3 flex flex-col gap-3">
        {model && (
          <PropField label="Model">
            <code className="text-xs font-mono text-foreground">{model}</code>
          </PropField>
        )}

        {outputFormat && (
          <PropField label="Output Format">
            <span className="text-xs capitalize">{outputFormat}</span>
          </PropField>
        )}

        {tools && tools.length > 0 && (
          <PropField label="Tools">
            <div className="flex flex-wrap gap-1">
              {tools.map((t) => (
                <span key={t} className="px-1.5 py-0.5 rounded text-[10px] bg-muted text-muted-foreground border border-border">
                  {t}
                </span>
              ))}
            </div>
          </PropField>
        )}

        {prompt && (
          <PropField label="Prompt">
            <div className={cn(fieldBoxClass, 'max-h-60 overflow-y-auto')}>
              <TemplateText text={prompt} />
            </div>
          </PropField>
        )}

        {systemPrompt && (
          <PropField label="System Prompt">
            <div className={cn(fieldBoxClass, 'max-h-40 overflow-y-auto')}>
              <TemplateText text={systemPrompt} />
            </div>
          </PropField>
        )}

        {outputTpl && (
          <PropField label="Output Template">
            <div className={cn(fieldBoxClass, 'max-h-40 overflow-y-auto')}>
              <TemplateText text={outputTpl} />
            </div>
          </PropField>
        )}

        {value && (
          <PropField label="Value">
            <div className={cn(fieldBoxClass, 'max-h-40 overflow-y-auto')}>
              {value}
            </div>
          </PropField>
        )}

        {extract?.mode && (
          <PropField label="Output Extraction">
            <span className="text-xs text-muted-foreground">
              {formatExtraction(extract)}
            </span>
          </PropField>
        )}

        {tool && (
          <PropField label="Tool">
            <code className="text-xs font-mono">{tool}</code>
          </PropField>
        )}
        {input && (
          <PropField label="Input">
            <div className={cn(fieldBoxClass, 'max-h-40 overflow-y-auto')}>
              <TemplateText text={input} />
            </div>
          </PropField>
        )}

        {assetId && (
          <PropField label="Asset ID">
            <code className="text-xs font-mono text-muted-foreground">{assetId}</code>
          </PropField>
        )}
      </div>
    </div>
  )
}

function PropField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <span className="text-[11px] font-medium text-muted-foreground">{label}</span>
      {children}
    </div>
  )
}

const tabTriggerClass = "rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 flex items-center gap-1.5 text-xs font-medium"

export function RunNodePanel({ selectedNode, run, onClose }: Props) {
  const [activeTab, setActiveTab] = useState('console')
  const prevNodeIdRef = useRef<string | null>(null)

  useEffect(() => {
    if (selectedNode && selectedNode.id !== prevNodeIdRef.current) {
      setActiveTab('properties')
    }
    prevNodeIdRef.current = selectedNode?.id ?? null
  }, [selectedNode])

  return (
    <div className="h-full flex flex-col bg-background border-l border-border">
      <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col min-h-0">
        <div className="flex items-center border-b border-border px-1">
          <TabsList className="h-10 bg-transparent p-0 gap-0 flex-1">
            <TabsTrigger value="properties" className={tabTriggerClass}>
              <Settings2 className="h-3.5 w-3.5 shrink-0" />
              Properties
            </TabsTrigger>
            <TabsTrigger value="console" className={tabTriggerClass}>
              <Terminal className="h-3.5 w-3.5 shrink-0" />
              Console
            </TabsTrigger>
            <TabsTrigger value="preview" className={tabTriggerClass}>
              <Eye className="h-3.5 w-3.5 shrink-0" />
              Preview
            </TabsTrigger>
          </TabsList>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer shrink-0 mr-1"
            title="Close Panel"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <TabsContent value="properties" className="flex-1 min-h-0 flex flex-col mt-0">
          {selectedNode && selectedNode.type !== 'groupNode' ? (
            <RunProperties node={selectedNode} />
          ) : (
            <div className="flex items-center justify-center h-32 text-xs text-muted-foreground p-3">
              Select a node to view its properties.
            </div>
          )}
        </TabsContent>

        <TabsContent value="console" className="flex-1 min-h-0 overflow-hidden mt-0">
          <RunConsole run={run} selectedNodeId={selectedNode?.id ?? null} />
        </TabsContent>

        <TabsContent value="preview" className="flex-1 min-h-0 overflow-hidden mt-0">
          <RunPreview run={run} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
