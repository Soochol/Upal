import { PromptBar } from './PromptBar'

type CanvasPromptBarProps = {
  onSubmit: (description: string) => void
  isGenerating: boolean
  hasNodes: boolean
}

export function CanvasPromptBar({ onSubmit, isGenerating, hasNodes }: CanvasPromptBarProps) {
  return (
    <PromptBar
      onSubmit={onSubmit}
      isGenerating={isGenerating}
      placeholder={hasNodes ? 'Edit these steps...' : 'Describe your workflow...'}
      positioning="absolute"
      hint="Upal can make mistakes, so double-check it"
    />
  )
}
