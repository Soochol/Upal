import { Loader2, Check, AlertCircle, Clock, Copy } from 'lucide-react'
import { Input } from '@/shared/ui/input'
import { ThemeToggle } from '@/shared/ui/ThemeToggle'
import type { SaveStatus } from '@/shared/hooks/useAutoSave'
import { RunWorkflowButton } from './RunWorkflowButton'

type WorkflowHeaderProps = {
  workflowName?: string
  onWorkflowNameChange?: (name: string) => void
  saveStatus?: SaveStatus
  onRun?: () => void
  isTemplate?: boolean
  templateName?: string
  onRemix?: () => void
  returnTo?: string | null
}

function SaveStatusIndicator({ saveStatus }: { saveStatus?: SaveStatus }) {
  if (!saveStatus) return null
  if (saveStatus === 'waiting') return (
    <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
      <Clock className="h-3 w-3" />
      Waiting to save
    </span>
  )
  if (saveStatus === 'saving') return (
    <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
      <Loader2 className="h-3 w-3 animate-spin" />
      Saving...
    </span>
  )
  if (saveStatus === 'saved') return (
    <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
      <Check className="h-3 w-3" />
      Saved
    </span>
  )
  if (saveStatus === 'error') return (
    <span className="flex items-center gap-1 text-[10px] text-destructive">
      <AlertCircle className="h-3 w-3" />
      Save failed
    </span>
  )
  return null
}

export function WorkflowHeader({ workflowName, onWorkflowNameChange, saveStatus, onRun, isTemplate, templateName, onRemix, returnTo }: WorkflowHeaderProps) {
  return (
    <div className="flex items-center justify-between flex-1 gap-3">
      <div className="flex items-center gap-2">
        {returnTo && (
          <>
            <button
              onClick={() => { window.location.href = decodeURIComponent(returnTo) }}
              className="inline-flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg bg-secondary text-secondary-foreground hover:bg-secondary/80 transition-colors mr-2 cursor-pointer border border-border shrink-0"
              title="Return to previous page after saving"
            >
              <Check className="h-3.5 w-3.5 text-success" />
              Save & Return
            </button>
            <div className="w-px h-5 bg-border mx-1" />
          </>
        )}
        {isTemplate ? (
          <>
            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-primary/10 border border-primary/20 text-xs font-medium text-primary">
              Template
            </span>
            <span className="text-sm font-semibold text-foreground truncate max-w-[200px]">
              {templateName || workflowName}
            </span>
          </>
        ) : (
          <>
            {workflowName !== undefined && onWorkflowNameChange && (
              <Input
                className="h-8 w-44 text-sm"
                placeholder="Workflow name..."
                value={workflowName}
                onChange={(e) => onWorkflowNameChange(e.target.value)}
              />
            )}
            <SaveStatusIndicator saveStatus={saveStatus} />
          </>
        )}
      </div>
      <div className="flex items-center gap-2">
        {isTemplate && onRemix && (
          <button
            onClick={onRemix}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg
              border border-primary/30 text-primary hover:bg-primary/10
              transition-colors duration-150"
          >
            <Copy className="size-3.5" />
            Remix
          </button>
        )}
        <RunWorkflowButton onRun={onRun ?? (() => { })} />
        <ThemeToggle />
      </div>
    </div>
  )
}
