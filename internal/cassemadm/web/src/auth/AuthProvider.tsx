/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useMemo, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import type { LoginResponse, User } from '../domain/types'
import { apiRequest, jsonBody } from '../lib/api'
import { clearSession, getSession, getStoredUser, storeSession } from '../lib/session'

type AuthContextValue = {
  session: string
  user: User | null
  login: (account: string, password: string, redirectTo?: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState(getSession())
  const [user, setUser] = useState<User | null>(getStoredUser())
  const navigate = useNavigate()

  const value = useMemo<AuthContextValue>(
    () => ({
      session,
      user,
      async login(account, password, redirectTo = '/dashboard') {
        const data = await apiRequest<LoginResponse>('/api/account/login', jsonBody({ account, password }))
        storeSession(data.session, data.user)
        setSession(data.session)
        setUser(data.user)
        navigate(redirectTo || '/dashboard', { replace: true })
      },
      logout() {
        clearSession()
        setSession('')
        setUser(null)
        navigate('/login', { replace: true })
      },
    }),
    [navigate, session, user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used within AuthProvider')
  return value
}
