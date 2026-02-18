import type { WorkflowDefinition } from './serializer'

const API_BASE = '/api'

export async function saveWorkflow(wf: WorkflowDefinition): Promise<void> {
  const res = await fetch(`${API_BASE}/workflows`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(wf),
  })
  if (!res.ok) {
    throw new Error(`Failed to save workflow: ${res.statusText}`)
  }
}

export async function loadWorkflow(name: string): Promise<WorkflowDefinition> {
  const res = await fetch(`${API_BASE}/workflows/${encodeURIComponent(name)}`)
  if (!res.ok) throw new Error('Workflow not found')
  return res.json()
}

export async function listWorkflows(): Promise<WorkflowDefinition[]> {
  const res = await fetch(`${API_BASE}/workflows`)
  if (!res.ok) throw new Error('Failed to list workflows')
  return res.json()
}

export type RunEvent = {
  type: string
  data: Record<string, unknown>
}

export async function runWorkflow(
  name: string,
  inputs: Record<string, string>,
  onEvent: (event: RunEvent) => void,
  onDone: (result: Record<string, unknown>) => void,
  onError: (error: Error) => void,
): Promise<void> {
  let res: Response
  try {
    res = await fetch(`${API_BASE}/workflows/${encodeURIComponent(name)}/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ inputs }),
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

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      // Parse SSE lines from buffer
      const lines = buffer.split('\n')
      // Keep the last potentially incomplete line in the buffer
      buffer = lines.pop() ?? ''

      let currentEventType = ''
      for (const line of lines) {
        if (line.startsWith('event: ')) {
          currentEventType = line.slice(7).trim()
        } else if (line.startsWith('data: ')) {
          const dataStr = line.slice(6)
          try {
            const data = JSON.parse(dataStr)
            const eventType = currentEventType || data.type || 'unknown'

            if (eventType === 'done') {
              onDone(data)
            } else {
              onEvent({ type: eventType, data })
            }
          } catch {
            // Non-JSON data line, emit as raw
            onEvent({ type: currentEventType || 'raw', data: { message: dataStr } })
          }
          currentEventType = ''
        }
        // Blank lines reset event type (SSE spec)
        if (line === '') {
          currentEventType = ''
        }
      }
    }
  } catch (err) {
    onError(err instanceof Error ? err : new Error(String(err)))
  }
}
