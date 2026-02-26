import { apiFetch } from '@/shared/api/client'
import type { PipelineSource, PipelineWorkflow, PipelineContext } from '@/entities/pipeline'
import type { ContentSession, ContentAngle, LLMAnalysis } from './types'

const BASE = '/api/content-sessions'

export async function fetchContentSessions(params?: {
  pipelineId?: string
  status?: string
  templateOnly?: boolean
  detail?: boolean
}): Promise<ContentSession[]> {
  const qs = new URLSearchParams()
  if (params?.pipelineId) qs.set('pipeline_id', params.pipelineId)
  if (params?.status) qs.set('status', params.status)
  if (params?.templateOnly) qs.set('template_only', 'true')
  if (params?.detail) qs.set('detail', 'true')
  const query = qs.toString() ? `?${qs}` : ''
  return apiFetch<ContentSession[]>(`${BASE}${query}`)
}

export async function fetchContentSession(id: string): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}`)
}

export async function approveSession(
  id: string,
  selectedWorkflows: string[],
): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ action: 'approve', selected_workflows: selectedWorkflows }),
  })
}

export async function produceSession(
  id: string,
  workflows: { name: string; channel_id?: string }[],
): Promise<{ session_id: string; status: string }> {
  return apiFetch(`${BASE}/${encodeURIComponent(id)}/produce`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ workflows }),
  })
}

export async function publishSession(
  id: string,
  runIds: string[],
): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}/publish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ run_ids: runIds }),
  })
}

export async function rejectSession(id: string): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ action: 'reject' }),
  })
}

export async function updateSessionAnalysis(
  id: string,
  data: { summary: string; insights: string[] },
): Promise<LLMAnalysis> {
  return apiFetch<LLMAnalysis>(`${BASE}/${encodeURIComponent(id)}/analysis`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function generateAngleWorkflow(
  sessionId: string,
  angleId: string,
): Promise<ContentAngle> {
  return apiFetch<ContentAngle>(
    `${BASE}/${encodeURIComponent(sessionId)}/generate-workflow`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ angle_id: angleId }),
    },
  )
}

export async function retryAnalyze(sessionId: string): Promise<void> {
  await apiFetch(`${BASE}/${encodeURIComponent(sessionId)}/retry-analyze`, {
    method: 'POST',
  })
}

export async function createDraftSession(data: {
  pipeline_id: string
  name?: string
  is_template?: boolean
  sources?: PipelineSource[]
  schedule?: string
  model?: string
  workflows?: PipelineWorkflow[]
  context?: PipelineContext
}): Promise<ContentSession> {
  return apiFetch<ContentSession>(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function deleteSession(id: string): Promise<void> {
  await apiFetch(`${BASE}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export async function updateSessionSettings(id: string, settings: {
  name?: string
  sources?: PipelineSource[]
  schedule?: string
  workflows?: PipelineWorkflow[]
  model?: string
  context?: PipelineContext
}): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}/settings`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
  })
}

export async function activateSession(id: string): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}/activate`, {
    method: 'POST',
  })
}

export async function deactivateSession(id: string): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}/deactivate`, {
    method: 'POST',
  })
}

export async function runSessionInstance(id: string, options?: { isTest?: boolean; limit?: number }): Promise<{ session_id: string }> {
  return apiFetch<{ session_id: string }>(`${BASE}/${encodeURIComponent(id)}/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(options ?? {}),
  })
}
