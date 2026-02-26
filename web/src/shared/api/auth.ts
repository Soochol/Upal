import { apiFetch, API_BASE } from './client'

export interface User {
  id: string
  email: string
  name: string
  avatar_url: string
  oauth_provider: string
}

export function fetchMe(): Promise<User> {
  return apiFetch<User>(`${API_BASE}/auth/me`)
}

export function refreshToken(): Promise<{ token: string }> {
  return apiFetch<{ token: string }>(`${API_BASE}/auth/refresh`, { method: 'POST' })
}

export function logout(): Promise<void> {
  return apiFetch<void>(`${API_BASE}/auth/logout`, { method: 'POST' })
}
