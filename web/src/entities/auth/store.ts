import { create } from 'zustand'
import { fetchMe, refreshToken, logout as apiLogout, type User } from '@/shared/api/auth'
import {
  setTokenGetter,
  setTokenRefreshCallback,
  setAuthExpiredCallback,
  tryRefresh,
} from '@/shared/api/client'

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

let refreshTimer: ReturnType<typeof setTimeout> | null = null

/** Decode JWT payload without verification (for reading exp). */
function decodeTokenExp(token: string): number | null {
  try {
    const payload = token.split('.')[1]
    const decoded = JSON.parse(atob(payload))
    return decoded.exp ?? null
  } catch {
    return null
  }
}

function scheduleProactiveRefresh(token: string) {
  if (refreshTimer) clearTimeout(refreshTimer)

  const exp = decodeTokenExp(token)
  if (!exp) return

  // Refresh 5 minutes before expiry
  const msUntilRefresh = (exp * 1000) - Date.now() - (5 * 60 * 1000)
  if (msUntilRefresh <= 0) {
    // Already near expiry, refresh now
    tryRefresh()
    return
  }

  refreshTimer = setTimeout(() => {
    tryRefresh()
  }, msUntilRefresh)
}

function isTokenExpiredOrNearExpiry(token: string): boolean {
  const exp = decodeTokenExp(token)
  if (!exp) return true
  // Consider expired if within 5 minutes of expiry
  return (exp * 1000) - Date.now() < 5 * 60 * 1000
}

export const useAuthStore = create<AuthState>((set, get) => {
  setTokenGetter(() => get().token)
  setTokenRefreshCallback((token) => {
    set({ token })
    scheduleProactiveRefresh(token)
  })
  setAuthExpiredCallback(() => {
    set({ token: null, user: null })
  })

  // Visibility change handler — refresh on tab return
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState !== 'visible') return
      const { token, user } = get()
      if (!token || !user) return
      if (isTokenExpiredOrNearExpiry(token)) {
        tryRefresh()
      }
    })
  }

  return {
    user: null,
    token: null,
    loading: true,
    initialized: false,

    setToken: (token) => {
      set({ token })
      scheduleProactiveRefresh(token)
    },

    init: async () => {
      const params = new URLSearchParams(window.location.search)
      const urlToken = params.get('token')
      if (urlToken) {
        set({ token: urlToken })
        scheduleProactiveRefresh(urlToken)
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
        scheduleProactiveRefresh(token)
        return true
      } catch {
        return false
      }
    },

    logout: async () => {
      if (refreshTimer) clearTimeout(refreshTimer)
      try { await apiLogout() } catch { /* ignore */ }
      set({ token: null, user: null })
    },
  }
})
