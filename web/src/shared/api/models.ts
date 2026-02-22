import { API_BASE, apiFetch } from './client'
import type { ModelInfo } from '../types'

export async function listModels(): Promise<ModelInfo[]> {
  return apiFetch<ModelInfo[]>(`${API_BASE}/models`)
}
