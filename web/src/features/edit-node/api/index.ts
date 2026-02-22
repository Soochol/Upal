import { API_BASE, apiFetch } from '@/shared/api/client'
import type { ConfigureNodeRequest, ConfigureNodeResponse } from '@/shared/types'

export async function configureNode(req: ConfigureNodeRequest): Promise<ConfigureNodeResponse> {
  return apiFetch<ConfigureNodeResponse>(`${API_BASE}/nodes/configure`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}
