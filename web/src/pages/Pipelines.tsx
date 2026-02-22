// web/src/pages/Pipelines.tsx
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState, useCallback } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Plus, GitBranch,
  CheckCircle2, AlertCircle, PauseCircle, Clock,
} from 'lucide-react'
import { Header } from '@/shared/ui/Header'
import {
  fetchPipelines, fetchPipeline, createPipeline, updatePipeline,
  deletePipeline, startPipeline, generatePipelineBundle,
  approvePipelineRun, rejectPipelineRun, generatePipelineThumbnail,
} from '@/entities/pipeline'
import { saveWorkflow } from '@/entities/workflow'
import { ApiError } from '@/shared/api'
import type { Pipeline, PipelineRun } from '@/shared/types'
import { PipelineCard, PipelineEditor, PipelineRunHistory, PipelineRunDetail } from '@/widgets/pipeline-editor'
import { PromptBar } from '@/widgets/workflow-canvas/ui/PromptBar'

// Suppress unused warning — referenced for type completeness
void ({ CheckCircle2, AlertCircle, PauseCircle, Clock } as Record<string, unknown>)

export default function Pipelines() {
  const queryClient = useQueryClient()
  const [selected, setSelected] = useState<Pipeline | null>(null)
  const [selectedRun, setSelectedRun] = useState<PipelineRun | null>(null)
  const [generateError, setGenerateError] = useState<string | null>(null)
  const [editorKey, setEditorKey] = useState(0)
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const thumbnailRequested = useRef<Set<string>>(new Set())

  const { data: pipelines = [], isLoading: loading } = useQuery({
    queryKey: ['pipelines'],
    queryFn: fetchPipelines,
  })

  // After list loads, generate thumbnails for pipelines that don't have one yet.
  useEffect(() => {
    if (loading) return
    const missing = pipelines.filter(
      (p) => !p.thumbnail_svg && !thumbnailRequested.current.has(p.id),
    )
    if (missing.length === 0) return

    let cancelled = false
    const runNext = async (i: number) => {
      if (cancelled || i >= missing.length) return
      const p = missing[i]
      thumbnailRequested.current.add(p.id)
      try {
        const svg = await generatePipelineThumbnail(p.id)
        if (!cancelled && svg) {
          queryClient.setQueryData<Pipeline[]>(['pipelines'], (old) =>
            old?.map(item => item.id === p.id ? { ...item, thumbnail_svg: svg } : item)
          )
        }
      } catch { /* skip — thumbnail is optional */ }
      runNext(i + 1)
    }
    runNext(0)
    return () => { cancelled = true }
  }, [loading, pipelines, queryClient])

  useEffect(() => {
    setSelectedRun(null)
    if (id) {
      fetchPipeline(id).then(setSelected).catch(() => navigate('/pipelines'))
    } else {
      setSelected(null)
    }
  }, [id, navigate])

  const createMutation = useMutation({
    mutationFn: createPipeline,
    onSuccess: (p) => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
      navigate(`/pipelines/${p.id}`)
    }
  })

  const updateMutation = useMutation({
    mutationFn: async ({ id, data }: { id: string, data: Pipeline }) => updatePipeline(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
      fetchPipeline(variables.id).then(setSelected)
    }
  })

  const deleteMutation = useMutation({
    mutationFn: deletePipeline,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
    }
  })

  const startMutation = useMutation({
    mutationFn: startPipeline,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
    }
  })

  const generateMutation = useMutation({
    mutationFn: async (description: string) => {
      const existingPipeline = selected
        ? { name: selected.name, description: selected.description, stages: selected.stages }
        : undefined
      const bundle = await generatePipelineBundle(description, existingPipeline)
      await Promise.all(
        bundle.workflows.map((wf) =>
          saveWorkflow(wf).catch((e) => {
            if (e instanceof ApiError && e.status === 409) return
            throw e
          }),
        ),
      )
      return bundle
    },
    onSuccess: async (bundle) => {
      if (selected) {
        await updatePipeline(selected.id, bundle.pipeline)
        const fresh = await fetchPipeline(selected.id)
        setSelected(fresh)
        setEditorKey((k) => k + 1)
        queryClient.invalidateQueries({ queryKey: ['pipelines'] })
      } else {
        const pipeline = await createPipeline(bundle.pipeline)
        queryClient.invalidateQueries({ queryKey: ['pipelines'] })
        navigate(`/pipelines/${pipeline.id}`)
      }
    },
    onError: (e) => {
      setGenerateError(e instanceof Error ? e.message : 'Generation failed')
    }
  })

  const handleCreate = () => createMutation.mutate({ name: 'New Pipeline', stages: [] })

  const handleGenerate = useCallback((description: string) => {
    setGenerateError(null)
    generateMutation.mutate(description)
  }, [generateMutation])

  const handleDelete = (pid: string) => {
    if (!confirm('Delete this pipeline?')) return
    deleteMutation.mutate(pid)
  }

  const handleStart = (pid: string) => startMutation.mutate(pid)

  const handleSave = async (p: Pipeline) => {
    await updateMutation.mutateAsync({ id: p.id, data: p })
  }

  return (
    <div className="flex flex-col h-screen bg-background">
      <Header />
      {/* ─── Floating prompt bar ─── */}
      <PromptBar
        onSubmit={handleGenerate}
        isGenerating={generateMutation.isPending}
        placeholder={selected ? 'Edit these stages...' : 'Describe your pipeline...'}
        positioning="fixed"
        error={generateError}
        onValueChange={() => setGenerateError(null)}
      />
      <main className="flex-1 overflow-y-auto pb-24">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 py-8">

          {selected ? (
            <>
              <PipelineEditor
                key={editorKey}
                pipeline={selected}
                onSave={handleSave}
                onBack={() => navigate('/pipelines')}
              />
              <div className="mt-6">
                {selectedRun ? (
                  <PipelineRunDetail
                    pipeline={selected}
                    run={selectedRun}
                    onBack={() => setSelectedRun(null)}
                    onApprove={async () => {
                      await approvePipelineRun(selected.id, selectedRun.id)
                      setSelectedRun(null)
                    }}
                    onReject={async () => {
                      await rejectPipelineRun(selected.id, selectedRun.id)
                      setSelectedRun(null)
                    }}
                  />
                ) : (
                  <PipelineRunHistory pipeline={selected} onSelectRun={setSelectedRun} />
                )}
              </div>
            </>
          ) : (
            <>
              {/* ─── Page header ─── */}
              <div className="flex items-start justify-between mb-6 gap-4">
                <div>
                  <h1 className="landing-display text-2xl font-bold tracking-tight">Pipelines</h1>
                  {!loading && (
                    <div className="flex items-center gap-5 mt-1.5">
                      <span className="text-sm text-muted-foreground">
                        <span className="text-foreground font-semibold tabular-nums">{pipelines.length}</span>
                        {' '}total
                      </span>
                    </div>
                  )}
                </div>
                <button
                  onClick={handleCreate}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-xl
                    bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer shrink-0"
                >
                  <Plus className="h-3.5 w-3.5" />
                  New Pipeline
                </button>
              </div>

              {loading ? (
                /* Skeleton */
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  {[1, 2, 3].map((i) => (
                    <div key={i} className="rounded-2xl bg-card border border-border animate-pulse overflow-hidden">
                      <div className="h-[68px] bg-muted/20 border-b border-border" />
                      <div className="p-4 space-y-2.5">
                        <div className="h-3.5 w-28 bg-muted/40 rounded" />
                        <div className="h-3 w-16 bg-muted/25 rounded" />
                      </div>
                    </div>
                  ))}
                </div>
              ) : pipelines.length > 0 ? (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  {/* Create new card */}
                  <button
                    onClick={handleCreate}
                    className="group rounded-2xl border-2 border-dashed border-border
                      hover:border-foreground/25 hover:bg-card/60 transition-all duration-200
                      flex flex-col items-center justify-center min-h-[164px] cursor-pointer"
                  >
                    <div className="w-9 h-9 rounded-xl bg-muted/30 flex items-center justify-center mb-2.5
                      group-hover:bg-muted/50 group-hover:scale-110 transition-all duration-200">
                      <Plus className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                    </div>
                    <span className="text-sm font-medium text-muted-foreground group-hover:text-foreground transition-colors">
                      New pipeline
                    </span>
                  </button>

                  {pipelines.map((p, i) => (
                    <div
                      key={p.id}
                      className="animate-in fade-in slide-in-from-bottom-2 duration-300"
                      style={{ animationDelay: `${i * 35}ms` }}
                    >
                      <PipelineCard
                        pipeline={p}
                        onClick={() => navigate(`/pipelines/${p.id}`)}
                        onStart={() => handleStart(p.id)}
                        onDelete={() => handleDelete(p.id)}
                      />
                    </div>
                  ))}
                </div>
              ) : (
                /* Empty state */
                <div className="text-center py-16">
                  <div className="w-14 h-14 rounded-2xl bg-muted/20 flex items-center justify-center mx-auto mb-5">
                    <GitBranch className="w-6 h-6 text-muted-foreground/40" />
                  </div>
                  <h3 className="landing-display text-lg font-semibold mb-2">No pipelines yet</h3>
                  <p className="text-sm text-muted-foreground max-w-xs mx-auto leading-relaxed">
                    Describe a pipeline above to generate one with AI, or create one manually.
                  </p>
                </div>
              )}
            </>
          )}

        </div>
      </main>
    </div>
  )
}
