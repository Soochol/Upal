import { API_BASE, apiFetch } from '@/shared/api/client'
import type { Pipeline, PipelineRun } from '../types'
import type { WorkflowDefinition } from '@/entities/workflow/lib/serializer'

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

export async function approvePipelineRun(pipelineId: string, runId: string): Promise<void> {
  return apiFetch<void>(
    `${API_BASE}/pipelines/${encodeURIComponent(pipelineId)}/runs/${encodeURIComponent(runId)}/approve`,
    { method: 'POST' }
  )
}

export async function rejectPipelineRun(pipelineId: string, runId: string): Promise<void> {
  return apiFetch<void>(
    `${API_BASE}/pipelines/${encodeURIComponent(pipelineId)}/runs/${encodeURIComponent(runId)}/reject`,
    { method: 'POST' }
  )
}

// Content pipeline: manual collect trigger
export async function collectPipeline(id: string, config?: { isTest?: boolean, limit?: number }): Promise<{ session_id: string }> {
  return apiFetch<{ session_id: string }>(
    `${API_BASE}/pipelines/${encodeURIComponent(id)}/collect`,
    {
      method: 'POST',
      ...(config ? { headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(config) } : {})
    },
  )
}

export type PipelineBundle = {
  pipeline: Omit<Pipeline, 'id' | 'created_at' | 'updated_at'>
  workflows: WorkflowDefinition[] | null
}

export async function generatePipelineBundle(
  description: string,
  existingPipeline?: Omit<Pipeline, 'id' | 'created_at' | 'updated_at'>,
): Promise<{ generation_id: string }> {
  const body: Record<string, unknown> = { description }
  if (existingPipeline) body.existing_pipeline = existingPipeline
  return apiFetch<{ generation_id: string }>(`${API_BASE}/generate-pipeline`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

