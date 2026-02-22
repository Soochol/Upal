import type { RunEvent } from '../types'

export const eventColorMap: Record<string, string> = {
  node_started:   'text-muted-foreground',
  tool_call:      'text-amber-500 dark:text-amber-400',
  tool_result:    'text-muted-foreground italic',
  node_completed: 'text-node-agent',
  node_skipped:   'text-muted-foreground/60 italic',
  node_waiting:   'text-amber-500 dark:text-amber-400',
  node_resumed:   'text-muted-foreground',
  done:           'text-node-output font-semibold',
  error:          'text-destructive',
  info:           'text-muted-foreground',
  log:            'text-muted-foreground/60 text-[11px]',
}

export function formatEvent(event: RunEvent): string {
  switch (event.type) {
    case 'node_started':
      return `[${event.nodeId}] started`
    case 'tool_call':
      return event.calls
        .map((c) => `[${event.nodeId}] ${c.name}(${JSON.stringify(c.args ?? {})})`)
        .join('\n')
    case 'tool_result':
      return event.results
        .map((r) => {
          const resp = JSON.stringify(r.response ?? {})
          const truncated = resp.length > 200 ? resp.slice(0, 200) + '...' : resp
          return `[${event.nodeId}] ${r.name} \u2192 ${truncated}`
        })
        .join('\n')
    case 'node_completed': {
      // Replace data URIs with compact summaries for log readability
      let output = event.output.replace(
        /data:(image\/[\w+-]+);base64,[A-Za-z0-9+/=]+/g,
        (_match, mimeType: string) => {
          const sizeKB = Math.round((_match.length * 3) / 4 / 1024)
          return `[image: ${mimeType}, ~${sizeKB}KB]`
        },
      )
      output = output.length > 300 ? output.slice(0, 300) + '...' : output
      const deltaKeys = Object.keys(event.stateDelta ?? {})
      const suffix = deltaKeys.length > 0 ? ` state: {${deltaKeys.join(', ')}}` : ''
      return `[${event.nodeId}] ${output}${suffix}`
    }
    case 'done':
      return event.error ? `status=${event.status}: ${event.error}` : `status=${event.status}`
    case 'error':
      return event.message
    case 'info':
      return event.message
    case 'node_skipped':
      return `[${event.nodeId}] skipped`
    case 'node_waiting':
      return `[${event.nodeId}] waiting`
    case 'node_resumed':
      return `[${event.nodeId}] resumed`
    case 'log':
      return `[${event.nodeId}] ${event.message}`
  }
}

/** Returns "+1.2s" offset from run start, or '' if timestamps are unavailable. */
export function formatRelativeTime(eventMs: number | undefined, runStartMs: number | null): string {
  if (!runStartMs || !eventMs) return ''
  const delta = (eventMs - runStartMs) / 1000
  if (delta < 0) return ''
  return `+${delta.toFixed(1)}s`
}

/** Same as formatEvent but without any character truncation.
 *  Use for "verbose" log level where the user wants to see the full content. */
export function formatEventVerbose(event: RunEvent): string {
  switch (event.type) {
    case 'node_started':
      return `[${event.nodeId}] started`
    case 'tool_call':
      return event.calls
        .map((c) => `[${event.nodeId}] ${c.name}(${JSON.stringify(c.args ?? {})})`)
        .join('\n')
    case 'tool_result':
      return event.results
        .map((r) => `[${event.nodeId}] ${r.name} \u2192 ${JSON.stringify(r.response ?? {})}`)
        .join('\n')
    case 'node_completed': {
      const deltaKeys = Object.keys(event.stateDelta ?? {})
      const suffix = deltaKeys.length > 0 ? ` state: {${deltaKeys.join(', ')}}` : ''
      return `[${event.nodeId}] ${event.output}${suffix}`
    }
    case 'done':
      return event.error ? `status=${event.status}: ${event.error}` : `status=${event.status}`
    case 'error':
      return event.message
    case 'info':
      return event.message
    case 'node_skipped':
      return `[${event.nodeId}] skipped`
    case 'node_waiting':
      return `[${event.nodeId}] waiting`
    case 'node_resumed':
      return `[${event.nodeId}] resumed`
    case 'log':
      return `[${event.nodeId}] ${event.message}`
  }
}
