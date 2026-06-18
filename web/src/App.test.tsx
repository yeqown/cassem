import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { StrictMode, type ComponentProps } from 'react'
import { MemoryRouter, RouterProvider, createMemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { CopyEnvDialog } from './features/envs/CopyEnvDialog'
import { ApiError } from './lib/api'
import { AppThemeProvider } from './AppThemeProvider'
import { routes } from './routes'

function renderRoute(path: string) {
  const entry = path.startsWith('/ui') ? path : `/ui${path}`
  const router = createMemoryRouter(routes, { basename: '/ui', initialEntries: [entry] })
  render(
    <AppThemeProvider>
      <RouterProvider router={router} />
    </AppThemeProvider>,
  )
  return router
}

function renderStrictRoute(path: string) {
  const entry = path.startsWith('/ui') ? path : `/ui${path}`
  const router = createMemoryRouter(routes, { basename: '/ui', initialEntries: [entry] })
  return {
    router,
    ...render(
      <StrictMode>
        <AppThemeProvider>
          <RouterProvider router={router} />
        </AppThemeProvider>
      </StrictMode>,
    ),
  }
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })

  return { promise, resolve }
}

function renderCopyEnvDialog(props: Partial<ComponentProps<typeof CopyEnvDialog>> = {}) {
  const onClose = vi.fn()
  const onBusyChange = vi.fn()
  const onFinished = vi.fn()

  return {
    onClose,
    onBusyChange,
    onFinished,
    ...render(
      <MemoryRouter>
        <CopyEnvDialog
          open
          appId="demo"
          envs={['prod']}
          onClose={onClose}
          onBusyChange={onBusyChange}
          onFinished={onFinished}
          {...props}
        />
      </MemoryRouter>,
    ),
  }
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

    if (url.includes('/api/apps/demo/envs/prod/elements/db.url/diff?')) {
      return Promise.resolve(createJsonResponse({
        errcode: 0,
        data: {
          base: { metadata: { key: 'db.url', usingVersion: 2, latestVersion: 3, unpublishedVersion: 3, contentType: 4 }, raw: btoa('v2'), version: 2, published: true },
          compare: { metadata: { key: 'db.url', usingVersion: 2, latestVersion: 3, unpublishedVersion: 3, contentType: 4 }, raw: btoa('v3'), version: 3, published: false },
          diff: 'value-v[31m2[0m[32m3[0m',
        },
      }))
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

  it('renders settings page from sidebar and saves local preferences', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const user = userEvent.setup()
    renderRoute('/settings')

    expect(await screen.findByRole('heading', { name: /settings/i })).toBeInTheDocument()
    for (const brand of screen.getAllByTestId('sidebar-brand')) {
      expect(brand).toHaveStyle({ backgroundColor: '#2454ff' })
    }
    expect(screen.getByRole('link', { name: /settings/i })).toHaveAttribute('href', '/ui/settings')
    expect(within(screen.getByRole('navigation', { name: /navigation/i })).getAllByRole('link').at(-1)).toHaveAccessibleName('Settings')

    const editorLayout = screen.getByTestId('editor-settings-layout')
    expect(editorLayout).toHaveAttribute('data-preview-position', 'right')
    expect(within(editorLayout).getAllByTestId(/editor-settings-panel|code-theme-preview-panel/).map((panel) => panel.getAttribute('data-testid'))).toEqual([
      'editor-settings-panel',
      'code-theme-preview-panel',
    ])
    const preview = within(editorLayout).getByLabelText('Code theme preview')
    expect(preview).toHaveAttribute('data-code-theme', 'github-light-plus')
    expect(preview).toHaveTextContent('"enabled": true')

    await user.click(screen.getByRole('combobox', { name: /code theme/i }))
    await user.click(await screen.findByRole('option', { name: 'One Dark' }))
    expect(preview).toHaveAttribute('data-code-theme', 'one-dark')
    await user.click(screen.getByRole('combobox', { name: /ui theme/i }))
    await user.click(await screen.findByRole('option', { name: 'Purple' }))
    for (const brand of screen.getAllByTestId('sidebar-brand')) {
      expect(brand).toHaveStyle({ backgroundColor: '#7c3aed' })
    }
    await user.click(screen.getByRole('switch', { name: /editor line wrapping/i }))
    expect(preview.querySelector('.cm-lineWrapping')).toBeNull()

    expect(JSON.parse(localStorage.getItem('cassem.settings') || '{}')).toEqual({
      codeTheme: 'one-dark',
      uiTheme: 'purple',
      editorLineWrapping: false,
    })
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
    expect(screen.getByRole('main')).toHaveStyle({ padding: '48px' })

    await user.click(screen.getByRole('button', { name: /account menu/i }))

    expect(await screen.findByText('Roles: superadmin')).toBeInTheDocument()
    expect(screen.getAllByText('superadmin@example.com').length).toBeGreaterThan(0)
  })

  it('renders apps page inside shell', () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    renderRoute('/apps')

    expect(screen.getByRole('heading', { name: /apps/i })).toBeInTheDocument()
    expect(screen.getByTestId('apps-title-icon')).toBeInTheDocument()
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
    expect(screen.getByRole('button', { name: /^search$/i })).toHaveStyle({ minWidth: '128px', height: '40px' })
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
    expect(screen.getByText(/environments separate app configuration/i)).toBeInTheDocument()
    const breadcrumbs = screen.getByRole('navigation', { name: /breadcrumb/i })
    expect(within(breadcrumbs).getByRole('link', { name: /^apps$/i })).toBeInTheDocument()
    expect(within(breadcrumbs).queryByRole('link', { name: /^demo$/i })).not.toBeInTheDocument()
    expect(within(breadcrumbs).getByText(/^demo$/i)).toBeInTheDocument()
    expect(screen.queryByText('App: demo')).not.toBeInTheDocument()
  })

  it('suggests common environment names and creates lowercase custom names', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: [] } }))
      }
      if (url.includes('/api/apps/demo/envs/qa') && init?.method === 'POST') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs')

    await user.click(screen.getByRole('button', { name: /add environment/i }))
    const dialog = await screen.findByRole('dialog')
    const input = within(dialog).getByRole('combobox', { name: /environment/i })
    await user.click(input)

    expect(await screen.findByRole('option', { name: 'dev' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'test' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'stage' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'prod' })).toBeInTheDocument()

    await user.clear(input)
    await user.type(input, 'QA')
    await user.click(within(dialog).getByRole('button', { name: /^create$/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/apps/demo/envs/qa', expect.objectContaining({ method: 'POST' }))
    })
  })

  it('creates an env copy task with default-selected source elements and target validation', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod', 'stage'] } }))
      }
      if (url.includes('/api/apps/demo/envs/prod/elements?limit=100')) {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [
              { metadata: { key: 'db.url', usingVersion: 2, contentType: 4 }, raw: btoa('mysql://demo') },
              { metadata: { key: 'feature.flag', usingVersion: 1, contentType: 1 }, raw: btoa('{"enabled":true}') },
            ],
            hasMore: false,
          },
        }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs')

    expect(await screen.findByRole('button', { name: /^copy$/i })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /^copy$/i }))

    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })
    expect(within(dialog).getByText(/task creation/i)).toBeInTheDocument()

    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))

    expect(await within(dialog).findByRole('checkbox', { name: /db\.url/i })).toBeChecked()
    expect(within(dialog).getByRole('checkbox', { name: /feature\.flag/i })).toBeChecked()
    expect(within(dialog).getByTestId('copy-summary-selected-elements-value')).toHaveTextContent('2')

    await user.type(within(dialog).getByRole('textbox', { name: /to env/i }), 'STAGE')

    expect(within(dialog).getByText(/environment already exists/i)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /start copy/i })).toBeDisabled()

    await user.clear(within(dialog).getByRole('textbox', { name: /to env/i }))
    await user.type(within(dialog).getByRole('textbox', { name: /to env/i }), 'QA')

    expect(within(dialog).getByRole('textbox', { name: /to env/i })).toHaveValue('qa')
    expect(within(dialog).getByRole('button', { name: /start copy/i })).toBeEnabled()
  })

  it('renders optimized env copy task creation controls', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method || 'GET'

      if (method === 'GET' && url === '/api/apps/demo/envs?limit=100') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod'] } }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: Array.from({ length: 6 }, (_, index) => ({
              metadata: { key: `item-${String(index + 1).padStart(2, '0')}`, usingVersion: 1, contentType: 4 },
              raw: `value-${index + 1}`,
            })),
            hasMore: false,
          },
        }))
      }

      throw new Error(`Unhandled request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs')

    await user.click(await screen.findByRole('button', { name: /^copy$/i }))
    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })

    expect(within(dialog).getByRole('combobox', { name: /source env/i })).toHaveAttribute('aria-required', 'true')
    expect(within(dialog).getByRole('textbox', { name: /to env/i })).toBeRequired()
    expect(within(dialog).getByRole('switch', { name: /copy empty elements/i })).not.toBeChecked()

    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))

    expect(await within(dialog).findByText(/Select all elements \(6\/6\)/i)).toBeInTheDocument()
    const elementsRegion = within(dialog).getByRole('region', { name: /copy elements list/i })
    expect(elementsRegion).toHaveStyle({ maxHeight: '240px', overflowY: 'auto' })

    await user.click(within(dialog).getByRole('checkbox', { name: /^item-01$/i }))
    expect(within(dialog).getByText(/Select all elements \(5\/6\)/i)).toBeInTheDocument()

    const summary = within(dialog).getByTestId('copy-summary-grid')
    expect(summary).toHaveStyle({ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))' })

    const sourceItem = within(summary).getByTestId('copy-summary-source-env')
    expect(sourceItem).toHaveStyle({ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)' })
    expect(within(summary).getByTestId('copy-summary-source-env-label')).toHaveStyle({ fontWeight: '700' })
    expect(within(summary).getByTestId('copy-summary-source-env-value')).toHaveTextContent('prod')
  })

  it('executes an env copy task, continues after element failure, and navigates to copied env', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method || 'GET'

      if (method === 'GET' && url === '/api/apps/demo/envs?limit=100') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod'] } }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [
              { metadata: { key: 'db.url', usingVersion: 1, contentType: 4 }, raw: btoa('mysql://demo') },
              { metadata: { key: 'empty.item', usingVersion: 1, contentType: 4 }, raw: '' },
              { metadata: { key: 'bad.item', usingVersion: 1, contentType: 1 }, raw: btoa('{"bad":true}') },
            ],
            hasMore: false,
          },
        }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements/db.url') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { metadata: { key: 'db.url', usingVersion: 1, contentType: 4 }, raw: 'mysql://demo' } }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements/empty.item') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { metadata: { key: 'empty.item', usingVersion: 1, contentType: 4 }, raw: '' } }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements/bad.item') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { metadata: { key: 'bad.item', usingVersion: 1, contentType: 1 }, raw: '{"bad":true}' } }))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa/elements/db.url') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa/elements/bad.item') {
        return Promise.resolve(createJsonResponse({ errcode: 2, errmsg: 'create failed' }, 500))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }

      throw new Error(`Unhandled request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    const router = renderRoute('/apps/demo/envs')

    await user.click(await screen.findByRole('button', { name: /^copy$/i }))
    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })

    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))
    expect(await within(dialog).findByRole('checkbox', { name: /db\.url/i })).toBeChecked()

    await user.type(within(dialog).getByRole('textbox', { name: /to env/i }), 'qa')
    await user.click(within(dialog).getByRole('button', { name: /start copy/i }))

    expect(await within(dialog).findByText(/task execution/i)).toBeInTheDocument()
    expect(await within(dialog).findByRole('progressbar', { name: /copy elements progress/i })).toHaveAttribute('aria-valuenow', '100')
    expect(within(dialog).getByText(/3\/3 elements processed/i)).toBeInTheDocument()

    const resultsPanel = within(dialog).getByRole('region', { name: /copy results/i })
    expect(within(resultsPanel).getByTestId('copy-results-success-count')).toHaveTextContent('Success 1')
    expect(within(resultsPanel).getByTestId('copy-results-success-count')).toHaveStyle({ color: 'rgb(46, 125, 50)' })
    expect(within(resultsPanel).getByTestId('copy-results-skipped-count')).toHaveTextContent('Skipped 1')
    expect(within(resultsPanel).getByTestId('copy-results-skipped-count')).toHaveStyle({ color: 'rgb(97, 97, 97)' })
    expect(within(resultsPanel).getByTestId('copy-results-failed-count')).toHaveTextContent('Failed 1')
    expect(within(resultsPanel).getByTestId('copy-results-failed-count')).toHaveStyle({ color: 'rgb(211, 47, 47)' })

    const failedTab = within(resultsPanel).getByRole('tab', { name: /failed 1/i })
    expect(failedTab).toHaveAttribute('aria-selected', 'true')
    const resultsTable = within(resultsPanel).getByRole('table', { name: /copy results/i })
    expect(within(resultsTable).getByRole('columnheader', { name: /^key$/i })).toBeInTheDocument()
    expect(within(resultsTable).getByRole('columnheader', { name: /^state$/i })).toBeInTheDocument()
    expect(within(resultsTable).queryByRole('columnheader', { name: /^result$/i })).not.toBeInTheDocument()
    expect(within(resultsTable).getByRole('columnheader', { name: /^reason$/i })).toBeInTheDocument()
    expect(within(resultsTable).getByRole('row', { name: /bad\.item failed create failed/i })).toHaveStyle({ backgroundColor: 'rgba(211, 47, 47, 0.08)' })
    expect(within(resultsTable).getByText('Failed')).toHaveStyle({ fontSize: 'inherit', lineHeight: 'inherit' })
    expect(within(resultsTable).queryByRole('row', { name: /db\.url success -/i })).not.toBeInTheDocument()

    await user.click(within(resultsPanel).getByRole('tab', { name: /success 1/i }))
    expect(within(resultsTable).getByRole('row', { name: /db\.url success -/i })).toHaveStyle({ backgroundColor: 'rgba(46, 125, 50, 0.08)' })
    expect(within(resultsTable).queryByRole('row', { name: /bad\.item failed create failed/i })).not.toBeInTheDocument()

    await user.click(within(resultsPanel).getByRole('tab', { name: /skipped 1/i }))
    expect(within(resultsTable).getByRole('row', { name: /empty\.item skipped empty element skipped/i })).toHaveStyle({ backgroundColor: 'rgba(97, 97, 97, 0.08)' })
    expect(within(resultsTable).getByText('Skipped')).toHaveStyle({ color: 'rgb(97, 97, 97)' })

    await user.click(within(dialog).getByRole('button', { name: /view copied env/i }))

    expect(router.state.location.pathname).toBe('/ui/apps/demo/envs/qa/elements')
  })

  it('keeps env copy complete when refresh fails after create and copy succeed', async () => {
    const onFinished = vi.fn().mockRejectedValue(new ApiError('refresh failed', 2, 500))
    const onBusyChange = vi.fn()

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method || 'GET'

      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [{ metadata: { key: 'db.url', usingVersion: 1, contentType: 4 }, raw: btoa('mysql://demo') }],
            hasMore: false,
          },
        }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements/db.url') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { metadata: { key: 'db.url', usingVersion: 1, contentType: 4 }, raw: 'mysql://demo' } }))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa/elements/db.url') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }

      throw new Error(`Unhandled request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderCopyEnvDialog({ onBusyChange, onFinished })

    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })
    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))
    await user.type(within(dialog).getByRole('textbox', { name: /to env/i }), 'qa')
    await user.click(within(dialog).getByRole('button', { name: /start copy/i }))

    expect(await within(dialog).findByRole('progressbar', { name: /copy elements progress/i })).toHaveAttribute('aria-valuenow', '100')
    expect(within(dialog).getByText(/1\/1 elements processed/i)).toBeInTheDocument()
    const resultsPanel = within(dialog).getByRole('region', { name: /copy results/i })
    expect(within(resultsPanel).getByTestId('copy-results-success-count')).toHaveTextContent('Success 1')
    expect(within(dialog).queryByText(/failed to create target environment/i)).not.toBeInTheDocument()
    expect(within(dialog).queryByText(/refresh failed/i)).not.toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /view copied env/i })).toBeEnabled()
    expect(onFinished).toHaveBeenCalledTimes(1)
    expect(onBusyChange).toHaveBeenLastCalledWith(false)
  })

  it('disables env copy creation when estimated copy is zero', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method || 'GET'

      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [
              { metadata: { key: 'empty.raw', usingVersion: 1, contentType: 4 }, raw: '' },
              { metadata: { key: 'no.using', contentType: 4 }, raw: 'value' },
            ],
            hasMore: false,
          },
        }))
      }

      throw new Error(`Unhandled request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderCopyEnvDialog({ envs: ['prod'] })

    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })
    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))

    expect(await within(dialog).findByRole('checkbox', { name: /empty\.raw/i })).toBeChecked()
    expect(within(dialog).getByRole('switch', { name: /copy empty elements/i })).toHaveAccessibleDescription(/empty elements will be skipped/i)
    expect(within(dialog).getByTestId('copy-summary-empty-elements-value')).toHaveTextContent('2')
    expect(within(dialog).getByTestId('copy-summary-estimated-skipped-value')).toHaveTextContent('2')
    expect(within(dialog).getByTestId('copy-summary-estimated-copy-value')).toHaveTextContent('0')

    await user.type(within(dialog).getByRole('textbox', { name: /to env/i }), 'qa')

    expect(within(dialog).getByText(/Estimated copy is 0/i)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /start copy/i })).toBeDisabled()

    await user.click(within(dialog).getByRole('switch', { name: /copy empty elements/i }))

    expect(within(dialog).getByRole('switch', { name: /copy empty elements/i })).toHaveAccessibleDescription(/empty elements will be copied/i)
    expect(within(dialog).getByTestId('copy-summary-estimated-skipped-value')).toHaveTextContent('0')
    expect(within(dialog).getByTestId('copy-summary-estimated-copy-value')).toHaveTextContent('2')
    expect(within(dialog).queryByText(/Estimated copy is 0/i)).not.toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /start copy/i })).toBeEnabled()
  })

  it('records copy-empty element failures when the server rejects empty raw content', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method || 'GET'

      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [{ metadata: { key: 'empty.raw', usingVersion: 1, contentType: 4 }, raw: '' }],
            hasMore: false,
          },
        }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements/empty.raw') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { metadata: { key: 'empty.raw', usingVersion: 1, contentType: 4 }, raw: '' } }))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa/elements/empty.raw') {
        return Promise.resolve(createJsonResponse({ errcode: 3, errmsg: 'raw is required' }, 400))
      }

      throw new Error(`Unhandled request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderCopyEnvDialog({ envs: ['prod'] })

    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })
    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))
    await user.click(within(dialog).getByRole('switch', { name: /copy empty elements/i }))
    await user.type(within(dialog).getByRole('textbox', { name: /to env/i }), 'qa')
    await user.click(within(dialog).getByRole('button', { name: /start copy/i }))

    const resultsPanel = await within(dialog).findByRole('region', { name: /copy results/i })
    expect(within(resultsPanel).getByTestId('copy-results-failed-count')).toHaveTextContent('Failed 1')
    expect(within(resultsPanel).getByTestId('copy-results-failed-count')).toHaveStyle({ color: 'rgb(211, 47, 47)' })
    const resultsTable = within(resultsPanel).getByRole('table', { name: /copy results/i })
    expect(within(resultsTable).getByRole('row', { name: /empty\.raw failed raw is required/i })).toHaveStyle({ backgroundColor: 'rgba(211, 47, 47, 0.08)' })
  })

  it('loads all source elements before creating an env copy task', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method || 'GET'

      if (method === 'GET' && url === '/api/apps/demo/envs?limit=100') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod'] } }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100&seek=next-page') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [{ metadata: { key: 'second', usingVersion: 1, contentType: 4 }, raw: '2' }],
            hasMore: false,
          },
        }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [{ metadata: { key: 'first', usingVersion: 1, contentType: 4 }, raw: '1' }],
            hasMore: true,
            nextSeek: 'next-page',
          },
        }))
      }

      throw new Error(`Unhandled request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs')

    await user.click(await screen.findByRole('button', { name: /^copy$/i }))
    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })
    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))

    expect(await within(dialog).findByRole('checkbox', { name: /^first$/i })).toBeChecked()
    expect(within(dialog).getByRole('checkbox', { name: /^second$/i })).toBeChecked()
    expect(
      fetchMock.mock.calls.some(([request, requestInit]) => {
        const requestUrl = String(request)
        const requestMethod = requestInit?.method || 'GET'
        return requestMethod === 'GET' && requestUrl === '/api/apps/demo/envs/prod/elements?limit=100&seek=next-page'
      }),
    ).toBe(true)
  })

  it('disables env page mutations while copy execution is running', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const createEnvRequest = createDeferred<Response>()
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method || 'GET'

      if (method === 'GET' && url === '/api/apps/demo/envs?limit=100') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod'] } }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [{ metadata: { key: 'db.url', usingVersion: 1, contentType: 4 }, raw: 'value' }],
            hasMore: false,
          },
        }))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa') {
        return createEnvRequest.promise
      }

      throw new Error(`Unhandled request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs')

    const copyButton = await screen.findByRole('button', { name: /^copy$/i })
    const addButton = screen.getByRole('button', { name: /add environment/i })
    const row = screen.getByRole('row', { name: /prod/i })
    const deleteButton = within(row).getByRole('button', { name: /^delete$/i })

    await user.click(copyButton)
    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })
    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))
    await user.type(within(dialog).getByRole('textbox', { name: /to env/i }), 'qa')
    await user.click(within(dialog).getByRole('button', { name: /start copy/i }))

    expect(await within(dialog).findByText(/^Create env$/i)).toBeInTheDocument()
    expect(within(dialog).getByText(/^doing$/i)).toBeInTheDocument()
    await waitFor(() => {
      expect(copyButton).toBeDisabled()
      expect(addButton).toBeDisabled()
      expect(deleteButton).toBeDisabled()
      expect(within(dialog).getByRole('button', { name: /^close$/i })).toBeDisabled()
    })

    await act(async () => {
      createEnvRequest.resolve(createJsonResponse({ errcode: 2, errmsg: 'create env failed' }, 500))
      await createEnvRequest.promise
    })

    expect(await within(dialog).findByText(/create env failed/i)).toBeInTheDocument()
    await waitFor(() => {
      expect(copyButton).toBeEnabled()
    })
  })

  it('posts decoded raw content when copying elements', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method || 'GET'

      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [{ metadata: { key: 'db.url', usingVersion: 1, contentType: 4 }, raw: btoa('mysql://demo') }],
            hasMore: false,
          },
        }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements/db.url') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { metadata: { key: 'db.url', usingVersion: 1, contentType: 4 }, raw: btoa('mysql://demo') } }))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa/elements/db.url') {
        expect(init?.body).toBe(JSON.stringify({ raw: 'mysql://demo', contentType: 4 }))
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }

      throw new Error(`Unhandled request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderCopyEnvDialog()

    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })
    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))
    await user.type(within(dialog).getByRole('textbox', { name: /to env/i }), 'qa')
    await user.click(within(dialog).getByRole('button', { name: /start copy/i }))

    expect(await within(dialog).findByRole('progressbar', { name: /copy elements progress/i })).toHaveAttribute('aria-valuenow', '100')
    expect(within(dialog).getByText(/1\/1 elements processed/i)).toBeInTheDocument()
    const resultsPanel = within(dialog).getByRole('region', { name: /copy results/i })
    expect(within(resultsPanel).getByTestId('copy-results-success-count')).toHaveTextContent('Success 1')
    expect(within(resultsPanel).getByTestId('copy-results-failed-count')).toHaveTextContent('Failed 0')
  })

  it('clears busy state when the copy dialog unmounts mid-copy', async () => {
    const detailRequest = createDeferred<Response>()
    const onBusyChange = vi.fn()

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method || 'GET'

      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements?limit=100') {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [{ metadata: { key: 'db.url', usingVersion: 1, contentType: 4 }, raw: btoa('mysql://demo') }],
            hasMore: false,
          },
        }))
      }
      if (method === 'GET' && url === '/api/apps/demo/envs/prod/elements/db.url') {
        return detailRequest.promise
      }
      if (method === 'POST' && url === '/api/apps/demo/envs/qa') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }

      throw new Error(`Unhandled request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    const { unmount } = renderCopyEnvDialog({ onBusyChange })

    const dialog = await screen.findByRole('dialog', { name: /copy environment/i })
    await user.click(within(dialog).getByRole('combobox', { name: /source env/i }))
    await user.click(await screen.findByRole('option', { name: /^prod$/i }))
    await user.type(within(dialog).getByRole('textbox', { name: /to env/i }), 'qa')
    await user.click(within(dialog).getByRole('button', { name: /start copy/i }))

    await waitFor(() => expect(onBusyChange).toHaveBeenCalledWith(true))
    unmount()

    expect(onBusyChange).toHaveBeenLastCalledWith(false)

    await act(async () => {
      detailRequest.resolve(createJsonResponse({ errcode: 0, data: { metadata: { key: 'db.url', usingVersion: 1, contentType: 4 }, raw: 'mysql://demo' } }))
      await detailRequest.promise
    })
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
    expect(screen.getByText(/elements are versioned configuration entries/i)).toBeInTheDocument()
    const breadcrumbs = screen.getByRole('navigation', { name: /breadcrumb/i })
    expect(within(breadcrumbs).getByRole('link', { name: /^apps$/i })).toBeInTheDocument()
    expect(within(breadcrumbs).getByRole('link', { name: /^demo$/i })).toBeInTheDocument()
    expect(within(breadcrumbs).queryByRole('link', { name: /^prod$/i })).not.toBeInTheDocument()
    expect(within(breadcrumbs).getByText(/^prod$/i)).toBeInTheDocument()
    expect(screen.queryByText(/app: demo/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/env: prod/i)).not.toBeInTheDocument()
  })

  it('keeps element rows scrolling inside the table card', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(createJsonResponse({
        errcode: 0,
        data: {
          elements: Array.from({ length: 16 }, (_, index) => ({
            metadata: { key: `row-${index + 1}`, latestVersion: 1, usingVersion: 1, unpublishedVersion: 0, contentType: 4 },
          })),
          hasMore: false,
        },
      })),
    )

    renderRoute('/apps/demo/envs/prod/elements')

    expect(await screen.findByText('row-16')).toBeInTheDocument()
    expect(screen.getByTestId('elements-table-scroll')).toHaveStyle({ maxHeight: '560px', overflowY: 'auto' })
  })

  it('aligns the elements search button with the input control', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(createJsonResponse({ errcode: 0, data: { elements: [], hasMore: false } })),
    )

    renderRoute('/apps/demo/envs/prod/elements')

    expect(await screen.findByRole('heading', { name: /elements/i })).toBeInTheDocument()
    const searchInput = screen.getByRole('textbox', { name: /search elements/i })
    const searchButton = screen.getByRole('button', { name: /^search$/i })
    const clearButton = screen.getByRole('button', { name: /^clear$/i })
    const pageSizeSelect = screen.getByRole('combobox', { name: /rows per page/i })
    expect(searchInput.closest('.MuiInputBase-root')).toHaveClass('MuiInputBase-sizeSmall')
    expect(searchButton).toHaveClass('MuiButton-contained')
    expect(searchButton).toHaveStyle({ minWidth: '128px', height: '40px' })
    expect(clearButton).toBeInTheDocument()
    expect(pageSizeSelect.closest('.MuiInputBase-root')).toHaveClass('MuiInputBase-sizeSmall')
    expect(screen.getAllByText('Rows per page').length).toBeGreaterThan(0)
  })

  it('searches elements by fuzzy query and appends more results', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs/prod/elements?limit=15&query=db&seek=db.pool')) {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [{ metadata: { key: 'db.password', latestVersion: 3, usingVersion: 2, unpublishedVersion: 0, contentType: 4 } }],
            hasMore: false,
          },
        }))
      }
      if (url.includes('/api/apps/demo/envs/prod/elements?limit=15&query=db')) {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            elements: [{ metadata: { key: 'db.url', latestVersion: 1, usingVersion: 1, unpublishedVersion: 0, contentType: 4 } }],
            hasMore: true,
            nextSeek: 'db.pool',
          },
        }))
      }
      if (url.includes('/api/apps/demo/envs/prod/elements?limit=15')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [], hasMore: false } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements')

    expect(await screen.findByRole('heading', { name: /elements/i })).toBeInTheDocument()
    await user.type(screen.getByRole('textbox', { name: /search elements/i }), 'db')
    await user.click(screen.getByRole('button', { name: /^search$/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/apps/demo/envs/prod/elements?limit=15&query=db', expect.any(Object))
    })
    expect(await screen.findByText('db.url')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /load more/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/apps/demo/envs/prod/elements?limit=15&query=db&seek=db.pool', expect.any(Object))
    })
    expect(await screen.findByText('db.password')).toBeInTheDocument()
    expect(screen.getByText('db.url')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /load more/i })).not.toBeInTheDocument()
  })

  it('changes the elements page size and reloads from the first page', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs/prod/elements?limit=30')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [{ metadata: { key: 'limit-30' } }], hasMore: false } }))
      }
      if (url.includes('/api/apps/demo/envs/prod/elements?limit=15')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [{ metadata: { key: 'limit-15' } }], hasMore: false } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements')

    expect(await screen.findByText('limit-15')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/apps/demo/envs/prod/elements?limit=15', expect.any(Object))

    await user.click(screen.getByRole('combobox', { name: /rows per page/i }))
    await user.click(await screen.findByRole('option', { name: '30' }))

    expect(await screen.findByText('limit-30')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/apps/demo/envs/prod/elements?limit=30', expect.any(Object))
  })

  it('renders element detail page with fixed content, versions, and operations tabs', async () => {
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

      if (url.includes('/api/admin/retention')) {
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            enabled: true,
            keepVersionCount: 20,
            keepVersionDays: 30,
            keepOperationDays: 180,
            versionPolicy: 'Versions keep current, draft, latest 20, and versions from the last 30 days.',
            operationPolicy: 'Operation logs keep 180 days.',
          },
        }))
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

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url')

    expect(await screen.findByRole('heading', { name: /element detail/i })).toBeInTheDocument()
    const breadcrumbs = screen.getByRole('navigation', { name: /breadcrumb/i })
    expect(within(breadcrumbs).getByRole('link', { name: /^demo$/i })).toBeInTheDocument()
    expect(within(breadcrumbs).getByRole('link', { name: /^prod$/i })).toBeInTheDocument()
    expect(within(breadcrumbs).queryByRole('link', { name: /^db\.url$/i })).not.toBeInTheDocument()
    expect(within(breadcrumbs).getByText(/^db\.url$/i)).toBeInTheDocument()
    expect(screen.queryByText(/app: demo \/ env: prod \/ key: db\.url/i)).not.toBeInTheDocument()
    expect(screen.getByLabelText('Current content')).toHaveTextContent('postgres://demo')
    expect(screen.queryByRole('tab', { name: /content/i })).not.toBeInTheDocument()
    await user.click(screen.getByRole('tab', { name: /versions/i }))
    expect(await screen.findByText('Versions keep current, draft, latest 20, and versions from the last 30 days.')).toBeInTheDocument()
    await user.click(screen.getByRole('tab', { name: /operations/i }))
    expect(await screen.findByText('Operation logs keep 180 days.')).toBeInTheDocument()
  })

  it('loads element detail when retention policy is unavailable', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'appdeveloper@example.com' }))

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
        if (url.includes('/api/admin/retention')) {
          return Promise.resolve(createJsonResponse({ errcode: 7, errmsg: 'permission denied' }, 403))
        }
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            metadata: { key: 'db.url', latestVersion: 1, usingVersion: 1, unpublishedVersion: 0, contentType: 4 },
            raw: btoa('postgres://demo'),
            version: 1,
            published: true,
          },
        }))
      }),
    )

    renderRoute('/apps/demo/envs/prod/elements/db.url')

    expect(await screen.findByRole('heading', { name: /element detail/i })).toBeInTheDocument()
    expect(screen.getByText('Latest: v1')).toBeInTheDocument()
    expect(screen.queryByText(/Versions keep current/i)).not.toBeInTheDocument()
  })

  it('loads element detail while retention policy is still pending', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    const retentionRequest = createDeferred<Response>()

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
        if (url.includes('/api/admin/retention')) {
          return retentionRequest.promise
        }
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            metadata: { key: 'db.url', latestVersion: 1, usingVersion: 1, unpublishedVersion: 0, contentType: 4 },
            raw: btoa('postgres://demo'),
            version: 1,
            published: true,
          },
        }))
      }),
    )

    renderRoute('/apps/demo/envs/prod/elements/db.url')

    expect(await screen.findByRole('heading', { name: /element detail/i })).toBeInTheDocument()
    expect(screen.getByText('Latest: v1')).toBeInTheDocument()
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
    expect(within(screen.getByTestId('element-detail-actions')).getByRole('button', { name: /new version/i })).toBeInTheDocument()
    expect(within(screen.getByTestId('element-detail-actions')).getByRole('link', { name: /publish/i })).toBeInTheDocument()
    expect(within(screen.getByTestId('element-detail-actions')).getByRole('link', { name: /rollback/i })).toBeInTheDocument()
  })

  it('keeps current content above the tabs and removes the content tab', async () => {
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
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            metadata: { key: 'db.url', latestVersion: 2, usingVersion: 1, unpublishedVersion: 0, contentType: 4 },
            raw: btoa('postgres://demo'),
            version: 1,
            published: true,
          },
        }))
      }),
    )

    renderRoute('/apps/demo/envs/prod/elements/db.url')

    const content = await screen.findByLabelText('Current content')
    const tablist = screen.getByRole('tablist', { name: /element detail tabs/i })
    expect(content).toHaveTextContent('postgres://demo')
    expect(screen.queryByRole('tab', { name: /content/i })).not.toBeInTheDocument()
    expect(within(tablist).getAllByRole('tab').map((tab) => tab.textContent)).toEqual(['Versions', 'Diff', 'Operations', 'Instances'])
  })

  it('applies saved code theme to element detail viewers and editors', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    localStorage.setItem('cassem.settings', JSON.stringify({ codeTheme: 'one-dark', editorLineWrapping: false }))

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
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            metadata: { key: 'db.url', latestVersion: 2, usingVersion: 1, unpublishedVersion: 0, contentType: 'JSON' },
            raw: btoa('{"enabled":true}'),
            version: 1,
            published: true,
          },
        }))
      }),
    )

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url')

    const currentContent = await screen.findByLabelText('Current content')
    expect(currentContent).toHaveAttribute('data-code-theme', 'one-dark')
    expect(currentContent.querySelector('.cm-lineWrapping')).toBeNull()

    await user.click(screen.getByRole('button', { name: /new version/i }))

    const editor = await screen.findByTestId('content-editor')
    expect(editor).toHaveAttribute('data-code-theme', 'one-dark')
    expect(editor.querySelector('.cm-lineWrapping')).toBeNull()
  })

  it('opens new version editing in a dialog and submits content updates', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [] } }))
      }
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/operations?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { operations: [] } }))
      }
      if (init?.method === 'PUT') {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: null }))
      }
      return Promise.resolve(createJsonResponse({
        errcode: 0,
        data: {
          metadata: { key: 'db.url', latestVersion: 2, usingVersion: 1, unpublishedVersion: 0, contentType: 4 },
          raw: btoa('postgres://demo'),
          version: 1,
          published: true,
        },
      }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url')

    await user.click(await screen.findByRole('button', { name: /new version/i }))
    const dialog = await screen.findByRole('dialog', { name: /new version/i })
    expect(within(dialog).getByLabelText('New version content')).toHaveTextContent('postgres://demo')

    await user.click(within(dialog).getByRole('button', { name: /submit/i }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/apps/demo/envs/prod/elements/db.url',
        expect.objectContaining({ method: 'PUT' }),
      )
    })
  })

  it('blocks new version creation when a draft already exists', async () => {
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
      return Promise.resolve(createJsonResponse({
        errcode: 0,
        data: {
          metadata: { key: 'db.url', latestVersion: 3, usingVersion: 2, unpublishedVersion: 3, contentType: 4 },
          raw: btoa('postgres://demo'),
          version: 2,
          published: true,
        },
      }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url')

    await user.click(await screen.findByRole('button', { name: /new version/i }))

    expect(await screen.findByText('Draft v3 already exists. Publish or rollback it before creating a new version.')).toBeInTheDocument()
    expect(screen.queryByRole('dialog', { name: /new version/i })).not.toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalledWith(
      '/api/apps/demo/envs/prod/elements/db.url',
      expect.objectContaining({ method: 'PUT' }),
    )
  })

  it('renders version history state, current marker, truncated preview, and full preview dialog', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const longPreview = 'checkout.enabled=true\ncheckout.rollout=50\ncheckout.regions=us-east,eu-west,ap-south\ncheckout.owner=payments'
    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input)
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
          return Promise.resolve(createJsonResponse({
            errcode: 0,
            data: {
              elements: [
                { metadata: { key: 'db.url', usingVersion: 2, contentType: 4 }, raw: btoa('postgres://demo'), version: 1, published: true },
                { metadata: { key: 'db.url', usingVersion: 2, contentType: 4 }, raw: btoa(longPreview), version: 2, published: true },
                { metadata: { key: 'db.url', usingVersion: 2, contentType: 4 }, raw: btoa('draft-value'), version: 3, published: false },
              ],
            },
          }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/operations?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { operations: [] } }))
        }
        return Promise.resolve(createJsonResponse({
          errcode: 0,
          data: {
            metadata: { key: 'db.url', latestVersion: 3, usingVersion: 2, unpublishedVersion: 3, contentType: 4 },
            raw: btoa(longPreview),
            version: 2,
            published: true,
          },
        }))
      }),
    )

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url')

    await user.click(await screen.findByRole('tab', { name: /versions/i }))

    expect(screen.getByRole('columnheader', { name: /state/i })).toBeInTheDocument()
    const currentRow = screen.getByTestId('version-row-2')
    expect(within(currentRow).getByText('v2')).toHaveStyle({ fontWeight: '700' })
    expect(within(currentRow).getByText('(current)')).toBeInTheDocument()
    expect(within(currentRow).getByText('Published')).toBeInTheDocument()
    expect(within(currentRow).queryByText(/Published.*current/i)).not.toBeInTheDocument()
    expect(within(screen.getByTestId('version-row-3')).getByText('Draft')).toBeInTheDocument()
    expect(screen.getAllByTestId('CheckCircleOutlineIcon').length).toBeGreaterThan(0)
    expect(screen.getAllByTestId('RadioButtonUncheckedIcon').length).toBeGreaterThan(0)
    expect(screen.getByTestId('version-preview-2')).toHaveStyle({ textOverflow: 'ellipsis' })

    await user.click(within(currentRow).getByRole('button', { name: /preview v2/i }))

    const dialog = await screen.findByRole('dialog', { name: /version v2 preview/i })
    const preview = within(dialog).getByLabelText('Version preview content')
    expect(preview).toHaveTextContent('checkout.enabled=true')
    expect(preview).toHaveTextContent('checkout.owner=payments')
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

    await user.click(await screen.findByRole('tab', { name: /diff/i }))
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
    expect(diff).toHaveAttribute('data-variant', 'split')
    expect(diff).not.toHaveTextContent(/\[31m|\[32m|\[0m/)

    const row = within(diff).getByTestId('diff-row-1')
    expect(row).toHaveAttribute('data-left-tone', 'removed')
    expect(row).toHaveAttribute('data-right-tone', 'added')
    expect(within(row).getAllByText('value-v')).toHaveLength(2)
    expect(within(row).getByText('1', { selector: 'span' })).toBeInTheDocument()
    expect(within(row).getByText('2', { selector: 'span' })).toBeInTheDocument()
  })

  it('shows empty diff message when compared element versions have no changes', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/apps/demo/envs/prod/elements/db.url/diff')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { diff: '' } }))
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
      return Promise.resolve(createJsonResponse({
        errcode: 0,
        data: {
          metadata: { key: 'db.url', latestVersion: 2, usingVersion: 2, unpublishedVersion: 0, contentType: 4 },
          raw: btoa('value-v2'),
          version: 2,
          published: true,
        },
      }))
    })
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url')

    await user.click(await screen.findByRole('tab', { name: /diff/i }))
    await user.click(screen.getByRole('button', { name: /show diff/i }))

    const diff = await screen.findByLabelText('Diff')
    expect(diff).toHaveAttribute('data-variant', 'split')
    expect(diff).toHaveTextContent('No differences returned for this comparison.')
    expect(screen.queryByText('Select two versions to compare.')).not.toBeInTheDocument()
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

  it('blocks privileged actions for superadmin users in the users list', async () => {
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
                { account: 'root@example.com', nickname: 'Root', status: 0, roles: ['superadmin'], bindingCount: 1 },
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

    const superadminRow = (await screen.findByText('root@example.com')).closest('tr')
    expect(superadminRow).not.toBeNull()
    expect(within(superadminRow as HTMLElement).getByRole('button', { name: /manage access/i })).toBeDisabled()
    expect(within(superadminRow as HTMLElement).getByRole('button', { name: /disable/i })).toBeDisabled()
    expect(within(superadminRow as HTMLElement).getByRole('button', { name: /reset password/i })).toBeDisabled()
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
      if (url.includes('/api/apps/demo/envs/prod/elements?limit=15')) {
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
        if (url.includes('/api/apps/demo/envs/prod/elements?limit=15')) {
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

  it('renders topology page with refresh control', async () => {
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

    renderRoute('/cluster/topology')

    expect(await screen.findByRole('heading', { name: /^topology$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument()
  })

  it('renders instances page with filter, lastSeen, and detail controls', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.spyOn(Date, 'now').mockReturnValue(1_700_000_065_000)

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        createJsonResponse({
          errcode: 0,
          data: {
            instances: [
              {
                clientId: 'instance-01',
                agentId: 'agent-a',
                clientIp: '10.0.0.1',
                lastRenewTimestamp: 1_700_000_000,
                targets: [
                  { app: 'demo', env: 'prod', key: 'db_url' },
                  { app: 'demo', env: 'prod', key: 'feature_flag' },
                ],
              },
              {
                clientId: 'instance-02',
                agentId: 'agent-b',
                clientIp: '10.0.0.2',
                targets: [
                  { app: 'test', env: 'default', key: 'ele1' },
                  { app: 'test', env: 'default', key: 'config' },
                ],
              },
            ],
          },
        }),
      ),
    )

    renderRoute('/cluster/instances')

    expect(await screen.findByRole('heading', { name: /instances/i })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: /^app$/i })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: /^env$/i })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: /^key$/i })).toBeInTheDocument()
    const filterRow = screen.getByTestId('instances-filter-row')
    expect(within(filterRow).getByRole('combobox', { name: /^app$/i })).toBeInTheDocument()
    expect(within(filterRow).getByRole('combobox', { name: /^env$/i })).toBeInTheDocument()
    expect(within(filterRow).getByRole('combobox', { name: /^key$/i })).toBeInTheDocument()
    expect(within(filterRow).getByRole('button', { name: /filter/i })).toBeInTheDocument()
    expect(within(filterRow).getByRole('button', { name: /refresh all/i })).toBeInTheDocument()

    expect(screen.getByRole('columnheader', { name: /targets/i })).toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: /^app$/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: /^env$/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: /^key$/i })).not.toBeInTheDocument()

    const instanceRow = await screen.findByRole('row', { name: /instance-01/i })
    expect(within(instanceRow).getAllByRole('link', { name: 'demo' })).toHaveLength(2)
    expect(within(instanceRow).getAllByRole('link', { name: 'prod' })).toHaveLength(2)
    expect(within(instanceRow).getByRole('link', { name: 'db_url' })).toHaveAttribute('href', '/ui/apps/demo/envs/prod/elements/db_url')
    expect(within(instanceRow).getByRole('link', { name: 'feature_flag' })).toHaveAttribute(
      'href',
      '/ui/apps/demo/envs/prod/elements/feature_flag',
    )
    expect(within(instanceRow).getByText('1m5s ago')).toBeInTheDocument()
    expect(within(instanceRow).getByRole('button', { name: /detail/i })).toBeInTheDocument()

    const joinedKeyRow = await screen.findByRole('row', { name: /instance-02/i })
    expect(within(joinedKeyRow).getAllByRole('link', { name: 'test' })).toHaveLength(2)
    expect(within(joinedKeyRow).getAllByRole('link', { name: 'default' })).toHaveLength(2)
    expect(within(joinedKeyRow).getByRole('link', { name: 'ele1' })).toHaveAttribute('href', '/ui/apps/test/envs/default/elements/ele1')
    expect(within(joinedKeyRow).getByRole('link', { name: 'config' })).toHaveAttribute('href', '/ui/apps/test/envs/default/elements/config')
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

    renderRoute('/cluster/topology')

    expect(await screen.findByText('agent-1')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-agent-1')).getByText('Agent')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-agent-1')).getByLabelText('Healthy')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-agent-1')).queryByText(/Addr:/i)).not.toBeInTheDocument()
  })

  it('loads cluster topology once under strict mode', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn(() =>
      Promise.resolve(
        createJsonResponse({
          errcode: 0,
          data: {
            dbs: [],
            agents: [{ agentId: 'agent-a', addr: '10.0.1.1:2030', ip: '10.0.1.1', health: 'healthy' }],
            instances: [],
          },
        }),
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    renderStrictRoute('/cluster/topology')

    expect(await screen.findByText('agent-a')).toBeInTheDocument()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))
  })

  it('loads initial page data once under strict mode', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const cases = [
      {
        path: '/apps/demo/envs',
        readyText: 'prod',
        urls: ['/api/apps/demo/envs?limit=100'],
      },
      {
        path: '/apps/demo/envs/prod/elements',
        readyText: 'db.url',
        urls: ['/api/apps/demo/envs/prod/elements?limit=15'],
      },
      {
        path: '/apps/demo/envs/prod/elements/db.url',
        readyText: 'Element detail',
        urls: [
          '/api/apps/demo/envs/prod/elements/db.url',
          '/api/apps/demo/envs/prod/elements/db.url/versions?limit=100',
          '/api/apps/demo/envs/prod/elements/db.url/operations?limit=100',
          '/api/admin/retention',
        ],
      },
      {
        path: '/users',
        readyText: 'alice@example.com',
        urls: ['/api/account/users?limit=100', '/api/account/acl/domains'],
      },
      {
        path: '/cluster/instances?app=demo&env=prod&key=db.url',
        readyText: 'instance-01',
        urls: [
          '/api/apps?limit=100',
          '/api/apps/demo/envs?limit=100',
          '/api/apps/demo/envs/prod/elements?limit=100',
          '/api/cluster/instances/filter?app=demo&env=prod&key=db.url',
        ],
      },
    ]

    for (const testCase of cases) {
      vi.restoreAllMocks()
      const fetchMock = vi.fn((input: RequestInfo | URL) => {
        const url = String(input)
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/versions?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [] } }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url/operations?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { operations: [] } }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements/db.url')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { metadata: { key: 'db.url', latestVersion: 1, usingVersion: 1, unpublishedVersion: 0, contentType: 4 }, raw: btoa('value') } }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [{ metadata: { key: 'db.url' } }] } }))
        }
        if (url.includes('/api/apps/demo/envs/prod/elements?limit=15')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { elements: [{ metadata: { key: 'db.url', latestVersion: 1, usingVersion: 1, unpublishedVersion: 0, contentType: 4 } }], hasMore: false } }))
        }
        if (url.includes('/api/apps/demo/envs?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { envs: ['prod'] } }))
        }
        if (url.includes('/api/apps?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { apps: [{ id: 'demo' }] } }))
        }
        if (url.includes('/api/cluster/instances/filter?app=demo&env=prod&key=db.url')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { instances: [{ clientId: 'instance-01', app: 'demo', env: 'prod', key: 'db.url' }] } }))
        }
        if (url.includes('/api/account/users?limit=100')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { users: [{ account: 'alice@example.com', nickname: 'Alice' }] } }))
        }
        if (url.includes('/api/account/acl/domains')) {
          return Promise.resolve(createJsonResponse({ errcode: 0, data: { domains: ['cluster'] } }))
        }
        return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
      })
      vi.stubGlobal('fetch', fetchMock)

      const { unmount } = renderStrictRoute(testCase.path)

      expect(await screen.findByText(testCase.readyText)).toBeInTheDocument()
      await waitFor(() => {
        testCase.urls.forEach((url) => {
          expect(fetchMock.mock.calls.filter(([input]) => String(input) === url)).toHaveLength(1)
        })
      })
      unmount()
    }
  })

  it('renders cluster topology with dbs, agents, instance aggregation, and health states', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/cluster/topology')) {
        return Promise.resolve(
          createJsonResponse({
            errcode: 0,
            data: {
              dbs: [
                { id: 'db-1', addr: '10.0.0.1:2021', ip: '10.0.0.1', health: 'healthy' },
                { id: 'db-2', addr: '10.0.0.2:2021', ip: '10.0.0.2', health: 'offline' },
              ],
              agents: [
                { agentId: 'agent-a', addr: '10.0.1.1:2030', ip: '10.0.1.1', health: 'healthy' },
                { agentId: 'agent-b', addr: '10.0.1.2:2030', ip: '10.0.1.2', health: 'unhealthy' },
              ],
              instances: [
                { clientId: 'instance-01', agentId: 'agent-a', clientIp: '10.0.2.1', health: 'healthy' },
                { clientId: 'instance-02', agentId: 'agent-a', clientIp: '10.0.2.2', health: 'healthy' },
                { clientId: 'instance-03', agentId: 'agent-a', clientIp: '10.0.2.3', health: 'healthy' },
                { clientId: 'instance-04', agentId: 'agent-a', clientIp: '10.0.2.4', health: 'healthy' },
                { clientId: 'instance-05', agentId: 'agent-a', clientIp: '10.0.2.5', health: 'healthy' },
                { clientId: 'instance-06', agentId: 'agent-a', clientIp: '10.0.2.6', health: 'healthy' },
                { clientId: 'instance-07', agentId: 'agent-b', clientIp: '10.0.2.7', health: 'offline' },
              ],
            },
          }),
        )
      }
      if (url.includes('/api/cluster/agents?limit=100')) {
        return Promise.resolve(createJsonResponse({ errcode: 0, data: { agents: [] } }))
      }
      return Promise.resolve(createJsonResponse({ errcode: 0, data: {} }))
    })
    vi.stubGlobal('fetch', fetchMock)

    renderRoute('/cluster/topology')

    expect(await screen.findByRole('heading', { name: /cluster topology/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /^topology$/i })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/cluster/topology', expect.any(Object))
    expect(screen.getByTestId('topology-graph')).toBeInTheDocument()
    expect(screen.getByTestId('topology-dbs')).toBeInTheDocument()
    expect(screen.getByTestId('topology-agents')).toBeInTheDocument()
    expect(screen.getByTestId('topology-instances')).toBeInTheDocument()
    expect(screen.getByTestId('topology-edge-db-1-agent-a')).toHaveAttribute('aria-label', 'db-1 connects to agent-a')
    expect(screen.getByTestId('topology-edge-db-1-agent-a')).toHaveAttribute('stroke-dasharray', '10 8')
    expect(screen.getByTestId('topology-edge-agent-a-instance-group-agent-a')).toHaveAttribute('aria-label', 'agent-a connects to 6 instances')
    expect(screen.getByTestId('topology-edge-agent-b-instance-07')).toHaveAttribute('aria-label', 'agent-b connects to instance-07')
    expect(screen.getByText('db-1')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-db-1')).getByText('DB')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-db-1')).queryByText(/IP:/i)).not.toBeInTheDocument()
    expect(screen.getByText('agent-a')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-agent-a')).getByText('Agent')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-agent-a')).queryByText(/IP:/i)).not.toBeInTheDocument()
    expect(screen.getByText('6 instances')).toBeInTheDocument()
    expect(screen.queryByText('instance-01')).not.toBeInTheDocument()
    expect(screen.getByText('instance-07')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-db-1')).getByLabelText('Healthy')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-agent-b')).getByLabelText('Unhealthy')).toBeInTheDocument()
    expect(within(screen.getByTestId('topology-node-instance-07')).getByLabelText('Offline')).toBeInTheDocument()
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
              instances: [{ id: 'instance-01@10.0.0.1', clientId: 'instance-01', agentId: 'agent-a', clientIp: '10.0.0.1' }],
            },
          }),
        )
      }

      if (url.includes('/api/cluster/instances/detail/instance-01%4010.0.0.1')) {
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
    expect(screen.queryByRole('option', { name: /super admin/i })).not.toBeInTheDocument()
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

  it('cancels publish workflow after confirmation and returns to element detail', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal('fetch', createWorkflowFetchMock())

    const user = userEvent.setup()
    const router = renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    expect(await screen.findByRole('combobox', { name: /^version$/i })).toBeInTheDocument()
    expect(screen.getByTestId('wizard-title-actions')).toHaveStyle({ justifyContent: 'space-between' })

    await user.click(await screen.findByRole('button', { name: /^cancel$/i }))

    expect(screen.getByRole('dialog', { name: /cancel workflow/i })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/ui/apps/demo/envs/prod/elements/db.url/publish')

    await user.click(screen.getByRole('button', { name: /^confirm cancel$/i }))

    expect(router.state.location.pathname).toBe('/ui/apps/demo/envs/prod/elements/db.url')
  })

  it('cancels rollback workflow after confirmation and returns to element detail', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal('fetch', createWorkflowFetchMock())

    const user = userEvent.setup()
    const router = renderRoute('/apps/demo/envs/prod/elements/db.url/rollback')

    expect(await screen.findByRole('combobox', { name: /target version/i })).toBeInTheDocument()
    expect(screen.getByTestId('wizard-title-actions')).toHaveStyle({ justifyContent: 'space-between' })

    await user.click(await screen.findByRole('button', { name: /^cancel$/i }))

    expect(screen.getByRole('dialog', { name: /cancel workflow/i })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/ui/apps/demo/envs/prod/elements/db.url/rollback')

    await user.click(screen.getByRole('button', { name: /^confirm cancel$/i }))

    expect(router.state.location.pathname).toBe('/ui/apps/demo/envs/prod/elements/db.url')
  })

  it('keeps workflow title and form content compact while aligned with the stepper visual width', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal('fetch', createWorkflowFetchMock())

    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    expect(await screen.findByRole('combobox', { name: /^version$/i })).toBeInTheDocument()
    expect(screen.getByTestId('wizard-title-actions')).toHaveStyle({ maxWidth: '1080px', marginLeft: 'auto', marginRight: 'auto' })
    expect(screen.getByTestId('wizard-surface')).toHaveStyle({ maxWidth: '1080px', marginLeft: 'auto', marginRight: 'auto' })
    expect(screen.getByTestId('wizard-stepper-frame')).toHaveStyle({ maxWidth: '1080px' })
    expect(screen.getByTestId('wizard-content-frame')).toHaveStyle({ maxWidth: '1080px', paddingLeft: 'calc(100% / 12)', paddingRight: 'calc(100% / 12)' })
    expect(screen.getByTestId('wizard-actions-frame')).toHaveStyle({ maxWidth: '1080px', paddingLeft: 'calc(100% / 12)', paddingRight: 'calc(100% / 12)' })
  })

  it('aligns publish impact confirmation fields in a compact grid', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
    vi.stubGlobal('fetch', createWorkflowFetchMock())

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    await user.click(await screen.findByRole('combobox', { name: /^version$/i }))
    await user.click(await screen.findByRole('option', { name: /^v3 draft$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(await screen.findByRole('button', { name: /^next$/i }))

    expect(screen.getByRole('heading', { name: /impact confirmation/i })).toBeInTheDocument()
    expect(screen.getByTestId('publish-impact-grid')).toHaveStyle({ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))' })
    expect(screen.getByTestId('publish-impact-app')).toBeInTheDocument()
    expect(screen.getByTestId('publish-impact-env')).toBeInTheDocument()
    expect(screen.getByTestId('publish-impact-key')).toBeInTheDocument()
    expect(screen.getByTestId('publish-impact-version')).toBeInTheDocument()
    expect(screen.getByTestId('publish-impact-strategy')).toBeInTheDocument()
    expect(screen.getByTestId('publish-impact-agent-ids')).toBeInTheDocument()
    expect(screen.getByTestId('publish-impact-instance-ids')).toBeInTheDocument()
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

  it('shows publish review diff before impact confirmation', async () => {
    localStorage.setItem('cassem.session', 'session')
    localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))

    const fetchMock = createWorkflowFetchMock()
    vi.stubGlobal('fetch', fetchMock)

    const user = userEvent.setup()
    renderRoute('/apps/demo/envs/prod/elements/db.url/publish')

    await user.click(await screen.findByRole('combobox', { name: /^version$/i }))
    await user.click(await screen.findByRole('option', { name: /^v3 draft$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))
    await user.click(screen.getByRole('button', { name: /^next$/i }))

    const diff = await screen.findByLabelText('Diff')
    expect(diff).toHaveAttribute('data-variant', 'split')
    expect(diff).not.toHaveTextContent(/\[31m|\[32m|\[0m/)
    expect(screen.getByRole('heading', { name: /review diff/i })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/apps/demo/envs/prod/elements/db.url/diff?base=0&compare=3',
      expect.any(Object),
    )

    await user.click(screen.getByRole('button', { name: /^next$/i }))

    expect(screen.getByRole('heading', { name: /impact confirmation/i })).toBeInTheDocument()
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
    expect(await screen.findByRole('heading', { name: /review diff/i })).toBeInTheDocument()
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
  }, 30000)

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

    const diff = await screen.findByLabelText('Diff')
    expect(diff).toHaveAttribute('data-variant', 'split')
    expect(diff).toHaveTextContent('changed')
    expect(screen.queryByDisplayValue('changed')).not.toBeInTheDocument()
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
