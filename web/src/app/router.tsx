import { BrowserRouter, Routes, Route } from 'react-router-dom'
import ProductLandingPage from '@/pages/landing/ProductLanding'
import LandingPage from '@/pages/landing'
import EditorPage from '@/pages/editor'
import RunsPage from '@/pages/runs'
import PipelinesPage from '@/pages/pipelines'
import ConnectionsPage from '@/pages/connections'
import { RunDetail } from '@/widgets/run-detail'
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
          <Route path="/pipelines" element={<PipelinesPage />} />
          <Route path="/pipelines/:id" element={<PipelinesPage />} />
          <Route path="/connections" element={<ConnectionsPage />} />
        </Routes>
      </BrowserRouter>
      <ToastContainer />
    </ErrorBoundary>
  )
}
