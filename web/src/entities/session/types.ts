import type { PipelineSource, PipelineWorkflow, PipelineContext, Stage } from '@/entities/pipeline'

export type SessionStatus = 'draft' | 'active' | 'archived'

export type Session = {
  id: string
  name: string
  description?: string
  sources?: PipelineSource[]
  schedule?: string
  model?: string
  workflows?: PipelineWorkflow[]
  context?: PipelineContext
  stages?: Stage[]
  status: SessionStatus
  thumbnail_svg?: string
  pending_run_count?: number
  last_collected_at?: string
  created_at: string
  updated_at: string
}
