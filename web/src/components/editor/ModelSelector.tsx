import { useEffect, useState } from 'react'
import {
  Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import { listModels, type ModelInfo } from '@/lib/api'
import { groupModelsByProvider } from '@/lib/utils'

type ModelSelectorProps = {
  value: string
  onChange: (value: string) => void
  placeholder?: string
}

export function ModelSelector({ value, onChange, placeholder = 'Select a model...' }: ModelSelectorProps) {
  const [models, setModels] = useState<ModelInfo[]>([])
  useEffect(() => {
    listModels().then(setModels).catch(() => setModels([]))
  }, [])
  const modelsByProvider = groupModelsByProvider(models)
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className="h-7 text-xs w-full" size="sm">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {Object.entries(modelsByProvider).map(([provider, providerModels]) => (
          <SelectGroup key={provider}>
            <SelectLabel>{provider}</SelectLabel>
            {providerModels.map((m) => (
              <SelectItem key={m.id} value={m.id} className="text-xs">{m.name}</SelectItem>
            ))}
          </SelectGroup>
        ))}
        {models.length === 0 && (
          <div className="px-2 py-4 text-xs text-muted-foreground text-center">
            No models available.<br />Configure providers in config.yaml
          </div>
        )}
      </SelectContent>
    </Select>
  )
}
