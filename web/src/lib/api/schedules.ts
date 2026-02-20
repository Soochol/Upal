import { API_BASE, apiFetch } from './client'
import type { Schedule } from './types'

export async function fetchSchedules(): Promise<Schedule[]> {
  return apiFetch<Schedule[]>(`${API_BASE}/schedules`)
}

export async function createSchedule(data: Partial<Schedule>): Promise<Schedule> {
  return apiFetch<Schedule>(`${API_BASE}/schedules`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function updateSchedule(id: string, data: Partial<Schedule>): Promise<Schedule> {
  return apiFetch<Schedule>(`${API_BASE}/schedules/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function deleteSchedule(id: string): Promise<void> {
  return apiFetch<void>(`${API_BASE}/schedules/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export async function pauseSchedule(id: string): Promise<void> {
  return apiFetch<void>(`${API_BASE}/schedules/${encodeURIComponent(id)}/pause`, { method: 'POST' })
}

export async function resumeSchedule(id: string): Promise<void> {
  return apiFetch<void>(`${API_BASE}/schedules/${encodeURIComponent(id)}/resume`, { method: 'POST' })
}
