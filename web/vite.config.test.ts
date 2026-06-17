// @vitest-environment node

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

  it('keeps production chunks coarse-grained', () => {
    const config = buildAppConfig('build', {})
    const output = config.build?.rollupOptions?.output
    const manualChunks = Array.isArray(output) ? undefined : output?.manualChunks

    expect(typeof manualChunks).toBe('function')
    if (typeof manualChunks !== 'function') return

    const chunkNames = new Set([
      manualChunks('/repo/web/node_modules/react/index.js', { getModuleInfo: () => null, getModuleIds: function* () {} }),
      manualChunks('/repo/web/node_modules/@mui/material/Button/index.js', { getModuleInfo: () => null, getModuleIds: function* () {} }),
      manualChunks('/repo/web/node_modules/@codemirror/view/dist/index.js', { getModuleInfo: () => null, getModuleIds: function* () {} }),
      manualChunks('/repo/web/src/routes.tsx', { getModuleInfo: () => null, getModuleIds: function* () {} }),
    ].filter(Boolean))

    expect([...chunkNames].sort()).toEqual(['vendor-editor', 'vendor-mui', 'vendor-react'])
  })
})
