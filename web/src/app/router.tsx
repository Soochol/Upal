import { lazy, Suspense } from 'react'
import type { ReactNode } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { useAuthStore } from '@/entities/auth'
import { LoginPage } from '@/pages/login'
import WorkflowsPage from '@/pages/workflows'
import RunsPage from '@/pages/runs'
import ConnectionsPage from '@/pages/connections'
import { RunViewer } from '@/pages/runs/RunViewer'
import PublishedPage from '@/pages/Published'
import { ErrorBoundary } from '@/shared/ui/ErrorBoundary'
import { ToastContainer } from '@/shared/ui/ToastContainer'
import { GlobalChatBar } from './GlobalChatBar'

const InboxPage = lazy(() => import('@/pages/inbox'))
const SessionsPage = lazy(() => import('@/pages/sessions'))
const SettingsPage = lazy(() => import('@/pages/settings'))

function AuthGuard({ children }: { children: ReactNode }) {
  const { user, loading, initialized } = useAuthStore()

  if (!initialized || loading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!user) return <Navigate to="/login" replace />
  return children
}

export function AppRouter() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          {/* Login — outside auth guard */}
          <Route path="/login" element={<LoginPage />} />

          {/* All other routes — protected */}
          <Route path="/" element={<AuthGuard><Navigate to="/sessions" replace /></AuthGuard>} />
          <Route path="/workflows" element={<AuthGuard><WorkflowsPage /></AuthGuard>} />
          <Route path="/runs" element={<AuthGuard><RunsPage /></AuthGuard>} />
          <Route path="/runs/:id" element={<AuthGuard><RunViewer /></AuthGuard>} />
          <Route path="/connections" element={<AuthGuard><ConnectionsPage /></AuthGuard>} />

          {/* Inbox */}
          <Route path="/inbox" element={<AuthGuard><Suspense fallback={null}><InboxPage /></Suspense></AuthGuard>} />
          <Route path="/publish-inbox" element={<Navigate to="/inbox" replace />} />

          {/* Sessions */}
          <Route path="/sessions" element={<AuthGuard><Suspense fallback={null}><SessionsPage /></Suspense></AuthGuard>} />

          {/* Content */}
          <Route path="/published" element={<AuthGuard><PublishedPage /></AuthGuard>} />

          {/* Settings */}
          <Route path="/settings" element={<AuthGuard><Suspense fallback={null}><SettingsPage /></Suspense></AuthGuard>} />
        </Routes>
        <GlobalChatBar />
      </BrowserRouter>
      <ToastContainer />
    </ErrorBoundary>
  )
}
