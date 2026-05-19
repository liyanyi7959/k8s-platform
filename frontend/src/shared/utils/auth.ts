const TOKEN_KEY = 'k8s_platform_token'
const TOKEN_EXPIRES_AT_KEY = 'k8s_platform_token_expires_at'
const SESSION_ID_KEY = 'k8s_platform_session_id'
const SESSION_USER_KEY = 'k8s_platform_session_user'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function getTokenExpiresAt(): number | null {
  const raw = localStorage.getItem(TOKEN_EXPIRES_AT_KEY)
  if (!raw) return null
  const ms = Number(raw)
  if (!Number.isFinite(ms) || ms <= 0) return null
  return ms
}

export function setTokenExpiresAt(expiresAtMs: number | null): void {
  if (!expiresAtMs) {
    localStorage.removeItem(TOKEN_EXPIRES_AT_KEY)
    return
  }
  localStorage.setItem(TOKEN_EXPIRES_AT_KEY, String(Math.floor(expiresAtMs)))
}

export function isTokenExpired(nowMs = Date.now()): boolean {
  const expiresAt = getTokenExpiresAt()
  if (!expiresAt) return false
  return nowMs >= expiresAt
}

export type SessionUser = { id: number; username: string }

export function getSessionId(): string | null {
  return localStorage.getItem(SESSION_ID_KEY)
}

export function setSessionId(id: string): void {
  localStorage.setItem(SESSION_ID_KEY, id)
}

export function createSessionId(): string {
  const cryptoAny =
    typeof crypto !== 'undefined' ? (crypto as unknown as { randomUUID?: () => string } | undefined) : undefined
  const uuid = cryptoAny?.randomUUID?.()
  if (uuid) return uuid
  return `${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`
}

export function getSessionUser(): SessionUser | null {
  const raw = localStorage.getItem(SESSION_USER_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as Partial<SessionUser>
    const id = Number(parsed.id)
    const username = String(parsed.username ?? '')
    if (!Number.isFinite(id) || id <= 0) return null
    if (!username) return null
    return { id, username }
  } catch {
    return null
  }
}

export function setSessionUser(user: SessionUser): void {
  localStorage.setItem(SESSION_USER_KEY, JSON.stringify(user))
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(TOKEN_EXPIRES_AT_KEY)
  localStorage.removeItem(SESSION_ID_KEY)
  localStorage.removeItem(SESSION_USER_KEY)
}

