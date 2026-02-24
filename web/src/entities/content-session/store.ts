import { create } from 'zustand'
import type { ContentSession, ContentSessionStatus } from './types'
import { fetchContentSessions, approveSession, rejectSession, produceSession } from './api'

export interface ContentSessionFilters {
  status?: ContentSessionStatus
  pipelineId?: string
}

interface ContentSessionStore {
  sessions: ContentSession[]
  filters: ContentSessionFilters
  loading: boolean
  error: string | null

  setFilters: (filters: ContentSessionFilters) => void
  fetchSessions: () => Promise<void>
  // Fetches only pending count — ignores active filters, safe for Header polling
  syncPendingCount: () => Promise<void>
  approveSession: (id: string, selectedAngles: string[]) => Promise<void>
  rejectSession: (id: string, reason?: string) => Promise<void>

  // Derived
  pendingCount: number
}

export const useContentSessionStore = create<ContentSessionStore>((set, get) => ({
  sessions: [],
  filters: {},
  loading: false,
  error: null,
  pendingCount: 0,

  setFilters: (filters) => set({ filters }),

  fetchSessions: async () => {
    set({ loading: true, error: null })
    try {
      const { filters } = get()
      const sessions = await fetchContentSessions({
        pipelineId: filters.pipelineId,
        status: filters.status,
      })
      // Only update pendingCount when fetching unfiltered (status not specified)
      const updates: Partial<ContentSessionStore> = { sessions, loading: false }
      if (!filters.status) {
        updates.pendingCount = sessions.filter((s) => s.status === 'pending_review').length
      }
      set(updates)
    } catch (e) {
      set({ error: e instanceof Error ? e.message : 'Failed to fetch sessions', loading: false })
    }
  },

  syncPendingCount: async () => {
    try {
      const pending = await fetchContentSessions({ status: 'pending_review' })
      set({ pendingCount: pending.length })
    } catch {
      // Badge polling failure is non-critical — fail silently
    }
  },

  approveSession: async (id, selectedAngles) => {
    const updated = await approveSession(id, selectedAngles)
    set((state) => {
      const sessions = state.sessions.map((s) => (s.id === id ? updated : s))
      return {
        sessions,
        pendingCount: sessions.filter((s) => s.status === 'pending_review').length,
      }
    })
    // Chain: trigger production with the selected workflows
    if (selectedAngles.length > 0) {
      const workflowRequests = selectedAngles.map(name => ({ name }))
      await produceSession(id, workflowRequests).catch(() => {
        // Production trigger failure is non-critical — session is already approved
      })
    }
  },

  rejectSession: async (id, reason) => {
    const updated = await rejectSession(id, reason)
    set((state) => {
      const sessions = state.sessions.map((s) => (s.id === id ? updated : s))
      return {
        sessions,
        pendingCount: sessions.filter((s) => s.status === 'pending_review').length,
      }
    })
  },
}))
