import { API_BASE } from './client'
import type { UploadResult } from './types'

export async function uploadFile(file: File): Promise<UploadResult> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch(`${API_BASE}/upload`, { method: 'POST', body: form })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`Upload failed: ${text || res.statusText}`)
  }
  return res.json()
}
