import { useRef, useState } from 'react'
import { ExternalLink, Upload } from 'lucide-react'
import { Label } from '@/shared/ui/label'
import { Button } from '@/shared/ui/button'
import { uploadFile } from '@/shared/api'
import type { AssetNodeConfig } from '@/shared/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'
import { fieldBox } from './NodeEditor'

export function AssetNodeEditor({ config, setConfig }: NodeEditorFieldProps<AssetNodeConfig>) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const result = await uploadFile(file)
      setConfig('file_id', result.id)
      setConfig('filename', result.filename)
      setConfig('content_type', result.content_type)
      setConfig('preview_text', result.preview_text ?? '')
    } catch (err) {
      console.error('AssetNodeEditor: upload failed', err)
    } finally {
      setUploading(false)
      e.target.value = ''
    }
  }

  return (
    <div className="space-y-3">
      {/* Upload button */}
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        onChange={handleFileChange}
      />
      <Button
        variant="outline"
        size="sm"
        className="w-full gap-2 text-xs"
        disabled={uploading}
        onClick={() => fileInputRef.current?.click()}
      >
        <Upload size={13} />
        {uploading ? 'Uploading…' : config.file_id ? 'Replace File' : 'Load File'}
      </Button>

      {/* Filename + open button */}
      {config.file_id && (
        <div className="space-y-1">
          <Label className="text-xs">File</Label>
          <div className="flex items-center gap-2">
            <p className="text-xs text-foreground font-medium truncate flex-1">
              {config.filename ?? config.file_id}
            </p>
            <a
              href={`/api/files/${config.file_id}/serve`}
              target="_blank"
              rel="noopener noreferrer"
              className="shrink-0 p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              title="Open file"
            >
              <ExternalLink size={13} />
            </a>
          </div>
        </div>
      )}

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
      {config.preview_text && (
        <div className="space-y-1">
          <Label className="text-xs">Preview</Label>
          <pre className={fieldBox + ' font-mono text-[10px] text-muted-foreground'}>
            {config.preview_text}
          </pre>
        </div>
      )}
    </div>
  )
}
