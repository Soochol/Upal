import { useState, useEffect, useCallback, useRef } from 'react'
import { Sparkles, ArrowRight, Loader2, AlertCircle } from 'lucide-react'
import { Link } from 'react-router-dom'

type CanvasPromptBarProps = {
  onSubmit: (description: string) => void
  isGenerating: boolean
  hasNodes: boolean
  autoFocusPrompt?: boolean
  hasDefaultLLM?: boolean
}

export function CanvasPromptBar({ onSubmit, isGenerating, hasNodes, autoFocusPrompt, hasDefaultLLM = true }: CanvasPromptBarProps) {
  const [expanded, setExpanded] = useState(false)
  const [value, setValue] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const wasGeneratingRef = useRef(false)

  useEffect(() => {
    if (autoFocusPrompt) setExpanded(true)
  }, [autoFocusPrompt])

  useEffect(() => {
    if (isGenerating) setExpanded(true)
  }, [isGenerating])

  // Clear input after generation finishes
  useEffect(() => {
    if (wasGeneratingRef.current && !isGenerating) {
      setValue('')
    }
    wasGeneratingRef.current = isGenerating
  }, [isGenerating])

  // Auto-focus input when expanding
  useEffect(() => {
    if (expanded) {
      // Wait for transition to start before focusing
      requestAnimationFrame(() => inputRef.current?.focus())
    }
  }, [expanded])

  const handleExpand = useCallback(() => setExpanded(true), [])

  const disabled = isGenerating || !hasDefaultLLM

  const handleSubmit = useCallback(() => {
    const trimmed = value.trim()
    if (!trimmed || disabled) return
    onSubmit(trimmed)
  }, [value, disabled, onSubmit])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSubmit()
      }
      if (e.key === 'Escape' && !value.trim()) {
        setExpanded(false)
      }
    },
    [handleSubmit, value],
  )

  const handleBlur = useCallback(
    (e: React.FocusEvent) => {
      if (isGenerating || value.trim()) return
      if (containerRef.current?.contains(e.relatedTarget as Node)) return
      setExpanded(false)
    },
    [isGenerating, value],
  )

  const placeholder = hasNodes ? 'Edit these steps...' : 'Describe your workflow...'

  return (
    <div
      ref={containerRef}
      onBlur={handleBlur}
      className="absolute bottom-6 left-1/2 -translate-x-1/2 z-20 px-4"
    >
      <div
        onMouseEnter={expanded ? undefined : handleExpand}
        className={[
          'prompt-bar flex items-center gap-2 rounded-xl border border-border/60 bg-card/80 backdrop-blur-xl shadow-lg px-4 py-2.5',
          'transition-all duration-300 ease-out',
          expanded
            ? 'w-[min(32rem,calc(100vw-2rem))] focus-within:border-ring/50 focus-within:shadow-xl'
            : 'w-11 cursor-pointer hover:shadow-xl hover:border-ring/50',
        ].join(' ')}
      >
        <Sparkles className="h-4 w-4 text-muted-foreground/60 shrink-0" />

        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={hasDefaultLLM ? placeholder : 'Settings에서 LLM을 설정해주세요'}
          disabled={disabled}
          tabIndex={expanded ? 0 : -1}
          className={[
            'flex-1 min-w-0 bg-transparent text-sm text-foreground placeholder:text-muted-foreground/50 outline-none disabled:opacity-50',
            'transition-opacity duration-200',
            expanded ? 'opacity-100' : 'opacity-0 w-0 pointer-events-none',
          ].join(' ')}
        />

        <button
          onClick={handleSubmit}
          disabled={!value.trim() || disabled}
          tabIndex={expanded ? 0 : -1}
          className={[
            'shrink-0 h-8 w-8 rounded-lg flex items-center justify-center bg-primary text-primary-foreground',
            'transition-all duration-200 hover:bg-primary/90 disabled:opacity-30 disabled:pointer-events-none',
            expanded ? 'opacity-100 scale-100' : 'opacity-0 scale-75 w-0 pointer-events-none',
          ].join(' ')}
        >
          {isGenerating ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <ArrowRight className="h-4 w-4" />
          )}
        </button>
      </div>

      {expanded && (
        !hasDefaultLLM ? (
          <Link
            to="/settings"
            className="absolute left-1/2 -translate-x-1/2 top-full mt-1.5 flex items-center gap-1 text-[11px] text-warning select-none whitespace-nowrap animate-in fade-in duration-300 hover:underline"
          >
            <AlertCircle className="size-3" />
            Settings에서 기본 LLM을 설정해주세요
          </Link>
        ) : (
          <p className="absolute left-1/2 -translate-x-1/2 top-full mt-1.5 text-[11px] text-muted-foreground/40 select-none whitespace-nowrap animate-in fade-in duration-300">
            Upal can make mistakes, so double-check it
          </p>
        )
      )}
    </div>
  )
}
