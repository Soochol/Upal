import { API_BASE, apiFetch } from './client'
import type { SchedulerStats } from './types'

export async function fetchSchedulerStats(): Promise<SchedulerStats> {
  return apiFetch<SchedulerStats>(`${API_BASE}/scheduler/stats`)
}
