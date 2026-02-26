import { create } from 'zustand'

type GenerationState = {
  isGenerating: boolean
  generationId: string | null
  error: string | null

  start: (generationId: string) => void
  clear: () => void
  fail: (error: string) => void
}

export const useGenerationStore = create<GenerationState>((set) => ({
  isGenerating: false,
  generationId: null,
  error: null,

  start: (generationId: string) => {
    set({ isGenerating: true, generationId, error: null })
  },

  clear: () => {
    set({ isGenerating: false, generationId: null, error: null })
  },

  fail: (error: string) => {
    set({ isGenerating: false, generationId: null, error })
  },
}))
