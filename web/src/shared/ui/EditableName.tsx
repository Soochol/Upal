import { useState, useRef, useEffect, useCallback, useImperativeHandle, forwardRef } from 'react'
import { Pencil } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

export interface EditableNameHandle {
  startEditing: () => void
}

interface EditableNameProps {
  value: string
  placeholder?: string
  onSave: (name: string) => void
  className?: string
  inputClassName?: string
  hideEditButton?: boolean
}

export const EditableName = forwardRef<EditableNameHandle, EditableNameProps>(function EditableName({
  value,
  placeholder = 'Untitled',
  onSave,
  className,
  inputClassName,
  hideEditButton,
}, ref) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(value)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => { setDraft(value) }, [value])

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [editing])

  const commit = useCallback(() => {
    setEditing(false)
    const trimmed = draft.trim()
    if (trimmed && trimmed !== value) onSave(trimmed)
    else setDraft(value)
  }, [draft, value, onSave])

  useImperativeHandle(ref, () => ({
    startEditing: () => setEditing(true),
  }), [])

  const cancel = useCallback(() => {
    setEditing(false)
    setDraft(value)
  }, [value])

  if (editing) {
    return (
      <input
        ref={inputRef}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') commit()
          if (e.key === 'Escape') cancel()
        }}
        onClick={(e) => e.stopPropagation()}
        placeholder={placeholder}
        className={cn(
          'bg-transparent border-b border-primary/40 outline-none text-sm font-semibold truncate min-w-0 w-full py-0 px-0',
          inputClassName,
        )}
      />
    )
  }

  return (
    <span
      className={cn('group/editable inline-flex items-center gap-1 min-w-0', className)}
      onDoubleClick={(e) => { e.stopPropagation(); setEditing(true) }}
    >
      <span className="truncate">{value || placeholder}</span>
      {!hideEditButton && (
        <button
          type="button"
          onClick={(e) => { e.stopPropagation(); setEditing(true) }}
          className="p-0.5 rounded text-muted-foreground/0 group-hover/editable:text-muted-foreground hover:!text-foreground transition-colors cursor-pointer shrink-0"
          title="Rename"
        >
          <Pencil className="h-3 w-3" />
        </button>
      )}
    </span>
  )
})
