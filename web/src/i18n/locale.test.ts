import { afterEach, describe, expect, it, vi } from 'vitest'
import { createLocaleFormatters, readLocale, translate } from './locale'
import { en, zhCN, type MessageKey } from './messages'

afterEach(() => vi.restoreAllMocks())

describe('locale data', () => {
  it('keeps every translated message and its interpolation parameters complete', () => {
    expect(Object.keys(zhCN).sort()).toEqual(Object.keys(en).sort())
    for (const key of Object.keys(en) as MessageKey[]) {
      expect(zhCN[key].trim(), key).not.toBe('')
      const parameters = (message: string) => [...message.matchAll(/\{(\w+)\}/g)].map((match) => match[1]).sort()
      expect(parameters(zhCN[key]), key).toEqual(parameters(en[key]))
    }
  })

  it('interpolates values as text without translating or recursively expanding user content', () => {
    expect(translate('zh-CN', 'common.roles', { roles: 'reader-{roles}' })).toBe('角色：reader-{roles}')
    expect(translate('en', 'common.roles', { roles: 'reader-{roles}' })).toBe('Roles: reader-{roles}')
  })

  it('formats dates in the chosen locale and handles invalid timestamps', () => {
    const options: Intl.DateTimeFormatOptions = { year: 'numeric', month: 'numeric', day: 'numeric', timeZone: 'UTC' }
    expect(createLocaleFormatters('en').formatDateTime('2026-01-02T12:00:00Z', options)).toBe('1/2/2026')
    expect(createLocaleFormatters('zh-CN').formatDateTime('2026-01-02T12:00:00Z', options)).toBe('2026/1/2')
    expect(createLocaleFormatters('zh-CN').formatDateTime('invalid')).toBe('-')
  })

  it('detects browser language even when storage reads are blocked', () => {
    vi.spyOn(localStorage, 'getItem').mockImplementation(() => { throw new DOMException('Storage blocked', 'SecurityError') })
    vi.spyOn(window.navigator, 'languages', 'get').mockReturnValue(['zh-CN', 'en-US'])
    expect(readLocale()).toBe('zh-CN')
  })
})
