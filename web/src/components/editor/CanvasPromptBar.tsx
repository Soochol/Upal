import { useState, useRef, useCallback } from 'react'
import { Sparkles, ArrowRight, Loader2 } from 'lucide-react'

type CanvasPromptBarProps = {
  onSubmit: (description: string) => void
  isGenerating: boolean
  hasNodes: boolean
}

export function CanvasPromptBar({ onSubmit, isGenerating, hasNodes }: CanvasPromptBarProps) {
  const [value, setValue] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const handleSubmit = useCallback(() => {
    const trimmed = value.trim()
    if (!trimmed || isGenerating) return
    onSubmit(trimmed)
    setValue('')
  }, [value, isGenerating, onSubmit])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSubmit()
      }
    },
    [handleSubmit],
  )

  const placeholder = hasNodes
    ? 'Edit these steps...'
    : 'Describe your workflow...'

  return (
    <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-20 flex flex-col items-center gap-1.5 w-full max-w-xl px-4">
      <div className="prompt-bar w-full flex items-center gap-2 rounded-xl border border-border/60 bg-card/80 backdrop-blur-xl shadow-lg px-4 py-2.5 transition-all duration-200 focus-within:border-ring/50 focus-within:shadow-xl">
        <Sparkles className="h-4 w-4 text-muted-foreground/60 shrink-0" />
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={isGenerating}
          className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground/50 outline-none disabled:opacity-50"
        />
        <button
          onClick={handleSubmit}
          disabled={!value.trim() || isGenerating}
          className="shrink-0 h-8 w-8 rounded-lg flex items-center justify-center bg-primary text-primary-foreground transition-all duration-150 hover:bg-primary/90 disabled:opacity-30 disabled:pointer-events-none"
        >
          {isGenerating ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <ArrowRight className="h-4 w-4" />
          )}
        </button>
      </div>
      <p className="text-[11px] text-muted-foreground/40 select-none">
        Upal can make mistakes, so double-check it
      </p>
    </div>
  )
}
