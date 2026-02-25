import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, ArrowRight, Check, Trash2, Plus, Sparkles, Loader2 } from 'lucide-react'
import { MainLayout } from '@/app/layout'
import { Separator } from '@/shared/ui/separator'
import { Textarea } from '@/shared/ui/textarea'
import { EditorialBriefForm } from '@/features/define-editorial-brief/EditorialBriefForm'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { createPipeline, generatePipelineBundle } from '@/entities/pipeline'
import { fetchPublishChannels } from '@/entities/publish-channel/api'
import { WorkflowPicker } from './WorkflowPicker'
import type { PipelineSource, PipelineContext, PipelineWorkflow } from '@/entities/pipeline'
import { humanReadableCron } from '@/shared/lib/cron'

type Step = 1 | 2 | 3 | 4

const STEP_LABELS: Record<Step, string> = {
  1: 'Basic Info',
  2: 'Editorial Brief',
  3: 'Sources & Schedule',
  4: 'Workflows', // New Step
}

// ─── Step indicators ──────────────────────────────────────────────────────────

function StepIndicator({ current, maxReached, setStep }: { current: Step, maxReached: Step, setStep: (s: Step) => void }) {
  const steps: Step[] = [1, 2, 3, 4]
  return (
    <div className="flex items-center gap-2 mb-8">
      {steps.map((step, i) => {
        const isCompleted = step < current
        const isCurrent = step === current
        const isClickable = step <= maxReached && !isCurrent

        return (
          <div key={step} className="flex items-center gap-2">
            <button
              onClick={() => isClickable && setStep(step)}
              disabled={!isClickable}
              className={`flex items-center gap-2 group ${isClickable ? 'cursor-pointer' : 'cursor-default'}`}
              type="button"
            >
              <div
                className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold
                  transition-all duration-300 ${isCompleted
                    ? 'bg-success text-success-foreground'
                    : isCurrent
                      ? 'bg-foreground text-background shadow-md transform scale-105'
                      : 'bg-muted text-muted-foreground'
                  } ${isClickable ? 'group-hover:ring-2 ring-foreground/20' : ''}`}
              >
                {isCompleted ? <Check className="h-3.5 w-3.5" /> : step}
              </div>
              <span className={`text-xs font-medium transition-colors ${isCurrent ? 'text-foreground font-semibold' : 'text-muted-foreground'
                } ${isClickable ? 'group-hover:text-foreground/80' : ''}`}>
                {STEP_LABELS[step]}
              </span>
            </button>
            {i < steps.length - 1 && (
              <div className={`w-8 h-px mx-1 transition-colors duration-500 ${step < current ? 'bg-success' : 'bg-border'}`} />
            )}
          </div>
        )
      })}
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function PipelineNewPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const shouldGenerate = searchParams.get('generate') === 'true'

  const [step, setStep] = useState<Step>(1)
  const [maxReached, setMaxReached] = useState<Step>(1)
  const [isGenerating, setIsGenerating] = useState(false)

  // Step 1 state
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')

  // Step 2 state
  const [context, setContext] = useState<PipelineContext>({
    purpose: '', target_audience: '', tone_style: '',
    focus_keywords: [], exclude_keywords: [], language: 'Korean',
  })

  // Step 3 state
  const [sources, setSources] = useState<PipelineSource[]>([])
  const [schedule, setSchedule] = useState('0 */6 * * *')
  const [customCron, setCustomCron] = useState(false)
  const [showAddModal, setShowAddModal] = useState(false)

  // Step 4 state
  const [workflows, setWorkflows] = useState<PipelineWorkflow[]>([])
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)

  const { data: channels = [] } = useQuery({
    queryKey: ['publish-channels'],
    queryFn: fetchPublishChannels,
  })

  const createMutation = useMutation({
    mutationFn: () => createPipeline({
      name,
      description,
      stages: [],
      sources,
      context,
      schedule,
      workflows,
    }),
    onSuccess: (pipeline) => {
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
      navigate(`/pipelines?p=${pipeline.id}`)
    },
  })

  // Handle auto-generation if requested
  useEffect(() => {
    if (!shouldGenerate) return
    const doGenerate = async () => {
      setIsGenerating(true)
      try {
        const bundle = await generatePipelineBundle("Generate a standard news curation pipeline")
        if (bundle.pipeline.name) setName(bundle.pipeline.name)
        if (bundle.pipeline.description) setDescription(bundle.pipeline.description)
        if (bundle.pipeline.context) setContext(bundle.pipeline.context)
        if (bundle.pipeline.sources) setSources(bundle.pipeline.sources)
        if (bundle.pipeline.schedule) setSchedule(bundle.pipeline.schedule)
        if (bundle.workflows && bundle.workflows.length > 0) {
          // Add generated workflows if returned
          const mapped = bundle.workflows.map((w: any) => ({ workflow_name: w.name, auto_select: true }))
          setWorkflows(mapped)
        }
      } catch (err) {
        console.error("AI Generation failed:", err)
      } finally {
        setIsGenerating(false)
      }
    }
    doGenerate()
  }, [shouldGenerate])

  const goNext = (target: Step) => {
    setStep(target)
    setMaxReached(Math.max(maxReached, target) as Step)
  }

  const canAdvanceStep1 = name.trim().length > 0

  return (
    <MainLayout
      headerContent={
        <div className="flex items-center gap-3">
          <button
            onClick={() => (step === 1 ? navigate('/pipelines') : setStep((s) => (s - 1) as Step))}
            className="p-1.5 rounded-md hover:bg-muted transition-colors cursor-pointer"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          <span className="font-semibold">New Pipeline</span>
          <Separator orientation="vertical" className="h-4" />
          <span className="text-sm text-muted-foreground">{STEP_LABELS[step]}</span>
        </div>
      }
    >
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 py-8 flex flex-col lg:flex-row gap-8 items-start">

          {/* Main Content Area */}
          <div className="flex-1 w-full max-w-2xl">
            <div className="flex items-center justify-between mb-6">
              <h1 className="text-2xl font-bold tracking-tight">Create Pipeline</h1>
              {isGenerating && (
                <div className="flex items-center gap-2 text-sm text-primary animate-pulse bg-primary/10 px-3 py-1.5 rounded-full">
                  <Sparkles className="h-4 w-4" /> Generating with AI...
                </div>
              )}
            </div>

            <StepIndicator current={step} maxReached={maxReached} setStep={setStep} />

            <div className="bg-card border border-border/60 rounded-2xl p-6 md:p-8 shadow-sm">
              {/* ── Step 1: Basic Info ── */}
              {step === 1 && (
                <div className="space-y-5 animate-in fade-in slide-in-from-right-4 duration-500">
                  <div>
                    <label className="block text-sm font-medium text-foreground mb-1.5">Pipeline name <span className="text-destructive">*</span></label>
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
                    <label className="block text-sm font-medium text-foreground mb-1.5">Description (optional)</label>
                    <Textarea
                      rows={3}
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      placeholder="Briefly describe the purpose of this pipeline…"
                      className="resize-none text-sm bg-background/50 focus:bg-background transition-all"
                    />
                  </div>
                  <div className="flex justify-end pt-4">
                    <button
                      onClick={() => goNext(2)}
                      disabled={!canAdvanceStep1}
                      className="flex items-center gap-1.5 px-5 py-2.5 rounded-xl text-sm font-medium
                        bg-foreground text-background hover:opacity-90 transition-all
                        disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer shadow-md shadow-foreground/10"
                    >
                      Next
                      <ArrowRight className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              )}

              {/* ── Step 2: Editorial Brief ── */}
              {step === 2 && (
                <div className="animate-in fade-in slide-in-from-right-4 duration-500">
                  <p className="text-sm text-muted-foreground mb-6 bg-muted/30 p-4 rounded-xl border border-border/50">
                    <Sparkles className="inline-block w-4 h-4 mr-2 text-primary" />
                    <strong>Editorial Brief</strong> gives the LLM guidelines when filtering news and writing content for this pipeline.
                  </p>
                  <EditorialBriefForm
                    initialContext={context}
                    submitLabel="Next Step"
                    onBack={() => setStep(1)}
                    onSkip={() => goNext(3)}
                    skipLabel="Skip for now"
                    onSave={async (ctx) => {
                      setContext(ctx)
                      goNext(3)
                    }}
                  />
                </div>
              )}

              {/* ── Step 3: Sources ── */}
              {step === 3 && (
                <div className="space-y-6 animate-in fade-in slide-in-from-right-4 duration-500">

                  <div>
                    <h3 className="text-sm font-semibold mb-3">Collection sources</h3>
                    {sources.length === 0 ? (
                      <button
                        type="button"
                        onClick={() => setShowAddModal(true)}
                        className="w-full flex flex-col items-center gap-2 py-10 rounded-xl border-2 border-dashed
                          border-border text-muted-foreground hover:border-foreground/30 hover:bg-muted/30 hover:text-foreground
                          transition-all cursor-pointer group"
                      >
                        <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center group-hover:scale-110 transition-transform">
                          <Plus className="h-5 w-5" />
                        </div>
                        <span className="text-sm font-medium">Add a source</span>
                        <span className="text-xs opacity-70">or add one later in settings</span>
                      </button>
                    ) : (
                      <div className="rounded-xl border border-border overflow-hidden bg-background">
                        {sources.map((src, i) => (
                          <div key={src.id} className="flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0 hover:bg-muted/20 transition-colors">
                            <SourceTypeBadge type={src.source_type} />
                            <span className="text-sm font-medium flex-1 truncate">{src.label}</span>
                            <span className="text-xs text-muted-foreground truncate max-w-[160px] hidden sm:block">
                              {src.url ?? src.subreddit ?? (src.keywords?.join(', ') ?? '')}
                            </span>
                            <button
                              onClick={() => setSources(sources.filter((_, j) => j !== i))}
                              className="p-1.5 rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors cursor-pointer"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        ))}
                        <button
                          type="button"
                          onClick={() => setShowAddModal(true)}
                          className="w-full flex items-center justify-center gap-1.5 px-4 py-3
                            text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted/50
                            transition-colors cursor-pointer border-t border-border"
                        >
                          <Plus className="h-4 w-4" />
                          Add another source
                        </button>
                      </div>
                    )}
                  </div>

                  <div>
                    <label className="block text-sm font-semibold mb-3">Collection schedule</label>
                    <div className="bg-muted/30 p-4 rounded-xl border border-border/50">
                      {!customCron ? (
                        <div className="flex items-center gap-3">
                          <select
                            value={schedule}
                            onChange={(e) => setSchedule(e.target.value)}
                            className="flex-1 rounded-xl border border-input bg-background px-3 py-2.5 text-sm
                              outline-none focus:ring-2 focus:ring-ring cursor-pointer"
                          >
                            <option value="0 * * * *">Every hour</option>
                            <option value="0 */3 * * *">Every 3 hours</option>
                            <option value="0 */6 * * *">Every 6 hours</option>
                            <option value="0 */12 * * *">Every 12 hours</option>
                            <option value="0 6 * * *">Daily at 6 AM</option>
                            <option value="0 8 * * 1">Every Monday at 8 AM</option>
                          </select>
                          <button
                            type="button"
                            onClick={() => setCustomCron(true)}
                            className="text-xs font-medium text-muted-foreground hover:text-foreground bg-background border border-border px-3 py-2.5 rounded-xl transition-colors cursor-pointer"
                          >
                            Custom
                          </button>
                        </div>
                      ) : (
                        <div className="flex flex-col gap-2">
                          <div className="flex items-center gap-3">
                            <input
                              type="text"
                              value={schedule}
                              onChange={(e) => setSchedule(e.target.value)}
                              placeholder="0 */6 * * *"
                              className="flex-1 rounded-xl border border-input bg-background px-3 py-2.5 text-sm
                                font-mono outline-none focus:ring-2 focus:ring-ring"
                            />
                            <button
                              type="button"
                              onClick={() => setCustomCron(false)}
                              className="text-xs font-medium text-muted-foreground hover:text-foreground bg-background border border-border px-3 py-2.5 rounded-xl transition-colors cursor-pointer"
                            >
                              Presets
                            </button>
                          </div>
                          <p className="text-xs text-muted-foreground mt-1 ml-1 flex items-center gap-1">
                            <Sparkles className="h-3 w-3 text-primary" /> Parse result: <span className="font-medium text-foreground">{humanReadableCron(schedule)}</span>
                          </p>
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-6 border-t border-border/50">
                    <button
                      type="button"
                      onClick={() => setStep(2)}
                      className="px-5 py-2.5 rounded-xl text-sm font-medium text-muted-foreground
                        hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
                    >
                      Back
                    </button>
                    <button
                      onClick={() => goNext(4)}
                      className="flex items-center gap-1.5 px-5 py-2.5 rounded-xl text-sm font-medium
                        bg-foreground text-background hover:opacity-90 transition-all cursor-pointer shadow-md shadow-foreground/10"
                    >
                      Next Step
                      <ArrowRight className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              )}

              {/* ── Step 4: Workflows ── */}
              {step === 4 && (
                <div className="space-y-6 animate-in fade-in slide-in-from-right-4 duration-500">
                  <p className="text-sm text-muted-foreground mb-6 bg-info/10 p-4 rounded-xl border border-info/20 text-info">
                    <strong>Attach Workflows:</strong> Workflows determine what happens to your collected data. You can always add or configure them later in settings.
                  </p>

                  {workflows.length === 0 ? (
                    <button
                      type="button"
                      onClick={() => setShowWorkflowPicker(true)}
                      className="w-full flex justify-between items-center px-5 py-4 rounded-xl border border-border bg-background hover:border-primary/50 hover:bg-primary/5 transition-all group cursor-pointer"
                    >
                      <div className="flex flex-col text-left">
                        <span className="text-sm font-semibold group-hover:text-primary transition-colors">Select a Workflow</span>
                        <span className="text-xs text-muted-foreground mt-0.5">Choose from your available workflow templates</span>
                      </div>
                      <div className="w-8 h-8 rounded-full bg-muted flex items-center justify-center group-hover:bg-primary/10 transition-colors">
                        <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-primary" />
                      </div>
                    </button>
                  ) : (
                    <div className="rounded-xl border border-border overflow-hidden bg-background">
                      {workflows.map((wf, i) => (
                        <div key={i} className="flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0 hover:bg-muted/20 transition-colors">
                          <Check className="h-4 w-4 text-success" />
                          <span className="text-sm font-medium flex-1 truncate">{wf.label || wf.workflow_name}</span>
                          <span className="text-muted-foreground text-xs">&rarr;</span>
                          <select
                            value={wf.channel_id || ''}
                            onChange={(e) => {
                              const updated = [...workflows]
                              updated[i] = { ...updated[i], channel_id: e.target.value || undefined }
                              setWorkflows(updated)
                            }}
                            className="w-48 rounded-lg border border-input bg-background px-2 py-1.5 text-xs outline-none focus:ring-2 focus:ring-ring cursor-pointer"
                          >
                            <option value="">No channel</option>
                            {channels.map((ch) => (
                              <option key={ch.id} value={ch.id}>{ch.name} ({ch.type})</option>
                            ))}
                          </select>
                          <button
                            onClick={() => setWorkflows(workflows.filter((_, j) => j !== i))}
                            className="p-1.5 rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors cursor-pointer"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={() => setShowWorkflowPicker(true)}
                        className="w-full flex items-center justify-center gap-1.5 px-4 py-3
                          text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted/50
                          transition-colors cursor-pointer border-t border-border"
                      >
                        <Plus className="h-4 w-4" />
                        Add another workflow
                      </button>
                    </div>
                  )}

                  <div className="flex items-center justify-between pt-6 border-t border-border/50 mt-8">
                    <button
                      type="button"
                      onClick={() => setStep(3)}
                      className="px-5 py-2.5 rounded-xl text-sm font-medium text-muted-foreground
                        hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
                    >
                      Back
                    </button>
                    <button
                      onClick={() => createMutation.mutate()}
                      disabled={createMutation.isPending}
                      className="flex items-center gap-2 px-6 py-2.5 rounded-xl text-sm font-bold
                        bg-primary text-primary-foreground hover:bg-primary/90 transition-all
                        disabled:opacity-60 disabled:cursor-not-allowed cursor-pointer shadow-lg shadow-primary/20"
                    >
                      {createMutation.isPending ? <><Loader2 className="w-4 h-4 animate-spin" /> Creating...</> : <><Check className="w-4 h-4" /> Create Pipeline</>}
                    </button>
                  </div>

                  {createMutation.isError && (
                    <div className="p-3 mt-4 rounded-xl bg-destructive/10 border border-destructive/20 text-sm text-destructive">
                      Failed to create: {createMutation.error instanceof Error ? createMutation.error.message : 'Unknown error'}
                    </div>
                  )}
                </div>
              )}

            </div>
          </div>

          {/* Right Sidebar Checklist / Summary (Hidden on small screens) */}
          <div className="hidden lg:block w-72 shrink-0 animate-in fade-in slide-in-from-bottom-8 duration-500 delay-150 sticky top-8">
            <div className="bg-card/50 backdrop-blur-sm border border-border/50 rounded-2xl p-5 shadow-sm">
              <h3 className="text-sm font-semibold mb-4 uppercase tracking-wider text-muted-foreground">Pipeline Summary</h3>

              <div className="space-y-4">
                <div className="flex gap-3">
                  <div className={`mt-0.5 shrink-0 ${name ? 'text-success' : 'text-muted-foreground/30'}`}>
                    <Check className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs font-medium text-muted-foreground uppercase">Name</p>
                    <p className={`text-sm truncate ${name ? 'text-foreground font-medium' : 'text-muted-foreground italic'}`}>
                      {name || 'Not set'}
                    </p>
                  </div>
                </div>

                <div className="flex gap-3">
                  <div className={`mt-0.5 shrink-0 ${(context.purpose || context.focus_keywords?.length) ? 'text-success' : 'text-muted-foreground/30'}`}>
                    <Check className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs font-medium text-muted-foreground uppercase">Brief</p>
                    <p className="text-sm text-foreground">
                      {context.focus_keywords?.length ? `${context.focus_keywords.length} keywords` : (context.purpose ? 'Purpose set' : <span className="text-muted-foreground italic">Not set</span>)}
                    </p>
                  </div>
                </div>

                <div className="flex gap-3">
                  <div className={`mt-0.5 shrink-0 ${sources.length > 0 ? 'text-success' : 'text-muted-foreground/30'}`}>
                    <Check className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs font-medium text-muted-foreground uppercase">Sources</p>
                    <p className="text-sm text-foreground">
                      {sources.length > 0 ? `${sources.length} sources` : <span className="text-muted-foreground italic">0 sources</span>}
                    </p>
                  </div>
                </div>

                <div className="flex gap-3">
                  <div className={`mt-0.5 shrink-0 ${workflows.length > 0 ? 'text-success' : 'text-muted-foreground/30'}`}>
                    <Check className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs font-medium text-muted-foreground uppercase">Workflows</p>
                    <p className="text-sm text-foreground">
                      {workflows.length > 0 ? `${workflows.length} attached` : <span className="text-muted-foreground italic">0 attached</span>}
                    </p>
                  </div>
                </div>
              </div>

            </div>
          </div>

          {showAddModal && (
            <AddSourceModal
              onAdd={(src) => setSources([...sources, src])}
              onClose={() => setShowAddModal(false)}
            />
          )}

          {showWorkflowPicker && (
            <WorkflowPicker
              existingWorkflows={workflows}
              onAdd={(wf) => setWorkflows([...workflows, wf])}
              onClose={() => setShowWorkflowPicker(false)}
            />
          )}

        </div>
      </div>
    </MainLayout>
  )
}
