import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Search, Plus, GitBranch, Rss, Sparkles, Clock, Loader2, Settings, Trash2,
} from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { cn } from '@/shared/lib/utils'
import { humanReadableCron } from '@/shared/lib/cron'
import { deletePipeline } from '@/entities/pipeline'
import { ConfirmDialog } from '@/shared/ui/ConfirmDialog'
import type { Pipeline } from '@/shared/types'

function isContentPipeline(p: Pipeline) {
  return (p.sources && p.sources.length > 0) || !!p.schedule || !!p.context
}

interface PipelineSidebarProps {
  pipelines: Pipeline[]
  selectedId: string | null
  onSelect: (id: string) => void
  isLoading: boolean
  onSettingsOpen?: () => void
  onDelete?: (id: string) => void
}

export function PipelineSidebar({ pipelines, selectedId, onSelect, isLoading, onSettingsOpen, onDelete }: PipelineSidebarProps) {
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<Pipeline | null>(null)

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deletePipeline(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
      setDeleteTarget(null)
      onDelete?.(id)
    },
  })

  const filtered = pipelines.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <div className="flex flex-col h-full animate-in fade-in duration-300">
      {/* Header */}
      <div className="p-4 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md shadow-sm z-10 space-y-3">
        {/* Label + New */}
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold">Pipelines</h2>
          <Link
            to="/pipelines/new"
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity shrink-0"
          >
            <Plus className="h-3 w-3" />
            New
          </Link>
        </div>

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
                    <span className="text-sm font-semibold truncate">{p.name}</span>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    {pendingCount > 0 && (
                      <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-warning/10 text-warning text-[10px] font-bold border border-warning/20 tabular-nums">
                        {pendingCount}
                      </span>
                    )}
                    <button
                      onClick={(e) => { e.stopPropagation(); onSelect(p.id); onSettingsOpen?.() }}
                      className="p-1 rounded-md text-muted-foreground/40 hover:text-foreground hover:bg-muted/50 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
                      title="Settings"
                    >
                      <Settings className="h-3.5 w-3.5" />
                    </button>
                    <button
                      onClick={(e) => { e.stopPropagation(); setDeleteTarget(p) }}
                      className="p-1 rounded-md text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
                      title="Delete pipeline"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
                {p.description && (
                  <p className="text-xs text-muted-foreground truncate">{p.description}</p>
                )}
                <div className="flex items-center gap-2 text-[10px] text-muted-foreground/60">
                  <span>{(p.stages ?? []).length} stages</span>
                  {p.schedule && (
                    <>
                      <span className="text-muted-foreground/30">·</span>
                      <span className="inline-flex items-center gap-0.5">
                        <Clock className="w-2.5 h-2.5" />
                        {humanReadableCron(p.schedule)}
                      </span>
                    </>
                  )}
                  {(p.sources?.length ?? 0) > 0 && (
                    <>
                      <span className="text-muted-foreground/30">·</span>
                      <span>{p.sources!.length} sources</span>
                    </>
                  )}
                </div>
              </button>
            )
          })
        )}
      </div>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="Delete pipeline"
        description={`"${deleteTarget?.name}" and all its sessions will be permanently deleted.`}
        isPending={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </div>
  )
}
