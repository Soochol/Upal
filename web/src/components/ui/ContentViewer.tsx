// web/src/components/ui/ContentViewer.tsx
import { useState } from 'react'
import { Code } from 'lucide-react'

type ContentKind = 'html' | 'json' | 'text'

function detectKind(value: string): ContentKind {
  const t = value.trim()
  if (/<[a-zA-Z][^>]*>/.test(t)) return 'html'
  try { JSON.parse(t); return 'json' } catch { /* */ }
  return 'text'
}

export function ContentViewer({ value }: { value: unknown }) {
  const [showSource, setShowSource] = useState(false)

  if (value === null || value === undefined) return null

  // Non-string → JSON tree
  if (typeof value !== 'string') {
    return (
      <pre className="text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-80 overflow-auto whitespace-pre-wrap break-all">
        {JSON.stringify(value, null, 2)}
      </pre>
    )
  }

  if (!value.trim()) return null

  const kind = detectKind(value)

  if (kind === 'html') {
    return (
      <div>
        {/* Toggle bar */}
        <div className="flex items-center gap-1 mb-2">
          {(['render', 'source'] as const).map((mode) => (
            <button
              key={mode}
              onClick={() => setShowSource(mode === 'source')}
              className={[
                'flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-md transition-colors cursor-pointer',
                (mode === 'source') === showSource
                  ? 'bg-foreground text-background'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted',
              ].join(' ')}
            >
              {mode === 'source' && <Code className="h-2.5 w-2.5" />}
              {mode === 'render' ? 'Render' : 'Source'}
            </button>
          ))}
        </div>

        {showSource ? (
          <pre className="text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-96 overflow-auto whitespace-pre-wrap break-all">
            {value}
          </pre>
        ) : (
          <iframe
            srcDoc={value}
            sandbox="allow-same-origin"
            className="w-full rounded-lg border border-border bg-white dark:bg-card"
            style={{ height: '420px' }}
            title="Rendered output"
          />
        )}
      </div>
    )
  }

  if (kind === 'json') {
    let pretty = value
    try { pretty = JSON.stringify(JSON.parse(value), null, 2) } catch { /* */ }
    return (
      <pre className="text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-80 overflow-auto whitespace-pre-wrap break-all">
        {pretty}
      </pre>
    )
  }

  return (
    <pre className="text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-80 overflow-auto whitespace-pre-wrap break-words">
      {value}
    </pre>
  )
}
