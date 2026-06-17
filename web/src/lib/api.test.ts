import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiRequest, buildQuery, jsonBody } from './api'

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
})

describe('api helpers', () => {
  it('builds repeated query parameters and skips empty values', () => {
    expect(buildQuery({
      limit: 100,
      key: ['a', 'b'],
      empty: '',
      nil: null,
      absent: undefined,
    })).toBe('?limit=100&key=a&key=b')
  })

  it('builds json request bodies', () => {
    expect(jsonBody({ key: 'value' })).toEqual({
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{"key":"value"}',
    })
  })

  it('unwraps CommonResponse data', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ errcode: 0, errmsg: 'success', data: { apps: [] } }),
    }))

    await expect(apiRequest('/api/apps')).resolves.toEqual({ apps: [] })
  })

  it('adds session header when stored', async () => {
    localStorage.setItem('cassem.session', 'abc')
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ errcode: 0, errmsg: 'success', data: null }),
    })
    vi.stubGlobal('fetch', fetchMock)

    await apiRequest('/api/apps')

    expect(fetchMock).toHaveBeenCalledWith('/api/apps', expect.objectContaining({
      headers: expect.objectContaining({ 'X-CASSEM-SESSION': 'abc' }),
    }))
  })

  it('throws ApiError for application errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ errcode: -2, errmsg: 'bad request' }),
    }))

    await expect(apiRequest('/api/apps')).rejects.toMatchObject({ code: -2, message: 'bad request' })
  })

  it('clears session on auth errors', async () => {
    localStorage.setItem('cassem.session', 'abc')
    localStorage.setItem('cassem.user', '{"account":"a@example.com"}')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ errcode: -1, errmsg: 'invalid session' }),
    }))

    await expect(apiRequest('/api/apps')).rejects.toBeInstanceOf(ApiError)
    expect(localStorage.getItem('cassem.session')).toBeNull()
    expect(localStorage.getItem('cassem.user')).toBeNull()
  })

  it('clears session when the response errcode indicates auth failure', async () => {
    localStorage.setItem('cassem.session', 'abc')
    localStorage.setItem('cassem.user', '{"account":"a@example.com"}')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      json: async () => ({ errcode: 16, errmsg: 'permission denied' }),
    }))

    await expect(apiRequest('/api/apps')).rejects.toBeInstanceOf(ApiError)
    expect(localStorage.getItem('cassem.session')).toBeNull()
    expect(localStorage.getItem('cassem.user')).toBeNull()
  })

  it.each([
    'session expired',
    'unauthenticated',
  ])('clears session when the error message indicates auth failure: %s', async (errmsg) => {
    localStorage.setItem('cassem.session', 'abc')
    localStorage.setItem('cassem.user', '{"account":"a@example.com"}')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      json: async () => ({ errcode: -1, errmsg }),
    }))

    await expect(apiRequest('/api/apps')).rejects.toBeInstanceOf(ApiError)
    expect(localStorage.getItem('cassem.session')).toBeNull()
    expect(localStorage.getItem('cassem.user')).toBeNull()
  })
})
