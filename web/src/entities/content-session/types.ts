// ContentSession entity types
// ContentSessionStatus and SourceType live in shared/types to avoid FSD violations
import type { ContentSessionStatus, SourceType } from '@/shared/types'
import type { PipelineSource, PipelineWorkflow, PipelineContext } from '@/entities/pipeline'
export type { ContentSessionStatus, SourceType }

export type SourceItem = {
  title: string
  url?: string
  score?: number       // points, upvotes, rank
  extra?: string       // e.g. subreddit, author
}

export type SourceFetch = {
  id: string
  tool: string         // "hn_rss", "reddit", "google_trends", etc.
  source_type: SourceType
  label: string        // display name
  count: number
  items: SourceItem[]
  fetched_at: string
}

export type ContentAngle = {
  id: string
  format: string       // "shorts", "blog", "newsletter", etc.
  title: string
  selected: boolean
  workflow_name?: string   // LLM-recommended workflow
  match_type?: 'matched' | 'generated' | 'none'
  rationale?: string
}

export type LLMAnalysis = {
  summary: string
  insights: string[]
  angles: ContentAngle[]
  score: number
  total_collected: number
  total_selected: number
}

export type WorkflowResult = {
  workflow_name: string
  run_id: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'published' | 'rejected'
  output_url?: string
  completed_at?: string
  channel_id?: string
  error_message?: string
  failed_node_id?: string
}

export type ContentSession = {
  id: string
  pipeline_id: string
  name?: string
  pipeline_name?: string
  session_number?: number
  trigger_type: 'schedule' | 'manual' | 'surge'
  status: ContentSessionStatus
  is_template?: boolean
  parent_session_id?: string
  schedule_id?: string
  source_count?: number
  // Session-level settings (override pipeline defaults)
  session_sources?: PipelineSource[]
  schedule?: string
  model?: string
  session_workflows?: PipelineWorkflow[]
  context?: PipelineContext
  // Composed data
  sources?: SourceFetch[]
  analysis?: LLMAnalysis
  workflow_results?: WorkflowResult[]
  created_at: string
  updated_at?: string
  reviewed_at?: string
  archived_at?: string
}
