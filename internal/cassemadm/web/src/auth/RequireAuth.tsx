import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { AuthProvider, useAuth } from './AuthProvider'

export function RequireAuth() {
  const { session } = useAuth()
  const location = useLocation()
  const from = `${location.pathname}${location.search}${location.hash}`

  if (!session) return <Navigate to="/login" replace state={{ from }} />
  return <Outlet />
}

export function AuthRoot() {
  return (
    <AuthProvider>
      <Outlet />
    </AuthProvider>
  )
}
