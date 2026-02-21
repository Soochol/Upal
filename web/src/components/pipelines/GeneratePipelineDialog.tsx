import { useState } from 'react'
import { Sparkles, Loader2, X } from 'lucide-react'
import { generatePipelineBundle, createPipeline } from '@/lib/api'
import { saveWorkflow } from '@/lib/api'
import { ApiError } from '@/lib/api/client'
import type { Pipeline } from '@/lib/api/types'

type Props = {
  open: boolean
  onClose: () => void
  onCreated: (pipeline: Pipeline) => void
}

export function GeneratePipelineDialog({ open, onClose, onCreated }: Props) {
  const [description, setDescription] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!open) return null

  const handleSubmit = async () => {
    if (!description.trim()) return
    setLoading(true)
    setError(null)
    try {
      const bundle = await generatePipelineBundle(description.trim())

      await Promise.all(
        bundle.workflows.map((wf) =>
          saveWorkflow(wf).catch((e) => {
            if (e instanceof ApiError && e.status === 409) return
            throw e
          }),
        ),
      )

      const pipeline = await createPipeline(bundle.pipeline)
      onCreated(pipeline)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : '알 수 없는 오류가 발생했습니다')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="relative w-full max-w-lg bg-background border border-border rounded-xl shadow-xl p-6">
        {/* Close button */}
        <button
          onClick={onClose}
          disabled={loading}
          className="absolute top-4 right-4 p-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors disabled:opacity-50"
        >
          <X className="h-4 w-4" />
        </button>

        {/* Header */}
        <div className="flex items-center gap-2 mb-4">
          <Sparkles className="h-5 w-5 text-primary" />
          <h2 className="text-base font-semibold">AI로 파이프라인 생성</h2>
        </div>

        {/* Description textarea */}
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={loading}
          rows={4}
          placeholder="예: 고객 문의를 받아 AI로 분류하고 답변을 검토 후 발송하는 파이프라인"
          className="w-full text-sm bg-muted/40 border border-border rounded-lg px-3 py-2.5 resize-none placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50 disabled:cursor-not-allowed"
        />

        {/* Error message */}
        {error && (
          <p className="mt-2 text-xs text-destructive">{error}</p>
        )}

        {/* Actions */}
        <div className="flex justify-end mt-4">
          <button
            onClick={handleSubmit}
            disabled={loading || !description.trim()}
            className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                생성 중...
              </>
            ) : (
              <>
                <Sparkles className="h-4 w-4" />
                생성하기
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
