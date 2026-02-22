// web/src/widgets/right-panel/ui/console/NodeOutputViewer.tsx
import { useState } from 'react'
import { Code } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { detectOutputKind } from './detectOutputKind'

type Props = { output: string }

export function NodeOutputViewer({ output }: Props) {
  const [showSource, setShowSource] = useState(false)
  const kind = detectOutputKind(output)

  if (kind === 'image') {
    return (
      <div className="p-2">
        <img
          src={output.trim()}
          alt="node output"
          className="max-w-full rounded-md border border-border"
          onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
        />
      </div>
    )
  }

  if (kind === 'audio') {
    return (
      <div className="p-2">
        <audio controls src={output.trim()} className="w-full h-8" />
      </div>
    )
  }

  if (kind === 'html') {
    return (
      <div className="p-2">
        <div className="flex items-center gap-1 mb-1.5">
          {(['render', 'source'] as const).map((mode) => (
            <button
              key={mode}
              onClick={() => setShowSource(mode === 'source')}
              className={cn(
                'flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-md transition-colors cursor-pointer',
                (mode === 'source') === showSource
                  ? 'bg-foreground text-background'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted',
              )}
            >
              {mode === 'source' && <Code className="h-2.5 w-2.5" />}
              {mode === 'render' ? 'Render' : 'Source'}
            </button>
          ))}
        </div>
        {showSource ? (
          <pre className="text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-64 overflow-auto whitespace-pre-wrap break-all">
            {output}
          </pre>
        ) : (
          <iframe
            srcDoc={output}
            sandbox="allow-same-origin"
            className="w-full rounded-lg border border-border bg-white dark:bg-card"
            style={{ height: '280px' }}
            title="Rendered output"
          />
        )}
      </div>
    )
  }

  if (kind === 'json') {
    let pretty = output
    try { pretty = JSON.stringify(JSON.parse(output), null, 2) } catch { /* keep raw */ }
    return (
      <pre className="m-2 text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-64 overflow-auto whitespace-pre-wrap break-all">
        {pretty}
      </pre>
    )
  }

  // text
  return (
    <pre className="m-2 text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-64 overflow-auto whitespace-pre-wrap break-words">
      {output}
    </pre>
  )
}
