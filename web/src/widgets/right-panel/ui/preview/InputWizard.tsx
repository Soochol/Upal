import type { Node } from '@xyflow/react'
import type { NodeData } from '@/entities/workflow'
import { Button } from '@/shared/ui/button'
import { Textarea } from '@/shared/ui/textarea'
import { Label } from '@/shared/ui/label'
import { Play, ArrowLeft, ChevronRight } from 'lucide-react'

type InputWizardProps = {
  sortedInputs: Node<NodeData>[]
  stepIndex: number
  inputs: Record<string, string>
  onInputChange: (nodeId: string, value: string) => void
  onNext: () => void
  onBack: () => void
}

export function InputWizard({
  sortedInputs,
  stepIndex,
  inputs,
  onInputChange,
  onNext,
  onBack,
}: InputWizardProps) {
  const currentInput = sortedInputs[stepIndex]
  if (!currentInput) return null

  return (
    <div className="p-3 space-y-2.5 shrink-0">
      <div className="flex items-center gap-1.5 mb-1">
        <button
          onClick={onBack}
          className="text-muted-foreground hover:text-foreground transition-colors"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
        </button>
        <p className="text-xs font-medium">
          Input {stepIndex + 1} of {sortedInputs.length}
        </p>
      </div>

      <div key={currentInput.id} className="input-step-enter space-y-1.5">
        <Label htmlFor={`preview-${currentInput.id}`} className="text-xs font-medium">
          {currentInput.data.label}
        </Label>
        {currentInput.data.description && (
          <p className="text-[10px] text-muted-foreground leading-relaxed">
            {currentInput.data.description}
          </p>
        )}
        <Textarea
          id={`preview-${currentInput.id}`}
          value={inputs[currentInput.id] ?? ''}
          onChange={(e) => onInputChange(currentInput.id, e.target.value)}
          placeholder={
            (currentInput.data.config.placeholder as string) ||
            `Enter value for ${currentInput.data.label}...`
          }
          className="min-h-[80px] resize-y text-xs"
          autoFocus
        />
      </div>

      {/* Step dots */}
      {sortedInputs.length > 1 && (
        <div className="flex justify-center gap-1.5 pt-0.5">
          {sortedInputs.map((_, i) => (
            <div
              key={i}
              className={`h-1.5 rounded-full transition-all duration-200 ${
                i < stepIndex
                  ? 'w-1.5 bg-node-input'
                  : i === stepIndex
                    ? 'w-3 bg-foreground'
                    : 'w-1.5 bg-border'
              }`}
            />
          ))}
        </div>
      )}

      <Button size="sm" className="w-full" onClick={onNext}>
        {stepIndex < sortedInputs.length - 1 ? (
          <>
            Next
            <ChevronRight className="h-3.5 w-3.5 ml-1.5" />
          </>
        ) : (
          <>
            <Play className="h-3.5 w-3.5 mr-1.5" />
            Run
          </>
        )}
      </Button>
    </div>
  )
}
