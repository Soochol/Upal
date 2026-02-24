// ContentSession entity types
// ContentSessionStatus and SourceType live in shared/types to avoid FSD violations
import type { ContentSessionStatus, SourceType } from '@/shared/types'
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
}

export type ContentSession = {
  id: string
  pipeline_id: string
  pipeline_name?: string
  session_number?: number
  trigger_type: 'schedule' | 'manual' | 'surge'
  status: ContentSessionStatus
  sources?: SourceFetch[]
  analysis?: LLMAnalysis
  workflow_results?: WorkflowResult[]
  created_at: string
  updated_at?: string
  archived_at?: string
}
