// web/src/pages/Pipelines.tsx
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Plus, Play, Loader2, Trash2, Clock, CheckCircle2,
  AlertCircle, PauseCircle, GitBranch, Zap, RefreshCw, Sparkles,
} from 'lucide-react'
import { Header } from '@/components/Header'
import {
  fetchPipelines, fetchPipeline, createPipeline, updatePipeline,
  deletePipeline, startPipeline,
} from '@/lib/api'
import type { Pipeline } from '@/lib/api/types'
import { PipelineEditor } from '@/components/pipelines/PipelineEditor'
import { PipelineRunHistory } from '@/components/pipelines/PipelineRunHistory'
import { PipelineTemplateGallery } from '@/components/pipelines/PipelineTemplateGallery'
import { GeneratePipelineDialog } from '@/components/pipelines/GeneratePipelineDialog'

const stageTypeIcons: Record<string, typeof CheckCircle2> = {
  workflow:  Play,
  approval:  PauseCircle,
  schedule:  Clock,
  trigger:   Zap,
  transform: RefreshCw,
}

const statusConfig: Record<string, { icon: typeof CheckCircle2; color: string }> = {
  idle:      { icon: Clock,         color: 'text-muted-foreground' },
  running:   { icon: Loader2,       color: 'text-info' },
  waiting:   { icon: PauseCircle,   color: 'text-warning' },
  completed: { icon: CheckCircle2,  color: 'text-success' },
  failed:    { icon: AlertCircle,   color: 'text-destructive' },
}
// Suppress unused warning — referenced for type completeness
void statusConfig

export default function Pipelines() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Pipeline | null>(null)
  const [generateOpen, setGenerateOpen] = useState(false)
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

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

  useEffect(() => {
    if (id) {
      fetchPipeline(id).then(setSelected).catch(() => navigate('/pipelines'))
    } else {
      setSelected(null)
    }
  }, [id])

  const handleCreate = async () => {
    try {
      const p = await createPipeline({ name: 'New Pipeline', stages: [] })
      navigate(`/pipelines/${p.id}`)
    } catch {
      // silent
    }
  }

  const handleTemplateCreated = (p: Pipeline) => {
    navigate(`/pipelines/${p.id}`)
  }

  const handleDelete = async (pid: string) => {
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
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto px-6 py-6 space-y-6">

          {selected ? (
            <>
              <PipelineEditor
                pipeline={selected}
                onSave={handleSave}
                onBack={() => navigate('/pipelines')}
              />
              <PipelineRunHistory pipeline={selected} />
            </>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <h1 className="text-xl font-semibold">Pipelines</h1>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setGenerateOpen(true)}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg border hover:bg-muted transition-colors"
                  >
                    <Sparkles className="h-4 w-4" />
                    생성
                  </button>
                  <button
                    onClick={handleCreate}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
                  >
                    <Plus className="h-4 w-4" />
                    New Pipeline
                  </button>
                </div>
              </div>

              {loading ? (
                <div className="flex items-center justify-center py-16">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <>
                  {pipelines.length === 0 && (
                    <div className="text-center py-8 text-muted-foreground">
                      <GitBranch className="h-10 w-10 mx-auto mb-3 opacity-40" />
                      <p className="text-sm">파이프라인이 없습니다</p>
                      <p className="text-xs mt-1">아래 템플릿으로 시작하거나 직접 만들어보세요</p>
                    </div>
                  )}

                  <PipelineTemplateGallery onCreated={handleTemplateCreated} />

                  {pipelines.length > 0 && (
                    <div className="space-y-3">
                      {pipelines.map((p) => (
                        <div
                          key={p.id}
                          className="border rounded-xl p-4 hover:border-foreground/20 transition-colors cursor-pointer"
                          onClick={() => navigate(`/pipelines/${p.id}`)}
                        >
                          <div className="flex items-center justify-between mb-2">
                            <h3 className="text-sm font-medium">{p.name}</h3>
                            <div className="flex items-center gap-1">
                              <button
                                onClick={(e) => { e.stopPropagation(); handleStart(p.id) }}
                                className="p-1.5 rounded-md hover:bg-muted transition-colors"
                                title="Start"
                              >
                                <Play className="h-3.5 w-3.5" />
                              </button>
                              <button
                                onClick={(e) => { e.stopPropagation(); handleDelete(p.id) }}
                                className="p-1.5 rounded-md hover:bg-muted transition-colors text-muted-foreground hover:text-destructive"
                                title="Delete"
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          </div>

                          {p.description && (
                            <p className="text-xs text-muted-foreground mb-2">{p.description}</p>
                          )}

                          {p.stages.length > 0 && (
                            <div className="flex items-center gap-1 text-xs text-muted-foreground">
                              {p.stages.map((stage, i) => {
                                const StageIcon = stageTypeIcons[stage.type] || GitBranch
                                return (
                                  <span key={stage.id} className="flex items-center gap-1">
                                    {i > 0 && <span className="text-border">→</span>}
                                    <span className="flex items-center gap-1 px-1.5 py-0.5 rounded bg-muted">
                                      <StageIcon className="h-3 w-3" />
                                      {stage.name || stage.type}
                                    </span>
                                  </span>
                                )
                              })}
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </>
              )}
              <GeneratePipelineDialog
                open={generateOpen}
                onClose={() => setGenerateOpen(false)}
                onCreated={(p) => { setGenerateOpen(false); handleTemplateCreated(p) }}
              />
            </>
          )}
        </div>
      </main>
    </div>
  )
}
