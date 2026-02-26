import { create } from 'zustand'
import { fetchMe, refreshToken, logout as apiLogout, type User } from '@/shared/api/auth'
import { setTokenGetter } from '@/shared/api/client'

interface AuthState {
  user: User | null
  token: string | null
  loading: boolean
  initialized: boolean
  setToken: (token: string) => void
  init: () => Promise<void>
  refresh: () => Promise<boolean>
  logout: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set, get) => {
  setTokenGetter(() => get().token)

  return {
    user: null,
    token: null,
    loading: true,
    initialized: false,

    setToken: (token) => set({ token }),

    init: async () => {
      const params = new URLSearchParams(window.location.search)
      const urlToken = params.get('token')
      if (urlToken) {
        set({ token: urlToken })
        window.history.replaceState({}, '', window.location.pathname)
      }

      if (!get().token) {
        const ok = await get().refresh()
        if (!ok) {
          set({ loading: false, initialized: true })
          return
        }
      }

      try {
        const user = await fetchMe()
        set({ user, loading: false, initialized: true })
      } catch {
        set({ token: null, user: null, loading: false, initialized: true })
      }
    },

    refresh: async () => {
      try {
        const { token } = await refreshToken()
        set({ token })
        return true
      } catch {
        return false
      }
    },

    logout: async () => {
      try { await apiLogout() } catch { /* ignore */ }
      set({ token: null, user: null })
    },
  }
})
