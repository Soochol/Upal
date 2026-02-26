import { useState, useEffect } from 'react'
import type { ReactNode } from 'react'
import { Zap, Box, Activity, Settings, Workflow, Globe, Menu, Inbox, X, LogOut } from 'lucide-react'
import { useLocation, NavLink } from 'react-router-dom'
import { useContentSessionStore } from '@/entities/content-session/store'
import { useAuthStore } from '@/entities/auth'
import { cn } from '@/shared/lib/utils'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/shared/ui/tooltip'
import { Separator } from '@/shared/ui/separator'

type MainLayoutProps = {
  children: ReactNode
  headerContent?: ReactNode
  bottomConsole?: ReactNode
}

const NAV_GROUPS = [
  {
    items: [
      { icon: Inbox, label: 'Inbox', to: '/inbox' },
    ],
  },
  {
    items: [
      { icon: Box, label: 'Workflows', to: '/workflows' },
      { icon: Workflow, label: 'Pipelines', to: '/pipelines' },
    ],
  },
  {
    items: [
      { icon: Globe, label: 'Published', to: '/published' },
    ],
  },
  {
    items: [
      { icon: Activity, label: 'Runs', to: '/runs' },
      { icon: Zap, label: 'Connections', to: '/connections' },
    ],
  },
]

function navLinkClass({ isActive }: { isActive: boolean }): string {
  return cn(
    'relative block px-3 py-2.5 rounded-xl transition-all duration-200 overflow-hidden',
    isActive
      ? 'bg-primary/10 text-primary shadow-sm'
      : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
  )
}

function NavBadge({ count }: { count: number }) {
  if (count <= 0) return null
  return (
    <span className="absolute -top-1.5 -right-2 flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full bg-destructive text-destructive-foreground text-[10px] font-bold leading-none">
      {count > 99 ? '99+' : count}
    </span>
  )
}

export function MainLayout({ children, headerContent, bottomConsole }: MainLayoutProps) {
  const [gnbVisible, setGnbVisible] = useState(true)
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const location = useLocation()
  const user = useAuthStore((s) => s.user)
  const authLogout = useAuthStore((s) => s.logout)
  const pendingCount = useContentSessionStore((s) => s.pendingCount)
  const publishReadyCount = useContentSessionStore((s) => s.publishReadyCount)
  const syncBadgeCounts = useContentSessionStore((s) => s.syncBadgeCounts)

  const badgeMap: Record<string, number> = {
    '/inbox': pendingCount + publishReadyCount,
  }

  useEffect(() => {
    syncBadgeCounts()
  }, [syncBadgeCounts])

  useEffect(() => {
    setMobileNavOpen(false) // eslint-disable-line react-hooks/set-state-in-effect
  }, [location.pathname])

  function handleMenuClick(): void {
    if (window.innerWidth < 768) {
      setMobileNavOpen(true)
    } else {
      setGnbVisible(!gnbVisible)
    }
  }

  function handleLogout(): void {
    authLogout().then(() => { window.location.href = '/login' })
  }

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-background text-foreground selection:bg-primary/20">

      <nav className={cn(
        'hidden md:flex flex-col py-4 bg-sidebar border-r border-sidebar-border z-50 transition-all duration-300 ease-[cubic-bezier(0.2,0.8,0.2,1)] overflow-hidden shadow-xl shrink-0',
        gnbVisible ? 'w-[240px]' : 'w-[64px] items-center',
      )}>
        <div className="mb-8 px-4 flex flex-row items-center gap-3 font-bold text-xl text-primary font-heading overflow-hidden whitespace-nowrap">
          <div className="size-8 min-w-8 rounded-lg bg-primary text-primary-foreground flex items-center justify-center">U</div>
          {gnbVisible && (
            <span className="transition-all duration-300 animate-in fade-in slide-in-from-left-2 overflow-hidden">
              Upal
            </span>
          )}
        </div>

        <TooltipProvider delayDuration={0}>
          <div className="flex flex-col flex-1 px-2">
            {NAV_GROUPS.map((group, gi) => (
              <div key={gi}>
                {gi > 0 && <Separator className="my-2 mx-1" />}
                <div className="flex flex-col gap-1">
                  {group.items.map((item) => (
                    <Tooltip key={item.to}>
                      <TooltipTrigger asChild>
                        <NavLink to={item.to} className={navLinkClass}>
                          <div className="flex flex-row items-center gap-3 whitespace-nowrap">
                            <span className="relative shrink-0">
                              <item.icon className="size-5 min-w-5" />
                              <NavBadge count={badgeMap[item.to] ?? 0} />
                            </span>
                            {gnbVisible && (
                              <span className="text-sm font-medium">{item.label}</span>
                            )}
                          </div>
                        </NavLink>
                      </TooltipTrigger>
                      {!gnbVisible && <TooltipContent side="right" className="font-medium">{item.label}</TooltipContent>}
                    </Tooltip>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </TooltipProvider>

        <div className="mt-auto px-2 flex flex-col gap-2">
          <NavLink to="/settings" className={navLinkClass}>
            <div className="flex flex-row items-center gap-3 whitespace-nowrap">
              <Settings className="size-5 min-w-5 shrink-0" />
              {gnbVisible && (
                <span className="text-sm font-medium">Settings</span>
              )}
            </div>
          </NavLink>
          {user && (
            <>
              <Separator className="mx-1" />
              <button
                className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground transition-all duration-200 overflow-hidden"
                onClick={handleLogout}
              >
                <img src={user.avatar_url} alt="" className="size-5 min-w-5 rounded-full shrink-0" />
                {gnbVisible && <>
                  <span className="truncate font-medium">{user.name}</span>
                  <LogOut className="size-4 ml-auto shrink-0 opacity-60" />
                </>}
              </button>
            </>
          )}
        </div>
      </nav>

      {mobileNavOpen && (
        <div className="fixed inset-0 z-[100] md:hidden">
          <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={() => setMobileNavOpen(false)} />
          <nav className="absolute inset-y-0 left-0 w-[280px] bg-sidebar border-r border-sidebar-border shadow-2xl flex flex-col py-4 animate-in slide-in-from-left duration-200">
            <div className="mb-6 px-4 flex items-center justify-between">
              <div className="flex items-center gap-3 font-bold text-xl text-primary font-heading">
                <div className="size-8 min-w-8 rounded-lg bg-primary text-primary-foreground flex items-center justify-center">U</div>
                <span>Upal</span>
              </div>
              <button onClick={() => setMobileNavOpen(false)} className="p-1.5 rounded-lg text-muted-foreground hover:bg-sidebar-accent transition-colors">
                <X className="size-5" />
              </button>
            </div>
            <div className="flex flex-col flex-1 px-2">
              {NAV_GROUPS.map((group, gi) => (
                <div key={gi}>
                  {gi > 0 && <Separator className="my-2 mx-1" />}
                  <div className="flex flex-col gap-1">
                    {group.items.map((item) => (
                      <NavLink key={item.to} to={item.to} className={navLinkClass}>
                        <div className="flex items-center gap-3">
                          <span className="relative shrink-0">
                            <item.icon className="size-5" />
                            <NavBadge count={badgeMap[item.to] ?? 0} />
                          </span>
                          <span className="text-sm font-medium">{item.label}</span>
                        </div>
                      </NavLink>
                    ))}
                  </div>
                </div>
              ))}
            </div>
            <div className="px-2 flex flex-col gap-2">
              <NavLink to="/settings" className={navLinkClass}>
                <div className="flex items-center gap-3">
                  <Settings className="size-5 shrink-0" />
                  <span className="text-sm font-medium">Settings</span>
                </div>
              </NavLink>
              {user && (
                <>
                  <Separator className="mx-1" />
                  <button
                    className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground transition-all duration-200"
                    onClick={handleLogout}
                  >
                    <img src={user.avatar_url} alt="" className="size-5 rounded-full shrink-0" />
                    <span className="truncate font-medium">{user.name}</span>
                    <LogOut className="size-4 ml-auto shrink-0 opacity-60" />
                  </button>
                </>
              )}
            </div>
          </nav>
        </div>
      )}

      <div className="flex flex-col flex-1 min-w-0 h-full relative z-0">

        <header className="h-[56px] min-h-[56px] border-b border-border bg-background/80 backdrop-blur-md flex items-center px-4 justify-between z-40 sticky top-0">
          <div className="flex-1 flex items-center gap-3 font-medium">
            <button
              onClick={handleMenuClick}
              className={cn(
                'p-2 -ml-2 rounded-lg transition-all duration-200',
                gnbVisible
                  ? 'md:bg-primary/10 md:text-primary text-muted-foreground'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
              )}
              title={gnbVisible ? 'Hide Menu' : 'Show Menu'}
            >
              <Menu className="size-5" />
            </button>
            <Separator orientation="vertical" className="h-4" />
            {headerContent || <span className="text-muted-foreground">Upal Workspace</span>}
          </div>
        </header>

        <div className="flex flex-1 overflow-hidden relative">
          <main className="flex-1 flex flex-col relative h-full bg-grid-pattern overflow-hidden">
            {children}
          </main>
        </div>

        {bottomConsole}

      </div>

    </div>
  )
}
