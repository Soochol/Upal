import { apiFetch } from '@/shared/api/client'
import type { ContentSession, LLMAnalysis } from './types'

const BASE = '/api/content-sessions'

export async function fetchContentSessions(params?: {
  pipelineId?: string
  status?: string
}): Promise<ContentSession[]> {
  const qs = new URLSearchParams()
  if (params?.pipelineId) qs.set('pipeline_id', params.pipelineId)
  if (params?.status) qs.set('status', params.status)
  const query = qs.toString() ? `?${qs}` : ''
  return apiFetch<ContentSession[]>(`${BASE}${query}`)
}

export async function fetchContentSession(id: string): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}`)
}

export async function approveSession(
  id: string,
  _selectedAngles: string[],
): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ action: 'approve' }),
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
