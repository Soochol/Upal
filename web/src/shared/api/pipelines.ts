import { apiFetch } from './client'
import type { Pipeline, PipelineRun } from '@/shared/types'
import type { WorkflowDefinition } from '@/entities/workflow'

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

export type PipelineBundle = {
  pipeline: Omit<Pipeline, 'id' | 'created_at' | 'updated_at'>
  workflows: WorkflowDefinition[]
}

export async function generatePipelineBundle(
  description: string,
  existingPipeline?: Omit<Pipeline, 'id' | 'created_at' | 'updated_at'>,
): Promise<PipelineBundle> {
  const body: Record<string, unknown> = { description }
  if (existingPipeline) body.existing_pipeline = existingPipeline
  return apiFetch<PipelineBundle>(`${API_BASE}/generate-pipeline`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function generatePipelineThumbnail(id: string): Promise<string> {
  const data = await apiFetch<{ thumbnail_svg: string }>(
    `${API_BASE}/pipelines/${encodeURIComponent(id)}/thumbnail`,
    { method: 'POST' },
  )
  return data.thumbnail_svg
}
