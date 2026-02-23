import { useState, useRef } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

type KeywordTagInputProps = {
  keywords: string[]
  onChange: (keywords: string[]) => void
  placeholder?: string
  className?: string
}

export function KeywordTagInput({
  keywords,
  onChange,
  placeholder = '키워드 입력 후 Enter',
  className,
}: KeywordTagInputProps) {
  const [input, setInput] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const addKeyword = (raw: string) => {
    const kw = raw.trim()
    if (kw && !keywords.includes(kw)) {
      onChange([...keywords, kw])
    }
    setInput('')
  }

  const removeKeyword = (kw: string) => {
    onChange(keywords.filter((k) => k !== kw))
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addKeyword(input)
    } else if (e.key === 'Backspace' && input === '' && keywords.length > 0) {
      onChange(keywords.slice(0, -1))
    }
  }

  return (
    <div
      className={cn(
        'flex flex-wrap gap-1.5 rounded-xl border border-input bg-background px-3 py-2 min-h-[40px] cursor-text',
        className,
      )}
      onClick={() => inputRef.current?.focus()}
    >
      {keywords.map((kw) => (
        <span
          key={kw}
          className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full
            bg-accent text-foreground text-xs font-medium"
        >
          {kw}
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); removeKeyword(kw) }}
            className="text-muted-foreground hover:text-foreground transition-colors"
          >
            <X className="h-3 w-3" />
          </button>
        </span>
      ))}
      <input
        ref={inputRef}
        type="text"
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={() => { if (input.trim()) addKeyword(input) }}
        placeholder={keywords.length === 0 ? placeholder : ''}
        className="flex-1 min-w-[80px] bg-transparent text-sm outline-none placeholder:text-muted-foreground"
      />
    </div>
  )
}
