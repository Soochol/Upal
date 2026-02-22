type EventHandler<T> = (data: T) => void

export function createEventBus<TMap extends Record<string, unknown>>() {
  const listeners = new Map<keyof TMap, Set<EventHandler<unknown>>>()

  return {
    on<K extends keyof TMap>(event: K, handler: EventHandler<TMap[K]>) {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(handler as EventHandler<unknown>)
      return () => listeners.get(event)?.delete(handler as EventHandler<unknown>)
    },
    emit<K extends keyof TMap>(event: K, data: TMap[K]) {
      listeners.get(event)?.forEach((h) => h(data))
    },
    clear() {
      listeners.clear()
    },
  }
}
