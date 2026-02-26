import { useState, useRef } from 'react'
import {
  Search, Plus, GitBranch, Sparkles, Trash2, Loader2, Check, X,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { templates } from '@/shared/lib/templates'
import type { TemplateDefinition } from '@/shared/lib/templates'
import type { WorkflowDefinition } from '@/entities/workflow'
import { EditableName } from '@/shared/ui/EditableName'

interface WorkflowSidebarProps {
  workflows: WorkflowDefinition[]
  selectedName: string | null
  onSelect: (name: string) => void
  onNew: (name: string) => void
  onDelete: (name: string) => void
  onRename: (oldName: string, newName: string) => void
  onTemplate: (tpl: TemplateDefinition) => void
  isLoading: boolean
  isCreating: boolean
  runningWorkflows: Set<string>
}

export function WorkflowSidebar({
  workflows,
  selectedName,
  onSelect,
  onNew,

  onDelete,
  onRename,
  onTemplate,
  isLoading,
  isCreating,
  runningWorkflows,
}: WorkflowSidebarProps) {
  const [search, setSearch] = useState('')
  const [isNaming, setIsNaming] = useState(false)
  const [newName, setNewName] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const filtered = workflows.filter((w) =>
    w.name.toLowerCase().includes(search.toLowerCase()),
  )

  const displayTemplates = templates.slice(0, 3)

  const handleCreate = () => {
    const trimmed = newName.trim()
    if (!trimmed || isCreating) return
    onNew(trimmed)
    setIsNaming(false)
    setNewName('')
  }

  const handleCancel = () => {
    setIsNaming(false)
    setNewName('')
  }

  return (
    <div className="flex flex-col h-full animate-in fade-in duration-300">
      {/* Header */}
      <div className="p-4 border-b border-border/50 shrink-0 bg-background/50 backdrop-blur-md shadow-sm z-10">
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="search"
              placeholder="Search workflows..."
              className="w-full h-9 pl-9 pr-4 rounded-lg bg-background border border-input text-sm outline-none focus:ring-1 focus:ring-ring transition-shadow"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <button
            onClick={() => { setIsNaming(true); setTimeout(() => inputRef.current?.focus(), 0) }}
            className="flex items-center gap-1 h-9 px-2.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity shrink-0 cursor-pointer"
          >
            <Plus className="h-3 w-3" />
            New
          </button>
        </div>
      </div>

      {/* Workflow list */}
      <div className="flex-1 overflow-y-auto p-3 space-y-1.5">
        {/* Inline creation input */}
        {isNaming && (
          <div className="flex items-center gap-2 px-3 py-2.5 rounded-xl border border-primary/40 bg-primary/5 mb-1">
            <div className="w-7 h-7 rounded-lg bg-card border border-white/5 flex items-center justify-center shrink-0">
              <GitBranch className="w-3.5 h-3.5 text-blue-400" />
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
              placeholder="Workflow name…"
              disabled={isCreating}
              className="flex-1 min-w-0 text-sm font-semibold bg-transparent outline-none placeholder:text-muted-foreground/50"
            />
            <button
              onClick={handleCreate}
              disabled={!newName.trim() || isCreating}
              className="p-0.5 rounded-md text-success hover:bg-success/10 transition-colors cursor-pointer disabled:opacity-30"
            >
              {isCreating ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
            </button>
            <button
              onClick={handleCancel}
              disabled={isCreating}
              className="p-0.5 rounded-md text-muted-foreground hover:bg-muted/50 transition-colors cursor-pointer"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        )}

        {isLoading ? (
          <div className="flex-1 flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
            <Loader2 className="w-5 h-5 animate-spin text-primary/50" />
            <span className="text-sm font-medium">Loading workflows...</span>
          </div>
        ) : filtered.length === 0 ? (
          workflows.length === 0 ? (
            // Global empty state
            <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-6 gap-4 text-center pt-16">
              <div className="w-14 h-14 rounded-2xl bg-muted/20 flex items-center justify-center">
                <GitBranch className="w-6 h-6 opacity-30" />
              </div>
              <div>
                <p className="font-medium text-foreground">No workflows yet</p>
                <p className="text-xs mt-1">Create your first workflow to get started.</p>
              </div>
              <div className="flex flex-col gap-2 w-full max-w-[200px]">
                <button
                  onClick={() => { setIsNaming(true); setTimeout(() => inputRef.current?.focus(), 0) }}
                  className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl bg-foreground text-background text-sm font-medium hover:opacity-90 transition-opacity cursor-pointer"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Create Workflow
                </button>
                <button
                  onClick={onGenerate}
                  className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl border border-border text-sm font-medium text-foreground hover:bg-muted/50 transition-all cursor-pointer"
                >
                  <Sparkles className="w-3.5 h-3.5" />
                  Generate with AI
                </button>
              </div>
            </div>
          ) : (
            // Search no results
            <div className="text-center py-12 px-4">
              <p className="text-sm text-muted-foreground">No workflows matching &ldquo;{search}&rdquo;</p>
            </div>
          )
        ) : (
          filtered.map((wf) => {
            const isSelected = selectedName === wf.name
            const isRunning = runningWorkflows.has(wf.name)

            return (
              <button
                key={wf.name}
                onClick={() => onSelect(wf.name)}
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
                      <GitBranch className="w-3.5 h-3.5 text-blue-400" />
                    </div>
                    <EditableName
                      value={wf.name}
                      placeholder="Untitled"
                      onSave={(name) => onRename(wf.name, name)}
                      className="text-sm font-semibold truncate"
                    />
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    {isRunning && (
                      <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-node-agent/10 text-node-agent text-[10px] font-bold border border-node-agent/20">
                        <span className="w-1.5 h-1.5 rounded-full bg-node-agent animate-pulse" />
                        Running
                      </span>
                    )}
                    <button
                      onClick={(e) => { e.stopPropagation(); onDelete(wf.name) }}
                      className="p-1 rounded-md text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
                      title="Delete workflow"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
                <div className="flex items-center gap-2 text-[10px] text-muted-foreground/60">
                  <span>{wf.nodes.length} node{wf.nodes.length !== 1 ? 's' : ''}</span>
                </div>
              </button>
            )
          })
        )}
      </div>

      {/* Templates section */}
      {displayTemplates.length > 0 && (
        <div className="border-t border-border/50 shrink-0">
          <div className="px-4 pt-3 pb-1">
            <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/60">
              Templates
            </span>
          </div>
          <div className="px-3 pb-3 space-y-1">
            {displayTemplates.map((tpl) => {
              const Icon = tpl.icon
              return (
                <button
                  key={tpl.id}
                  onClick={() => onTemplate(tpl)}
                  className="w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left hover:bg-muted/50 transition-colors cursor-pointer group"
                >
                  <div className={cn('w-6 h-6 rounded-md flex items-center justify-center shrink-0', tpl.color)}>
                    <Icon className="w-3 h-3" />
                  </div>
                  <span className="text-xs font-medium text-muted-foreground group-hover:text-foreground transition-colors truncate">
                    {tpl.title}
                  </span>
                </button>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
