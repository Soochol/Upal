// web/src/lib/api/connections.ts
import { apiFetch } from './client'
import type { Connection, ConnectionCreate } from './types'

export async function listConnections(): Promise<Connection[]> {
  return apiFetch<Connection[]>('/api/connections')
}

export async function createConnection(conn: ConnectionCreate): Promise<Connection> {
  return apiFetch<Connection>('/api/connections', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(conn),
  })
}

export async function deleteConnection(id: string): Promise<void> {
  await apiFetch<void>(`/api/connections/${id}`, { method: 'DELETE' })
}
