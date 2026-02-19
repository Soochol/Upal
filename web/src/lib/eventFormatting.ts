import type { RunEvent } from '@/stores/executionStore'

export const eventColorMap: Record<string, string> = {
  agent: 'text-node-agent',
  event: 'text-muted-foreground',
  done: 'text-node-output font-semibold',
  error: 'text-destructive',
  info: 'text-muted-foreground',
}

export function formatEvent(event: RunEvent): string {
  const data = event.data
  // UI-generated events (info, error with message)
  if (data.message && typeof data.message === 'string') return data.message

  const parts: string[] = []

  // ADK Event: show author (node ID) — Go struct uses PascalCase
  if (data.Author) parts.push(`[${data.Author}]`)

  // Show text content from genai.Content
  const content = data.Content as { parts?: { text?: string }[] } | undefined
  if (content?.parts) {
    for (const part of content.parts) {
      if (part.text) parts.push(part.text)
    }
  }

  // Show state delta keys from actions
  const actions = data.Actions as { StateDelta?: Record<string, unknown> } | undefined
  if (actions?.StateDelta) {
    const keys = Object.keys(actions.StateDelta)
    if (keys.length > 0) parts.push(`state: {${keys.join(', ')}}`)
  }

  // Done/error event fields
  if (data.status) parts.push(`status=${data.status}`)
  if (data.error) parts.push(`error: ${data.error}`)

  if (parts.length === 0) return JSON.stringify(data)
  return parts.join(' ')
}
