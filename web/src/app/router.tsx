import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate, useParams } from 'react-router-dom'
import ProductLandingPage from '@/pages/landing/ProductLanding'
import WorkflowsPage from '@/pages/workflows'
import RunsPage from '@/pages/runs'
import PipelinesPage from '@/pages/pipelines'
import PipelineNewPage from '@/pages/pipelines/PipelineNew'
import ConnectionsPage from '@/pages/connections'
import { RunViewer } from '@/pages/runs/RunViewer'
import PublishedPage from '@/pages/Published'
import { ErrorBoundary } from '@/shared/ui/ErrorBoundary'
import { ToastContainer } from '@/shared/ui/ToastContainer'

const InboxPage = lazy(() => import('@/pages/inbox'))
const SettingsPage = lazy(() => import('@/pages/settings'))

function PipelineRedirect() {
  const { id, sessionId } = useParams()
  if (sessionId) {
    return <Navigate to={`/pipelines?p=${id}&s=${sessionId}`} replace />
  }
  return <Navigate to={`/pipelines?p=${id}`} replace />
}

export function AppRouter() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<ProductLandingPage />} />
          <Route path="/workflows" element={<WorkflowsPage />} />
          <Route path="/runs" element={<RunsPage />} />
          <Route path="/runs/:id" element={<RunViewer />} />
          <Route path="/connections" element={<ConnectionsPage />} />

          {/* Inbox */}
          <Route path="/inbox" element={<Suspense fallback={null}><InboxPage /></Suspense>} />
          <Route path="/publish-inbox" element={<Navigate to="/inbox" replace />} />

          {/* Pipelines */}
          <Route path="/pipelines" element={<PipelinesPage />} />
          <Route path="/pipelines/new" element={<PipelineNewPage />} />
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
