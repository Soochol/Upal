import { useWorkflowStore } from '@/entities/workflow'
import type { NodeData } from '@/entities/workflow'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Ungroup } from 'lucide-react'

const colorOptions = [
  { value: 'purple', label: 'Purple', class: 'bg-purple-400' },
  { value: 'blue', label: 'Blue', class: 'bg-blue-400' },
  { value: 'green', label: 'Green', class: 'bg-green-400' },
  { value: 'orange', label: 'Orange', class: 'bg-orange-400' },
  { value: 'pink', label: 'Pink', class: 'bg-pink-400' },
]

type GroupEditorProps = {
  groupId: string
  data: NodeData
  onClose: () => void
}

export function GroupEditor({ groupId, data, onClose }: GroupEditorProps) {
  const updateGroupLabel = useWorkflowStore((s) => s.updateGroupLabel)
  const updateGroupColor = useWorkflowStore((s) => s.updateGroupColor)
  const removeGroup = useWorkflowStore((s) => s.removeGroup)

  const label = data.label ?? 'Group'
  const color = (data.config.color as string) ?? 'purple'

  return (
    <div className="p-4 space-y-4">
      <div className="space-y-2">
        <Label htmlFor="group-label">Group Label</Label>
        <Input
          id="group-label"
          value={label}
          onChange={(e) => updateGroupLabel(groupId, e.target.value)}
        />
      </div>

      <div className="space-y-2">
        <Label>Color</Label>
        <div className="flex gap-2">
          {colorOptions.map((opt) => (
            <button
              key={opt.value}
              className={`w-7 h-7 rounded-full ${opt.class} transition-all ${
                color === opt.value ? 'ring-2 ring-ring ring-offset-2 ring-offset-background' : 'opacity-60 hover:opacity-100'
              }`}
              onClick={() => updateGroupColor(groupId, opt.value)}
              title={opt.label}
            />
          ))}
        </div>
      </div>

      <Separator />

      <Button
        variant="outline"
        size="sm"
        className="w-full"
        onClick={() => {
          removeGroup(groupId)
          onClose()
        }}
      >
        <Ungroup className="h-4 w-4 mr-2" />
        Ungroup Nodes
      </Button>
    </div>
  )
}
