import { API_BASE, apiFetch } from './client'
import type { RunRecord, RunListResponse, RunEvent, ToolCall } from './types'
import type { WorkflowDefinition } from '../serializer'

export async function fetchRuns(limit = 20, offset = 0): Promise<RunListResponse> {
  return apiFetch<RunListResponse>(`${API_BASE}/runs?limit=${limit}&offset=${offset}`)
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
      return { type: 'node_started', nodeId }
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
    case 'log':
      return { type: 'log', nodeId, message: data.message as string }
    default:
      return { type: 'info', message: JSON.stringify(data) }
  }
}

export async function runWorkflow(
  name: string,
  inputs: Record<string, string>,
  onEvent: (event: RunEvent) => void,
  onDone: (result: Record<string, unknown>) => void,
  onError: (error: Error) => void,
  workflow?: WorkflowDefinition,
): Promise<void> {
  let res: Response
  try {
    res = await fetch(`${API_BASE}/workflows/${encodeURIComponent(name)}/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ inputs, workflow }),
    })
  } catch (err) {
    onError(err instanceof Error ? err : new Error(String(err)))
    return
  }

  if (!res.ok) {
    onError(new Error(`Run failed: ${res.statusText}`))
    return
  }

  const reader = res.body?.getReader()
  if (!reader) {
    onError(new Error('No response body'))
    return
  }

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
        if (trimmed.startsWith('event: ')) {
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
    onError(err instanceof Error ? err : new Error(String(err)))
  }
}
