import { API_BASE, apiFetch } from './client'
import type { Trigger } from './types'

export async function createTrigger(data: Partial<Trigger>): Promise<{ trigger: Trigger; webhook_url: string }> {
  return apiFetch<{ trigger: Trigger; webhook_url: string }>(`${API_BASE}/triggers`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function fetchTriggers(workflowName: string): Promise<Trigger[]> {
  return apiFetch<Trigger[]>(`${API_BASE}/workflows/${encodeURIComponent(workflowName)}/triggers`)
}

export async function deleteTrigger(id: string): Promise<void> {
  return apiFetch<void>(`${API_BASE}/triggers/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
