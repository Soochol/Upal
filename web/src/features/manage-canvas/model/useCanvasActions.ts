import { useCallback } from 'react'
import { useWorkflowStore } from '@/entities/workflow'
import { useUIStore } from '@/entities/ui'
import { deleteFile } from '@/shared/api'
import type { NodeData } from '@/entities/workflow'
import type { Node } from '@xyflow/react'

/**
 * Orchestrates canvas mutations that touch multiple stores.
 * Components call these instead of calling stores directly.
 */
export function useCanvasActions() {
  const workflowStore = useWorkflowStore()
  const selectNode = useUIStore((s) => s.selectNode)

  /**
   * Delete a node + its backing file (if asset) + deselect.
   * Cross-store: workflow store removes node, ui store clears selection.
   */
  const deleteNode = useCallback((id: string) => {
    const node = useWorkflowStore.getState().getNode(id) as Node<NodeData> | undefined
    if (node?.data.nodeType === 'asset') {
      const fileId = node.data.config.file_id as string | undefined
      if (fileId) {
        deleteFile(fileId).catch((err) =>
          console.error(`Failed to delete file ${fileId}:`, err),
        )
      }
    }
    useWorkflowStore.getState().removeNode(id)
    if (useUIStore.getState().selectedNodeId === id) {
      selectNode(null)
    }
  }, [selectNode])

  /**
   * Create a group from selected node IDs, then select the group node.
   */
  const createGroup = useCallback((nodeIds: string[]) => {
    const groupId = useWorkflowStore.getState().createGroup(nodeIds)
    if (groupId) selectNode(groupId)
  }, [selectNode])

  /**
   * Remove a group node and release its children, then deselect.
   */
  const removeGroup = useCallback((groupId: string) => {
    useWorkflowStore.getState().removeGroup(groupId)
    selectNode(null)
  }, [selectNode])

  return {
    deleteNode,
    createGroup,
    removeGroup,
    // Convenience pass-throughs to workflow store (no cross-store concern)
    addNode: workflowStore.addNode,
    updateNodeConfig: workflowStore.updateNodeConfig,
    updateNodeLabel: workflowStore.updateNodeLabel,
    applyAutoLayout: workflowStore.applyAutoLayout,
    updateGroupLabel: workflowStore.updateGroupLabel,
    updateGroupColor: workflowStore.updateGroupColor,
  }
}
