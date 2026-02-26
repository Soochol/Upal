import type { SourceFetch, LLMAnalysis } from '@/entities/content-session'

export type RunStatus =
  | 'collecting' | 'analyzing' | 'pending_review'
  | 'approved' | 'rejected' | 'producing' | 'published' | 'error'

export type WorkflowRunStatus = 'pending' | 'running' | 'success' | 'failed' | 'published' | 'rejected'

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
  session_name?: string
  run_number?: number
  status: RunStatus
  trigger_type: 'schedule' | 'manual' | 'surge'
  source_count?: number
  sources?: SourceFetch[]
  analysis?: LLMAnalysis
  workflow_runs?: WorkflowRun[]
  created_at: string
  reviewed_at?: string
}
