import { API_BASE, apiFetch } from './client'
import type { WorkflowDefinition } from '../serializer'

export async function suggestWorkflowName(wf: WorkflowDefinition): Promise<string> {
  const data = await apiFetch<{ name: string }>(`${API_BASE}/workflows/suggest-name`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(wf),
  })
  return data.name
}

export async function saveWorkflow(wf: WorkflowDefinition, originalName?: string): Promise<void> {
  if (originalName) {
    await apiFetch<void>(`${API_BASE}/workflows/${encodeURIComponent(originalName)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(wf),
    })
  } else {
    await apiFetch<void>(`${API_BASE}/workflows`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(wf),
    })
  }
}

export async function loadWorkflow(name: string): Promise<WorkflowDefinition> {
  return apiFetch<WorkflowDefinition>(`${API_BASE}/workflows/${encodeURIComponent(name)}`)
}

export async function deleteWorkflow(name: string): Promise<void> {
  return apiFetch<void>(`${API_BASE}/workflows/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  })
}

export async function listWorkflows(): Promise<WorkflowDefinition[]> {
  return apiFetch<WorkflowDefinition[]>(`${API_BASE}/workflows`)
}

export async function generateWorkflow(
  description: string,
  model?: string,
  existingWorkflow?: WorkflowDefinition,
): Promise<WorkflowDefinition> {
  const body: Record<string, unknown> = { description }
  if (model) body.model = model
  if (existingWorkflow) body.existing_workflow = existingWorkflow
  return apiFetch<WorkflowDefinition>(`${API_BASE}/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}
