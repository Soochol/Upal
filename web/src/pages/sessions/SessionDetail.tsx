import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Settings, History } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { fetchSession } from '@/entities/session'
import { SessionConfigView } from './SessionConfigView'
import { RunHistoryPanel } from './RunHistoryPanel'

type Tab = 'setup' | 'runs'

interface SessionDetailProps {
  sessionId: string
}

export function SessionDetail({ sessionId }: SessionDetailProps) {
  const [activeTab, setActiveTab] = useState<Tab>('setup')

  const { data: session } = useQuery({
    queryKey: ['session', sessionId],
    queryFn: () => fetchSession(sessionId),
    enabled: !!sessionId,
  })

  const tabs: { value: Tab; label: string; icon: typeof Settings }[] = [
    { value: 'setup', label: 'Setup', icon: Settings },
    { value: 'runs', label: 'Runs', icon: History },
  ]

  return (
    <div className="flex flex-col h-full">
      {/* Tab header */}
      <div className="flex items-center gap-1 px-4 py-2 border-b border-border/50 bg-background/50 backdrop-blur-md shrink-0">
        {session && (
          <h2 className="text-sm font-semibold text-foreground mr-4 truncate">{session.name}</h2>
        )}
        <div className="flex items-center gap-1">
          {tabs.map((tab) => {
            const Icon = tab.icon
            const isActive = activeTab === tab.value
            return (
              <button
                key={tab.value}
                onClick={() => setActiveTab(tab.value)}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer',
                  isActive
                    ? 'bg-foreground text-background'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted/50',
                )}
              >
                <Icon className="h-3.5 w-3.5" />
                {tab.label}
              </button>
            )
          })}
        </div>
      </div>

      {/* Tab content */}
      <div className="flex-1 min-h-0 overflow-y-auto">
        {activeTab === 'setup' && (
          <SessionConfigView sessionId={sessionId} />
        )}
        {activeTab === 'runs' && (
          <RunHistoryPanel sessionId={sessionId} />
        )}
      </div>
    </div>
  )
}
