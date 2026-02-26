import { apiFetch } from '@/shared/api/client'
import type { AIProvider, AIProviderCreate } from './types'

const BASE = '/api/ai-providers'

export async function listAIProviders(): Promise<AIProvider[]> {
  return apiFetch<AIProvider[]>(BASE)
}

export async function createAIProvider(data: AIProviderCreate): Promise<AIProvider> {
  return apiFetch<AIProvider>(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export async function deleteAIProvider(id: string): Promise<void> {
  await apiFetch<void>(`${BASE}/${id}`, { method: 'DELETE' })
}

export async function setAIProviderDefault(id: string): Promise<AIProvider[]> {
  return apiFetch<AIProvider[]>(`${BASE}/${id}/default`, { method: 'PUT' })
}
