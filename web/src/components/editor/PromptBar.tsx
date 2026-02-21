import { useState, useRef, useCallback } from 'react'
import { Sparkles, ArrowRight, Loader2 } from 'lucide-react'

type Props = {
  onSubmit: (value: string) => void
  isGenerating: boolean
  placeholder?: string
  positioning?: 'absolute' | 'fixed'
  hint?: string
  error?: string | null
  onValueChange?: () => void
}

export function PromptBar({
  onSubmit,
  isGenerating,
  placeholder = 'Describe...',
  positioning = 'absolute',
  hint,
  error,
  onValueChange,
}: Props) {
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

  const posClass = positioning === 'fixed'
    ? 'fixed bottom-6 left-1/2 -translate-x-1/2 z-20'
    : 'absolute bottom-6 left-1/2 -translate-x-1/2 z-20'

  return (
    <div className={`${posClass} flex flex-col items-center gap-1.5 w-full max-w-xl px-4`}>
      <div className="prompt-bar w-full flex items-center gap-2 rounded-xl border border-border/60 bg-card/80 backdrop-blur-xl shadow-lg px-4 py-2.5 transition-all duration-200 focus-within:border-ring/50 focus-within:shadow-xl">
        <Sparkles className="h-4 w-4 text-muted-foreground/60 shrink-0" />
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => { setValue(e.target.value); onValueChange?.() }}
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
      {error
        ? <p className="text-[11px] text-destructive/80 select-none">{error}</p>
        : hint && <p className="text-[11px] text-muted-foreground/40 select-none">{hint}</p>
      }
    </div>
  )
}
