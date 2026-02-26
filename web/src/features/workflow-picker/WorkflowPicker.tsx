import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { X, Search, Plus, Loader2, Check } from 'lucide-react'
import { listWorkflows } from '@/entities/workflow'
import type { SessionWorkflow } from '@/entities/session'

type WorkflowListItem = {
  name: string
  description?: string
}

// --- Mode A: pipeline add (legacy) ---
interface AddModeProps {
  mode?: 'add'
  existingWorkflows: SessionWorkflow[]
  onAdd: (workflow: SessionWorkflow) => void
  onClose: () => void
  title?: string
}

// --- Mode B: simple select (returns name only) ---
interface SelectModeProps {
  mode: 'select'
  currentWorkflow?: string
  onSelect: (workflowName: string) => void
  onClose: () => void
  title?: string
}

type WorkflowPickerProps = AddModeProps | SelectModeProps

export function WorkflowPicker(props: WorkflowPickerProps) {
  const { onClose, title } = props
  const [search, setSearch] = useState('')

  const { data: workflows = [], isLoading } = useQuery<WorkflowListItem[]>({
    queryKey: ['workflows'],
    queryFn: () => listWorkflows() as Promise<WorkflowListItem[]>,
  })

  const isSelectMode = props.mode === 'select'
  const existingNames = !isSelectMode
    ? new Set((props as AddModeProps).existingWorkflows.map(ew => ew.workflow_name))
    : undefined

  const filtered = workflows.filter(w => {
    if (existingNames?.has(w.name)) return false
    if (!search) return true
    return w.name.toLowerCase().includes(search.toLowerCase()) ||
      w.description?.toLowerCase().includes(search.toLowerCase())
  })

  const handleClick = (w: WorkflowListItem) => {
    if (isSelectMode) {
      (props as SelectModeProps).onSelect(w.name)
    } else {
      (props as AddModeProps).onAdd({ workflow_name: w.name, auto_select: true })
    }
    onClose()
  }

  const currentWorkflow = isSelectMode ? (props as SelectModeProps).currentWorkflow : undefined

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-md mx-4 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="text-sm font-semibold">{title ?? (isSelectMode ? 'Select Workflow' : 'Add Workflow')}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Search & Actions */}
        <div className="px-5 py-3 border-b border-border flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            <input
              type="search"
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search workflows..."
              autoFocus
              className="w-full pl-8 pr-3 py-2 rounded-lg border border-input bg-background
                text-sm outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
            />
          </div>
          <button
            onClick={() => {
              const returnTo = encodeURIComponent(window.location.pathname)
              window.open(`/workflows?returnTo=${returnTo}`, '_blank')
            }}
            className="flex items-center justify-center gap-1.5 px-3 py-2 text-sm font-medium
              bg-foreground text-background hover:opacity-90 transition-opacity rounded-lg whitespace-nowrap"
          >
            <Plus className="h-3.5 w-3.5" />
            Create New
          </button>
        </div>

        {/* List */}
        <div className="max-h-80 overflow-y-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            </div>
          ) : filtered.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-8">
              {search ? 'No matching workflows.' : 'No workflows available.'}
            </p>
          ) : (
            filtered.map(w => {
              const isCurrent = currentWorkflow === w.name
              return (
                <button
                  key={w.name}
                  onClick={() => handleClick(w)}
                  className={`w-full flex items-center gap-3 px-5 py-3 border-b border-border last:border-b-0
                    hover:bg-muted/30 transition-colors text-left cursor-pointer
                    ${isCurrent ? 'bg-success/5' : ''}`}
                >
                  {isCurrent ? (
                    <Check className="h-3.5 w-3.5 text-success shrink-0" />
                  ) : (
                    <Plus className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                  )}
                  <div className="flex-1 min-w-0">
                    <span className={`text-sm font-medium ${isCurrent ? 'text-success' : ''}`}>{w.name}</span>
                    {w.description && (
                      <p className="text-xs text-muted-foreground truncate mt-0.5">{w.description}</p>
                    )}
                  </div>
                </button>
              )
            })
          )}
        </div>
      </div>
    </div>
  )
}
