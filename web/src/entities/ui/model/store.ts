import { create } from 'zustand'

export type Toast = {
  id: string
  message: string
  variant: 'error' | 'success' | 'info'
}

type UIState = {
  selectedNodeId: string | null
  selectNode: (id: string | null) => void

  forcePreviewTab: boolean
  setForcePreviewTab: (force: boolean) => void

  toasts: Toast[]
  addToast: (message: string, variant?: Toast['variant']) => void
  dismissToast: (id: string) => void
}

let toastId = 0

export const useUIStore = create<UIState>((set) => ({
  selectedNodeId: null,
  selectNode: (id) => set({ selectedNodeId: id }),

  forcePreviewTab: false,
  setForcePreviewTab: (force) => set({ forcePreviewTab: force }),

  toasts: [],
  addToast: (message, variant = 'error') => {
    const id = `toast-${++toastId}`
    set((s) => ({ toasts: [...s.toasts, { id, message, variant }] }))
    setTimeout(() => {
      set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }))
    }, 5000)
  },
  dismissToast: (id) => {
    set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }))
  },
}))
