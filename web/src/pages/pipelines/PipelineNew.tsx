import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { ArrowLeft, ArrowRight, Check, Trash2, Plus } from 'lucide-react'
import { MainLayout } from '@/app/layout'
import { Separator } from '@/shared/ui/separator'
import { Textarea } from '@/shared/ui/textarea'
import { EditorialBriefForm } from '@/features/define-editorial-brief/EditorialBriefForm'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { createPipeline } from '@/entities/pipeline'
import type { PipelineSource, PipelineContext } from '@/shared/types'

type Step = 1 | 2 | 3

const STEP_LABELS: Record<Step, string> = {
  1: 'Basic Info',
  2: 'Editorial Brief',
  3: 'Sources',
}

// ─── Step indicators ──────────────────────────────────────────────────────────

function StepIndicator({ current }: { current: Step }) {
  const steps: Step[] = [1, 2, 3]
  return (
    <div className="flex items-center gap-2 mb-8">
      {steps.map((step, i) => (
        <div key={step} className="flex items-center gap-2">
          <div
            className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold
              transition-colors ${step < current
                ? 'bg-success text-success-foreground'
                : step === current
                  ? 'bg-foreground text-background'
                  : 'bg-muted text-muted-foreground'
              }`}
          >
            {step < current ? <Check className="h-3.5 w-3.5" /> : step}
          </div>
          <span className={`text-xs font-medium ${step === current ? 'text-foreground' : 'text-muted-foreground'}`}>
            {STEP_LABELS[step]}
          </span>
          {i < steps.length - 1 && (
            <div className={`w-8 h-px mx-1 ${step < current ? 'bg-success' : 'bg-border'}`} />
          )}
        </div>
      ))}
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function PipelineNewPage() {
  const navigate = useNavigate()
  const [step, setStep] = useState<Step>(1)

  // Step 1 state
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')

  // Step 2 state (editorial brief — deferred to inline form)
  const [context, setContext] = useState<PipelineContext>({
    purpose: '', target_audience: '', tone_style: '',
    focus_keywords: [], exclude_keywords: [], language: 'Korean',
  })

  // Step 3 state
  const [sources, setSources] = useState<PipelineSource[]>([])
  const [schedule, setSchedule] = useState('0 */6 * * *')
  const [customCron, setCustomCron] = useState(false)
  const [showAddModal, setShowAddModal] = useState(false)

  const createMutation = useMutation({
    mutationFn: () => createPipeline({
      name,
      description,
      stages: [],
      sources,
      context,
      schedule,
    }),
    onSuccess: (pipeline) => {
      navigate(`/pipelines/${pipeline.id}`)
    },
  })

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
        <div className="max-w-2xl mx-auto px-4 sm:px-6 py-8">
          <h1 className="text-2xl font-bold tracking-tight mb-6">New Pipeline</h1>

          <StepIndicator current={step} />

          {/* ── Step 1: Basic Info ── */}
          {step === 1 && (
            <div className="space-y-5">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Pipeline name *</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g. AI News Daily"
                  autoFocus
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none
                    focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Description (optional)</label>
                <Textarea
                  rows={3}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Briefly describe the purpose of this pipeline…"
                  className="resize-none text-sm"
                />
              </div>
              <div className="flex justify-end pt-2">
                <button
                  onClick={() => setStep(2)}
                  disabled={!canAdvanceStep1}
                  className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
                    bg-foreground text-background hover:opacity-90 transition-opacity
                    disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
                >
                  Next
                  <ArrowRight className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          )}

          {/* ── Step 2: Editorial Brief ── */}
          {step === 2 && (
            <div>
              <p className="text-sm text-muted-foreground mb-5">
                Guidelines the LLM uses when filtering news and writing content.
              </p>
              <EditorialBriefForm
                initialContext={context}
                submitLabel="Next"
                onBack={() => setStep(1)}
                onSkip={() => setStep(3)}
                skipLabel="Skip"
                onSave={async (ctx) => {
                  setContext(ctx)
                  setStep(3)
                }}
              />
            </div>
          )}

          {/* ── Step 3: Sources ── */}
          {step === 3 && (
            <div className="space-y-5">
              <h3 className="text-sm font-medium">Collection sources</h3>

              {sources.length === 0 ? (
                <button
                  type="button"
                  onClick={() => setShowAddModal(true)}
                  className="w-full flex flex-col items-center gap-2 py-8 rounded-xl border border-dashed
                    border-border text-muted-foreground hover:border-foreground/40 hover:text-foreground
                    transition-colors cursor-pointer group"
                >
                  <Plus className="h-5 w-5 group-hover:scale-110 transition-transform" />
                  <span className="text-sm">Add a source</span>
                  <span className="text-xs opacity-70">or add one later in the pipeline settings</span>
                </button>
              ) : (
                <div className="rounded-xl border border-border overflow-hidden">
                  {sources.map((src, i) => (
                    <div key={src.id} className="flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0">
                      <SourceTypeBadge type={src.source_type} />
                      <span className="text-sm font-medium flex-1">{src.label}</span>
                      <span className="text-xs text-muted-foreground truncate max-w-[160px]">
                        {src.url ?? src.subreddit ?? (src.keywords?.join(', ') ?? '')}
                      </span>
                      <button
                        onClick={() => setSources(sources.filter((_, j) => j !== i))}
                        className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  ))}
                  <button
                    type="button"
                    onClick={() => setShowAddModal(true)}
                    className="w-full flex items-center justify-center gap-1.5 px-4 py-2.5
                      text-xs text-muted-foreground hover:text-foreground hover:bg-muted/50
                      transition-colors cursor-pointer border-t border-border"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    Add source
                  </button>
                </div>
              )}

              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Collection schedule</label>
                {!customCron ? (
                  <div className="flex items-center gap-2">
                    <select
                      value={schedule}
                      onChange={(e) => setSchedule(e.target.value)}
                      className="flex-1 rounded-xl border border-input bg-background px-3 py-2 text-sm
                        outline-none focus:ring-1 focus:ring-ring"
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
                      className="text-xs text-muted-foreground hover:text-foreground transition-colors whitespace-nowrap cursor-pointer"
                    >
                      Custom
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={schedule}
                      onChange={(e) => setSchedule(e.target.value)}
                      placeholder="0 */6 * * *"
                      className="flex-1 rounded-xl border border-input bg-background px-3 py-2 text-sm
                        font-mono outline-none focus:ring-1 focus:ring-ring"
                    />
                    <button
                      type="button"
                      onClick={() => setCustomCron(false)}
                      className="text-xs text-muted-foreground hover:text-foreground transition-colors whitespace-nowrap cursor-pointer"
                    >
                      Presets
                    </button>
                  </div>
                )}
              </div>

              <div className="flex items-center justify-between pt-2">
                <button
                  type="button"
                  onClick={() => setStep(2)}
                  className="px-4 py-2 rounded-xl text-sm font-medium text-muted-foreground
                    hover:text-foreground transition-colors cursor-pointer"
                >
                  Back
                </button>
                <button
                  onClick={() => createMutation.mutate()}
                  disabled={createMutation.isPending}
                  className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
                    bg-foreground text-background hover:opacity-90 transition-opacity
                    disabled:opacity-60 disabled:cursor-not-allowed cursor-pointer"
                >
                  {createMutation.isPending ? 'Creating…' : 'Create Pipeline'}
                </button>
              </div>

              {createMutation.isError && (
                <p className="text-sm text-destructive">
                  Failed to create: {createMutation.error instanceof Error ? createMutation.error.message : 'Unknown error'}
                </p>
              )}
            </div>
          )}

          {showAddModal && (
            <AddSourceModal
              onAdd={(src) => setSources([...sources, src])}
              onClose={() => setShowAddModal(false)}
            />
          )}

        </div>
      </div>
    </MainLayout>
  )
}
