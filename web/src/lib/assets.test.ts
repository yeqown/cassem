import { describe, expect, it } from 'vitest'
import { assetUrl } from './assets'

describe('assetUrl', () => {
  it('uses the embedded UI base for public assets', () => {
    expect(assetUrl('logo.svg', '/ui/')).toBe('/ui/logo.svg')
    expect(assetUrl('/login-topology.svg', '/ui/')).toBe('/ui/login-topology.svg')
  })

  it('keeps root asset URLs in dev', () => {
    expect(assetUrl('logo.svg', '/')).toBe('/logo.svg')
  })
})
