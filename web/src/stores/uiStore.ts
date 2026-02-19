import { create } from 'zustand'

type UIState = {
  selectedNodeId: string | null
  selectNode: (id: string | null) => void

  forcePreviewTab: boolean
  setForcePreviewTab: (force: boolean) => void
}

export const useUIStore = create<UIState>((set) => ({
  selectedNodeId: null,
  selectNode: (id) => set({ selectedNodeId: id }),

  forcePreviewTab: false,
  setForcePreviewTab: (force) => set({ forcePreviewTab: force }),
}))
