import { BrowserRouter, Routes, Route } from 'react-router-dom'
import ProductLanding from '@/pages/ProductLanding'
import Landing from '@/pages/Landing'
import Editor from '@/pages/Editor'
import Runs from '@/pages/Runs'
import Schedules from '@/pages/Schedules'
import Pipelines from '@/pages/Pipelines'
import RunDetail from '@/components/runs/RunDetail'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { ToastContainer } from '@/components/ToastContainer'

export default function App() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<ProductLanding />} />
          <Route path="/workflows" element={<Landing />} />
          <Route path="/editor" element={<Editor />} />
          <Route path="/runs" element={<Runs />} />
          <Route path="/runs/:id" element={<RunDetail />} />
          <Route path="/schedules" element={<Schedules />} />
          <Route path="/pipelines" element={<Pipelines />} />
          <Route path="/pipelines/:id" element={<Pipelines />} />
        </Routes>
      </BrowserRouter>
      <ToastContainer />
    </ErrorBoundary>
  )
}
