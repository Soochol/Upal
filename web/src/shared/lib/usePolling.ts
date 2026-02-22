import { useEffect, useRef, useState } from 'react'

export function usePolling<T>(
  fetchFn: () => Promise<T>,
  intervalMs: number,
  enabled: boolean,
): { data: T | null; isLoading: boolean } {
  const [data, setData] = useState<T | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const fetchRef = useRef(fetchFn)
  fetchRef.current = fetchFn

  useEffect(() => {
    if (!enabled) return

    let cancelled = false

    const poll = async () => {
      setIsLoading(true)
      try {
        const result = await fetchRef.current()
        if (!cancelled) setData(result)
      } catch {
        // silently ignore polling errors
      } finally {
        if (!cancelled) setIsLoading(false)
      }
    }

    poll()
    const id = setInterval(poll, intervalMs)

    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [intervalMs, enabled])

  return { data, isLoading }
}
