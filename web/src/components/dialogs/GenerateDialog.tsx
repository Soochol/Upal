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
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Sparkles, Loader2 } from 'lucide-react'

export type GenerateDialogProps = {
  open: boolean
  onGenerate: (description: string, model?: string) => void
  onOpenChange: (open: boolean) => void
  isGenerating: boolean
}

export function GenerateDialog({
  open,
  onGenerate,
  onOpenChange,
  isGenerating,
}: GenerateDialogProps) {
  const [description, setDescription] = useState('')
  const [model, setModel] = useState('')

  return (
    <Dialog open={open} onOpenChange={isGenerating ? undefined : onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Generate Workflow</DialogTitle>
          <DialogDescription>
            Describe what you want your workflow to do, and AI will generate the
            nodes and connections for you.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="e.g., Read a PDF and summarize its key points, then translate the summary to Korean"
              className="min-h-[100px]"
              autoFocus
              disabled={isGenerating}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="model">Model (optional)</Label>
            <Input
              id="model"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              placeholder="openai/gpt-4o (uses default if empty)"
              disabled={isGenerating}
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isGenerating}
          >
            Cancel
          </Button>
          <Button
            onClick={() => onGenerate(description, model || undefined)}
            disabled={!description.trim() || isGenerating}
          >
            {isGenerating ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <Sparkles className="h-4 w-4 mr-2" />
            )}
            {isGenerating ? 'Generating...' : 'Generate'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
