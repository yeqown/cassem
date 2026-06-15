import { describe, expect, it } from 'vitest'
import { buildAppConfig } from './vite.config'

describe('buildAppConfig', () => {
  it('uses root base and API proxy for dev server mode', () => {
    const config = buildAppConfig('serve', { CASSEMADM_API_TARGET: 'http://127.0.0.1:20218' })

    expect(config.base).toBe('/')
    expect(config.server).toMatchObject({
      host: '0.0.0.0',
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:20218',
          changeOrigin: true,
        },
      },
    })
  })

  it('uses embedded /ui/ base for build mode', () => {
    const config = buildAppConfig('build', {})

    expect(config.base).toBe('/ui/')
    expect(config.server).toBeUndefined()
  })
})
