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

function truncate(text: string, maxLen: number): string {
  return text.length > maxLen ? text.slice(0, maxLen) + '...' : text
}

/**
 * Format a run event as a log line.
 * When `verbose` is true, outputs full content without truncation.
 */
export function formatEvent(event: RunEvent, verbose = false): string {
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
          return `[${event.nodeId}] ${r.name} \u2192 ${verbose ? resp : truncate(resp, 200)}`
        })
        .join('\n')
    case 'node_completed': {
      let output = event.output
      if (!verbose) {
        output = output.replace(
          /data:(image\/[\w+-]+);base64,[A-Za-z0-9+/=]+/g,
          (_match, mimeType: string) => {
            const sizeKB = Math.round((_match.length * 3) / 4 / 1024)
            return `[image: ${mimeType}, ~${sizeKB}KB]`
          },
        )
        output = truncate(output, 300)
      }
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
