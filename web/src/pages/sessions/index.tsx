import { useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Settings, ArrowLeft } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { fetchSessions, createSession } from '@/entities/session'
import { useUIStore } from '@/entities/ui'
import { SessionSidebar } from './SessionSidebar'
import { SessionDetail } from './SessionDetail'

export default function SessionsPage() {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const addToast = useUIStore((s) => s.addToast)

  const selectedId = searchParams.get('id')

  const selectSession = useCallback((id: string) => {
    setSearchParams({ id })
  }, [setSearchParams])

  // Data
  const { data: sessions = [], isLoading } = useQuery({
    queryKey: ['sessions'],
    queryFn: fetchSessions,
  })

  const createMutation = useMutation({
    mutationFn: (name: string) => createSession({ name }),
    onSuccess: (session) => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
      selectSession(session.id)
    },
    onError: (err) => addToast(`Failed: ${err instanceof Error ? err.message : 'unknown'}`),
  })

  // Auto-select first session
  useEffect(() => {
    if (!selectedId && sessions.length > 0) selectSession(sessions[0].id)
  }, [sessions, selectedId, selectSession])

  // Mobile level
  const showDetail = !!selectedId

  return (
    <MainLayout headerContent={<span className="font-semibold tracking-tight">Sessions</span>}>
      <div className="flex h-full w-full overflow-hidden bg-background">
        {/* Sidebar */}
        <div className={cn(
          'w-full md:w-[340px] 2xl:w-[400px] shrink-0 md:border-r border-border',
          'bg-sidebar/30 backdrop-blur-xl flex flex-col',
          showDetail ? 'hidden md:flex' : 'flex',
        )}>
          <SessionSidebar
            sessions={sessions}
            selectedId={selectedId}
            onSelect={selectSession}
            onDeselect={() => setSearchParams({})}
            onCreate={(name) => createMutation.mutate(name)}
            isLoading={isLoading}
          />
        </div>

        {/* Detail */}
        <div className={cn(
          'flex-1 min-w-0 flex flex-col',
          showDetail ? 'flex' : 'hidden md:flex',
        )}>
          {selectedId ? (
            <>
              <div className="md:hidden px-4 py-2 border-b border-border/50 shrink-0">
                <button
                  onClick={() => setSearchParams({})}
                  className="flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
                >
                  <ArrowLeft className="w-4 h-4" />
                  Back
                </button>
              </div>
              <SessionDetail sessionId={selectedId} />
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-muted-foreground flex-col gap-3">
              <div className="size-14 rounded-full bg-muted/30 flex items-center justify-center shrink-0 border border-border/50">
                <Settings className="w-6 h-6 opacity-30" />
              </div>
              <div className="text-center">
                <p className="font-medium text-foreground">Select a session</p>
                <p className="text-sm">Choose a session to view settings and runs.</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </MainLayout>
  )
}
