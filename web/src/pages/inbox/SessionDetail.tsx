import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ExternalLink, CheckCircle, XCircle, CheckSquare, Square, Loader2 } from 'lucide-react'
import { Header } from '@/shared/ui/Header'
import { BreadcrumbNav } from '@/shared/ui/BreadcrumbNav'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { fetchContentSession } from '@/entities/content-session/api'
import { useContentSessionStore } from '@/entities/content-session/store'
import type { ContentSession, ContentAngle } from '@/entities/content-session'

// ─── Source panel ─────────────────────────────────────────────────────────────

function SourcePanel({ session }: { session: ContentSession }) {
  const isCollecting = session.status === 'collecting'
  const sources = session.sources ?? []

  return (
    <div className="space-y-6">
      {isCollecting && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          소스 수집 중...
        </div>
      )}
      {sources.map((src) => (
        <div key={src.id}>
          <div className="flex items-center gap-2 mb-2">
            <SourceTypeBadge type={src.source_type} />
            <span className="text-sm font-medium">{src.label}</span>
            <span className="text-xs text-muted-foreground">· {src.count}개 수집</span>
          </div>
          <div className="rounded-xl border border-border overflow-hidden">
            {src.items.map((item, i) => (
              <div
                key={i}
                className="flex items-center gap-3 px-4 py-2.5 border-b border-border last:border-b-0
                  hover:bg-muted/30 transition-colors"
              >
                {item.score != null && (
                  <span className="text-xs text-muted-foreground tabular-nums w-16 shrink-0">
                    {item.extra ?? item.score}
                  </span>
                )}
                <span className="text-sm flex-1 truncate">{item.title}</span>
                {item.url && (
                  <a
                    href={item.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-muted-foreground hover:text-foreground transition-colors shrink-0"
                  >
                    <ExternalLink className="h-3.5 w-3.5" />
                  </a>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
      {sources.length === 0 && !isCollecting && (
        <p className="text-sm text-muted-foreground">수집된 소스가 없습니다.</p>
      )}
    </div>
  )
}

// ─── Analysis panel ───────────────────────────────────────────────────────────

function AnalysisPanel({
  session,
  angles,
  onToggleAngle,
}: {
  session: ContentSession
  angles: ContentAngle[]
  onToggleAngle: (id: string) => void
}) {
  if (!session.analysis) {
    return <p className="text-sm text-muted-foreground">분석 결과가 없습니다.</p>
  }
  const { analysis } = session

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
          총 {analysis.total_collected}개 수집 → {analysis.total_selected}개 선별
        </p>
      </div>

      <div>
        <h3 className="text-sm font-semibold mb-2">요약</h3>
        <div className="rounded-xl border border-border bg-muted/20 px-4 py-3 text-sm leading-relaxed">
          {analysis.summary}
        </div>
      </div>

      <div>
        <h3 className="text-sm font-semibold mb-2">핵심 인사이트</h3>
        <ul className="space-y-1.5">
          {analysis.insights.map((insight, i) => (
            <li key={i} className="flex items-start gap-2 text-sm">
              <span className="text-muted-foreground mt-0.5 shrink-0">•</span>
              {insight}
            </li>
          ))}
        </ul>
      </div>

      <div>
        <h3 className="text-sm font-semibold mb-2">추천 콘텐츠 각도</h3>
        <div className="rounded-xl border border-border overflow-hidden">
          {angles.map((angle) => (
            <button
              key={angle.id}
              onClick={() => onToggleAngle(angle.id)}
              className="w-full flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0
                hover:bg-muted/30 transition-colors text-left cursor-pointer"
            >
              {angle.selected
                ? <CheckSquare className="h-4 w-4 text-primary shrink-0" />
                : <Square className="h-4 w-4 text-muted-foreground shrink-0" />
              }
              <span className="text-xs text-muted-foreground w-12 shrink-0">{angle.format}</span>
              <span className="text-sm">{angle.title}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

// ─── Results panel ────────────────────────────────────────────────────────────

function ResultsPanel({ session }: { session: ContentSession }) {
  const results = session.workflow_results ?? []
  if (results.length === 0) {
    return <p className="text-sm text-muted-foreground">워크플로우 실행 결과가 없습니다.</p>
  }

  const statusLabel: Record<string, string> = {
    pending: '대기 중',
    running: '실행 중...',
    success: '완료',
    failed: '실패',
  }
  const statusClass: Record<string, string> = {
    pending: 'text-muted-foreground',
    running: 'text-info',
    success: 'text-success',
    failed: 'text-destructive',
  }

  return (
    <div className="space-y-3">
      {results.map((result) => (
        <div
          key={result.run_id}
          className="rounded-xl border border-border bg-card px-4 py-3 flex items-center justify-between"
        >
          <div className="flex items-center gap-3">
            <span className="text-sm font-medium">{result.workflow_name}</span>
            <span className={`text-xs font-medium ${statusClass[result.status] ?? ''}`}>
              [{statusLabel[result.status] ?? result.status}]
            </span>
          </div>
          {result.output_url && result.status === 'success' && (
            <a
              href={result.output_url}
              className="text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              결과 보기 →
            </a>
          )}
        </div>
      ))}
    </div>
  )
}

// ─── Reject dialog ────────────────────────────────────────────────────────────

function RejectDialog({
  onConfirm,
  onCancel,
  isLoading,
}: {
  onConfirm: (reason: string) => void
  onCancel: () => void
  isLoading: boolean
}) {
  const [reason, setReason] = useState('')

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onCancel} />
      <div className="relative bg-card border border-border rounded-2xl p-5 w-full max-w-sm mx-4 shadow-xl">
        <h3 className="text-sm font-semibold mb-1">세션 거절</h3>
        <p className="text-xs text-muted-foreground mb-3">거절 사유를 입력하세요 (선택)</p>
        <textarea
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="예: 관련성 낮음, 다음 주 재수집 예정..."
          rows={3}
          className="w-full rounded-xl border border-border bg-muted/20 px-3 py-2 text-sm
            placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-border
            resize-none mb-3"
        />
        <div className="flex items-center justify-end gap-2">
          <button
            onClick={onCancel}
            className="px-3 py-1.5 rounded-lg text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            취소
          </button>
          <button
            onClick={() => onConfirm(reason)}
            disabled={isLoading}
            className="flex items-center gap-1.5 px-4 py-1.5 rounded-xl text-sm font-medium
              bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors cursor-pointer
              disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            거절 확인
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

type Tab = 'source' | 'analysis' | 'results'

export default function SessionDetailPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<Tab>('source')
  const [showRejectDialog, setShowRejectDialog] = useState(false)

  const { data: session, isLoading, error } = useQuery({
    queryKey: ['content-session', sessionId],
    queryFn: () => fetchContentSession(sessionId!),
    enabled: !!sessionId,
    refetchInterval: (query) =>
      query.state.data?.status === 'collecting' ? 3000 : false,
  })

  const [angles, setAngles] = useState<ContentAngle[]>([])
  const { approveSession, rejectSession } = useContentSessionStore()
  const [isApproving, setIsApproving] = useState(false)
  const [isRejecting, setIsRejecting] = useState(false)

  // Reset angles when navigating to a different session
  useEffect(() => {
    if (session?.analysis?.angles) {
      setAngles(session.analysis.angles)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session?.id])

  const handleApprove = async () => {
    setIsApproving(true)
    try {
      await approveSession(sessionId!, angles.filter((a) => a.selected).map((a) => a.id))
      navigate('/inbox')
    } finally {
      setIsApproving(false)
    }
  }

  const handleReject = async (reason: string) => {
    setIsRejecting(true)
    try {
      await rejectSession(sessionId!, reason || undefined)
      navigate('/inbox')
    } finally {
      setIsRejecting(false)
    }
  }

  const toggleAngle = (id: string) => {
    setAngles((prev) => prev.map((a) => (a.id === id ? { ...a, selected: !a.selected } : a)))
  }

  if (isLoading) {
    return (
      <div className="flex flex-col h-screen bg-background">
        <Header />
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      </div>
    )
  }

  if (error || !session) {
    return (
      <div className="flex flex-col h-screen bg-background">
        <Header />
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-muted-foreground">세션을 불러올 수 없습니다.</p>
        </div>
      </div>
    )
  }

  const createdAt = new Date(session.created_at).toLocaleString('ko-KR', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })

  const selectedCount = angles.filter((a) => a.selected).length
  const isPending = session.status === 'pending_review'

  const tabs: { value: Tab; label: string }[] = [
    { value: 'source', label: '소스' },
    { value: 'analysis', label: 'LLM 분석' },
    { value: 'results', label: '결과물' },
  ]

  return (
    <div className="flex flex-col h-screen bg-background">
      <Header />
      <main className="flex-1 overflow-y-auto pb-24">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8">

          <BreadcrumbNav
            className="mb-5"
            items={[
              { label: 'Content Inbox', to: '/inbox' },
              ...(session.pipeline_name ? [{ label: session.pipeline_name, to: `/pipelines/${session.pipeline_id}` }] : []),
              { label: session.session_number != null ? `Session ${session.session_number}` : session.id },
            ]}
          />

          <div className="flex items-start justify-between gap-4 mb-6">
            <div>
              <div className="flex items-center gap-2 flex-wrap mb-1">
                <h1 className="text-xl font-bold">
                  {session.session_number != null ? `Session ${session.session_number}` : session.id}
                </h1>
                <StatusBadge status={session.status} />
              </div>
              <p className="text-sm text-muted-foreground">
                {createdAt} · trigger: {session.trigger_type}
                {session.analysis && (
                  <>
                    {' '}· Score: <ScoreIndicator score={session.analysis.score} />
                  </>
                )}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-1 p-1 rounded-xl bg-muted/30 w-fit mb-6">
            {tabs.map(({ value, label }) => (
              <button
                key={value}
                onClick={() => setActiveTab(value)}
                className={`px-4 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer ${
                  activeTab === value
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {label}
              </button>
            ))}
          </div>

          {activeTab === 'source' && <SourcePanel session={session} />}
          {activeTab === 'analysis' && (
            <AnalysisPanel
              session={session}
              angles={angles}
              onToggleAngle={toggleAngle}
            />
          )}
          {activeTab === 'results' && <ResultsPanel session={session} />}

        </div>
      </main>

      {isPending && (
        <div className="fixed bottom-0 left-0 right-0 border-t border-border bg-background/95 backdrop-blur-sm px-4 py-3">
          <div className="max-w-4xl mx-auto flex items-center justify-between gap-4">
            <p className="text-sm text-muted-foreground">
              {selectedCount}개 포맷 선택됨
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setShowRejectDialog(true)}
                disabled={isRejecting}
                className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
                  bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors cursor-pointer
                  disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <XCircle className="h-4 w-4" />
                거절
              </button>
              <button
                onClick={() => void handleApprove()}
                disabled={selectedCount === 0 || isApproving}
                className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
                  bg-success/15 text-success hover:bg-success/25 transition-colors cursor-pointer
                  disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isApproving
                  ? <Loader2 className="h-4 w-4 animate-spin" />
                  : <CheckCircle className="h-4 w-4" />
                }
                승인하고 선택된 워크플로우 실행
              </button>
            </div>
          </div>
        </div>
      )}

      {showRejectDialog && (
        <RejectDialog
          onConfirm={(reason) => void handleReject(reason)}
          onCancel={() => setShowRejectDialog(false)}
          isLoading={isRejecting}
        />
      )}
    </div>
  )
}
