// web/src/lib/api/pipelines.ts
import { apiFetch } from './client'
import type { Pipeline, PipelineRun } from './types'

const API_BASE = '/api'

export async function fetchPipelines(): Promise<Pipeline[]> {
  return apiFetch<Pipeline[]>(`${API_BASE}/pipelines`)
}

export async function fetchPipeline(id: string): Promise<Pipeline> {
  return apiFetch<Pipeline>(`${API_BASE}/pipelines/${encodeURIComponent(id)}`)
}

export async function createPipeline(data: Partial<Pipeline>): Promise<Pipeline> {
  return apiFetch<Pipeline>(`${API_BASE}/pipelines`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function updatePipeline(id: string, data: Partial<Pipeline>): Promise<Pipeline> {
  return apiFetch<Pipeline>(`${API_BASE}/pipelines/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function deletePipeline(id: string): Promise<void> {
  return apiFetch<void>(`${API_BASE}/pipelines/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export async function startPipeline(id: string): Promise<PipelineRun> {
  return apiFetch<PipelineRun>(`${API_BASE}/pipelines/${encodeURIComponent(id)}/start`, {
    method: 'POST',
  })
}

export async function fetchPipelineRuns(id: string): Promise<PipelineRun[]> {
  return apiFetch<PipelineRun[]>(`${API_BASE}/pipelines/${encodeURIComponent(id)}/runs`)
}

export async function approvePipelineStage(pipelineId: string, stageId: string): Promise<void> {
  return apiFetch<void>(
    `${API_BASE}/pipelines/${encodeURIComponent(pipelineId)}/stages/${encodeURIComponent(stageId)}/approve`,
    { method: 'POST' }
  )
}

export async function rejectPipelineStage(pipelineId: string, stageId: string): Promise<void> {
  return apiFetch<void>(
    `${API_BASE}/pipelines/${encodeURIComponent(pipelineId)}/stages/${encodeURIComponent(stageId)}/reject`,
    { method: 'POST' }
  )
}
