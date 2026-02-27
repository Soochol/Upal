import { useRef, useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Loader2, Check, X, Trash2, Search, FileText } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { EditableName } from '@/shared/ui/EditableName'
import { updateSession, deleteSession as deleteSessionApi } from '@/entities/session'
import type { Session } from '@/entities/session'

const SESSION_STATUS_DOT: Record<string, string> = {
  draft: 'bg-muted-foreground/30',
  active: 'bg-success',
  archived: 'bg-muted-foreground/40',
}

interface SessionSidebarProps {
  sessions: Session[]
  selectedId: string | null
  onSelect: (id: string) => void
  onDeselect: () => void
  onCreate: (name: string) => void
  isLoading: boolean
}

export function SessionSidebar({ sessions, selectedId, onSelect, onDeselect, onCreate, isLoading }: SessionSidebarProps) {
  const queryClient = useQueryClient()
  const [isCreating, setIsCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [search, setSearch] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const filteredSessions = useMemo(() => {
    if (!search) return sessions
    const q = search.toLowerCase()
    return sessions.filter(s => s.name.toLowerCase().includes(q))
  }, [sessions, search])

  const renameMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => updateSession(id, { name }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['sessions'] }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteSessionApi(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
      if (selectedId === id) onDeselect()
    },
  })

  const handleCreate = () => {
    const trimmed = newName.trim()
    if (!trimmed) return
    onCreate(trimmed)
    setIsCreating(false)
    setNewName('')
  }

  const handleCancel = () => { setIsCreating(false); setNewName('') }

  return (
    <div className="flex flex-col h-full animate-in fade-in duration-300">
      <div className="px-3 py-3 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md shadow-sm z-10 space-y-2">
        <span className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">Sessions</span>
        {!isLoading && sessions.length > 0 && (
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <input
                type="search"
                placeholder="Search sessions..."
                className="w-full h-8 pl-8 pr-3 rounded-lg bg-background border border-input text-sm outline-none focus:ring-1 focus:ring-ring transition-shadow"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <button
              onClick={() => { setIsCreating(true); setTimeout(() => inputRef.current?.focus(), 0) }}
              className="flex items-center gap-1 h-8 px-2 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity shrink-0 cursor-pointer"
            >
              <Plus className="h-3 w-3" />
              New
            </button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-0.5">
        {isCreating && (
          <div className="flex items-center gap-2 px-2.5 py-2 rounded-lg border border-primary/40 bg-primary/5 mb-1">
            <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground/60" />
            <input
              ref={inputRef}
              type="text"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreate()
                if (e.key === 'Escape') handleCancel()
              }}
              onBlur={() => { if (!newName.trim()) handleCancel() }}
              placeholder="Session name..."
              className="flex-1 min-w-0 text-sm font-medium bg-transparent outline-none placeholder:text-muted-foreground/50"
            />
            <button onClick={handleCreate} disabled={!newName.trim()} className="p-0.5 rounded-md text-success hover:bg-success/10 transition-colors cursor-pointer disabled:opacity-30">
              <Check className="h-3.5 w-3.5" />
            </button>
            <button onClick={handleCancel} className="p-0.5 rounded-md text-muted-foreground hover:bg-muted/50 transition-colors cursor-pointer">
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        )}
        {isLoading ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
            <Loader2 className="w-5 h-5 animate-spin text-primary/50" />
          </div>
        ) : sessions.length === 0 ? (
          <div className="flex flex-col items-center justify-center text-muted-foreground p-4 gap-4 text-center pt-12">
            <button
              onClick={() => { setIsCreating(true); setTimeout(() => inputRef.current?.focus(), 0) }}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
            >
              <Plus className="h-3.5 w-3.5" />
              New Session
            </button>
          </div>
        ) : (
          filteredSessions.map((s) => {
            const isSelected = selectedId === s.id
            return (
              <div
                key={s.id}
                onClick={() => onSelect(s.id)}
                className={cn(
                  'group w-full text-left p-3 rounded-xl transition-all duration-200 cursor-pointer border min-h-[80px]',
                  isSelected
                    ? 'bg-primary/5 border-primary/20 shadow-sm'
                    : 'bg-transparent border-transparent hover:bg-muted/50',
                )}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-1.5 min-w-0">
                    <span className={cn('w-1.5 h-1.5 rounded-full shrink-0', SESSION_STATUS_DOT[s.status] ?? 'bg-muted-foreground/30')} />
                    <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground/60" />
                    <EditableName
                      value={s.name}
                      placeholder="Untitled"
                      onSave={(name) => renameMutation.mutate({ id: s.id, name })}
                      className={cn('text-sm font-semibold', isSelected ? 'text-primary' : 'text-foreground')}
                    />
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    <button
                      onClick={(e) => { e.stopPropagation(); if (confirm(`Delete "${s.name}"?`)) deleteMutation.mutate(s.id) }}
                      className="p-0.5 rounded-md text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
                      title="Delete"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                </div>
                <p className="text-[10px] text-muted-foreground/50 mt-1 pl-6 capitalize">{s.status}</p>
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
