import { useState, useRef, createRef } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Search, Plus, GitBranch, Rss, Sparkles, Loader2, Pencil,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { EditableName } from '@/shared/ui/EditableName'
import type { EditableNameHandle } from '@/shared/ui/EditableName'
import type { Pipeline } from '@/entities/pipeline'
import { updatePipeline } from '@/entities/pipeline/api'

type PipelineTab = 'all' | 'content'

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
  const [search, setSearch] = useState('')
  const [activeTab, setActiveTab] = useState<PipelineTab>('all')
  const editableRefs = useRef<Map<string, React.RefObject<EditableNameHandle | null>>>(new Map())

  const renameMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => updatePipeline(id, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
    },
  })

  const contentPipelines = pipelines.filter(isContentPipeline)
  const displayPipelines = activeTab === 'content' ? contentPipelines : pipelines
  const filtered = displayPipelines.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <div className="flex flex-col h-full animate-in fade-in duration-300">
      {/* Header */}
      <div className="p-4 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md shadow-sm z-10 space-y-3">
        {/* Search */}
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="search"
            placeholder="Search pipelines..."
            className="w-full h-9 pl-9 pr-4 rounded-lg bg-background border border-input text-sm outline-none focus:ring-1 focus:ring-ring transition-shadow"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        {/* Tabs + New button */}
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-1 p-0.5 rounded-lg bg-muted/30 shrink-0">
            <button
              onClick={() => setActiveTab('all')}
              className={cn(
                'px-2.5 py-1 rounded-md text-xs font-medium transition-colors cursor-pointer',
                activeTab === 'all'
                  ? 'bg-foreground text-background'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              All
            </button>
            <button
              onClick={() => setActiveTab('content')}
              className={cn(
                'flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium transition-colors cursor-pointer',
                activeTab === 'content'
                  ? 'bg-foreground text-background'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              Content
              {contentPipelines.length > 0 && (
                <span className={cn(
                  'text-[10px] font-bold tabular-nums px-1 rounded-full',
                  activeTab === 'content' ? 'bg-background/20' : 'bg-muted-foreground/20',
                )}>
                  {contentPipelines.length}
                </span>
              )}
            </button>
          </div>

          <Link
            to="/pipelines/new"
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity shrink-0"
          >
            <Plus className="h-3 w-3" />
            New
          </Link>
        </div>
      </div>

      {/* Pipeline list */}
      <div className="flex-1 overflow-y-auto p-3 space-y-1.5">
        {isLoading ? (
          <div className="flex-1 flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
            <Loader2 className="w-5 h-5 animate-spin text-primary/50" />
            <span className="text-sm font-medium">Loading pipelines...</span>
          </div>
        ) : filtered.length === 0 ? (
          pipelines.length === 0 ? (
            // Global empty state
            <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-6 gap-4 text-center pt-16">
              <div className="w-14 h-14 rounded-2xl bg-muted/20 flex items-center justify-center">
                <GitBranch className="w-6 h-6 opacity-30" />
              </div>
              <div>
                <p className="font-medium text-foreground">No pipelines yet</p>
                <p className="text-xs mt-1">Create your first pipeline to get started.</p>
              </div>
              <div className="flex flex-col gap-2 w-full max-w-[200px]">
                <Link
                  to="/pipelines/new"
                  className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl bg-foreground text-background text-sm font-medium hover:opacity-90 transition-opacity"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Create Pipeline
                </Link>
                <Link
                  to="/pipelines/new?generate=true"
                  className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl border border-border text-sm font-medium text-foreground hover:bg-muted/50 transition-all"
                >
                  <Sparkles className="w-3.5 h-3.5" />
                  Generate with AI
                </Link>
              </div>
            </div>
          ) : (
            // Search no results
            <div className="text-center py-12 px-4">
              <p className="text-sm text-muted-foreground">No pipelines matching &ldquo;{search}&rdquo;</p>
            </div>
          )
        ) : (
          filtered.map((p) => {
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
                  'group w-full text-left p-3.5 rounded-xl border transition-all duration-200 cursor-pointer flex flex-col gap-1.5',
                  isSelected
                    ? 'bg-primary/5 border-primary/40 shadow-sm ring-1 ring-primary/20'
                    : 'bg-card border-border/60 hover:border-primary/40 hover:bg-muted/50',
                )}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <div className="w-7 h-7 rounded-lg bg-card border border-white/5 flex items-center justify-center shrink-0">
                      {isContent
                        ? <Rss className="w-3.5 h-3.5 text-purple-400" />
                        : <GitBranch className="w-3.5 h-3.5 text-blue-400" />}
                    </div>
                    <EditableName
                      ref={editRef}
                      value={p.name}
                      placeholder="Untitled Pipeline"
                      onSave={(name) => renameMutation.mutate({ id: p.id, name })}
                      className="text-sm font-semibold"
                      hideEditButton
                    />
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    {pendingCount > 0 && (
                      <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-warning/10 text-warning text-[10px] font-bold border border-warning/20 tabular-nums">
                        {pendingCount}
                      </span>
                    )}
                    <button
                      onClick={(e) => { e.stopPropagation(); editRef.current?.startEditing() }}
                      className="p-1 rounded-md text-muted-foreground/40 hover:text-foreground hover:bg-muted/50 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
                      title="Rename"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
                {p.description && (
                  <p className="text-xs text-muted-foreground truncate">{p.description}</p>
                )}
                <div className="flex items-center gap-2 text-[10px] text-muted-foreground/60">
                  <span>{(p.stages ?? []).length} stages</span>
                </div>
              </button>
            )
          })
        )}
      </div>
    </div>
  )
}
