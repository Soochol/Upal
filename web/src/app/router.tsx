import { BrowserRouter, Routes, Route } from 'react-router-dom'
import ProductLandingPage from '@/pages/landing/ProductLanding'
import LandingPage from '@/pages/landing'
import EditorPage from '@/pages/editor'
import RunsPage from '@/pages/runs'
import PipelinesPage from '@/pages/pipelines'
import PipelineDetailPage from '@/pages/pipelines/PipelineDetail'
import PipelineNewPage from '@/pages/pipelines/PipelineNew'
import ConnectionsPage from '@/pages/connections'
import { RunDetail } from '@/widgets/run-detail'
import InboxPage from '@/pages/Inbox'
import SessionDetailPage from '@/pages/SessionDetail'
import PublishedPage from '@/pages/Published'
import { ErrorBoundary } from '@/shared/ui/ErrorBoundary'
import { ToastContainer } from '@/shared/ui/ToastContainer'

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

          {/* Pipelines */}
          <Route path="/pipelines" element={<PipelinesPage />} />
          <Route path="/pipelines/new" element={<PipelineNewPage />} />
          <Route path="/pipelines/:id" element={<PipelineDetailPage />} />

          {/* Content operations */}
          <Route path="/inbox" element={<InboxPage />} />
          <Route path="/inbox/:sessionId" element={<SessionDetailPage />} />
          <Route path="/published" element={<PublishedPage />} />
        </Routes>
      </BrowserRouter>
      <ToastContainer />
    </ErrorBoundary>
  )
}
