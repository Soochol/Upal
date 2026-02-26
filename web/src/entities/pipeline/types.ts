export type PipelineSourceType =
  | 'rss'
  | 'hn'
  | 'reddit'
  | 'google_trends'
  | 'social'
  | 'http'

export type PipelineSource = {
  id: string
  type: PipelineSourceType
  source_type: 'static' | 'signal'
  label: string
  // type-specific config
  url?: string             // rss, http
  subreddit?: string       // reddit
  min_score?: number       // reddit, hn
  keywords?: string[]      // google_trends, social
  accounts?: string[]      // social: follow account handles
  geo?: string             // google_trends: country code
  limit?: number
}

export type PipelineContext = {
  purpose: string
  target_audience: string
  tone_style: string
  focus_keywords: string[]
  exclude_keywords: string[]
  language: string
}

export type PipelineWorkflow = {
  workflow_name: string
  label?: string
  auto_select?: boolean
  channel_id?: string
}

export type Pipeline = {
  id: string
  name: string
  description?: string
  stages: Stage[]
  thumbnail_svg?: string
  last_collected_at?: string
  pending_session_count?: number
  created_at: string
  updated_at: string
}

export type Stage = {
  id: string
  name: string
  type: 'workflow' | 'approval' | 'notification' | 'schedule' | 'trigger' | 'transform' | 'collect'
  config: StageConfig
  depends_on?: string[]
}

export type CollectSource = {
  id: string
  type: 'rss' | 'http' | 'scrape'
  url: string
  limit?: number
  method?: string
  headers?: Record<string, string>
  body?: string
  selector?: string
  attribute?: string
  scrape_limit?: number
}

export type StageConfig = {
  workflow_name?: string
  input_mapping?: Record<string, string>
  message?: string
  connection_id?: string
  subject?: string
  timeout?: number
  cron?: string
  timezone?: string
  schedule_id?: string
  trigger_id?: string
  expression?: string
  sources?: CollectSource[]
}

export type PipelineRun = {
  id: string
  pipeline_id: string
  status: 'pending' | 'running' | 'waiting' | 'completed' | 'failed'
  current_stage?: string
  stage_results?: Record<string, StageResult>
  started_at: string
  completed_at?: string
}

export type StageResult = {
  stage_id: string
  status: 'pending' | 'running' | 'waiting' | 'completed' | 'skipped' | 'failed'
  output?: Record<string, unknown>
  error?: string
  started_at: string
  completed_at?: string
}

