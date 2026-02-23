import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { ArrowLeft, ArrowRight, Check } from 'lucide-react'
import { Header } from '@/shared/ui/Header'
import { Textarea } from '@/shared/ui/textarea'
import { EditorialBriefForm } from '@/features/define-editorial-brief/EditorialBriefForm'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { createPipeline } from '@/entities/pipeline'
import type { PipelineSource, PipelineContext } from '@/shared/types'
import { Trash2, Plus } from 'lucide-react'

type Step = 1 | 2 | 3

const STEP_LABELS: Record<Step, string> = {
  1: '기본 정보',
  2: 'Editorial Brief',
  3: '소스 설정',
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
              transition-colors ${
              step < current
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
    focus_keywords: [], exclude_keywords: [], language: '한국어',
  })

  // Step 3 state
  const [sources, setSources] = useState<PipelineSource[]>([])
  const [schedule, setSchedule] = useState('0 */6 * * *')
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
    <div className="flex flex-col h-screen bg-background">
      <Header />
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto px-4 sm:px-6 py-8">

          {/* Breadcrumb / back */}
          <button
            onClick={() => (step === 1 ? navigate('/pipelines') : setStep((s) => (s - 1) as Step))}
            className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground
              transition-colors mb-6 cursor-pointer"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            {step === 1 ? 'Pipelines' : STEP_LABELS[(step - 1) as Step]}
          </button>

          <h1 className="text-2xl font-bold tracking-tight mb-6">새 파이프라인</h1>

          <StepIndicator current={step} />

          {/* ── Step 1: 기본 정보 ── */}
          {step === 1 && (
            <div className="space-y-5">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">파이프라인 이름 *</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="예: IT AI Pipeline"
                  autoFocus
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none
                    focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">설명 (선택)</label>
                <Textarea
                  rows={3}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="이 파이프라인의 목적을 간단히 설명해 주세요..."
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
                  다음
                  <ArrowRight className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          )}

          {/* ── Step 2: Editorial Brief ── */}
          {step === 2 && (
            <div>
              <p className="text-sm text-muted-foreground mb-5">
                LLM이 뉴스를 선별하고 콘텐츠를 작성할 때 참고하는 가이드라인입니다.
              </p>
              <EditorialBriefForm
                initialContext={context}
                onSave={async (ctx) => {
                  setContext(ctx)
                  setStep(3)
                }}
              />
            </div>
          )}

          {/* ── Step 3: 소스 설정 ── */}
          {step === 3 && (
            <div className="space-y-5">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium">수집 소스</h3>
                <button
                  onClick={() => setShowAddModal(true)}
                  className="flex items-center gap-1 text-xs text-muted-foreground
                    hover:text-foreground transition-colors cursor-pointer"
                >
                  <Plus className="h-3.5 w-3.5" />
                  소스 추가
                </button>
              </div>

              {sources.length === 0 ? (
                <p className="text-sm text-muted-foreground py-6 text-center rounded-xl border border-dashed border-border">
                  소스를 추가하거나, 나중에 파이프라인 설정에서 추가할 수 있습니다.
                </p>
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
                </div>
              )}

              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">수집 스케줄 (cron)</label>
                <input
                  type="text"
                  value={schedule}
                  onChange={(e) => setSchedule(e.target.value)}
                  placeholder="0 */6 * * *"
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm
                    font-mono outline-none focus:ring-1 focus:ring-ring"
                />
              </div>

              <div className="flex justify-end pt-2">
                <button
                  onClick={() => createMutation.mutate()}
                  disabled={createMutation.isPending}
                  className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
                    bg-foreground text-background hover:opacity-90 transition-opacity
                    disabled:opacity-60 disabled:cursor-not-allowed cursor-pointer"
                >
                  {createMutation.isPending ? '생성 중…' : '파이프라인 생성'}
                </button>
              </div>

              {createMutation.isError && (
                <p className="text-sm text-destructive">
                  생성 실패: {createMutation.error instanceof Error ? createMutation.error.message : '알 수 없는 오류'}
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
      </main>
    </div>
  )
}
