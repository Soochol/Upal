import { apiFetch } from '@/shared/api/client'
import type { ContentSession, ContentAngle, LLMAnalysis } from './types'

const BASE = '/api/content-sessions'

export async function fetchContentSessions(params?: {
  pipelineId?: string
  status?: string
  archivedOnly?: boolean
  includeArchived?: boolean
}): Promise<ContentSession[]> {
  const qs = new URLSearchParams()
  if (params?.pipelineId) qs.set('pipeline_id', params.pipelineId)
  if (params?.status) qs.set('status', params.status)
  if (params?.archivedOnly) qs.set('archived_only', 'true')
  if (params?.includeArchived) qs.set('include_archived', 'true')
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

export async function rejectWorkflowResult(
  id: string,
  runId: string,
): Promise<{ status: string }> {
  return apiFetch<{ status: string }>(`${BASE}/${encodeURIComponent(id)}/reject-result`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ run_id: runId }),
  })
}

export async function rejectSession(id: string, _reason?: string): Promise<ContentSession> {
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

export async function archiveSession(id: string): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}/archive`, {
    method: 'POST',
  })
}

export async function unarchiveSession(id: string): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}/unarchive`, {
    method: 'POST',
  })
}

export async function deleteSession(id: string): Promise<void> {
  await apiFetch(`${BASE}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export type SessionEvent =
  | { type: 'source_fetched'; tool: string; count: number }
  | { type: 'analysis_complete'; score: number; summary: string }
  | { type: 'status_changed'; status: string }

export function subscribeToSession(
  sessionId: string,
  onEvent: (e: SessionEvent) => void,
): () => void {
  const es = new EventSource(`${BASE}/${encodeURIComponent(sessionId)}/events`)
  es.onmessage = (e) => {
    try {
      onEvent(JSON.parse(e.data) as SessionEvent)
    } catch {
      // ignore malformed events
    }
  }
  return () => es.close()
}
