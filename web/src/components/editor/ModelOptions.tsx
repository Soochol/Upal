import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Slider } from '@/components/ui/slider'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { OptionSchema } from '@/lib/api'

type ModelOptionsProps = {
  options: OptionSchema[]
  values: Record<string, unknown>
  onChange: (key: string, value: unknown) => void
}

export function ModelOptions({ options, values, onChange }: ModelOptionsProps) {
  if (!options || options.length === 0) return null

  return (
    <div className="space-y-2.5">
      {options.map((opt) => {
        const value = values[opt.key] ?? opt.default
        switch (opt.type) {
          case 'slider':
            return (
              <div key={opt.key} className="space-y-1.5">
                <div className="flex items-center justify-between">
                  <Label className="text-xs">{opt.label}</Label>
                  <span className="text-[10px] text-muted-foreground tabular-nums">
                    {value != null ? String(value) : 'default'}
                  </span>
                </div>
                <Slider
                  min={opt.min ?? 0}
                  max={opt.max ?? 1}
                  step={opt.step ?? 0.1}
                  value={value != null ? [Number(value)] : undefined}
                  defaultValue={[opt.default != null ? Number(opt.default) : (opt.min ?? 0)]}
                  onValueChange={([v]) => onChange(opt.key, v)}
                />
              </div>
            )
          case 'number':
            return (
              <div key={opt.key} className="space-y-1">
                <Label className="text-xs">{opt.label}</Label>
                <Input
                  type="number"
                  className="h-7 text-xs"
                  min={opt.min}
                  max={opt.max}
                  step={opt.step}
                  value={value != null ? String(value) : ''}
                  placeholder={opt.default != null ? `${opt.default}` : 'default'}
                  onChange={(e) => {
                    const v = e.target.value === '' ? undefined : Number(e.target.value)
                    onChange(opt.key, v)
                  }}
                />
              </div>
            )
          case 'select':
            return (
              <div key={opt.key} className="space-y-1">
                <Label className="text-xs">{opt.label}</Label>
                <Select
                  value={value != null ? String(value) : undefined}
                  onValueChange={(v) => onChange(opt.key, v)}
                >
                  <SelectTrigger className="h-7 text-xs w-full" size="sm">
                    <SelectValue placeholder="default" />
                  </SelectTrigger>
                  <SelectContent>
                    {opt.choices?.map((c) => (
                      <SelectItem key={String(c.value)} value={String(c.value)} className="text-xs">
                        {c.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )
          default:
            return null
        }
      })}
    </div>
  )
}
