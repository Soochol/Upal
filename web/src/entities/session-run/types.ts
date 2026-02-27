import type { SessionSource, SessionWorkflow, SessionContext } from '@/entities/session/types'

export type RunStatus =
  | 'draft' | 'collecting' | 'analyzing' | 'pending_review'
  | 'approved' | 'rejected' | 'producing' | 'published' | 'error'

export type WorkflowRunStatus = 'pending' | 'running' | 'success' | 'failed' | 'published' | 'rejected'

export type SourceItem = {
  title: string
  url?: string
  score?: number
  extra?: string
}

export type SourceFetch = {
  id: string
  tool: string
  source_type: 'static' | 'signal'
  label: string
  count: number
  items: SourceItem[]
  fetched_at: string
}

export type ContentAngle = {
  id: string
  format: string
  title: string
  selected: boolean
  workflow_name?: string
  match_type?: 'matched' | 'generated' | 'none'
  rationale?: string
}

export type SourceHighlight = {
  source_id: string
  title: string
  key_points: string[]
}

export type LLMAnalysis = {
  summary: string
  source_highlights?: SourceHighlight[]
  insights: string[]
  angles: ContentAngle[]
  score: number
  total_collected: number
  total_selected: number
}

export type WorkflowRun = {
  workflow_name: string
  run_id: string
  status: WorkflowRunStatus
  channel_id?: string
  output_url?: string
  completed_at?: string
  error_message?: string
  failed_node_id?: string
}

export type Run = {
  id: string
  session_id: string
  name?: string
  session_name?: string
  run_number?: number
  status: RunStatus
  trigger_type: 'schedule' | 'manual' | 'surge'
  source_count?: number
  // Config (moved from Session to Run)
  run_sources?: SessionSource[]
  run_workflows?: SessionWorkflow[]
  context?: SessionContext
  schedule?: string
  schedule_active?: boolean
  // Composed data
  sources?: SourceFetch[]
  analysis?: LLMAnalysis
  workflow_runs?: WorkflowRun[]
  created_at: string
  reviewed_at?: string
}
