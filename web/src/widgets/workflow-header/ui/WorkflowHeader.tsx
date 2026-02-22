import { Header } from '@/shared/ui/Header'
import type { SaveStatus } from '@/features/manage-canvas'

type WorkflowHeaderProps = {
  workflowName?: string
  onWorkflowNameChange?: (name: string) => void
  saveStatus?: SaveStatus
}

export function WorkflowHeader({ workflowName, onWorkflowNameChange, saveStatus }: WorkflowHeaderProps) {
  return (
    <Header
      workflowName={workflowName}
      onWorkflowNameChange={onWorkflowNameChange}
      saveStatus={saveStatus}
    />
  )
}
