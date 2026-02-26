import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from 'react'

type Direction = 'horizontal' | 'vertical'

type UseResizeDragOptions = {
  direction: Direction
  /** min/max/initial as ratio (0–1) of window dimension */
  min: number
  max: number
  initial: number
}

function getWindowWidth() { return window.innerWidth }
function getWindowHeight() { return window.innerHeight }
function subscribeResize(cb: () => void) {
  window.addEventListener('resize', cb)
  return () => window.removeEventListener('resize', cb)
}

export function useResizeDrag({ direction, min, max, initial }: UseResizeDragOptions) {
  const windowSize = useSyncExternalStore(
    subscribeResize,
    direction === 'horizontal' ? getWindowWidth : getWindowHeight,
  )

  // Store ratio (0–1) so the panel scales with window size
  const [ratio, setRatio] = useState(initial)
  const isResizing = useRef(false)

  const size = Math.round(Math.min(max, Math.max(min, ratio)) * windowSize)

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    isResizing.current = true
    document.body.style.cursor = direction === 'horizontal' ? 'col-resize' : 'row-resize'
    document.body.style.userSelect = 'none'
  }, [direction])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing.current) return
      const total = direction === 'horizontal' ? window.innerWidth : window.innerHeight
      const px = direction === 'horizontal'
        ? window.innerWidth - e.clientX
        : window.innerHeight - e.clientY
      const newRatio = Math.min(max, Math.max(min, px / total))
      setRatio(newRatio)
    }
    const handleMouseUp = () => {
      if (!isResizing.current) return
      isResizing.current = false
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [direction, min, max])

  return { size, handleMouseDown }
}
