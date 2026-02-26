import { useCallback, useMemo } from 'react'
import {
  useAutoSave as useGenericAutoSave,
  type SaveStatus,
} from '@/shared/hooks/useAutoSave'
import { useWorkflowStore } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import {
  serializeWorkflow,
  saveWorkflow,
  suggestWorkflowName,
} from '@/entities/workflow'

export type { SaveStatus }

type CanvasSnapshot = {
  nodes: Array<{ id: string; data: unknown; position: { x: number; y: number } }>
  edges: unknown[]
  workflowName: string
}

function snapshotEqual(a: CanvasSnapshot, b: CanvasSnapshot): boolean {
  return JSON.stringify(a) === JSON.stringify(b)
}

export function useAutoSave() {
  const nodes = useWorkflowStore((s) => s.nodes)
  const edges = useWorkflowStore((s) => s.edges)
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const isTemplate = useWorkflowStore((s) => s.isTemplate)
  const originalName = useWorkflowStore((s) => s.originalName)
  const setWorkflowName = useWorkflowStore((s) => s.setWorkflowName)
  const setOriginalName = useWorkflowStore((s) => s.setOriginalName)
  const isRunning = useExecutionStore((s) => s.isRunning)

  const data: CanvasSnapshot = useMemo(
    () => ({
      nodes: nodes.map((n) => ({ id: n.id, data: n.data, position: n.position })),
      edges,
      workflowName: workflowName ?? '',
    }),
    [nodes, edges, workflowName],
  )

  const onSave = useCallback(
    async (snapshot: CanvasSnapshot) => {
      let name = snapshot.workflowName
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
    },
    [nodes, edges, originalName, setWorkflowName, setOriginalName],
  )

  const enabled = !isTemplate && !isRunning && nodes.length > 0

  const { saveStatus, saveNow, markClean } = useGenericAutoSave<CanvasSnapshot>({
    data,
    onSave,
    delay: 2000,
    isEqual: snapshotEqual,
    enabled,
  })

  return { saveStatus, saveNow, markClean }
}
