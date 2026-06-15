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

function createWorkflowVersions() {
  return {
    elements: [
      { metadata: { key: 'db.url', usingVersion: 2, latestVersion: 3, unpublishedVersion: 3, contentType: 4 }, raw: btoa('v1'), version: 1, published: true },
      { metadata: { key: 'db.url', usingVersion: 2, latestVersion: 3, unpublishedVersion: 3, contentType: 4 }, raw: btoa('v2'), version: 2, published: true },
      { metadata: { key: 'db.url', usingVersion: 2, latestVersion: 3, unpublishedVersion: 3, contentType: 4 }, raw: btoa('v3'), version: 3, published: false },
    ],
  }
}

function createWorkflowFetchMock(extra?: (input: RequestInfo | URL, init?: RequestInit) => Response | Promise<Response> | undefined) {
  return vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const response = extra?.(input, init)
    if (response) return Promise.resolve(response)

    const url = String(input)
    if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
      return Promise.resolve(createJsonResponse({ errcode: 0, data: createWorkflowVersions() }))
    }

    if (url.includes('/api/cluster/agents?limit=100')) {
      return Promise.resolve(createJsonResponse({ errcode: 0, data: { agents: [{ agentId: 'agent-a' }, { agentId: 'agent-b' }] } }))
    }

    if (url.includes('/api/cluster/instances/filter?app=demo&env=prod&key=db.url')) {
      return Promise.resolve(createJsonResponse({ errcode: 0, data: { instances: [{ clientId: 'instance-01', agentId: 'agent-a' }, { clientId: 'instance-02', agentId: 'agent-b' }] } }))
    }

    return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
  })
}

afterEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('app shell routing', () => {
  it('renders login when unauthenticated', () => {
    renderRoute('/login')

    expect(screen.getByTestId('login-background')).toHaveStyle({ backgroundColor: 'rgb(238, 247, 245)' })
    expect(screen.getByTestId('login-background').getAttribute('style')).toContain('login-topology')
    expect(screen.getByTestId('login-card')).toHaveAttribute('data-visual', 'glass')
    expect(screen.getByTestId('login-brand')).toHaveStyle({ flexDirection: 'row' })
    expect(screen.getByRole('img', { name: /cassem logo/i })).toHaveAttribute('src', '/logo.svg')
    expect(screen.getByRole('heading', { name: /configuration center/i })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: /cassem admin/i })).not.toBeInTheDocument()
    expect(screen.queryByText(/use your cassemadm account email/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/enter the password currently assigned/i)).not.toBeInTheDocument()
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
    expect(screen.getByRole('img', { name: /cassem logo/i })).toHaveAttribute('src', '/logo.svg')
    screen.getAllByTestId('sidebar-logo').forEach((logo) => {
      expect(logo).not.toHaveClass('MuiAvatar-root')
    })
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

  it('paginates, searches, and renders app metadata columns', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps?limit=15&query=demo&seek=next-demo')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'demo-next', description: 'Next page', createdAt: 1710003600, creator: 'bob', owner: 'team-b' }], hasMore: false } }))
      }
      if (url.includes('/api/apps?limit=15&query=demo')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'demo', description: 'Demo app', createdAt: 1710000000, creator: 'alice', owner: 'team-a' }], hasMore: true, nextSeek: 'next-demo' } }))
      }
      if (url.includes('/api/apps?limit=30&query=demo')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'demo-large', description: 'Large page', createdAt: 1710007200, creator: 'carol', owner: 'team-c' }], hasMore: false } }))
      }
      if (url.includes('/api/apps?limit=15')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'alpha', description: 'Alpha app', createdAt: 1709996400, creator: 'root', owner: 'platform' }], hasMore: false } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps')

    expect(await screen.findByText('alpha')).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: /^app id$/i })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: /created at/i })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: /creator/i })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: /owner/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^search$/i })).toHaveClass('MuiButton-contained')
    expect(within(screen.getByRole('button', { name: /^search$/i })).getByTestId('SearchIcon')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^search$/i })).toHaveStyle({ minWidth: '128px' })
    expect(screen.getByText('root')).toBeInTheDocument()
    expect(screen.getByText('platform')).toBeInTheDocument()
    expect(within(screen.getByRole('link', { name: /^envs$/i })).getByTestId('DnsIcon')).toBeInTheDocument()
    expect(within(screen.getByRole('button', { name: /^delete$/i })).getByTestId('DeleteOutlineIcon')).toBeInTheDocument()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    await user.type(screen.getByRole('textbox', { name: /search apps/i }), 'demo')
    await user.click(screen.getByRole('button', { name: /^search$/i }))

    expect(await screen.findByText('demo')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/apps?limit=15&query=demo', expect.any(Object))
    expect(screen.getByText('alice')).toBeInTheDocument()
    expect(screen.getByText('team-a')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /^next$/i }))

    expect(await screen.findByText('demo-next')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/apps?limit=15&query=demo&seek=next-demo', expect.any(Object))

    await user.click(screen.getByRole('button', { name: /^previous$/i }))

    expect(await screen.findByText('demo')).toBeInTheDocument()

    await user.click(screen.getByRole('combobox', { name: /rows per page/i }))
    await user.click(await screen.findByRole('option', { name: '30' }))

    expect(await screen.findByText('demo-large')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/apps?limit=30&query=demo', expect.any(Object))
  })

  it('keeps the latest apps response when refresh overlaps the initial load', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const initialLoad = createDeferred<Response>()
    const refreshLoad = createDeferred<Response>()
    let appsGetRequests = 0
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps?limit=15')) {
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

  it('renders envs page context through breadcrumbs only', () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    renderRoute('/apps/demo/envs')

    expect(screen.getByRole('heading', { name: /environments/i })).toBeInTheDocument()
    const breadcrumbs = screen.getByRole('navigation', { name: /breadcrumb/i })
    expect(within(breadcrumbs).getByRole('link', { name: /^demo$/i })).toBeInTheDocument()
    expect(screen.queryByText('App: demo')).not.toBeInTheDocument()
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

    expect(await screen.findByRole('cell', { name: 'demo' })).toBeInTheDocument()

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

    await waitFor(() => {
      expect(screen.getAllByText('other').some((node) => node.closest('td'))).toBe(true)
    })

    await act(async () => {
      createRequest.resolve(createJsonResponse({ errcode: 0, data: null }))
      await createRequest.promise
    })

    expect(demoEnvLoads).toBe(1)
    expect(screen.getAllByText('other').some((node) => node.closest('td'))).toBe(true)
    expect(screen.queryByText('stale-demo')).not.toBeInTheDocument()
  })

  it('renders elements page context through breadcrumbs only', async () => {
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
    const breadcrumbs = screen.getByRole('navigation', { name: /breadcrumb/i })
    expect(within(breadcrumbs).getByRole('link', { name: /^demo$/i })).toBeInTheDocument()
    expect(within(breadcrumbs).getByRole('link', { name: /^prod$/i })).toBeInTheDocument()
    expect(screen.queryByText(/app: demo/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/env: prod/i)).not.toBeInTheDocument()
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
    const breadcrumbs = screen.getByRole('navigation', { name: /breadcrumb/i })
    expect(within(breadcrumbs).getByRole('link', { name: /^demo$/i })).toBeInTheDocument()
    expect(within(breadcrumbs).getByRole('link', { name: /^prod$/i })).toBeInTheDocument()
    expect(within(breadcrumbs).getByRole('link', { name: /^db\.url$/i })).toBeInTheDocument()
    expect(screen.queryByText(/app: demo \/ env: prod \/ key: db\.url/i)).not.toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /content/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /versions/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /operations/i })).toBeInTheDocument()
  })

  it('renders element detail status in the title toolbar', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input)
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [] } }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/operations?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { operations: [] } }))
        }
        return Promise.resolve(
          createJsonResponse({
            errcode: 0,
            data: {
              metadata: { key: 'db.url', latestVersion: 2, usingVersion: 1, unpublishedVersion: 0, contentType: 4 },
              raw: btoa('postgres://demo'),
              version: 1,
              published: true,
            },
          }),
        )
      }),
    )

    renderRoute('/apps/demo/envs/prod/elements/db.url')

    expect(await screen.findByRole('heading', { name: /element detail/i })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: /^status$/i })).not.toBeInTheDocument()
    expect(screen.getByText('Latest: v2')).toBeInTheDocument()
    expect(screen.getByText('Current: v1')).toBeInTheDocument()
    expect(screen.getByText('Draft: -')).toBeInTheDocument()
    expect(screen.getByText('Type: PLAINTEXT')).toBeInTheDocument()
    expect(screen.getByTestId('element-detail-actions')).toHaveStyle({ alignSelf: 'flex-end' })
  })

  it('compares element versions with dropdowns and renders readable diff text', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const esc = String.fromCharCode(27)
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/diff')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { diff: `value-v${esc}[31m1${esc}[0m${esc}[32m2${esc}[0m` } }))
      }
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [
              { metadata: { key: 'db.url', contentType: 4 }, raw: btoa('value-v1'), version: 1, published: true },
              { metadata: { key: 'db.url', contentType: 4 }, raw: btoa('value-v2'), version: 2, published: true },
            ],
          },
        }))
      }
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/operations?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { operations: [] } }))
      }
      return Promise.resolve(
        createJsonResponse({
          errcode: 0,
          data: {
            metadata: { key: 'db.url', latestVersion: 2, usingVersion: 2, unpublishedVersion: 0, contentType: 4 },
            raw: btoa('value-v2'),
            version: 2,
            published: true,
          },
        }),
      )
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url')

    await user.click(await screen.findByRole('tab', { name: /versions/i }))
    await user.click(screen.getByRole('combobox', { name: /base/i }))
    await user.click(await screen.findByRole('option', { name: 'v1' }))
    await user.click(screen.getByRole('combobox', { name: /compare/i }))
    await user.click(await screen.findByRole('option', { name: 'v2' }))
    await user.click(screen.getByRole('button', { name: /show diff/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/apps/demo/envs/prod/elements/db.url/diff?base=1&compare=2',
        expect.any(Object),
      )
    })
    const diff = await screen.findByLabelText('Diff')
    expect(within(diff).getByText('value-v')).toBeInTheDocument()
    expect(within(diff).getByText('1')).toBeInTheDocument()
    expect(within(diff).getByText('2')).toBeInTheDocument()
    expect(diff).not.toHaveTextContent(/\[31m|\[32m|\[0m/)
  })

  it('shows operation action, actor, time, and version change in order', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input)
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [] } }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/operations?limit=100')) {
          return Promise.resolve(createJsonResponse({
            errcode: 0,
            data: {
              operations: [
                { operator: 'alice@example.com', op: 1, lastVersion: 1, currentVersion: 2, operatedAt: 1_781_259_956_333_000_000 },
                { operator: 'system', op: 'PUBLISH', lastVersion: 2, currentVersion: 2, operatedAt: 1_781_259_956_334_000_000 },
              ],
            },
          }))
        }
        return Promise.resolve(
          createJsonResponse({
            errcode: 0,
            data: {
              metadata: { key: 'db.url', latestVersion: 2, usingVersion: 2, unpublishedVersion: 0, contentType: 4 },
              raw: btoa('value-v2'),
              version: 2,
              published: true,
            },
          }),
        )
      }),
    )

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url')

    await user.click(await screen.findByRole('tab', { name: /operations/i }))
    const table = screen.getByRole('table')
    expect(within(table).getAllByRole('columnheader').map((cell) => cell.textContent)).toEqual(['Time', 'Operation', 'Operator', 'Version change'])
    expect(within(table).getByText('SET')).toBeInTheDocument()
    expect(within(table).getByText('alice@example.com')).toBeInTheDocument()
    expect(within(table).getByText('v1 → v2')).toBeInTheDocument()
    expect(within(table).getByText('PUBLISH')).toBeInTheDocument()
    expect(within(table).getByText('system')).toBeInTheDocument()
    expect(within(table).getByText('-')).toBeInTheDocument()
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

  it('renders user status with Material chips and blocks disabling disabled users', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input)
        if (url.includes('/api/account/users?limit=100')) {
          return Promise.resolve(createJsonResponse({
            errcode: 0,
            data: {
              users: [
                { account: 'alice@example.com', nickname: 'Alice', status: 0 },
                { account: 'bob@example.com', nickname: 'Bob', status: 1 },
              ],
            },
          }))
        }
        if (url.includes('/api/account/acl/domains')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { domains: [] } }))
        }
        return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
      }),
    )

    renderRoute('/users')

    expect(await screen.findByRole('heading', { name: /users/i })).toBeInTheDocument()
    const enabledRow = screen.getByText('alice@example.com').closest('tr')
    const disabledRow = screen.getByText('bob@example.com').closest('tr')

    expect(enabledRow).not.toBeNull()
    expect(disabledRow).not.toBeNull()
    expect(within(enabledRow as HTMLElement).getByText('Enabled')).toBeInTheDocument()
    expect(within(disabledRow as HTMLElement).getByText('Disabled')).toBeInTheDocument()
    expect(within(enabledRow as HTMLElement).getByRole('button', { name: /disable/i })).toBeEnabled()
    expect(within(disabledRow as HTMLElement).getByRole('button', { name: /disable/i })).toBeDisabled()
  })

  it('uses a Material confirmation dialog before disabling a user', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/account/users?limit=100')) {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: { users: [{ account: 'alice@example.com', nickname: 'Alice', status: 0 }] },
        }))
      }
      if (url.includes('/api/account/disable?account=alice%40example.com')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      if (url.includes('/api/account/acl/domains')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { domains: [] } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/users')

    const row = (await screen.findByText('alice@example.com')).closest('tr')
    expect(row).not.toBeNull()
    await user.click(within(row as HTMLElement).getByRole('button', { name: /disable/i }))

    expect(confirmSpy).not.toHaveBeenCalled()
    const dialog = await screen.findByRole('dialog', { name: /disable user/i })
    expect(within(dialog).getByText(/alice@example.com/i)).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: /^disable$/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/account/disable?account=alice%40example.com', expect.any(Object))
    })
  })

  it('uses a Material confirmation dialog before deleting an app', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (url.includes('/api/apps?limit=15')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'demo', description: 'Demo app' }] } }))
      }
      if (url.includes('/api/apps/demo') && init?.method === 'DELETE') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps')

    await screen.findByText('demo')
    await user.click(screen.getByRole('button', { name: /^delete$/i }))

    expect(confirmSpy).not.toHaveBeenCalled()
    const dialog = await screen.findByRole('dialog', { name: /delete app/i })
    expect(within(dialog).getByText(/demo/i)).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/apps/demo', expect.objectContaining({ method: 'DELETE' }))
    })
  })

  it('uses a Material confirmation dialog before deleting an environment', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod'] } }))
      }
      if (url.includes('/api/apps/demo/envs/prod') && init?.method === 'DELETE') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs')

    await screen.findByText('prod')
    await user.click(screen.getByRole('button', { name: /^delete$/i }))

    expect(confirmSpy).not.toHaveBeenCalled()
    const dialog = await screen.findByRole('dialog', { name: /delete environment/i })
    expect(within(dialog).getByText(/prod/i)).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/apps/demo/envs/prod', expect.objectContaining({ method: 'DELETE' }))
    })
  })

  it('uses a Material confirmation dialog before deleting an element', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs/prod/elements?limit=100')) {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: { elements: [{ metadata: { key: 'db.url', latestVersion: 1, usingVersion: 1, unpublishedVersion: 0, contentType: 4 } }] },
        }))
      }
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url') && init?.method === 'DELETE') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements')

    await screen.findByText('db.url')
    const row = screen.getByText('db.url').closest('tr')
    expect(row).not.toBeNull()
    const cells = within(row as HTMLElement).getAllByRole('cell')
    expect(cells[1]).toHaveTextContent('v1')
    expect(cells[2]).toHaveTextContent('v1')
    expect(cells[3]).toHaveTextContent('-')

    await user.click(screen.getByRole('button', { name: /^delete$/i }))

    expect(confirmSpy).not.toHaveBeenCalled()
    const dialog = await screen.findByRole('dialog', { name: /delete element/i })
    expect(within(dialog).getByText(/db.url/i)).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/apps/demo/envs/prod/elements/db.url', expect.objectContaining({ method: 'DELETE' }))
    })
  })

  it.each([
    {
      path: '/apps',
      current: 'Apps',
      links: [],
    },
    {
      path: '/apps/demo/envs',
      current: 'Environments',
      links: [{ name: 'Apps', href: '/ui/apps' }],
    },
    {
      path: '/apps/demo/envs/prod/elements',
      current: 'Elements',
      links: [
        { name: 'Apps', href: '/ui/apps' },
        { name: 'demo', href: '/ui/apps/demo/envs' },
      ],
    },
    {
      path: '/apps/demo/envs/prod/elements/db.url',
      current: 'Detail',
      links: [
        { name: 'Apps', href: '/ui/apps' },
        { name: 'demo', href: '/ui/apps/demo/envs' },
        { name: 'prod', href: '/ui/apps/demo/envs/prod/elements' },
      ],
    },
    {
      path: '/apps/demo/envs/prod/elements/db.url/publish',
      current: 'Publish',
      links: [
        { name: 'Apps', href: '/ui/apps' },
        { name: 'demo', href: '/ui/apps/demo/envs' },
        { name: 'prod', href: '/ui/apps/demo/envs/prod/elements' },
        { name: 'db.url', href: '/ui/apps/demo/envs/prod/elements/db.url' },
      ],
    },
    {
      path: '/apps/demo/envs/prod/elements/db.url/rollback',
      current: 'Rollback',
      links: [
        { name: 'Apps', href: '/ui/apps' },
        { name: 'demo', href: '/ui/apps/demo/envs' },
        { name: 'prod', href: '/ui/apps/demo/envs/prod/elements' },
        { name: 'db.url', href: '/ui/apps/demo/envs/prod/elements/db.url' },
      ],
    },
  ])('renders app breadcrumbs on $path', async ({ path, current, links }) => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input)
        if (url.includes('/api/apps?limit=15')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'demo', description: 'Demo app' }] } }))
        }
        if (url.includes('/api/apps/demo/envs?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod'] } }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [] } }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/operations?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { operations: [] } }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url')) {
          return Promise.resolve(createJsonResponse({
            errcode: 0,
            data: {
              metadata: { key: 'db.url', latestVersion: 1, usingVersion: 1, unpublishedVersion: 0, contentType: 4 },
              raw: btoa('postgres://demo'),
              version: 1,
              published: true,
            },
          }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [] } }))
        }
        return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
      }),
    )

    renderRoute(path)

    const breadcrumbs = await screen.findByRole('navigation', { name: /breadcrumb/i })
    for (const link of links) {
      expect(within(breadcrumbs).getByRole('link', { name: link.name })).toHaveAttribute('href', link.href)
    }
    expect(within(breadcrumbs).getByText(current)).toBeInTheDocument()
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
    expect(screen.getByRole('combobox', { name: /^app$/i })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: /^env$/i })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: /^key$/i })).toBeInTheDocument()
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

  it('cascades instance filter candidates from app to env to key', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'demo' }, { id: 'other' }] } }))
      }
      if (url.includes('/api/apps/demo/envs?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod', 'stage'] } }))
      }
      if (url.includes('/api/apps/demo/envs/prod/elements?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [{ metadata: { key: 'db.url' } }, { metadata: { key: 'feature.flag' } }] } }))
      }
      if (url.includes('/api/cluster/instances/filter?app=demo&env=prod&key=db.url')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { instances: [{ clientId: 'instance-01', agentId: 'agent-a', clientIp: '10.0.0.1' }] } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: { instances: [] } }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/cluster/instances')

    expect(await screen.findByRole('heading', { name: /instances/i })).toBeInTheDocument()
    await user.click(await screen.findByRole('combobox', { name: /^app$/i }))
    await user.click(await screen.findByRole('option', { name: /^demo$/i }))
    await user.click(await screen.findByRole('combobox', { name: /^env$/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))
    await user.click(await screen.findByRole('combobox', { name: /^key$/i }))
    await user.click(await screen.findByRole('option', { name: /^db\.url$/i }))
    await user.click(screen.getByRole('button', { name: /^filter$/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/cluster/instances/filter?app=demo&env=prod&key=db.url', expect.any(Object))
    })
    expect(await screen.findByText('instance-01')).toBeInTheDocument()
  })

  it('falls back to full instance list when filter input is incomplete', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn().mockResolvedValue(createJsonResponse({ errcode: 0, data: { instances: [] } }))
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/cluster/instances')

    expect(await screen.findByRole('heading', { name: /instances/i })).toBeInTheDocument()
    await user.type(screen.getByRole('combobox', { name: /^app$/i }), 'demo')
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
    expect(screen.getByRole('combobox', { name: /^app$/i })).toHaveValue('demo')
    expect(screen.getByRole('combobox', { name: /^env$/i })).toHaveValue('prod')
    expect(screen.getByRole('combobox', { name: /^key$/i })).toHaveValue('db.url')
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
    const disableDialog = await screen.findByRole('dialog', { name: /disable user/i })
    await user.click(within(disableDialog).getByRole('button', { name: /^disable$/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/account/disable?account=alice%40example.com', expect.any(Object))
    })
    await waitFor(() => expect(screen.queryByRole('dialog', { name: /disable user/i })).not.toBeInTheDocument())

    await user.click(screen.getByRole('button', { name: /reset password/i }))
    const dialog = await screen.findByRole('dialog', { name: /reset password/i })
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

  it('enables publishing a newer published version after rollback', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal('fetch', createWorkflowFetchMock((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
        return createJsonResponse({
          errcode: 0,
          data: {
            elements: [
              { metadata: { key: 'db.url', usingVersion: 1, latestVersion: 2, unpublishedVersion: 0, contentType: 4 }, raw: btoa('v1'), version: 1, published: true },
              { metadata: { key: 'db.url', usingVersion: 1, latestVersion: 2, unpublishedVersion: 0, contentType: 4 }, raw: btoa('v2'), version: 2, published: true },
            ],
          },
        })
      }
    }))

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    expect(screen.getByRole('heading', { name: /publish element/i })).toBeInTheDocument()
    const versionSelect = await screen.findByRole('combobox', { name: /^version$/i })
    expect(screen.queryByRole('spinbutton', { name: /^version$/i })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^next$/i })).toBeDisabled()

    await user.click(versionSelect)

    const listbox = await screen.findByRole('listbox')
    const currentPublishedOption = within(listbox).getByLabelText('v1 published')
    const newerPublishedOption = within(listbox).getByLabelText('v2 published')
    expect(currentPublishedOption).toHaveAttribute('aria-disabled', 'true')
    expect(newerPublishedOption).not.toHaveAttribute('aria-disabled', 'true')
    expect(within(currentPublishedOption).getByTestId('version-status-current')).toBeInTheDocument()
    expect(within(currentPublishedOption).getByText('current')).toBeInTheDocument()
    expect(within(currentPublishedOption).getByTestId('version-status-published')).toBeInTheDocument()
    expect(within(newerPublishedOption).getByTestId('version-status-published')).toBeInTheDocument()

    await user.click(newerPublishedOption)

    expect(screen.getByRole('button', { name: /^next$/i })).toBeEnabled()
  })

  it('renders rollback version candidates but only enables older published versions', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal('fetch', createWorkflowFetchMock())

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/rollback')

    expect(screen.getByRole('heading', { name: /rollback element/i })).toBeInTheDocument()
    const versionSelect = await screen.findByRole('combobox', { name: /target version/i })
    expect(screen.queryByRole('spinbutton', { name: /target version/i })).not.toBeInTheDocument()

    await user.click(versionSelect)

    const listbox = await screen.findByRole('listbox')
    const olderPublishedOption = within(listbox).getByLabelText('v1 published')
    const currentPublishedOption = within(listbox).getByLabelText('v2 published')
    const draftOption = within(listbox).getByLabelText('v3 draft')
    expect(olderPublishedOption).not.toHaveAttribute('aria-disabled', 'true')
    expect(currentPublishedOption).toHaveAttribute('aria-disabled', 'true')
    expect(draftOption).toHaveAttribute('aria-disabled', 'true')
    expect(within(olderPublishedOption).getByTestId('version-status-published')).toBeInTheDocument()
    expect(within(currentPublishedOption).getByTestId('version-status-current')).toBeInTheDocument()
    expect(within(currentPublishedOption).getByText('current')).toBeInTheDocument()
    expect(within(draftOption).getByTestId('version-status-draft')).toBeInTheDocument()
  })

  it('requires at least one gray publish target before allowing progression', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal('fetch', createWorkflowFetchMock())

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    await user.click(await screen.findByRole('combobox', { name: /^version$/i }))
    await user.click(await screen.findByRole('option', { name: /^v3 draft$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('radio', { name: /gray publish/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))

    const wizardActions = screen.getByTestId('wizard-actions')
    const nextButton = within(wizardActions).getByRole('button', { name: /^next$/i })

    expect(within(wizardActions).getByRole('button', { name: /^back$/i })).toBeInTheDocument()
    expect(screen.getAllByText(/^Gray publish requires at least one agent or instance target\.$/i)).toHaveLength(1)
    expect(nextButton).toBeDisabled()

    await user.click(screen.getByRole('combobox', { name: /instance ids/i }))
    await user.click(await screen.findByRole('option', { name: /^instance-01$/i }))

    expect(screen.queryByText(/^Gray publish requires at least one agent or instance target\.$/i)).not.toBeInTheDocument()
    expect(nextButton).toBeEnabled()
  })

  it('hides explicit target inputs for full publish', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal('fetch', createWorkflowFetchMock())

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    await user.click(await screen.findByRole('combobox', { name: /^version$/i }))
    await user.click(await screen.findByRole('option', { name: /^v3 draft$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))

    expect(screen.queryByRole('combobox', { name: /agent ids/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: /instance ids/i })).not.toBeInTheDocument()
    expect(screen.getByText(/full publish does not use explicit agent or instance targets/i)).toBeInTheDocument()
  })

  it('submits gray publish targets from candidate multiselects', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = createWorkflowFetchMock((input, init) => {
      if (String(input).includes('/api/apps/demo/envs/prod/elements/db.url/publish') && init?.method === 'POST') {
        return createJsonResponse({ errcode: 0, data: null })
      }
      return undefined
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    await user.click(await screen.findByRole('combobox', { name: /^version$/i }))
    await user.click(await screen.findByRole('option', { name: /^v3 draft$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('radio', { name: /gray publish/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('combobox', { name: /agent ids/i }))
    await user.click(await screen.findByRole('option', { name: /^agent-a$/i }))
    await user.click(screen.getByRole('combobox', { name: /instance ids/i }))
    await user.click(await screen.findByRole('option', { name: /^instance-01$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('button', { name: /^publish$/i }))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/apps/demo/envs/prod/elements/db.url/publish',
      expect.objectContaining({
        body: JSON.stringify({
          version: 3,
          publishMode: 1,
          agentId: ['agent-a'],
          instanceId: ['instance-01'],
        }),
      }),
    )
  })

  it('shows rollback diff loading state while fetching comparison', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const diffRequest = createDeferred<Response>()
    const fetchMock = createWorkflowFetchMock((input) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/diff?')) {
        return diffRequest.promise
      }
      return undefined
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/rollback')

    await user.click(await screen.findByRole('combobox', { name: /target version/i }))
    await user.click(await screen.findByRole('option', { name: /^v1 published$/i }))
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

  it('disables rollback progression when there are no older target candidates', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    vi.stubGlobal(
      'fetch',
      createWorkflowFetchMock((input) => {
        if (String(input).includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
          return createJsonResponse({
            errcode: 0,
            data: {
              elements: [
                { metadata: { key: 'db.url', usingVersion: 1, latestVersion: 1, unpublishedVersion: 0, contentType: 4 }, raw: btoa('v1'), version: 1, published: true },
              ],
            },
          })
        }
        return undefined
      }),
    )

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/rollback')

    const versionSelect = await screen.findByRole('combobox', { name: /target version/i })
    expect(screen.getByRole('button', { name: /^next$/i })).toBeDisabled()

    await user.click(versionSelect)

    expect(await screen.findByText(/no published rollback target version available/i)).toBeInTheDocument()
    const listbox = await screen.findByRole('listbox')
    const currentPublishedOption = within(listbox).getByLabelText('v1 published')
    expect(currentPublishedOption).toHaveAttribute('aria-disabled', 'true')
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
