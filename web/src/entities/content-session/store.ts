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
  // Fetches badge counts (pending_review + approved) — safe for sidebar polling
  syncBadgeCounts: () => Promise<void>
  approveSession: (id: string, selectedAngles: string[], channelMap?: Record<string, string>) => Promise<void>
  rejectSession: (id: string, reason?: string) => Promise<void>

  // Derived
  pendingCount: number
  publishReadyCount: number
}

export const useContentSessionStore = create<ContentSessionStore>((set, get) => ({
  sessions: [],
  filters: {},
  loading: false,
  error: null,
  pendingCount: 0,
  publishReadyCount: 0,

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

  syncBadgeCounts: async () => {
    try {
      const [pending, approved, producing, errored] = await Promise.all([
        fetchContentSessions({ status: 'pending_review' }),
        fetchContentSessions({ status: 'approved' }),
        fetchContentSessions({ status: 'producing' }),
        fetchContentSessions({ status: 'error' }),
      ])
      set({ pendingCount: pending.length, publishReadyCount: approved.length + producing.length + errored.length })
    } catch {
      // Badge polling failure is non-critical — fail silently
    }
  },

  approveSession: async (id, selectedAngles, channelMap) => {
    const updated = await approveSession(id, selectedAngles)
    set((state) => {
      const sessions = state.sessions.map((s) => (s.id === id ? updated : s))
      return {
        sessions,
        pendingCount: sessions.filter((s) => s.status === 'pending_review').length,
        publishReadyCount: sessions.filter((s) => s.status === 'approved' || s.status === 'producing').length,
      }
    })
    // Chain: trigger production with the selected workflows
    if (selectedAngles.length > 0) {
      const workflowRequests = selectedAngles.map(name => ({
        name,
        ...(channelMap?.[name] ? { channel_id: channelMap[name] } : {}),
      }))
      await produceSession(id, workflowRequests).catch((err) => {
        console.error('Failed to trigger production:', err)
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
        publishReadyCount: sessions.filter((s) => s.status === 'approved' || s.status === 'producing').length,
      }
    })
  },
}))
