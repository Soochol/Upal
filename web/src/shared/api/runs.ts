import { API_BASE, apiFetch } from './client'
import type { RunRecord, RunListResponse, RunEvent, ToolCall } from '@/shared/types'
import type { WorkflowDefinition } from '@/entities/workflow'

export async function fetchRuns(limit = 20, offset = 0, status = ''): Promise<RunListResponse> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  if (status) params.set('status', status)
  return apiFetch<RunListResponse>(`${API_BASE}/runs?${params}`)
}

export async function fetchWorkflowRuns(name: string, limit = 20, offset = 0): Promise<RunListResponse> {
  return apiFetch<RunListResponse>(`${API_BASE}/workflows/${encodeURIComponent(name)}/runs?limit=${limit}&offset=${offset}`)
}

export async function fetchRun(id: string): Promise<RunRecord> {
  return apiFetch<RunRecord>(`${API_BASE}/runs/${encodeURIComponent(id)}`)
}

// parseSSEPayload converts a backend SSE event (snake_case) into a typed RunEvent (camelCase).
function parseSSEPayload(eventType: string, data: Record<string, unknown>): RunEvent {
  const nodeId = data.node_id as string
  switch (eventType) {
    case 'node_started':
      return { type: 'node_started', nodeId, startedAt: data.started_at as number | undefined }
    case 'tool_call':
      return { type: 'tool_call', nodeId, calls: data.calls as ToolCall[] }
    case 'tool_result':
      return { type: 'tool_result', nodeId, results: data.results as ToolCall[] }
    case 'node_completed':
      return {
        type: 'node_completed',
        nodeId,
        output: data.output as string,
        stateDelta: (data.state_delta ?? {}) as Record<string, unknown>,
      }
    case 'node_skipped':
      return { type: 'node_skipped', nodeId }
    case 'node_waiting':
      return { type: 'node_waiting', nodeId }
    case 'node_resumed':
      return { type: 'node_resumed', nodeId }
    case 'log':
      return { type: 'log', nodeId, message: data.message as string }
    default:
      return { type: 'info', message: JSON.stringify(data) }
  }
}

// startRun kicks off a workflow execution in the background and returns the run ID.
export async function startRun(
  name: string,
  inputs: Record<string, string>,
  workflow?: WorkflowDefinition,
): Promise<{ run_id: string }> {
  const res = await fetch(`${API_BASE}/workflows/${encodeURIComponent(name)}/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ inputs, workflow }),
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(body || `Run failed: ${res.statusText}`)
  }
  return res.json()
}

// connectToRunEvents subscribes to a run's SSE event stream.
// Supports reconnection via lastSeq and cancellation via AbortSignal.
export async function connectToRunEvents(
  runId: string,
  onEvent: (event: RunEvent) => void,
  onDone: (result: Record<string, unknown>) => void,
  onError: (error: Error) => void,
  options?: { lastSeq?: number; signal?: AbortSignal },
): Promise<void> {
  const headers: Record<string, string> = {}
  if (options?.lastSeq !== undefined) {
    headers['Last-Event-ID'] = String(options.lastSeq)
  }

  let res: Response
  try {
    res = await fetch(`${API_BASE}/runs/${encodeURIComponent(runId)}/events`, {
      headers,
      signal: options?.signal,
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') return
    onError(err instanceof Error ? err : new Error(String(err)))
    return
  }

  if (!res.ok) {
    onError(new Error(`Event stream failed: ${res.statusText}`))
    return
  }

  const reader = res.body?.getReader()
  if (!reader) {
    onError(new Error('No response body'))
    return
  }

  // Cancel the reader when abort signal fires.
  options?.signal?.addEventListener('abort', () => reader.cancel(), { once: true })

  const decoder = new TextDecoder()
  let buffer = ''
  let currentEventType = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''

      for (const line of lines) {
        const trimmed = line.replace(/\r$/, '')
        if (trimmed.startsWith('id: ')) {
          // SSE id field — tracked automatically by the server seq numbers
          continue
        } else if (trimmed.startsWith('event: ')) {
          currentEventType = trimmed.slice(7).trim()
        } else if (trimmed.startsWith('data: ')) {
          const dataStr = trimmed.slice(6)
          try {
            const data = JSON.parse(dataStr)

            if (currentEventType === 'done') {
              onDone(data)
            } else {
              onEvent(parseSSEPayload(currentEventType, data))
            }
          } catch {
            // Non-JSON data line, skip
          }
          currentEventType = ''
        }
        if (trimmed === '') {
          currentEventType = ''
        }
      }
    }
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') return
    onError(err instanceof Error ? err : new Error(String(err)))
  }
}
