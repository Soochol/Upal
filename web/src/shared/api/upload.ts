import { API_BASE } from './client'
import type { UploadResult } from '@/shared/types'

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

export async function deleteFile(fileId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/files/${fileId}`, { method: 'DELETE' })
  if (!res.ok && res.status !== 404) throw new Error(`deleteFile: ${res.status}`)
}
