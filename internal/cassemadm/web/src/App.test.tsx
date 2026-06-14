import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { routes } from './routes'

function renderRoute(path: string) {
  const entry = path.startsWith('/ui') ? path : `/ui${path}`
  const router = createMemoryRouter(routes, { basename: '/ui', initialEntries: [entry] })
  render(<RouterProvider router={router} />)
  return router
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })

  return { promise, resolve }
}

function createJsonResponse<T>(payload: T, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => payload,
  } as Response
}

afterEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('app shell routing', () => {
  it('renders login when unauthenticated', () => {
    renderRoute('/login')

    expect(screen.getByRole('heading', { name: /cassem admin/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
  })

  it('redirects protected routes to login when unauthenticated', () => {
    const router = renderRoute('/dashboard')

    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/ui/login')
  })

  it('navigates to the saved protected route after login', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          errcode: 0,
          data: {
            session: 'session',
            user: { account: 'superadmin@example.com', nickname: 'Super Admin' },
          },
        }),
      }),
    )

    const user = userEvent.setup()
    renderRoute('/apps')

    await user.type(screen.getByRole('textbox', { name: /account/i }), 'superadmin@example.com')
    await user.type(screen.getByLabelText(/password/i), 'password')
    await user.click(screen.getByRole('button', { name: /login/i }))

    expect(await screen.findByRole('heading', { name: /apps/i })).toBeInTheDocument()
  })

  it('preserves query and hash when redirecting to login and back', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          errcode: 0,
          data: {
            session: 'session',
            user: { account: 'superadmin@example.com', nickname: 'Super Admin' },
          },
        }),
      }),
    )

    const user = userEvent.setup()
    const router = renderRoute('/cluster/instances?app=demo#node')

    await user.type(screen.getByRole('textbox', { name: /account/i }), 'superadmin@example.com')
    await user.type(screen.getByLabelText(/password/i), 'password')
    await user.click(screen.getByRole('button', { name: /login/i }))

    expect(await screen.findByRole('heading', { name: /instances/i })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/ui/cluster/instances')
    expect(router.state.location.search).toBe('?app=demo')
    expect(router.state.location.hash).toBe('#node')
  })

  it('renders branded shell navigation and account menu when authenticated', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem(
      'cassem.user',
      JSON.stringify({ account: 'superadmin@example.com', nickname: 'Super Admin', roles: ['superadmin'] }),
    )

    const user = userEvent.setup()
    renderRoute('/dashboard')

    expect(screen.getByRole('heading', { name: /dashboard/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /apps/i })).toHaveAttribute('href', '/ui/apps')
    expect(screen.getAllByText('CASSEM').length).toBeGreaterThan(0)
    expect(screen.queryByText('Cassem Admin')).not.toBeInTheDocument()
    expect(screen.getAllByText('Super Admin').length).toBeGreaterThan(0)

    await user.click(screen.getByRole('button', { name: /account menu/i }))

    expect(await screen.findByText('Roles: superadmin')).toBeInTheDocument()
    expect(screen.getAllByText('superadmin@example.com').length).toBeGreaterThan(0)
  })

  it('renders apps page inside shell', () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    renderRoute('/apps')

    expect(screen.getByRole('heading', { name: /apps/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /add app/i })).toBeInTheDocument()
  })

  it('keeps the latest apps response when refresh overlaps the initial load', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const initialLoad = createDeferred<Response>()
    const refreshLoad = createDeferred<Response>()
    let appsGetRequests = 0
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps?limit=100')) {
        appsGetRequests += 1
        return appsGetRequests === 1 ? initialLoad.promise : refreshLoad.promise
      }

      if (url.includes('/api/apps/demo')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }

      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps')

    await user.click(screen.getByRole('button', { name: /add app/i }))
    await user.type(screen.getByLabelText(/app id/i), 'demo')
    await user.type(screen.getByLabelText(/description/i), 'Demo app')
    await user.click(screen.getByRole('button', { name: /^create$/i }))

    await waitFor(() => expect(appsGetRequests).toBe(2))

    await act(async () => {
      await refreshLoad.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'latest', description: 'new' }] } }))
      await refreshLoad.promise
    })

    expect(await screen.findByText('latest')).toBeInTheDocument()

    await act(async () => {
      await initialLoad.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'stale', description: 'old' }] } }))
      await initialLoad.promise
    })

    expect(screen.getByText('latest')).toBeInTheDocument()
    expect(screen.queryByText('stale')).not.toBeInTheDocument()
  })

  it('renders envs page with app context', () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    renderRoute('/apps/demo/envs')

    expect(screen.getByRole('heading', { name: /environments/i })).toBeInTheDocument()
    expect(screen.getByText(/demo/i)).toBeInTheDocument()
  })

  it('keeps the latest environments response when refresh overlaps the initial load', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const initialLoad = createDeferred<Response>()
    const refreshLoad = createDeferred<Response>()
    let envRequests = 0
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs?limit=100')) {
        envRequests += 1
        return envRequests === 1 ? initialLoad.promise : refreshLoad.promise
      }

      if (url.includes('/api/apps/demo/envs/prod')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }

      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs')

    await user.click(screen.getByRole('button', { name: /add environment/i }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText(/environment/i), 'prod')
    await user.click(within(dialog).getByRole('button', { name: /^create$/i }))

    await waitFor(() => expect(envRequests).toBe(2))

    await act(async () => {
      await refreshLoad.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod'] } }))
      await refreshLoad.promise
    })

    expect(await screen.findByText('prod')).toBeInTheDocument()

    await act(async () => {
      await initialLoad.resolve(createJsonResponse({ errcode: 0, data: { envs: ['stale'] } }))
      await initialLoad.promise
    })

    expect(screen.getByText('prod')).toBeInTheDocument()
    expect(screen.queryByText('stale')).not.toBeInTheDocument()
  })

  it('does not reload the original app after navigating away during a create request', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const initialDemoLoad = createDeferred<Response>()
    const otherAppLoad = createDeferred<Response>()
    const createRequest = createDeferred<Response>()
    const staleDemoReload = createDeferred<Response>()
    let demoEnvLoads = 0
    let otherEnvLoads = 0
    let createRequests = 0

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs?limit=100')) {
        demoEnvLoads += 1
        return demoEnvLoads === 1 ? initialDemoLoad.promise : staleDemoReload.promise
      }

      if (url.includes('/api/apps/other/envs?limit=100')) {
        otherEnvLoads += 1
        return otherAppLoad.promise
      }

      if (url.includes('/api/apps/demo/envs/prod') && init?.method === 'POST') {
        createRequests += 1
        return createRequest.promise
      }

      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    const router = renderRoute('/apps/demo/envs')

    await waitFor(() => expect(demoEnvLoads).toBe(1))

    await act(async () => {
      initialDemoLoad.resolve(createJsonResponse({ errcode: 0, data: { envs: ['demo'] } }))
      await initialDemoLoad.promise
    })

    expect(await screen.findByText('demo')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /add environment/i }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText(/environment/i), 'prod')
    await user.click(within(dialog).getByRole('button', { name: /^create$/i }))

    await waitFor(() => expect(createRequests).toBe(1))

    await act(async () => {
      await router.navigate('/apps/other/envs')
    })

    await waitFor(() => expect(otherEnvLoads).toBe(1))

    await act(async () => {
      otherAppLoad.resolve(createJsonResponse({ errcode: 0, data: { envs: ['other'] } }))
      await otherAppLoad.promise
    })

    expect(await screen.findByText('other')).toBeInTheDocument()

    await act(async () => {
      createRequest.resolve(createJsonResponse({ errcode: 0, data: null }))
      await createRequest.promise
    })

    expect(demoEnvLoads).toBe(1)
    expect(screen.getByText('other')).toBeInTheDocument()
    expect(screen.queryByText('stale-demo')).not.toBeInTheDocument()
  })

  it('renders elements page with app and environment context', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        createJsonResponse({
          errcode: 0,
          data: {
            elements: [],
          },
        }),
      ),
    )

    renderRoute('/apps/demo/envs/prod/elements')

    expect(await screen.findByRole('heading', { name: /elements/i })).toBeInTheDocument()
    expect(screen.getByText(/app: demo/i)).toBeInTheDocument()
    expect(screen.getByText(/env: prod/i)).toBeInTheDocument()
  })

  it('renders element detail page with content, versions, and operations tabs', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)

      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [] } }))
      }

      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/operations?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { operations: [] } }))
      }

      if (url.includes('/api/apps/demo/envs/prod/elements/db.url')) {
        return Promise.resolve(
          createJsonResponse({
            errcode: 0,
            data: {
              metadata: { key: 'db.url', latestVersion: 1, usingVersion: 1, unpublishedVersion: 0, contentType: 4 },
              raw: btoa('postgres://demo'),
              version: 1,
              published: true,
            },
          }),
        )
      }

      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    renderRoute('/apps/demo/envs/prod/elements/db.url')

    expect(await screen.findByRole('heading', { name: /element detail/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /content/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /versions/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /operations/i })).toBeInTheDocument()
  })

  it('renders users page with list and add action', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(createJsonResponse({ errcode: 0, data: { users: [{ account: 'alice@example.com', nickname: 'Alice', roles: ['admin'] }] } })),
    )

    renderRoute('/users')

    expect(await screen.findByRole('heading', { name: /users/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /add user/i })).toBeInTheDocument()
    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Admin')).toBeInTheDocument()
  })

  it('renders users page with access management action', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        createJsonResponse({ errcode: 0, data: { users: [{ account: 'alice@example.com', nickname: 'Alice', roles: ['appdeveloper'], bindingCount: 1 }] } }),
      ),
    )

    renderRoute('/users')

    expect(await screen.findByRole('heading', { name: /users/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /manage access/i })).toBeInTheDocument()
    expect(screen.getByText('Developer')).toBeInTheDocument()
  })

  it('renders agents page with refresh control', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        createJsonResponse({
          errcode: 0,
          data: {
            agents: [],
          },
        }),
      ),
    )

    renderRoute('/cluster/agents')

    expect(await screen.findByRole('heading', { name: /agents/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument()
  })

  it('renders instances page with filter and detail controls', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        createJsonResponse({
          errcode: 0,
          data: {
            instances: [{ clientId: 'instance-01', agentId: 'agent-a', clientIp: '10.0.0.1' }],
          },
        }),
      ),
    )

    renderRoute('/cluster/instances')

    expect(await screen.findByRole('heading', { name: /instances/i })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: /^app$/i })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: /^env$/i })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: /^key$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /filter/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /refresh all/i })).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: /detail/i })).toBeInTheDocument()
  })

  it('accepts raw array agents responses from the cluster endpoint', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => [{ agentId: 'agent-1', addr: '127.0.0.1:7001' }],
      } as Response),
    )

    renderRoute('/cluster/agents')

    expect(await screen.findByText('agent-1')).toBeInTheDocument()
    expect(screen.getByText('127.0.0.1:7001')).toBeInTheDocument()
  })

  it('falls back to full instance list when filter input is incomplete', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse({ errcode: 0, data: { instances: [] } }))
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/cluster/instances')

    expect(await screen.findByRole('heading', { name: /instances/i })).toBeInTheDocument()
    await user.type(screen.getByRole('textbox', { name: /^app$/i }), 'demo')
    await user.click(screen.getByRole('button', { name: /^filter$/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/cluster/instances?limit=100', expect.any(Object))
    })
    expect(screen.queryByText(/app, env, and key are all required for filtered instance lookup/i)).not.toBeInTheDocument()
  })

  it('seeds instance filters from the query string and loads detail inline', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)

      if (url.includes('/api/cluster/instances/filter?app=demo&env=prod&key=db.url')) {
        return Promise.resolve(
          createJsonResponse({
            errcode: 0,
            data: {
              instances: [{ clientId: 'instance-01', agentId: 'agent-a', clientIp: '10.0.0.1' }],
            },
          }),
        )
      }

      if (url.includes('/api/cluster/instances/detail/instance-01')) {
        return Promise.resolve(
          createJsonResponse({
            errcode: 0,
            data: {
              clientId: 'instance-01',
              agentId: 'agent-a',
              watching: { app: 'demo', env: 'prod', key: 'db.url' },
            },
          }),
        )
      }

      return Promise.resolve(createJsonResponse({ errcode: 0, data: { instances: [] } }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/cluster/instances?app=demo&env=prod&key=db.url')

    expect(await screen.findByRole('heading', { name: /instances/i })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: /^app$/i })).toHaveValue('demo')
    expect(screen.getByRole('textbox', { name: /^env$/i })).toHaveValue('prod')
    expect(screen.getByRole('textbox', { name: /^key$/i })).toHaveValue('db.url')
    expect(await screen.findByText('instance-01')).toBeInTheDocument()

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith('/api/cluster/instances/filter?app=demo&env=prod&key=db.url', expect.any(Object)),
    )

    await user.click(screen.getByRole('button', { name: /detail/i }))

    expect(await screen.findByRole('heading', { name: /instance detail/i })).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText(/"clientId": "instance-01"/i, { selector: 'pre' })).toBeInTheDocument()
    })
  })

  it('submits add user requests to the account add endpoint', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/account/users?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { users: [] } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/users')

    await screen.findByRole('button', { name: /add user/i })
    await user.click(screen.getByRole('button', { name: /add user/i }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByRole('textbox', { name: /account/i }), 'alice@example.com')
    await user.type(within(dialog).getByRole('textbox', { name: /nickname/i }), 'Alice')
    await user.type(within(dialog).getByLabelText(/password/i), 'secret-123')
    await user.click(within(dialog).getByRole('button', { name: /^create$/i }))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/account/add',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ account: 'alice@example.com', password: 'secret-123', nickname: 'Alice' }),
      }),
    )
    expect(await screen.findByText(/user added successfully/i)).toBeInTheDocument()
  })

  it('submits disable and reset actions from the users list', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const confirmMock = vi.fn().mockReturnValue(true)
    vi.stubGlobal('confirm', confirmMock)
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/account/users?limit=100')) {
        return Promise.resolve(
          createJsonResponse({ errcode: 0, data: { users: [{ account: 'alice@example.com', nickname: 'Alice', roles: ['admin'] }] } }),
        )
      }
      if (url.includes('/api/account/acl/domains')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { domains: ['cluster', 'demo/*', 'demo/prod'] } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/users')

    await screen.findByText('alice@example.com')
    await user.click(screen.getByRole('button', { name: /^disable$/i }))

    expect(fetchMock).toHaveBeenCalledWith('/api/account/disable?account=alice%40example.com', expect.any(Object))

    await user.click(screen.getByRole('button', { name: /reset password/i }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText(/password/i), 'new-secret')
    await user.click(within(dialog).getByRole('button', { name: /^reset$/i }))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/account/reset',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ account: 'alice@example.com', password: 'new-secret' }),
      }),
    )
    expect(await screen.findByText(/password reset successfully/i)).toBeInTheDocument()
  })

  it('assigns ACL bindings from the users access dialog with structured domain selection', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/account/users?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { users: [{ account: 'alice@example.com', nickname: 'Alice', roles: ['admin'], bindingCount: 1 }] } }))
      }
      if (url.includes('/api/account/acl/domains')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { domains: ['cluster', 'demo/*', 'demo/prod'] } }))
      }
      if (url.includes('/api/account/users/alice%40example.com/acl')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { bindings: [{ role: 'admin', domain: 'cluster' }] } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/users')

    await screen.findByText('alice@example.com')
    await user.click(screen.getByRole('button', { name: /manage access/i }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('combobox', { name: /role/i }))
    await user.click(await screen.findByRole('option', { name: /developer/i }))
    await user.click(within(dialog).getByRole('combobox', { name: /scope/i }))
    await user.click(await screen.findByRole('option', { name: /entire app/i }))
    await user.click(within(dialog).getByRole('combobox', { name: /^app$/i }))
    await user.click(await screen.findByRole('option', { name: /^demo$/i }))
    await user.click(within(dialog).getByRole('button', { name: /add binding/i }))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/account/acl/assign?account=alice%40example.com&role=appdeveloper&domain=demo%2F*',
      expect.any(Object),
    )
  })

  it('revokes ACL bindings from the users access dialog', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/account/users?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { users: [{ account: 'alice@example.com', nickname: 'Alice', roles: ['admin'], bindingCount: 1 }] } }))
      }
      if (url.includes('/api/account/acl/domains')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { domains: ['cluster', 'demo/*', 'demo/prod'] } }))
      }
      if (url.includes('/api/account/users/alice%40example.com/acl')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { bindings: [{ role: 'admin', domain: 'cluster' }] } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/users')

    await screen.findByText('alice@example.com')
    await user.click(screen.getByRole('button', { name: /manage access/i }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: /revoke/i }))

    expect(fetchMock).toHaveBeenCalledWith('/api/account/acl/revoke?account=alice%40example.com&role=admin&domain=cluster', expect.any(Object))
  })

  it('renders publish wizard route', () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    expect(screen.getByRole('heading', { name: /publish element/i })).toBeInTheDocument()
    expect(screen.getByRole('spinbutton', { name: /^version$/i })).toBeInTheDocument()
  })

  it('renders rollback wizard route', () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    renderRoute('/apps/demo/envs/prod/elements/db.url/rollback')

    expect(screen.getByRole('heading', { name: /rollback element/i })).toBeInTheDocument()
    expect(screen.getByRole('spinbutton', { name: /target version/i })).toBeInTheDocument()
  })

  it('requires a positive integer publish version before continuing', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    const versionInput = screen.getByRole('spinbutton', { name: /^version$/i })
    const nextButton = screen.getByRole('button', { name: /^next$/i })

    expect(nextButton).toBeDisabled()

    await user.type(versionInput, '0')
    expect(screen.getByText(/version must be a positive integer within uint32 range/i)).toBeInTheDocument()
    expect(nextButton).toBeDisabled()

    await user.clear(versionInput)
    await user.type(versionInput, '1.5')
    expect(screen.getByText(/version must be a positive integer within uint32 range/i)).toBeInTheDocument()
    expect(nextButton).toBeDisabled()

    await user.clear(versionInput)
    await user.type(versionInput, '-1')
    expect(screen.getByText(/version must be a positive integer within uint32 range/i)).toBeInTheDocument()
    expect(nextButton).toBeDisabled()

    await user.clear(versionInput)
    await user.type(versionInput, '4294967296')
    expect(screen.getByText(/version must be a positive integer within uint32 range/i)).toBeInTheDocument()
    expect(nextButton).toBeDisabled()

    await user.clear(versionInput)
    await user.type(versionInput, '2')

    expect(nextButton).toBeEnabled()
  })

  it('requires a positive integer rollback target version before loading the diff', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/rollback')

    const versionInput = screen.getByRole('spinbutton', { name: /target version/i })
    const nextButton = screen.getByRole('button', { name: /^next$/i })

    expect(nextButton).toBeDisabled()

    await user.type(versionInput, '0')
    expect(screen.getByText(/version must be a positive integer within uint32 range/i)).toBeInTheDocument()
    expect(nextButton).toBeDisabled()

    await user.clear(versionInput)
    await user.type(versionInput, '2.5')
    expect(screen.getByText(/version must be a positive integer within uint32 range/i)).toBeInTheDocument()
    expect(nextButton).toBeDisabled()

    await user.clear(versionInput)
    await user.type(versionInput, '-1')
    expect(screen.getByText(/version must be a positive integer within uint32 range/i)).toBeInTheDocument()
    expect(nextButton).toBeDisabled()

    await user.clear(versionInput)
    await user.type(versionInput, '4294967296')
    expect(screen.getByText(/version must be a positive integer within uint32 range/i)).toBeInTheDocument()
    expect(nextButton).toBeDisabled()

    await user.clear(versionInput)
    await user.type(versionInput, '4')

    expect(nextButton).toBeEnabled()
  })

  it('requires at least one gray publish target before allowing progression', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    await user.type(screen.getByRole('spinbutton', { name: /^version$/i }), '3')
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('radio', { name: /gray publish/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))

    const nextButton = screen.getByRole('button', { name: /^next$/i })

    expect(screen.getAllByText(/^Gray publish requires at least one agent or instance target\.$/i)).toHaveLength(2)
    expect(nextButton).toBeDisabled()

    await user.type(screen.getByLabelText(/instance ids/i), 'instance-01')

    expect(screen.queryByText(/^Gray publish requires at least one agent or instance target\.$/i)).not.toBeInTheDocument()
    expect(nextButton).toBeEnabled()
  })

  it('hides explicit target inputs for full publish', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    await user.type(screen.getByRole('spinbutton', { name: /^version$/i }), '3')
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))

    expect(screen.queryByLabelText(/agent ids/i)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/instance ids/i)).not.toBeInTheDocument()
    expect(screen.getByText(/full publish does not use explicit agent or instance targets/i)).toBeInTheDocument()
  })

  it('splits gray publish targets on newlines before submit', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse({ errcode: 0, data: null }))
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    await user.type(screen.getByRole('spinbutton', { name: /^version$/i }), '3')
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('radio', { name: /gray publish/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.type(screen.getByLabelText(/agent ids/i), 'agent-a{enter}agent-b')
    await user.type(screen.getByLabelText(/instance ids/i), 'instance-01{enter}instance-02')
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('button', { name: /^publish$/i }))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/apps/demo/envs/prod/elements/db.url/publish',
      expect.objectContaining({
        body: JSON.stringify({
          version: 3,
          publishMode: 1,
          agentId: ['agent-a', 'agent-b'],
          instanceId: ['instance-01', 'instance-02'],
        }),
      }),
    )
  })

  it('shows rollback diff loading state while fetching comparison', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const diffRequest = createDeferred<Response>()
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/diff?')) {
        return diffRequest.promise
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/rollback')

    await user.type(screen.getByRole('spinbutton', { name: /target version/i }), '1')
    await user.click(screen.getByRole('button', { name: /^next$/i }))

    expect(await screen.findByText(/loading diff/i)).toBeInTheDocument()

    await act(async () => {
      diffRequest.resolve(
        createJsonResponse({
          errcode: 0,
          data: {
            base: {
              metadata: { key: 'db.url', usingVersion: 2, latestVersion: 3, unpublishedVersion: 3, contentType: 4 },
              version: 2,
              raw: btoa('before'),
              published: true,
            },
            compare: {
              metadata: { key: 'db.url', usingVersion: 2, latestVersion: 3, unpublishedVersion: 3, contentType: 4 },
              version: 1,
              raw: btoa('after'),
              published: true,
            },
            diff: 'changed',
          },
        }),
      )
      await diffRequest.promise
    })

    expect(await screen.findByDisplayValue('changed')).toBeInTheDocument()
  })

  it('blocks rollback diff review when target is not older than live version', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        createJsonResponse({
          errcode: 0,
          data: {
            base: {
              metadata: { key: 'db.url', usingVersion: 2, latestVersion: 3, unpublishedVersion: 3, contentType: 4 },
              version: 2,
              raw: btoa('before'),
              published: true,
            },
            compare: {
              metadata: { key: 'db.url', usingVersion: 2, latestVersion: 3, unpublishedVersion: 3, contentType: 4 },
              version: 2,
              raw: btoa('before'),
              published: true,
            },
            diff: '',
          },
        }),
      ),
    )

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/rollback')

    await user.type(screen.getByRole('spinbutton', { name: /target version/i }), '2')
    await user.click(screen.getByRole('button', { name: /^next$/i }))

    expect(await screen.findByText(/rollback target must be older than the current live version/i)).toBeInTheDocument()
    expect(screen.queryByText(/loading diff/i)).not.toBeInTheDocument()
  })

  it('blocks rollback diff review when no live using version is available', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        createJsonResponse({
          errcode: 0,
          data: {
            base: {
              metadata: { key: 'db.url', latestVersion: 3, unpublishedVersion: 3, contentType: 4 },
              raw: btoa('draft'),
              published: false,
            },
            compare: {
              metadata: { key: 'db.url', latestVersion: 3, unpublishedVersion: 3, contentType: 4 },
              version: 1,
              raw: btoa('older'),
              published: true,
            },
            diff: '',
          },
        }),
      ),
    )

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/rollback')

    await user.type(screen.getByRole('spinbutton', { name: /target version/i }), '4')
    await user.click(screen.getByRole('button', { name: /^next$/i }))

    expect(await screen.findByText(/unable to determine the current version for diff review/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^next$/i })).toBeEnabled()
    expect(screen.queryByText(/loading diff/i)).not.toBeInTheDocument()
  })

  it('logs out from the shell', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    renderRoute('/dashboard')

    await userEvent.click(screen.getByRole('button', { name: /account menu/i }))
    await userEvent.click(screen.getByRole('menuitem', { name: /logout/i }))

    expect(localStorage.getItem('cassem.session')).toBeNull()
    expect(localStorage.getItem('cassem.user')).toBeNull()
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
    expect(screen.queryByRole('menuitem', { name: /logout/i })).not.toBeInTheDocument()
  })
})
