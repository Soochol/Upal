export const API_BASE = '/api'

let getToken: (() => string | null) = () => null
let onTokenRefreshed: ((token: string) => void) | null = null

export function setTokenGetter(fn: () => string | null) {
  getToken = fn
}

export function setTokenRefreshCallback(fn: (token: string) => void) {
  onTokenRefreshed = fn
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

let refreshPromise: Promise<string | null> | null = null

async function tryRefresh(): Promise<string | null> {
  if (refreshPromise) return refreshPromise
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${API_BASE}/auth/refresh`, { method: 'POST' })
      if (!res.ok) return null
      const { token } = await res.json()
      if (token && onTokenRefreshed) onTokenRefreshed(token)
      return token as string
    } catch {
      return null
    } finally {
      refreshPromise = null
    }
  })()
  return refreshPromise
}

export async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  const token = getToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const res = await fetch(url, { ...init, headers })

  if (res.status === 401 && !url.includes('/api/auth/')) {
    const newToken = await tryRefresh()
    if (newToken) {
      const retryHeaders = new Headers(init?.headers)
      retryHeaders.set('Authorization', `Bearer ${newToken}`)
      const retry = await fetch(url, { ...init, headers: retryHeaders })
      if (!retry.ok) {
        const text = await retry.text().catch(() => retry.statusText)
        throw new ApiError(retry.status, text || retry.statusText)
      }
      if (retry.status === 204 || retry.headers.get('content-length') === '0') {
        return undefined as T
      }
      return retry.json()
    }
  }

  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new ApiError(res.status, text || res.statusText)
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') {
    return undefined as T
  }
  return res.json()
}
