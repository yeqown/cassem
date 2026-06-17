import type { CommonResponse } from '../domain/types'
import { clearSession, getSession } from './session'

type Primitive = string | number | boolean
type QueryValue = Primitive | Primitive[] | undefined | null

export class ApiError extends Error {
  code: number
  status: number
  auth: boolean

  constructor(message: string, code: number, status: number, auth = false) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    this.auth = auth
  }
}

export function buildQuery(params: Record<string, QueryValue>) {
  const search = new URLSearchParams()

  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return

    if (Array.isArray(value)) {
      value.forEach((item) => search.append(key, String(item)))
      return
    }

    search.append(key, String(value))
  })

  const query = search.toString()
  return query ? `?${query}` : ''
}

function isAuthError(status: number, payload: CommonResponse<unknown>) {
  return status === 401 || payload.errcode === 16 || /unauthenticated|session expired|invalid session/i.test(payload.errmsg || '')
}

function normalizeHeaders(headers?: HeadersInit) {
  if (!headers) return {}
  if (headers instanceof Headers) return Object.fromEntries(headers.entries())
  if (Array.isArray(headers)) return Object.fromEntries(headers)
  return { ...headers }
}

export async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = normalizeHeaders(options.headers)
  headers.Accept = 'application/json'

  const session = getSession()
  if (session) headers['X-CASSEM-SESSION'] = session

  const response = await fetch(path, { ...options, headers })
  const payload = await response.json().catch(() => ({ errcode: -1, errmsg: `HTTP ${response.status}` })) as CommonResponse<T>
  const auth = isAuthError(response.status, payload)

  if (auth) clearSession()

  if (!response.ok || payload.errcode !== 0) {
    throw new ApiError(payload.errmsg || `HTTP ${response.status}`, payload.errcode ?? -1, response.status, auth)
  }

  return payload.data as T
}

export function jsonBody<T>(body: T): RequestInit {
  return {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }
}
