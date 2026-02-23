import { create } from 'zustand'
import type { SurgeAlert } from './types'
import { fetchSurges, dismissSurge, createSessionFromSurge } from './api'

interface SurgeStore {
  surges: SurgeAlert[]
  loading: boolean
  error: string | null
  fetchSurges: () => Promise<void>
  dismissSurge: (id: string) => Promise<void>
  createSessionFromSurge: (surgeId: string) => Promise<string>
}

export const useSurgeStore = create<SurgeStore>((set, get) => ({
  surges: [],
  loading: false,
  error: null,

  fetchSurges: async () => {
    set({ loading: true, error: null })
    try {
      const surges = await fetchSurges()
      set({ surges, loading: false })
    } catch (e) {
      set({ loading: false, error: e instanceof Error ? e.message : 'Failed to fetch surges' })
    }
  },

  dismissSurge: async (id) => {
    await dismissSurge(id)
    set((state) => ({
      surges: state.surges.filter((s) => s.id !== id),
    }))
  },

  createSessionFromSurge: async (surgeId) => {
    const { session_id } = await createSessionFromSurge(surgeId)
    get().fetchSurges()
    return session_id
  },
}))
