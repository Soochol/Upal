import { API_BASE, apiFetch } from './client'
import type { ToolInfo } from './types'

export async function listTools(): Promise<ToolInfo[]> {
  return apiFetch<ToolInfo[]>(`${API_BASE}/tools`)
}
