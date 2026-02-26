import { apiFetch } from '@/shared/api/client'
import type { Session } from './types'

const BASE = '/api/sessions'

export async function createSession(data: Partial<Session>): Promise<Session> {
  return apiFetch<Session>(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function fetchSessions(): Promise<Session[]> {
  return apiFetch<Session[]>(BASE)
}

export async function fetchSession(id: string): Promise<Session> {
  return apiFetch<Session>(`${BASE}/${encodeURIComponent(id)}`)
}

export async function updateSession(id: string, data: Partial<Session>): Promise<Session> {
  return apiFetch<Session>(`${BASE}/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function deleteSession(id: string): Promise<void> {
  await apiFetch(`${BASE}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export async function activateSession(id: string): Promise<Session> {
  return apiFetch<Session>(`${BASE}/${encodeURIComponent(id)}/activate`, {
    method: 'POST',
  })
}

export async function deactivateSession(id: string): Promise<Session> {
  return apiFetch<Session>(`${BASE}/${encodeURIComponent(id)}/deactivate`, {
    method: 'POST',
  })
}
