import { API_BASE, apiFetch } from '@/shared/api/client'
import type { ConfigurePipelineRequest, ConfigurePipelineResponse } from '@/shared/types'

export async function configurePipeline(
  pipelineId: string,
  req: ConfigurePipelineRequest,
): Promise<ConfigurePipelineResponse> {
  return apiFetch<ConfigurePipelineResponse>(
    `${API_BASE}/pipelines/${encodeURIComponent(pipelineId)}/configure`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    },
  )
}
