// web/src/pages/Pipelines.tsx
import { useEffect, useRef, useState, useCallback } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Plus, GitBranch,
  CheckCircle2, AlertCircle, PauseCircle, Clock,
} from 'lucide-react'
import { Header } from '@/components/Header'
import {
  fetchPipelines, fetchPipeline, createPipeline, updatePipeline,
  deletePipeline, startPipeline, generatePipelineBundle, saveWorkflow,
  approvePipelineRun, rejectPipelineRun, generatePipelineThumbnail,
} from '@/lib/api'
import { ApiError } from '@/lib/api/client'
import type { Pipeline, PipelineRun } from '@/lib/api/types'
import { PipelineCard } from '@/components/pipelines/PipelineCard'
import { PipelineEditor } from '@/components/pipelines/PipelineEditor'
import { PipelineRunHistory } from '@/components/pipelines/PipelineRunHistory'
import { PipelineRunDetail } from '@/components/pipelines/PipelineRunDetail'
import { PromptBar } from '@/components/editor/PromptBar'

// Suppress unused warning — referenced for type completeness
void ({ CheckCircle2, AlertCircle, PauseCircle, Clock } as Record<string, unknown>)

export default function Pipelines() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Pipeline | null>(null)
  const [selectedRun, setSelectedRun] = useState<PipelineRun | null>(null)
  const [isGenerating, setIsGenerating] = useState(false)
  const [generateError, setGenerateError] = useState<string | null>(null)
  const [editorKey, setEditorKey] = useState(0)
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const thumbnailRequested = useRef<Set<string>>(new Set())

  const reload = async () => {
    try {
      const data = await fetchPipelines()
      setPipelines(data)
    } catch {
      // silent
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { reload() }, [])

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
        if (!cancelled) {
          setPipelines((prev) =>
            prev.map((item) => (item.id === p.id ? { ...item, thumbnail_svg: svg } : item)),
          )
        }
      } catch { /* skip — thumbnail is optional */ }
      runNext(i + 1)
    }
    runNext(0)
    return () => { cancelled = true }
  }, [loading]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    setSelectedRun(null)
    if (id) {
      fetchPipeline(id).then(setSelected).catch(() => navigate('/pipelines'))
    } else {
      setSelected(null)
    }
  }, [id]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleCreate = async () => {
    try {
      const p = await createPipeline({ name: 'New Pipeline', stages: [] })
      navigate(`/pipelines/${p.id}`)
    } catch {
      // silent
    }
  }

  const handleGenerate = useCallback(async (description: string) => {
    setIsGenerating(true)
    setGenerateError(null)
    try {
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
      if (selected) {
        // Edit mode: update existing pipeline and remount editor
        await updatePipeline(selected.id, bundle.pipeline)
        const fresh = await fetchPipeline(selected.id)
        setSelected(fresh)
        setEditorKey((k) => k + 1)
      } else {
        // Create mode: create new pipeline and navigate to it
        const pipeline = await createPipeline(bundle.pipeline)
        navigate(`/pipelines/${pipeline.id}`)
      }
    } catch (e) {
      setGenerateError(e instanceof Error ? e.message : 'Generation failed')
    } finally {
      setIsGenerating(false)
    }
  }, [isGenerating, selected, navigate])

  const handleDelete = async (pid: string) => {
    if (!confirm('Delete this pipeline?')) return
    try {
      await deletePipeline(pid)
      reload()
    } catch {
      // silent
    }
  }

  const handleStart = async (pid: string) => {
    try {
      await startPipeline(pid)
      reload()
    } catch {
      // silent
    }
  }

  const handleSave = async (p: Pipeline) => {
    await updatePipeline(p.id, p)
    setSelected(p)
    reload()
  }

  return (
    <div className="flex flex-col h-screen bg-background">
      <Header />
      {/* ─── Floating prompt bar ─── */}
      <PromptBar
        onSubmit={handleGenerate}
        isGenerating={isGenerating}
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
