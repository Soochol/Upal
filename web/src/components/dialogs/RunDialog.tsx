import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Play } from 'lucide-react'

export type RunDialogProps = {
  open: boolean
  inputNodes: Array<{ id: string; label: string }>
  onRun: (inputs: Record<string, string>) => void
  onOpenChange: (open: boolean) => void
}

export function RunDialog({ open, inputNodes, onRun, onOpenChange }: RunDialogProps) {
  const [inputs, setInputs] = useState<Record<string, string>>({})

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Run Workflow</DialogTitle>
          <DialogDescription>
            Provide input values for each input node in your workflow.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          {inputNodes.map((node) => (
            <div key={node.id} className="space-y-1.5">
              <Label htmlFor={node.id}>{node.label}</Label>
              <Textarea
                id={node.id}
                value={inputs[node.id] ?? ''}
                onChange={(e) => setInputs({ ...inputs, [node.id]: e.target.value })}
                placeholder={`Enter value for ${node.label}...`}
                autoFocus={inputNodes[0]?.id === node.id}
                className="min-h-[60px] resize-y"
              />
            </div>
          ))}
          {inputNodes.length === 0 && (
            <p className="text-sm text-muted-foreground">
              No input nodes found. The workflow will run with empty inputs.
            </p>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={() => onRun(inputs)}>
            <Play className="h-4 w-4 mr-2" />
            Run
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
