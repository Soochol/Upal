import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { IteratorNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function IteratorNodeEditor({ config, setConfig }: NodeEditorFieldProps<IteratorNodeConfig>) {
  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label htmlFor="iter-source" className="text-xs">Source</Label>
        <Input
          id="iter-source"
          className="h-7 text-xs font-mono"
          value={config.source ?? ''}
          placeholder='e.g. {{upstream_node}} or ["a","b","c"]'
          onChange={(e) => setConfig('source', e.target.value)}
        />
        <p className="text-[10px] text-muted-foreground">
          Template that resolves to a JSON array or newline-separated items.
        </p>
      </div>

      <div className="space-y-1">
        <Label htmlFor="iter-key" className="text-xs">Item Key</Label>
        <Input
          id="iter-key"
          className="h-7 text-xs"
          value={config.item_key ?? ''}
          placeholder="Default: <nodeId>_item"
          onChange={(e) => setConfig('item_key', e.target.value)}
        />
        <p className="text-[10px] text-muted-foreground">
          Session state key for the current item during iteration.
        </p>
      </div>

      <div className="space-y-1">
        <Label htmlFor="iter-max" className="text-xs">Max Iterations</Label>
        <Input
          id="iter-max"
          className="h-7 text-xs"
          type="number"
          value={config.max_iterations ?? 100}
          onChange={(e) => setConfig('max_iterations', Number(e.target.value))}
        />
      </div>
    </div>
  )
}
