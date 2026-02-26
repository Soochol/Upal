import { useRef, createRef, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, GitBranch, Rss, Loader2, Pencil, Check, X, Trash2,
} from 'lucide-react'
import { createPipeline, deletePipeline } from '@/entities/pipeline'
import { createDraftSession } from '@/entities/content-session/api'
import { cn } from '@/shared/lib/utils'
import { EditableName } from '@/shared/ui/EditableName'
import type { EditableNameHandle } from '@/shared/ui/EditableName'
import type { Pipeline } from '@/entities/pipeline'
import { updatePipeline } from '@/entities/pipeline/api'

function isContentPipeline(p: Pipeline) {
  return (p.stages ?? []).some(s => s.type === 'collect')
}

interface PipelineSidebarProps {
  pipelines: Pipeline[]
  selectedId: string | null
  onSelect: (id: string) => void
  onDeselect: () => void
  isLoading: boolean
}

export function PipelineSidebar({ pipelines, selectedId, onSelect, onDeselect, isLoading }: PipelineSidebarProps) {
  const queryClient = useQueryClient()
  const editableRefs = useRef<Map<string, React.RefObject<EditableNameHandle | null>>>(new Map())
  const [isCreating, setIsCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const renameMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => updatePipeline(id, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
    },
  })

  const createMutation = useMutation({
    mutationFn: async (name: string) => {
      const pipeline = await createPipeline({ name, stages: [] })
      await createDraftSession({
        pipeline_id: pipeline.id,
        name: `${name} Template`,
        is_template: true,
      })
      return pipeline
    },
    onSuccess: (pipeline) => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
      setIsCreating(false)
      setNewName('')
      onSelect(pipeline.id)
    },
  })

  const handleCreate = () => {
    const trimmed = newName.trim()
    if (!trimmed || createMutation.isPending) return
    createMutation.mutate(trimmed)
  }

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deletePipeline(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
      queryClient.removeQueries({ queryKey: ['content-sessions', { pipelineId: id }] })
      if (selectedId === id) onDeselect()
      setConfirmDeleteId(null)
    },
    onError: () => {
      setConfirmDeleteId(null)
    },
  })

  const handleCancel = () => {
    setIsCreating(false)
    setNewName('')
  }

  return (
    <div className="flex flex-col h-full animate-in fade-in duration-300">
      {/* Header */}
      <div className="px-3 py-3 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md shadow-sm z-10 flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">Pipelines</span>
        <button
          onClick={() => { setIsCreating(true); setTimeout(() => inputRef.current?.focus(), 0) }}
          className="flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity shrink-0 cursor-pointer"
        >
          <Plus className="h-3 w-3" />
          New
        </button>
      </div>

      {/* Pipeline list */}
      <div className="flex-1 overflow-y-auto p-2 space-y-0.5">
        {isCreating && (
          <div className="flex items-center gap-2 px-2.5 py-2 rounded-lg border border-primary/40 bg-primary/5 mb-1">
            <div className="w-6 h-6 rounded-md bg-card border border-white/5 flex items-center justify-center shrink-0">
              <GitBranch className="w-3 h-3 text-blue-400" />
            </div>
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
              placeholder="Pipeline name…"
              disabled={createMutation.isPending}
              className="flex-1 min-w-0 text-sm font-medium bg-transparent outline-none placeholder:text-muted-foreground/50"
            />
            <button
              onClick={handleCreate}
              disabled={!newName.trim() || createMutation.isPending}
              className="p-0.5 rounded-md text-success hover:bg-success/10 transition-colors cursor-pointer disabled:opacity-30"
            >
              {createMutation.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
            </button>
            <button
              onClick={handleCancel}
              disabled={createMutation.isPending}
              className="p-0.5 rounded-md text-muted-foreground hover:bg-muted/50 transition-colors cursor-pointer"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        )}
        {isLoading ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
            <Loader2 className="w-5 h-5 animate-spin text-primary/50" />
          </div>
        ) : pipelines.length === 0 ? (
          <div className="flex flex-col items-center justify-center text-muted-foreground p-4 gap-3 text-center pt-12">
            <div className="w-10 h-10 rounded-xl bg-muted/20 flex items-center justify-center">
              <GitBranch className="w-4 h-4 opacity-30" />
            </div>
            <p className="text-xs">No pipelines yet</p>
          </div>
        ) : (
          pipelines.map((p) => {
            const isSelected = selectedId === p.id
            const isContent = isContentPipeline(p)
            const pendingCount = p.pending_session_count ?? 0

            if (!editableRefs.current.has(p.id)) {
              editableRefs.current.set(p.id, createRef<EditableNameHandle>())
            }
            const editRef = editableRefs.current.get(p.id)!

            return (
              <div
                key={p.id}
                onClick={() => onSelect(p.id)}
                className={cn(
                  'group w-full text-left px-2.5 py-2 rounded-lg border transition-all duration-200 cursor-pointer',
                  'flex flex-wrap items-center gap-2',
                  isSelected
                    ? 'bg-primary/5 border-primary/40 shadow-sm'
                    : 'bg-transparent border-transparent hover:bg-muted/50',
                )}
              >
                <div className="w-6 h-6 rounded-md bg-card border border-white/5 flex items-center justify-center shrink-0">
                  {isContent
                    ? <Rss className="w-3 h-3 text-purple-400" />
                    : <GitBranch className="w-3 h-3 text-blue-400" />}
                </div>
                <EditableName
                  ref={editRef}
                  value={p.name}
                  placeholder="Untitled"
                  onSave={(name) => renameMutation.mutate({ id: p.id, name })}
                  className="text-sm font-medium flex-1 min-w-0 truncate"
                  hideEditButton
                />
                <div className="flex items-center gap-1 shrink-0">
                  {pendingCount > 0 && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded-full bg-warning/10 text-warning text-[10px] font-bold border border-warning/20 tabular-nums">
                      {pendingCount}
                    </span>
                  )}
                  <button
                    onClick={(e) => { e.stopPropagation(); editRef.current?.startEditing() }}
                    className="p-0.5 rounded-md text-muted-foreground/40 hover:text-foreground hover:bg-muted/50 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
                    title="Rename"
                  >
                    <Pencil className="h-3 w-3" />
                  </button>
                  <button
                    onClick={(e) => { e.stopPropagation(); setConfirmDeleteId(p.id) }}
                    className="p-0.5 rounded-md text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
                    title="Delete"
                  >
                    <Trash2 className="h-3 w-3" />
                  </button>
                </div>
                {confirmDeleteId === p.id && (
                  <div
                    className="flex items-center gap-1.5 mt-1 ml-8 text-xs animate-in fade-in duration-200"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <span className="text-destructive font-medium">Delete?</span>
                    <button
                      onClick={() => deleteMutation.mutate(p.id)}
                      disabled={deleteMutation.isPending}
                      className="px-1.5 py-0.5 rounded bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors cursor-pointer font-medium"
                    >
                      {deleteMutation.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : 'Yes'}
                    </button>
                    <button
                      onClick={() => setConfirmDeleteId(null)}
                      className="px-1.5 py-0.5 rounded text-muted-foreground hover:bg-muted/50 transition-colors cursor-pointer"
                    >
                      No
                    </button>
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
