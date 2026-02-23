import { apiFetch } from '@/shared/api/client'
import type { SurgeAlert } from './types'

const BASE = '/api/surges'

export async function fetchSurges(): Promise<SurgeAlert[]> {
  return apiFetch<SurgeAlert[]>(BASE)
}

export async function dismissSurge(id: string): Promise<void> {
  return apiFetch<void>(`${BASE}/${encodeURIComponent(id)}/dismiss`, { method: 'POST' })
}

export async function createSessionFromSurge(
  surgeId: string,
): Promise<{ session_id: string }> {
  return apiFetch<{ session_id: string }>(`${BASE}/${encodeURIComponent(surgeId)}/create-session`, {
    method: 'POST',
  })
}
