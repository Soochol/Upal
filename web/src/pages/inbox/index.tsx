import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Flame, X, CheckCircle, XCircle, ExternalLink, Loader2 } from 'lucide-react'
import { Header } from '@/shared/ui/Header'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { useContentSessionStore } from '@/entities/content-session'
import { useSurgeStore } from '@/entities/surge'
import type { ContentSession, ContentSessionStatus } from '@/entities/content-session'
import type { SurgeAlert } from '@/entities/surge'

// ─── Surge banner ─────────────────────────────────────────────────────────────

function SurgeBanner({
  surge,
  onDismiss,
  onCreateSession,
  isCreating,
}: {
  surge: SurgeAlert
  onDismiss: () => void
  onCreateSession: () => void
  isCreating: boolean
}) {
  return (
    <div className="rounded-xl border border-warning/30 bg-warning/8 px-4 py-3 flex items-center gap-3 mb-6">
      <Flame className="h-4 w-4 text-warning shrink-0" />
      <span className="text-sm flex-1">
        <span className="font-semibold text-warning">{surge.keyword}</span>
        {' '}언급{' '}
        <span className="font-semibold">{surge.multiplier}배 급증</span> 감지
        {' — '}
        <span className="text-muted-foreground">{surge.pipeline_name}</span>
      </span>
      <button
        onClick={onCreateSession}
        disabled={isCreating}
        className="flex items-center gap-1 text-xs font-medium px-2.5 py-1 rounded-lg bg-warning/20 text-warning
          hover:bg-warning/30 transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
      >
        {isCreating ? <Loader2 className="h-3 w-3 animate-spin" /> : null}
        세션 생성
      </button>
      <button
        onClick={onDismiss}
        className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}

// ─── Source summary chips ─────────────────────────────────────────────────────

function SourceSummary({ sources }: { sources: ContentSession['sources'] }) {
  const statics = sources.filter((s) => s.source_type === 'static')
  const signals = sources.filter((s) => s.source_type === 'signal')
  if (statics.length === 0 && signals.length === 0) return null

  return (
    <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
      {statics.length > 0 && (
        <span className="flex items-center gap-1.5">
          <SourceTypeBadge type="static" />
          {statics.map((s) => `${s.label}(${s.count})`).join('  ')}
        </span>
      )}
      {signals.length > 0 && (
        <span className="flex items-center gap-1.5">
          <SourceTypeBadge type="signal" />
          {signals.map((s) => s.label).join(' · ')}
        </span>
      )}
    </div>
  )
}

// ─── Session card ─────────────────────────────────────────────────────────────

function SessionCard({
  session,
  onView,
  onApprove,
  onReject,
  isApproving,
  isRejecting,
}: {
  session: ContentSession
  onView: () => void
  onApprove?: () => void
  onReject?: () => void
  isApproving?: boolean
  isRejecting?: boolean
}) {
  const isPending = session.status === 'pending_review'
  const createdAt = new Date(session.created_at).toLocaleString('ko-KR', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
  })

  return (
    <div className="rounded-xl border border-border bg-card p-4 space-y-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs font-semibold text-foreground">{session.pipeline_name}</span>
          <span className="text-muted-foreground/50 text-xs">·</span>
          <span className="text-xs text-muted-foreground">Session {session.session_number}</span>
          <span className="text-muted-foreground/50 text-xs">·</span>
          <span className="text-xs text-muted-foreground">{createdAt}</span>
          <StatusBadge status={session.status} />
        </div>
        {session.analysis && (
          <div className="flex items-center gap-1 shrink-0">
            <span className="text-[11px] text-muted-foreground">Score:</span>
            <ScoreIndicator score={session.analysis.score} />
          </div>
        )}
      </div>

      {session.analysis?.summary && (
        <p className="text-sm font-medium leading-snug">"{session.analysis.summary}"</p>
      )}

      <SourceSummary sources={session.sources} />

      {session.analysis && (
        <p className="text-xs text-muted-foreground">
          → 선별: {session.analysis.total_selected}개 / 총 {session.analysis.total_collected}개 수집
        </p>
      )}

      <div className="flex items-center justify-between pt-1">
        <button
          onClick={onView}
          className="flex items-center gap-1 text-xs font-medium text-muted-foreground
            hover:text-foreground transition-colors cursor-pointer"
        >
          <ExternalLink className="h-3.5 w-3.5" />
          세션 보기
        </button>
        {isPending && (
          <div className="flex items-center gap-2">
            <button
              onClick={onApprove}
              disabled={isApproving}
              className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium
                bg-success/15 text-success hover:bg-success/25 transition-colors cursor-pointer
                disabled:opacity-60 disabled:cursor-not-allowed"
            >
              {isApproving
                ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
                : <CheckCircle className="h-3.5 w-3.5" />
              }
              승인
            </button>
            <button
              onClick={onReject}
              disabled={isRejecting}
              className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium
                bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors cursor-pointer
                disabled:opacity-60 disabled:cursor-not-allowed"
            >
              {isRejecting
                ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
                : <XCircle className="h-3.5 w-3.5" />
              }
              거절
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

// ─── Status tabs ──────────────────────────────────────────────────────────────

const STATUS_TABS: { label: string; value: ContentSessionStatus | 'all' }[] = [
  { label: '전체', value: 'all' },
  { label: '대기 중', value: 'pending_review' },
  { label: '승인됨', value: 'approved' },
  { label: '제작 중', value: 'producing' },
  { label: '완료', value: 'published' },
]

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function InboxPage() {
  const navigate = useNavigate()
  const [activeFilter, setActiveFilter] = useState<ContentSessionStatus | 'all'>('all')

  const { sessions, loading, pendingCount, fetchSessions } = useContentSessionStore()
  const { surges, fetchSurges, dismissSurge, createSessionFromSurge } = useSurgeStore()

  // Load data on mount
  useEffect(() => {
    void fetchSessions()
    void fetchSurges()
  }, [fetchSessions, fetchSurges])

  // First undismissed surge
  const activeSurge = surges.find((s) => !s.dismissed) ?? null

  const [approvingId, setApprovingId] = useState<string | null>(null)
  const [rejectingId, setRejectingId] = useState<string | null>(null)

  const handleApprove = async (id: string) => {
    setApprovingId(id)
    try {
      // Quick-approve from list sends no selected angles;
      // the server uses the session's recommended angles as default.
      await useContentSessionStore.getState().approveSession(id, [])
    } finally {
      setApprovingId(null)
    }
  }

  const handleReject = async (id: string) => {
    setRejectingId(id)
    try {
      await useContentSessionStore.getState().rejectSession(id)
    } finally {
      setRejectingId(null)
    }
  }

  const surgeCreateMutation = useMutation({
    mutationFn: (surgeId: string) => createSessionFromSurge(surgeId),
    onSuccess: (sessionId) => navigate(`/inbox/${sessionId}`),
  })

  const filtered =
    activeFilter === 'all' ? sessions : sessions.filter((s) => s.status === activeFilter)

  return (
    <div className="flex flex-col h-screen bg-background">
      <Header />
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8">

          {/* Surge banner */}
          {activeSurge && (
            <SurgeBanner
              surge={activeSurge}
              onDismiss={() => dismissSurge(activeSurge.id)}
              onCreateSession={() => surgeCreateMutation.mutate(activeSurge.id)}
              isCreating={surgeCreateMutation.isPending}
            />
          )}

          {/* Page header */}
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-2xl font-bold tracking-tight">Content Inbox</h1>
          </div>

          {/* Filter tabs */}
          <div className="flex items-center mb-6">
            <div className="flex items-center gap-1 p-1 rounded-xl bg-muted/30 flex-wrap">
              {STATUS_TABS.map(({ label, value }) => {
                const count =
                  value === 'pending_review' ? pendingCount
                  : value === 'all' ? sessions.length
                  : undefined
                return (
                  <button
                    key={value}
                    onClick={() => setActiveFilter(value)}
                    className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer ${
                      activeFilter === value
                        ? 'bg-background text-foreground shadow-sm'
                        : 'text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    {label}
                    {count != null && count > 0 && (
                      <span className="ml-1.5 text-[10px] font-semibold">{count}</span>
                    )}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Session list */}
          {loading ? (
            <div className="flex justify-center py-16">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : filtered.length === 0 ? (
            <div className="text-center py-16 text-sm text-muted-foreground">
              해당하는 세션이 없습니다.
            </div>
          ) : (
            <div className="space-y-3">
              {filtered.map((session) => (
                <SessionCard
                  key={session.id}
                  session={session}
                  onView={() => navigate(`/inbox/${session.id}`)}
                  onApprove={() => void handleApprove(session.id)}
                  onReject={() => void handleReject(session.id)}
                  isApproving={approvingId === session.id}
                  isRejecting={rejectingId === session.id}
                />
              ))}
            </div>
          )}

        </div>
      </main>
    </div>
  )
}
