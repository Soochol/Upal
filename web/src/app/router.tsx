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

const ReviewInboxPage = lazy(() => import('@/pages/inbox'))
const PublishInboxPage = lazy(() => import('@/pages/publish-inbox'))

function PipelineRedirect() {
  const { id, sessionId } = useParams()
  const params = new URLSearchParams()
  if (id) params.set('p', id)
  if (sessionId) params.set('s', sessionId)
  return <Navigate to={`/pipelines?${params.toString()}`} replace />
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

          {/* Inbox (Unified Reviews) */}
          <Route path="/inbox" element={<Suspense fallback={null}><ReviewInboxPage /></Suspense>} />
          <Route path="/publish-inbox" element={<Suspense fallback={null}><PublishInboxPage /></Suspense>} />

          {/* Pipelines */}
          <Route path="/pipelines" element={<PipelinesPage />} />
          <Route path="/pipelines/new" element={<PipelineNewPage />} />
          <Route path="/pipelines/:id" element={<PipelineRedirect />} />
          <Route path="/pipelines/:id/sessions/:sessionId" element={<PipelineRedirect />} />

          {/* Content */}
          <Route path="/published" element={<PublishedPage />} />
        </Routes>
      </BrowserRouter>
      <ToastContainer />
    </ErrorBoundary>
  )
}
