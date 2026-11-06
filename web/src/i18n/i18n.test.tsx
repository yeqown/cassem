import { act, cleanup, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AppThemeProvider } from '../AppThemeProvider'
import { ToastProvider } from '../components/ToastProvider'
import { routes } from '../routes'

function renderPage(path = '/settings') {
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  return {
    router,
    ...render(
      <AppThemeProvider>
        <ToastProvider><RouterProvider router={router} /></ToastProvider>
      </AppThemeProvider>,
    ),
  }
}

beforeEach(() => {
  vi.spyOn(window.navigator, 'languages', 'get').mockReturnValue(['en-US'])
  localStorage.setItem('cassem.session', 'session')
  localStorage.setItem('cassem.user', JSON.stringify({ account: 'superadmin@example.com' }))
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  localStorage.clear()
})

describe('interface language', () => {
  it('switches the current page and navigation from the control before the account icon, then restores the choice on reload', async () => {
    const user = userEvent.setup()
    const page = renderPage()
    const languageButton = screen.getByRole('button', { name: 'Language' })
    const accountButton = screen.getByRole('button', { name: 'account menu' })
    expect(languageButton.compareDocumentPosition(accountButton) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()

    await user.click(languageButton)
    await user.click(screen.getByRole('menuitemradio', { name: '简体中文' }))
    expect(screen.getByRole('heading', { name: '设置', level: 1 })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '使用指南' })).toBeInTheDocument()
    expect(document.documentElement).toHaveAttribute('lang', 'zh-CN')
    expect(page.router.state.location.pathname).toBe('/settings')

    page.unmount()
    renderPage()
    expect(screen.getByRole('heading', { name: '设置', level: 1 })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '语言' }))
    await user.click(screen.getByRole('menuitemradio', { name: 'English' }))
    expect(screen.getByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    expect(document.documentElement).toHaveAttribute('lang', 'en')
  })

  it('uses browser Chinese on first visit, while a saved English preference wins', () => {
    vi.spyOn(window.navigator, 'languages', 'get').mockReturnValue(['zh-CN', 'en-US'])
    const page = renderPage()
    expect(screen.getByRole('heading', { name: '设置', level: 1 })).toBeInTheDocument()
    page.unmount()
    localStorage.setItem('cassem.locale', 'en')
    renderPage()
    expect(screen.getByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
  })

  it('preserves a draft and the open dialog while switching language', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ errcode: 0, data: { apps: [] } }) }))
    const user = userEvent.setup()
    renderPage('/apps')
    await user.click(await screen.findByRole('button', { name: /add app/i }))
    const dialog = screen.getByRole('dialog')
    const input = within(dialog).getByRole('textbox', { name: /app id/i })
    await user.type(input, 'my-untouched-app')

    act(() => {
      localStorage.setItem('cassem.locale', 'zh-CN')
      window.dispatchEvent(new StorageEvent('storage', { key: 'cassem.locale', newValue: 'zh-CN' }))
    })
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(input).toHaveValue('my-untouched-app')
    expect(within(screen.getByRole('dialog')).getByRole('button', { name: '取消' })).toBeInTheDocument()
  })

  it('supports switching with the keyboard and returns focus to the trigger', async () => {
    const user = userEvent.setup()
    renderPage('/docs')
    screen.getByRole('button', { name: 'Language' }).focus()
    await user.keyboard('{Enter}')
    expect(screen.getByRole('menuitemradio', { name: 'English' })).toHaveAttribute('aria-checked', 'true')
    await user.keyboard('{ArrowDown}{Enter}')
    expect(screen.getByRole('heading', { name: 'Cassem 使用指南', level: 1 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '语言' })).toHaveFocus()
  })

  it('keeps switching usable when storage writes are blocked', async () => {
    const user = userEvent.setup()
    renderPage('/docs')
    vi.spyOn(localStorage, 'setItem').mockImplementation(() => { throw new DOMException('Storage blocked', 'SecurityError') })
    await user.click(screen.getByRole('button', { name: 'Language' }))
    await user.click(screen.getByRole('menuitemradio', { name: '简体中文' }))
    expect(screen.getByRole('heading', { name: 'Cassem 使用指南', level: 1 })).toBeInTheDocument()
  })

  it('falls back to English for unsupported saved and browser languages', () => {
    localStorage.setItem('cassem.locale', 'invalid')
    vi.spyOn(window.navigator, 'languages', 'get').mockReturnValue(['fr-FR'])
    renderPage('/docs')
    expect(screen.getByRole('heading', { name: 'Cassem Guide', level: 1 })).toBeInTheDocument()
    expect(document.documentElement).toHaveAttribute('lang', 'en')
  })

  it('lets visitors switch the login form without losing credentials they have typed', async () => {
    localStorage.removeItem('cassem.session')
    localStorage.removeItem('cassem.user')
    const user = userEvent.setup()
    renderPage('/login')
    await user.type(screen.getByRole('textbox', { name: 'Account' }), 'reader@example.com')
    await user.click(screen.getByRole('button', { name: 'Language' }))
    await user.click(screen.getByRole('menuitemradio', { name: '简体中文' }))
    expect(screen.getByRole('textbox', { name: '账号' })).toHaveValue('reader@example.com')
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument()
  })

  it('uses the current language when a request started before switching later fails', async () => {
    let rejectRequest!: (reason: Error) => void
    vi.stubGlobal('fetch', vi.fn(() => new Promise((_resolve, reject) => { rejectRequest = reject })))
    const user = userEvent.setup()
    renderPage('/apps')
    await user.click(screen.getByRole('button', { name: 'Language' }))
    await user.click(screen.getByRole('menuitemradio', { name: '简体中文' }))
    await act(async () => rejectRequest(new TypeError('Network unavailable')))
    expect(await screen.findByRole('alert')).toHaveTextContent('加载应用失败')
  })
})
