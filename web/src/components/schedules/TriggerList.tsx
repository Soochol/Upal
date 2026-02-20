import { useState } from 'react'
import { Webhook, Plus } from 'lucide-react'
import { createTrigger, type Trigger } from '@/lib/api'
import { TriggerRow } from './TriggerRow'

type Props = {
  triggers: Trigger[]
  workflowNames: string[]
  onDelete: (id: string) => void
  onCreated: () => void
}

export function TriggerList({ triggers, workflowNames, onDelete, onCreated }: Props) {
  const [adding, setAdding] = useState(false)
  const [workflowName, setWorkflowName] = useState('')
  const [creating, setCreating] = useState(false)

  const handleCreate = async () => {
    if (!workflowName) return
    setCreating(true)
    try {
      await createTrigger({
        workflow_name: workflowName,
        type: 'webhook',
        enabled: true,
      })
      setAdding(false)
      setWorkflowName('')
      onCreated()
    } catch {
      // silently handle
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="space-y-3">
      {triggers.length === 0 ? (
        <div className="text-center py-10">
          <Webhook className="h-8 w-8 text-muted-foreground/40 mx-auto mb-2" />
          <p className="text-sm text-muted-foreground">No webhook triggers configured.</p>
        </div>
      ) : (
        <div className="border border-border rounded-lg overflow-hidden">
          {triggers.map((t) => (
            <TriggerRow key={t.id} trigger={t} onDelete={onDelete} />
          ))}
        </div>
      )}

      {adding ? (
        <div className="border border-border rounded-lg p-4 bg-card space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium text-foreground">New Webhook Trigger</h3>
            <button
              onClick={() => setAdding(false)}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Cancel
            </button>
          </div>
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Workflow</label>
            <select
              value={workflowName}
              onChange={(e) => setWorkflowName(e.target.value)}
              className="w-full h-8 px-2 rounded-md border border-border bg-background text-sm text-foreground"
            >
              <option value="">Select workflow...</option>
              {workflowNames.map((n) => (
                <option key={n} value={n}>{n}</option>
              ))}
            </select>
          </div>
          <button
            onClick={handleCreate}
            disabled={!workflowName || creating}
            className="w-full h-8 rounded-md bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50 transition-opacity"
          >
            {creating ? 'Creating...' : 'Create Trigger'}
          </button>
        </div>
      ) : (
        <button
          onClick={() => setAdding(true)}
          className="flex items-center gap-1.5 px-3 py-2 text-sm text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded-md transition-colors w-full"
        >
          <Plus className="h-3.5 w-3.5" />
          Add Webhook
        </button>
      )}
    </div>
  )
}
