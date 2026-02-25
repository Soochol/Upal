import { useState, useEffect, useRef, useCallback } from 'react'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type SaveStatus = 'idle' | 'waiting' | 'saving' | 'saved' | 'error'

export type UseAutoSaveOptions<T> = {
  /** Current data to watch for changes */
  data: T
  /** Async save function called with the latest data */
  onSave: (data: T) => Promise<void>
  /** Debounce delay in ms (default: 2000) */
  delay?: number
  /** Custom equality check; defaults to JSON.stringify comparison */
  isEqual?: (a: T, b: T) => boolean
  /** Whether auto-save is active (default: true) */
  enabled?: boolean
  /** Save on component unmount if dirty (default: false) */
  saveOnUnmount?: boolean
  /**
   * Fire-and-forget save for browser close / refresh.
   * Caller should use `fetch(..., { keepalive: true })`.
   * Only called when data is dirty at time of beforeunload.
   */
  onBeforeUnloadSave?: (data: T) => void
  /** Called on save error */
  onError?: (error: unknown) => void
  /** Auto-dismiss "saved" status after this many ms (default: 2000, 0 = never) */
  savedDismissMs?: number
}

export type UseAutoSaveReturn = {
  saveStatus: SaveStatus
  /** Trigger an immediate save, bypassing debounce */
  saveNow: () => Promise<void>
  /** Reset dirty baseline to current data (call after external sync) */
  markClean: () => void
  isDirty: boolean
}

// ---------------------------------------------------------------------------
// Default comparator
// ---------------------------------------------------------------------------

function jsonEqual<T>(a: T, b: T): boolean {
  return JSON.stringify(a) === JSON.stringify(b)
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useAutoSave<T>(options: UseAutoSaveOptions<T>): UseAutoSaveReturn {
  const {
    data,
    onSave,
    delay = 2000,
    isEqual = jsonEqual,
    enabled = true,
    saveOnUnmount = false,
    onBeforeUnloadSave,
    onError,
    savedDismissMs = 2000,
  } = options

  // ── Refs: always-current values for async/cleanup contexts ──────────
  const dataRef = useRef(data)
  const onSaveRef = useRef(onSave)
  const onErrorRef = useRef(onError)
  const onBeforeUnloadSaveRef = useRef(onBeforeUnloadSave)
  const isEqualRef = useRef(isEqual)

  // Sync refs every render (synchronous — no effect needed)
  dataRef.current = data
  onSaveRef.current = onSave
  onErrorRef.current = onError
  onBeforeUnloadSaveRef.current = onBeforeUnloadSave
  isEqualRef.current = isEqual

  // ── Clean baseline ──────────────────────────────────────────────────
  const cleanRef = useRef<T>(data)
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')

  // ── Timers ──────────────────────────────────────────────────────────
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const savedDismissTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isSavingRef = useRef(false)

  // ── Dirty state ─────────────────────────────────────────────────────
  const isDirty = !isEqual(data, cleanRef.current)
  const isDirtyRef = useRef(isDirty)
  isDirtyRef.current = isDirty

  // ── Core save ───────────────────────────────────────────────────────
  const performSave = useCallback(async () => {
    if (isSavingRef.current) return
    const currentData = dataRef.current

    // Re-check dirty (may have been markClean'd between schedule and execution)
    if (isEqualRef.current(currentData, cleanRef.current)) return

    isSavingRef.current = true
    if (savedDismissTimerRef.current) clearTimeout(savedDismissTimerRef.current)
    setSaveStatus('saving')

    try {
      await onSaveRef.current(currentData)
      cleanRef.current = currentData
      isDirtyRef.current = false
      setSaveStatus('saved')
      if (savedDismissMs > 0) {
        savedDismissTimerRef.current = setTimeout(
          () => setSaveStatus('idle'),
          savedDismissMs,
        )
      }
    } catch (err) {
      setSaveStatus('error')
      onErrorRef.current?.(err)
    } finally {
      isSavingRef.current = false
    }
  }, [savedDismissMs])

  // ── Debounced auto-save ─────────────────────────────────────────────
  useEffect(() => {
    if (!enabled || !isDirty) return

    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current)
    setSaveStatus('waiting')
    debounceTimerRef.current = setTimeout(() => { void performSave() }, delay)

    return () => {
      if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current)
    }
  }, [data, enabled, isDirty, delay, performSave]) // eslint-disable-line react-hooks/exhaustive-deps

  // ── saveNow ─────────────────────────────────────────────────────────
  const saveNow = useCallback(async () => {
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current)
    await performSave()
  }, [performSave])

  // ── markClean ───────────────────────────────────────────────────────
  const markClean = useCallback(() => {
    cleanRef.current = dataRef.current
    isDirtyRef.current = false
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current)
    if (saveStatus === 'waiting') setSaveStatus('idle')
  }, [saveStatus])

  // ── Unmount save ────────────────────────────────────────────────────
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current)
      if (savedDismissTimerRef.current) clearTimeout(savedDismissTimerRef.current)
      if (saveOnUnmount && isDirtyRef.current) {
        void onSaveRef.current(dataRef.current)
      }
    }
  }, [saveOnUnmount])

  // ── beforeunload save ───────────────────────────────────────────────
  useEffect(() => {
    if (!onBeforeUnloadSave) return
    const handler = () => {
      if (!isDirtyRef.current) return
      onBeforeUnloadSaveRef.current?.(dataRef.current)
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [onBeforeUnloadSave])

  return { saveStatus, saveNow, markClean, isDirty }
}
