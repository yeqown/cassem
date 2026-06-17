import { describe, expect, it } from 'vitest'
import { resolveRouterBasename } from './routerBase'

describe('resolveRouterBasename', () => {
  it('keeps root basename for standalone dev server', () => {
    expect(resolveRouterBasename('/')).toBe('/')
  })

  it('trims trailing slash for embedded production base', () => {
    expect(resolveRouterBasename('/ui/')).toBe('/ui')
  })
})
