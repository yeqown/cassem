/// <reference types="node" />

import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const webRoot = __dirname
const publicDir = resolve(webRoot, 'public')

describe('pwa icons', () => {
  it('declares favicon, apple touch icon, and manifest links', () => {
    const html = readFileSync(resolve(webRoot, 'index.html'), 'utf8')

    expect(html).toContain('<link rel="icon" type="image/svg+xml" href="/logo.svg" />')
    expect(html).toContain('<link rel="apple-touch-icon" href="/apple-touch-icon.png" />')
    expect(html).toContain('<link rel="manifest" href="/manifest.webmanifest" />')
  })

  it('publishes svg and png icons for install surfaces', () => {
    expect(existsSync(resolve(publicDir, 'logo.svg'))).toBe(true)
    expect(existsSync(resolve(publicDir, 'apple-touch-icon.png'))).toBe(true)
    expect(existsSync(resolve(publicDir, 'icon-192.png'))).toBe(true)
    expect(existsSync(resolve(publicDir, 'icon-512.png'))).toBe(true)
  })

  it('uses product topology tones for the login background', () => {
    const topology = readFileSync(resolve(publicDir, 'login-topology.svg'), 'utf8')

    expect(topology).toContain('#40A798')
    expect(topology).toContain('#476269')
    expect(topology).not.toContain('#1976d2')
    expect(topology).not.toContain('#42a5f5')
  })

  it('describes pwa icons in the web manifest', () => {
    const manifest = JSON.parse(readFileSync(resolve(publicDir, 'manifest.webmanifest'), 'utf8'))

    expect(manifest.name).toBe('Cassem Admin')
    expect(manifest.icons).toEqual([
      { src: 'logo.svg', type: 'image/svg+xml', sizes: 'any', purpose: 'any' },
      { src: 'icon-192.png', type: 'image/png', sizes: '192x192', purpose: 'any maskable' },
      { src: 'icon-512.png', type: 'image/png', sizes: '512x512', purpose: 'any maskable' },
    ])
  })
})
