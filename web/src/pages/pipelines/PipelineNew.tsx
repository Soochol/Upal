import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { ArrowLeft, Check, Loader2 } from 'lucide-react'
import { MainLayout } from '@/app/layout'
import { Separator } from '@/shared/ui/separator'
import { Textarea } from '@/shared/ui/textarea'
import { createPipeline } from '@/entities/pipeline'

export default function PipelineNewPage() {
  const navigate = useNavigate()

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')

  const createMutation = useMutation({
    mutationFn: () => createPipeline({ name, description }),
    onSuccess: (pipeline) => {
      navigate(`/pipelines?p=${pipeline.id}`)
    },
  })

  const canCreate = name.trim().length > 0

  return (
    <MainLayout
      headerContent={
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/pipelines')}
            className="p-1.5 rounded-md hover:bg-muted transition-colors cursor-pointer"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          <span className="font-semibold">New Pipeline</span>
          <Separator orientation="vertical" className="h-4" />
          <span className="text-sm text-muted-foreground">Create</span>
        </div>
      }
    >
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-lg mx-auto px-4 sm:px-6 py-12">

          <h1 className="text-2xl font-bold tracking-tight mb-8">Create Pipeline</h1>

          <div className="bg-card border border-border/60 rounded-2xl p-6 md:p-8 shadow-sm">
            <div className="space-y-5">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">
                  Pipeline name <span className="text-destructive">*</span>
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g. AI News Daily"
                  autoFocus
                  className="w-full rounded-xl border border-input bg-background/50 px-3 py-2.5 text-sm outline-none
                    focus:ring-2 focus:ring-ring focus:bg-background transition-all placeholder:text-muted-foreground/50"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">
                  Description (optional)
                </label>
                <Textarea
                  rows={3}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Briefly describe the purpose of this pipeline..."
                  className="resize-none text-sm bg-background/50 focus:bg-background transition-all"
                />
              </div>

              <div className="flex items-center justify-between pt-4 border-t border-border/50">
                <button
                  type="button"
                  onClick={() => navigate('/pipelines')}
                  className="px-5 py-2.5 rounded-xl text-sm font-medium text-muted-foreground
                    hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  onClick={() => createMutation.mutate()}
                  disabled={!canCreate || createMutation.isPending}
                  className="flex items-center gap-2 px-6 py-2.5 rounded-xl text-sm font-bold
                    bg-primary text-primary-foreground hover:bg-primary/90 transition-all
                    disabled:opacity-60 disabled:cursor-not-allowed cursor-pointer shadow-lg shadow-primary/20"
                >
                  {createMutation.isPending
                    ? <><Loader2 className="w-4 h-4 animate-spin" /> Creating...</>
                    : <><Check className="w-4 h-4" /> Create Pipeline</>
                  }
                </button>
              </div>

              {createMutation.isError && (
                <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-sm text-destructive">
                  Failed to create: {createMutation.error instanceof Error ? createMutation.error.message : 'Unknown error'}
                </div>
              )}
            </div>
          </div>

        </div>
      </div>
    </MainLayout>
  )
}
