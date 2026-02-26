export type SessionStatus = 'draft' | 'active' | 'archived'

export type SessionSourceType =
  | 'rss'
  | 'hn'
  | 'reddit'
  | 'google_trends'
  | 'social'
  | 'http'
  | 'research'

export type SessionSource = {
  id: string
  type: SessionSourceType
  source_type: 'static' | 'signal' | 'research'
  label: string
  url?: string
  subreddit?: string
  min_score?: number
  keywords?: string[]
  accounts?: string[]
  geo?: string
  limit?: number
  topic?: string
  depth?: 'light' | 'deep'
  model?: string
}

export type SessionContext = {
  prompt?: string
  language?: string
  research_depth?: 'light' | 'deep'
  research_model?: string
}

export type SessionWorkflow = {
  workflow_name: string
  label?: string
  auto_select?: boolean
  channel_id?: string
}

export type CollectSource = {
  id: string
  type: 'rss' | 'http' | 'scrape' | 'social' | 'research'
  url: string
  limit?: number
  method?: string
  headers?: Record<string, string>
  body?: string
  selector?: string
  attribute?: string
  scrape_limit?: number
  keywords?: string[]
  accounts?: string[]
  topic?: string
  model?: string
  depth?: 'light' | 'deep'
  max_searches?: number
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

export type Stage = {
  id: string
  name: string
  type: 'workflow' | 'approval' | 'notification' | 'schedule' | 'trigger' | 'transform' | 'collect'
  config: StageConfig
  depends_on?: string[]
}

export type Session = {
  id: string
  name: string
  description?: string
  sources?: SessionSource[]
  schedule?: string
  model?: string
  workflows?: SessionWorkflow[]
  context?: SessionContext
  stages?: Stage[]
  status: SessionStatus
  thumbnail_svg?: string
  pending_run_count?: number
  last_collected_at?: string
  created_at: string
  updated_at: string
}
