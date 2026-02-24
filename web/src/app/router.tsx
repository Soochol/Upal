import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import ProductLandingPage from '@/pages/landing/ProductLanding'
import LandingPage from '@/pages/landing'
import EditorPage from '@/pages/Editor'
import RunsPage from '@/pages/runs'
import PipelinesPage from '@/pages/pipelines'
import PipelineDetailPage from '@/pages/pipelines/PipelineDetail'
import PipelineNewPage from '@/pages/pipelines/PipelineNew'
import ConnectionsPage from '@/pages/connections'
import { RunDetail } from '@/widgets/run-detail'
import PublishedPage from '@/pages/Published'
import { ErrorBoundary } from '@/shared/ui/ErrorBoundary'
import { ToastContainer } from '@/shared/ui/ToastContainer'

const ReviewInboxPage = lazy(() => import('@/pages/inbox'))
const PublishInboxPage = lazy(() => import('@/pages/publish-inbox'))

export function AppRouter() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<ProductLandingPage />} />
          <Route path="/workflows" element={<LandingPage />} />
          <Route path="/editor" element={<EditorPage />} />
          <Route path="/runs" element={<RunsPage />} />
          <Route path="/runs/:id" element={<RunDetail />} />
          <Route path="/connections" element={<ConnectionsPage />} />

          {/* Inbox (Unified Reviews) */}
          <Route path="/inbox" element={<Suspense fallback={null}><ReviewInboxPage /></Suspense>} />
          <Route path="/publish-inbox" element={<Suspense fallback={null}><PublishInboxPage /></Suspense>} />

          {/* Pipelines */}
          <Route path="/pipelines" element={<PipelinesPage />} />
          <Route path="/pipelines/new" element={<PipelineNewPage />} />
          <Route path="/pipelines/:id" element={<PipelineDetailPage />} />
          <Route path="/pipelines/:id/sessions/*" element={<Navigate to=".." replace />} />

          {/* Content */}
          <Route path="/published" element={<PublishedPage />} />
        </Routes>
      </BrowserRouter>
      <ToastContainer />
    </ErrorBoundary>
  )
}
