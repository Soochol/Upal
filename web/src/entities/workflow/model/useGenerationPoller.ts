import { useEffect, useRef } from 'react'
import { useGenerationStore } from './generationStore'
import { pollGeneration } from '../api/index'
import { ApiError } from '@/shared/api/client'

const POLL_INTERVAL = 1500
const MAX_CONSECUTIVE_ERRORS = 20

export function useGenerationPoller<T = unknown>(
  onComplete: (result: T) => void,
  onError: (msg: string) => void,
) {
  const generationId = useGenerationStore((s) => s.generationId)
  const isGenerating = useGenerationStore((s) => s.isGenerating)
  const clear = useGenerationStore((s) => s.clear)
  const fail = useGenerationStore((s) => s.fail)
  const onCompleteRef = useRef(onComplete)
  const onErrorRef = useRef(onError)
  onCompleteRef.current = onComplete
  onErrorRef.current = onError

  useEffect(() => {
    if (!generationId || !isGenerating) return

    let timer: ReturnType<typeof setInterval> | null = null
    let cancelled = false
    let consecutiveErrors = 0

    const poll = async () => {
      try {
        const res = await pollGeneration<T>(generationId)
        if (cancelled) return
        consecutiveErrors = 0
        if (res.status === 'completed' && res.result !== undefined) {
          clear()
          onCompleteRef.current(res.result)
        } else if (res.status === 'failed') {
          clear()
          onErrorRef.current(res.error ?? 'Generation failed')
        }
      } catch (err) {
        if (cancelled) return
        if (err instanceof ApiError && err.status === 404) {
          fail('Generation not found — it may have expired')
          onErrorRef.current('Generation not found — it may have expired')
          return
        }
        consecutiveErrors++
        if (consecutiveErrors >= MAX_CONSECUTIVE_ERRORS) {
          fail('Lost connection to server')
          onErrorRef.current('Lost connection to server')
        }
      }
    }

    poll()
    timer = setInterval(poll, POLL_INTERVAL)

    return () => {
      cancelled = true
      if (timer) clearInterval(timer)
    }
  }, [generationId, isGenerating, clear, fail])
}
