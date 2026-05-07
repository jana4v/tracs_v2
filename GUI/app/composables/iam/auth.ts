import { computed } from 'vue'

export interface IamUser {
  id: string
  username: string
  full_name?: string
  email?: string
  roles?: string[]
}

interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
  user: IamUser
}

interface RefreshResponse {
  access_token: string
  expires_in: number
  token_type: string
}

interface StoredSession {
  accessToken: string
  refreshToken: string
  expiresAt: number
  user: IamUser | null
}

const STORAGE_KEY = 'iam_auth_v1'

function readStorage(): StoredSession | null {
  if (!import.meta.client)
    return null
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw)
      return null
    const parsed = JSON.parse(raw) as StoredSession
    if (!parsed?.accessToken || !parsed?.refreshToken || !parsed?.expiresAt)
      return null
    return parsed
  }
  catch {
    return null
  }
}

function writeStorage(session: StoredSession | null) {
  if (!import.meta.client)
    return
  if (!session) {
    window.localStorage.removeItem(STORAGE_KEY)
    return
  }
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session))
}

function baseApiRoot() {
  if (!import.meta.client)
    return ''
  return `${window.location.protocol}//${window.location.hostname}/iam/api/v1`
}

function candidateUrls(path: string): string[] {
  const normalized = path.startsWith('/') ? path : `/${path}`
  const root = baseApiRoot()
  if (!root)
    return []

  if (normalized.startsWith('/iam/')) {
    return [`${root}${normalized}`]
  }

  return [`${root}/iam${normalized}`, `${root}${normalized}`]
}

async function fetchWithFallback<T>(path: string, options: any): Promise<T> {
  const urls = candidateUrls(path)
  let lastError: any
  console.log('[FetchWithFallback] Trying URLs for path:', path, urls)

  for (const url of urls) {
    try {
      console.log('[FetchWithFallback] Fetching:', url)
      return (await $fetch(url, options)) as T
    }
    catch (err: any) {
      console.warn('[FetchWithFallback] Failed:', url, err)
      lastError = err
      const status = Number(err?.statusCode ?? err?.status ?? 0)
      if (status !== 404 && status !== 405) {
        throw err
      }
    }
  }

  console.error('[FetchWithFallback] All URLs failed:', lastError)
  throw lastError ?? new Error('IAM request failed')
}

export function useIamAuth() {
  const session = useState<StoredSession | null>('iam-auth-session', () => readStorage())

  const isLoggedIn = computed(() => Boolean(session.value?.accessToken))
  const user = computed(() => session.value?.user ?? null)
  const accessToken = computed(() => session.value?.accessToken ?? '')

  function setSession(next: StoredSession | null) {
    session.value = next
    writeStorage(next)
  }

  function clearSession() {
    setSession(null)
  }

  function tokenExpiringSoon() {
    if (!session.value?.expiresAt)
      return true
    return Date.now() + 20_000 >= session.value.expiresAt
  }

  async function login(username: string, password: string) {
    console.log('[Auth] Attempting login for:', username)
    const resp = await fetchWithFallback<LoginResponse>('/auth/login', {
      method: 'POST',
      body: { username, password },
    })

    console.log('[Auth] Login successful, storing session')
    setSession({
      accessToken: resp.access_token,
      refreshToken: resp.refresh_token,
      expiresAt: Date.now() + Math.max(5, resp.expires_in) * 1000,
      user: resp.user ?? null,
    })

    return resp.user
  }

  async function refreshIfNeeded(force = false) {
    if (!session.value?.refreshToken)
      return false
    if (!force && !tokenExpiringSoon())
      return true

    const resp = await fetchWithFallback<RefreshResponse>('/auth/refresh', {
      method: 'POST',
      body: { refresh_token: session.value.refreshToken },
    })

    setSession({
      accessToken: resp.access_token,
      refreshToken: session.value.refreshToken,
      expiresAt: Date.now() + Math.max(5, resp.expires_in) * 1000,
      user: session.value.user,
    })

    return true
  }

  async function logout() {
    const refreshToken = session.value?.refreshToken
    const token = session.value?.accessToken

    if (refreshToken && token) {
      try {
        await fetchWithFallback('/auth/logout', {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
          body: { refresh_token: refreshToken },
        })
      }
      catch {
        // Ignore logout transport errors and clear local session anyway.
      }
    }

    clearSession()
  }

  return {
    session,
    user,
    accessToken,
    isLoggedIn,
    login,
    logout,
    clearSession,
    refreshIfNeeded,
  }
}
