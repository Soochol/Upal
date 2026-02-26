import { useRef, createRef } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, GitBranch, Rss, Loader2, Pencil,
} from 'lucide-react'
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
  isLoading: boolean
}

export function PipelineSidebar({ pipelines, selectedId, onSelect, isLoading }: PipelineSidebarProps) {
  const queryClient = useQueryClient()
  const editableRefs = useRef<Map<string, React.RefObject<EditableNameHandle | null>>>(new Map())

  const renameMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => updatePipeline(id, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
    },
  })

  return (
    <div className="flex flex-col h-full animate-in fade-in duration-300">
      {/* Header */}
      <div className="px-3 py-3 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md shadow-sm z-10 flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">Pipelines</span>
        <Link
          to="/pipelines/new"
          className="flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity shrink-0"
        >
          <Plus className="h-3 w-3" />
          New
        </Link>
      </div>

      {/* Pipeline list */}
      <div className="flex-1 overflow-y-auto p-2 space-y-0.5">
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
              <button
                key={p.id}
                onClick={() => onSelect(p.id)}
                className={cn(
                  'group w-full text-left px-2.5 py-2 rounded-lg border transition-all duration-200 cursor-pointer',
                  'flex items-center gap-2',
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
                </div>
              </button>
            )
          })
        )}
      </div>
    </div>
  )
}
