import type { WorkflowDefinition } from './serializer'

const API_BASE = '/api'

// --- Model Discovery ---

export type ModelInfo = {
  id: string
  provider: string
  name: string
}

export async function listModels(): Promise<ModelInfo[]> {
  const res = await fetch(`${API_BASE}/models`)
  if (!res.ok) throw new Error('Failed to list models')
  return res.json()
}

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

export async function generateWorkflow(
  description: string,
  model?: string,
  existingWorkflow?: WorkflowDefinition,
): Promise<WorkflowDefinition> {
  const body: Record<string, unknown> = { description }
  if (model) body.model = model
  if (existingWorkflow) body.existing_workflow = existingWorkflow
  const res = await fetch(`${API_BASE}/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`Generation failed: ${text || res.statusText}`)
  }
  return res.json()
}

export type UploadResult = {
  id: string
  filename: string
  content_type: string
  size: number
}

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

// --- AI Node Configuration ---

export type ConfigureNodeRequest = {
  node_type: string
  node_id: string
  current_config: Record<string, unknown>
  label: string
  description: string
  message: string
  history?: { role: string; content: string }[]
  upstream_nodes: { id: string; type: string; label: string }[]
}

export type ConfigureNodeResponse = {
  config: Record<string, unknown>
  label?: string
  description?: string
  explanation: string
}

export async function configureNode(req: ConfigureNodeRequest): Promise<ConfigureNodeResponse> {
  const res = await fetch(`${API_BASE}/nodes/configure`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
  if (!res.ok) throw new Error(await res.text() || res.statusText)
  return res.json()
}

// --- Workflow Execution ---

export type RunEvent = {
  type: string        // 'agent' | 'info' | 'error' | 'done'
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

            if (currentEventType === 'done') {
              onDone(data)
            } else {
              // Derive event type from ADK Event content
              const type = data.author ? 'agent' : (currentEventType || 'event')
              onEvent({ type, data })
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
