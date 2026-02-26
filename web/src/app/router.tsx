import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate, useParams } from 'react-router-dom'
import WorkflowsPage from '@/pages/workflows'
import RunsPage from '@/pages/runs'
import PipelinesPage from '@/pages/pipelines'
import ConnectionsPage from '@/pages/connections'
import { RunViewer } from '@/pages/runs/RunViewer'
import PublishedPage from '@/pages/Published'
import { ErrorBoundary } from '@/shared/ui/ErrorBoundary'
import { ToastContainer } from '@/shared/ui/ToastContainer'

const InboxPage = lazy(() => import('@/pages/inbox'))
const SettingsPage = lazy(() => import('@/pages/settings'))

function PipelineRedirect() {
  const { id, sessionId } = useParams()
  const search = sessionId ? `?p=${id}&s=${sessionId}` : `?p=${id}`
  return <Navigate to={`/pipelines${search}`} replace />
}

export function AppRouter() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Navigate to="/pipelines" replace />} />
          <Route path="/workflows" element={<WorkflowsPage />} />
          <Route path="/runs" element={<RunsPage />} />
          <Route path="/runs/:id" element={<RunViewer />} />
          <Route path="/connections" element={<ConnectionsPage />} />

          {/* Inbox */}
          <Route path="/inbox" element={<Suspense fallback={null}><InboxPage /></Suspense>} />
          <Route path="/publish-inbox" element={<Navigate to="/inbox" replace />} />

          {/* Pipelines */}
          <Route path="/pipelines" element={<PipelinesPage />} />
          <Route path="/pipelines/:id" element={<PipelineRedirect />} />
          <Route path="/pipelines/:id/sessions/:sessionId" element={<PipelineRedirect />} />

          {/* Content */}
          <Route path="/published" element={<PublishedPage />} />

          {/* Settings */}
          <Route path="/settings" element={<Suspense fallback={null}><SettingsPage /></Suspense>} />
        </Routes>
      </BrowserRouter>
      <ToastContainer />
    </ErrorBoundary>
  )
}
