import { BrowserRouter, Routes, Route, Navigate, useParams } from 'react-router-dom'
import ProductLanding from '@/pages/ProductLanding'
import WorkflowsPage from '@/pages/workflows'
import Runs from '@/pages/Runs'
import Pipelines from '@/pages/Pipelines'
import PublishedPage from '@/pages/published'
import Connections from '@/pages/Connections'
import { RunViewer } from '@/pages/runs/RunViewer'
import { ErrorBoundary } from '@/shared/ui/ErrorBoundary'
import { ToastContainer } from '@/shared/ui/ToastContainer'

function PipelineRedirect() {
  const { id, sessionId } = useParams()
  const params = new URLSearchParams()
  if (id) params.set('p', id)
  if (sessionId) params.set('s', sessionId)
  return <Navigate to={`/pipelines?${params.toString()}`} replace />
}

export default function App() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<ProductLanding />} />
          <Route path="/workflows" element={<WorkflowsPage />} />
          <Route path="/runs" element={<Runs />} />
          <Route path="/runs/:id" element={<RunViewer />} />
          <Route path="/pipelines" element={<Pipelines />} />
          <Route path="/pipelines/:id" element={<PipelineRedirect />} />
          <Route path="/pipelines/:id/sessions/:sessionId" element={<PipelineRedirect />} />
          <Route path="/published" element={<PublishedPage />} />
          <Route path="/connections" element={<Connections />} />
        </Routes>
      </BrowserRouter>
      <ToastContainer />
    </ErrorBoundary>
  )
}
