import { apiFetch } from '@/shared/api/client'
import type { PublishedContent, PublishedChannel } from './types'

const BASE = '/api/published'

export async function fetchPublishedContent(params?: {
  pipelineId?: string
  channel?: PublishedChannel
}): Promise<PublishedContent[]> {
  const qs = new URLSearchParams()
  if (params?.pipelineId) qs.set('pipeline_id', params.pipelineId)
  if (params?.channel) qs.set('channel', params.channel)
  const query = qs.toString() ? `?${qs}` : ''
  return apiFetch<PublishedContent[]>(`${BASE}${query}`)
}
