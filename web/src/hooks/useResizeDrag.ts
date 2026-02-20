import { useCallback, useEffect, useRef, useState } from 'react'

type Direction = 'horizontal' | 'vertical'

type UseResizeDragOptions = {
  direction: Direction
  min: number
  max: number
  initial: number
}

export function useResizeDrag({ direction, min, max, initial }: UseResizeDragOptions) {
  const [size, setSize] = useState(initial)
  const isResizing = useRef(false)

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    isResizing.current = true
    document.body.style.cursor = direction === 'horizontal' ? 'col-resize' : 'row-resize'
    document.body.style.userSelect = 'none'
  }, [direction])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing.current) return
      const newSize =
        direction === 'horizontal'
          ? window.innerWidth - e.clientX
          : window.innerHeight - e.clientY
      setSize(Math.min(max, Math.max(min, newSize)))
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
