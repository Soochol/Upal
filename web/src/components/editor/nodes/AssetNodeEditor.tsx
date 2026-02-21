import { Label } from '@/components/ui/label'
import type { AssetNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'
import { fieldBox } from './NodeEditor'

export function AssetNodeEditor({ config }: NodeEditorFieldProps<AssetNodeConfig>) {
  return (
    <div className="space-y-3">
      {/* Filename */}
      <div className="space-y-1">
        <Label className="text-xs">File</Label>
        <p className="text-xs text-foreground font-medium truncate">
          {config.filename ?? 'No file'}
        </p>
      </div>

      {/* Content type */}
      {config.content_type && (
        <div className="space-y-1">
          <Label className="text-xs">Type</Label>
          <span className="inline-block text-[10px] font-mono px-1.5 py-0.5 rounded bg-muted text-muted-foreground border border-border">
            {config.content_type}
          </span>
        </div>
      )}

      {/* Preview text */}
      <div className="space-y-1">
        <Label className="text-xs">Preview</Label>
        {config.preview_text ? (
          <pre className={fieldBox + ' font-mono text-[10px] text-muted-foreground'}>
            {config.preview_text}
          </pre>
        ) : (
          <p className="text-xs text-muted-foreground italic">No preview available.</p>
        )}
      </div>

      <p className="text-[10px] text-muted-foreground">
        Asset nodes are read-only. Upload a new file to replace.
      </p>
    </div>
  )
}
