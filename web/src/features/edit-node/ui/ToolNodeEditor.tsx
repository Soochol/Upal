import { useEffect, useState } from 'react'
import { Label } from '@/shared/ui/label'
import { listTools } from '@/shared/api'
import type { ToolInfo } from '@/shared/types'
import type { ToolNodeConfig } from '@/shared/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function ToolNodeEditor({ config, setConfig }: NodeEditorFieldProps<ToolNodeConfig>) {
  const [tools, setTools] = useState<ToolInfo[]>([])

  useEffect(() => {
    listTools().then(setTools).catch(() => {})
  }, [])

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label className="text-xs">Tool</Label>
        <select
          value={config.tool ?? ''}
          onChange={(e) => setConfig('tool', e.target.value || undefined)}
          className="w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs outline-none focus:ring-1 focus:ring-ring"
        >
          <option value="">Select a tool...</option>
          {tools.map((t) => (
            <option key={t.name} value={t.name}>{t.name}</option>
          ))}
        </select>
        {config.tool && tools.find(t => t.name === config.tool) && (
          <p className="text-[10px] text-muted-foreground mt-1">
            {tools.find(t => t.name === config.tool)!.description}
          </p>
        )}
      </div>
    </div>
  )
}
