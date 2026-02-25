import { useMemo } from 'react'
import { useWorkflowStore, serializeWorkflow, saveWorkflow, suggestWorkflowName } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { useAutoSave as useGenericAutoSave } from '@/shared/hooks/useAutoSave'

export type { SaveStatus } from '@/shared/hooks/useAutoSave'

// ---------------------------------------------------------------------------
// Canvas snapshot type (only the fields relevant for dirty-tracking)
// ---------------------------------------------------------------------------

type CanvasSnapshot = {
  nodes: { id: string; data: unknown; position: unknown }[]
  edges: unknown[]
  workflowName: string
}

function snapshotEqual(a: CanvasSnapshot, b: CanvasSnapshot): boolean {
  return JSON.stringify(a) === JSON.stringify(b)
}

// ---------------------------------------------------------------------------
// flushSave — imperative save, reads directly from Zustand stores.
// Kept as a standalone function so callers outside React can use it.
// ---------------------------------------------------------------------------

async function flushSave(): Promise<void> {
  if (useWorkflowStore.getState().isTemplate) return
  const { nodes, edges, workflowName, originalName, setWorkflowName, setOriginalName } =
    useWorkflowStore.getState()

  if (nodes.length === 0) return

  let name = workflowName
  if (!name) {
    const tempWf = serializeWorkflow('untitled', nodes, edges)
    try {
      name = await suggestWorkflowName(tempWf)
    } catch {
      name = 'untitled-workflow'
    }
    setWorkflowName(name)
  }

  const wf = serializeWorkflow(name, nodes, edges)
  await saveWorkflow(wf, originalName || undefined)

  if (originalName !== name) {
    setOriginalName(name)
  }
}

// ---------------------------------------------------------------------------
// useAutoSave — wraps the generic hook with canvas-specific selectors
// ---------------------------------------------------------------------------

export function useAutoSave() {
  const nodes = useWorkflowStore(s => s.nodes)
  const edges = useWorkflowStore(s => s.edges)
  const workflowName = useWorkflowStore(s => s.workflowName)
  const isTemplate = useWorkflowStore(s => s.isTemplate)
  const isRunning = useExecutionStore(s => s.isRunning)

  const data: CanvasSnapshot = useMemo(() => ({
    nodes: nodes.map(n => ({ id: n.id, data: n.data, position: n.position })),
    edges,
    workflowName,
  }), [nodes, edges, workflowName])

  const { saveStatus, saveNow, markClean } = useGenericAutoSave({
    data,
    onSave: async () => { await flushSave() },
    delay: 2000,
    isEqual: snapshotEqual,
    enabled: !isTemplate && !isRunning && nodes.length > 0,
  })

  return { saveStatus, saveNow, markClean }
}
