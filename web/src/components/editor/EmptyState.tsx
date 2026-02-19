import { Workflow, Plus, Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/button'

type EmptyStateProps = {
  onAddNode: () => void
  onGenerate: () => void
}

export function EmptyState({ onAddNode, onGenerate }: EmptyStateProps) {
  return (
    <div className="absolute inset-0 flex items-center justify-center pointer-events-none z-10">
      <div className="flex flex-col items-center text-center max-w-sm pointer-events-auto">
        <div className="h-16 w-16 rounded-2xl bg-muted flex items-center justify-center mb-6">
          <Workflow className="h-8 w-8 text-muted-foreground" />
        </div>
        <h2 className="text-lg font-semibold text-foreground mb-2">
          Build your workflow
        </h2>
        <p className="text-sm text-muted-foreground mb-6 leading-relaxed">
          Add nodes from the sidebar and connect them to create an AI-powered
          workflow. Or let AI generate one for you.
        </p>
        <div className="flex gap-3">
          <Button variant="outline" onClick={onAddNode}>
            <Plus className="h-4 w-4 mr-1.5" />
            Add first node
          </Button>
          <Button onClick={onGenerate}>
            <Sparkles className="h-4 w-4 mr-1.5" />
            Generate with AI
          </Button>
        </div>
      </div>
    </div>
  )
}
