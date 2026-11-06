import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { afterEach, expect, it, vi } from 'vitest'
import { AppThemeProvider } from '../AppThemeProvider'
import { ToastProvider } from '../components/ToastProvider'
import { routes } from '../routes'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  localStorage.clear()
})

it('preserves the publish step, version, and gray targets when switching language without reloading workflow data', async () => {
  localStorage.setItem('cassem.session', 'session')
  localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
  localStorage.setItem('cassem.locale', 'en')
  const responses: Record<string, unknown> = {
    '/api/apps/demo/envs/prod/elements/db.url/versions?limit=100': {
      elements: [
        { metadata: { key: 'db.url', usingVersion: 1, latestVersion: 2, unpublishedVersion: 2, contentType: 4 }, raw: btoa('old value'), version: 1, published: true },
        { metadata: { key: 'db.url', usingVersion: 1, latestVersion: 2, unpublishedVersion: 2, contentType: 4 }, raw: btoa('new value'), version: 2, published: false },
      ],
    },
    '/api/cluster/agents?limit=100': { agents: [{ agentId: 'agent-a' }] },
    '/api/cluster/instances/filter?app=demo&env=prod&key=db.url': { instances: [{ clientId: 'instance-01', agentId: 'agent-a' }] },
  }
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input)
    if (!Object.hasOwn(responses, url)) throw new Error(`Unexpected request: ${url}`)
    return new Response(JSON.stringify({ errcode: 0, data: responses[url] }), { status: 200 })
  })
  vi.stubGlobal('fetch', fetchMock)
  const path = '/apps/demo/envs/prod/elements/db.url/publish'
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  render(
    <AppThemeProvider>
      <ToastProvider><RouterProvider router={router} /></ToastProvider>
    </AppThemeProvider>,
  )
  const user = userEvent.setup()
  await user.click(await screen.findByRole('combobox', { name: 'Version' }))
  await user.click(await screen.findByRole('option', { name: 'v2 draft' }))
  await user.click(screen.getByRole('button', { name: 'Next' }))
  await user.click(screen.getByRole('radio', { name: /Gray publish/ }))
  await user.click(screen.getByRole('button', { name: 'Next' }))
  await user.click(screen.getByRole('combobox', { name: 'Agent IDs' }))
  await user.click(await screen.findByRole('option', { name: 'agent-a' }))
  await user.click(screen.getByRole('combobox', { name: 'Instance IDs' }))
  await user.click(await screen.findByRole('option', { name: 'instance-01' }))
  expect(fetchMock).toHaveBeenCalledTimes(3)

  await user.click(screen.getByRole('button', { name: 'Language' }))
  await user.click(screen.getByRole('menuitemradio', { name: '简体中文' }))
  expect(screen.getByRole('heading', { name: '指定发布目标' })).toBeInTheDocument()
  expect(screen.getByText('agent-a')).toBeInTheDocument()
  expect(screen.getByText('instance-01')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '下一步' })).toBeEnabled()
  expect(router.state.location.pathname).toBe(path)
  expect(fetchMock).toHaveBeenCalledTimes(3)

  await user.click(screen.getByRole('button', { name: '下一步' }))
  expect(await screen.findByRole('heading', { name: '检查差异' })).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: '下一步' }))
  expect(screen.getByTestId('publish-impact-version')).toHaveTextContent('v2 草稿')
  expect(screen.getByTestId('publish-impact-strategy')).toHaveTextContent('灰度发布')
  expect(screen.getByTestId('publish-impact-agent-ids')).toHaveTextContent('agent-a')
  expect(screen.getByTestId('publish-impact-instance-ids')).toHaveTextContent('instance-01')
  expect(fetchMock).toHaveBeenCalledTimes(3)
}, 15000)
