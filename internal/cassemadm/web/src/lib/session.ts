import type { User } from '../domain/types'

const sessionKey = 'cassem.session'
const userKey = 'cassem.user'

export function getSession() {
  return localStorage.getItem(sessionKey) || ''
}

export function getStoredUser(): User | null {
  const raw = localStorage.getItem(userKey)
  if (!raw) return null

  try {
    return JSON.parse(raw) as User
  } catch {
    localStorage.removeItem(userKey)
    return null
  }
}

export function storeSession(session: string, user: User) {
  localStorage.setItem(sessionKey, session)
  localStorage.setItem(userKey, JSON.stringify(user))
}

export function clearSession() {
  localStorage.removeItem(sessionKey)
  localStorage.removeItem(userKey)
}
