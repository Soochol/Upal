import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { SubWorkflowNodeConfig } from '@/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function SubWorkflowNodeEditor({ config, setConfig }: NodeEditorFieldProps<SubWorkflowNodeConfig>) {
  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label htmlFor="subwf-name" className="text-xs">Workflow Name</Label>
        <Input
          id="subwf-name"
          className="h-7 text-xs"
          value={config.workflow_name ?? ''}
          placeholder="e.g. my-child-workflow"
          onChange={(e) => setConfig('workflow_name', e.target.value)}
        />
        <p className="text-[10px] text-muted-foreground">
          Name of the saved workflow to execute as a child.
        </p>
      </div>

      <div className="space-y-1">
        <Label className="text-xs">Input Mapping</Label>
        <p className="text-[10px] text-muted-foreground">
          Map parent state values to child input nodes using {'{{node_id}}'} templates.
          Configure via the JSON config or AI Chat.
        </p>
      </div>
    </div>
  )
}
