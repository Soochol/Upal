import type { RunInputNodeConfig } from '@/shared/lib/nodeConfigs'
import type { NodeEditorFieldProps } from './NodeEditor'

export function RunInputNodeEditor({ }: NodeEditorFieldProps<RunInputNodeConfig>) {
  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">
        This node receives data from pipeline runs automatically.
        When the workflow is triggered by a pipeline, the run brief is injected here.
      </p>
    </div>
  )
}
