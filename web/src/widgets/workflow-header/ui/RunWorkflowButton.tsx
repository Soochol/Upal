import { Play } from 'lucide-react'

type RunWorkflowButtonProps = {
  onRun: () => void
  disabled?: boolean
}

export function RunWorkflowButton({ onRun, disabled }: RunWorkflowButtonProps) {
  return (
    <button
      onClick={onRun}
      disabled={disabled}
      className="h-8 px-3.5 rounded-lg bg-primary text-primary-foreground text-xs font-semibold
                 flex items-center gap-1.5
                 shadow-sm shadow-primary/25
                 hover:shadow-md hover:shadow-primary/35 hover:brightness-105
                 disabled:opacity-40 disabled:shadow-none disabled:cursor-not-allowed
                 transition-all duration-150 active:scale-[0.97]"
    >
      <Play className="h-3 w-3" fill="currentColor" />
      Run
    </button>
  )
}
