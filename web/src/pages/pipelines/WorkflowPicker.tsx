import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { X, Search, Plus, Loader2 } from 'lucide-react'
import { apiFetch } from '@/shared/api/client'
import type { PipelineWorkflow } from '@/shared/types'

type WorkflowListItem = {
  name: string
  description?: string
}

interface WorkflowPickerProps {
  existingWorkflows: PipelineWorkflow[]
  onAdd: (workflow: PipelineWorkflow) => void
  onClose: () => void
}

export function WorkflowPicker({ existingWorkflows, onAdd, onClose }: WorkflowPickerProps) {
  const [search, setSearch] = useState('')

  const { data: workflows = [], isLoading } = useQuery<WorkflowListItem[]>({
    queryKey: ['workflows'],
    queryFn: () => apiFetch<WorkflowListItem[]>('/api/workflows'),
  })

  const existingNames = new Set(existingWorkflows.map(w => w.workflow_name))

  const filtered = workflows.filter(w => {
    if (existingNames.has(w.name)) return false
    if (!search) return true
    return w.name.toLowerCase().includes(search.toLowerCase()) ||
      w.description?.toLowerCase().includes(search.toLowerCase())
  })

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-md mx-4 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="text-sm font-semibold">Add Workflow</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Search */}
        <div className="px-5 py-3 border-b border-border">
          <div className="relative">
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
            filtered.map(w => (
              <button
                key={w.name}
                onClick={() => {
                  onAdd({ workflow_name: w.name, auto_select: true })
                  onClose()
                }}
                className="w-full flex items-center gap-3 px-5 py-3 border-b border-border last:border-b-0
                  hover:bg-muted/30 transition-colors text-left cursor-pointer"
              >
                <Plus className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <span className="text-sm font-medium">{w.name}</span>
                  {w.description && (
                    <p className="text-xs text-muted-foreground truncate mt-0.5">{w.description}</p>
                  )}
                </div>
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
