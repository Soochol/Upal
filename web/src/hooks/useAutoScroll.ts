import { useEffect, useRef } from 'react'

/**
 * Returns a ref to attach to a sentinel element (e.g. an empty div at the bottom).
 * Scrolls the sentinel into view whenever `deps` changes.
 */
export function useAutoScroll<T extends HTMLElement = HTMLDivElement>(
  deps: unknown,
) {
  const ref = useRef<T>(null)

  useEffect(() => {
    ref.current?.scrollIntoView({ behavior: 'smooth' })
  }, [deps])

  return ref
}
