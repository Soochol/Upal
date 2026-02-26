import { apiFetch } from '@/shared/api/client'
import type { Run, SourceFetch, LLMAnalysis } from './types'

const SESSIONS_BASE = '/api/sessions'
const RUNS_BASE = '/api/session-runs'

// --- Session-scoped run routes -----------------------------------------------

export async function createRun(sessionId: string): Promise<Run> {
  return apiFetch<Run>(`${SESSIONS_BASE}/${encodeURIComponent(sessionId)}/runs`, {
    method: 'POST',
  })
}

export async function fetchSessionRuns(sessionId: string): Promise<Run[]> {
  return apiFetch<Run[]>(`${SESSIONS_BASE}/${encodeURIComponent(sessionId)}/runs`)
}

// --- Top-level session-run routes --------------------------------------------

export async function fetchRuns(params?: { status?: string }): Promise<Run[]> {
  const qs = new URLSearchParams()
  if (params?.status) qs.set('status', params.status)
  const query = qs.toString() ? `?${qs}` : ''
  return apiFetch<Run[]>(`${RUNS_BASE}${query}`)
}

export async function fetchRun(id: string): Promise<Run> {
  return apiFetch<Run>(`${RUNS_BASE}/${encodeURIComponent(id)}`)
}

export async function produceRun(
  id: string,
  workflows: { name: string; channel_id?: string }[],
): Promise<{ run_id: string; status: string }> {
  return apiFetch(`${RUNS_BASE}/${encodeURIComponent(id)}/produce`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ workflows }),
  })
}

export async function publishRun(
  id: string,
  runIds: string[],
): Promise<Run> {
  return apiFetch<Run>(`${RUNS_BASE}/${encodeURIComponent(id)}/publish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ run_ids: runIds }),
  })
}

export async function rejectRun(id: string): Promise<Run> {
  return apiFetch<Run>(`${RUNS_BASE}/${encodeURIComponent(id)}/reject`, {
    method: 'POST',
  })
}

export async function fetchRunSources(id: string): Promise<SourceFetch[]> {
  return apiFetch<SourceFetch[]>(`${RUNS_BASE}/${encodeURIComponent(id)}/sources`)
}

export async function fetchRunAnalysis(id: string): Promise<LLMAnalysis> {
  return apiFetch<LLMAnalysis>(`${RUNS_BASE}/${encodeURIComponent(id)}/analysis`)
}

export async function updateRunAnalysis(
  id: string,
  data: { summary: string; insights: string[] },
): Promise<LLMAnalysis> {
  return apiFetch<LLMAnalysis>(`${RUNS_BASE}/${encodeURIComponent(id)}/analysis`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}
