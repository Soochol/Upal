import { apiFetch } from '@/shared/api/client'
import type { PublishChannel } from './types'

const BASE = '/api/publish-channels'

export async function fetchPublishChannels(): Promise<PublishChannel[]> {
  return apiFetch<PublishChannel[]>(BASE)
}

export async function createPublishChannel(ch: Omit<PublishChannel, 'id'>): Promise<PublishChannel> {
  return apiFetch<PublishChannel>(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(ch),
  })
}

export async function updatePublishChannel(id: string, ch: Partial<PublishChannel>): Promise<PublishChannel> {
  return apiFetch<PublishChannel>(`${BASE}/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(ch),
  })
}

export async function deletePublishChannel(id: string): Promise<void> {
  await apiFetch(`${BASE}/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
