import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitest/config'

const root = fileURLToPath(new URL('.', import.meta.url))

export default defineConfig({
  root,
  test: {
    environment: 'node',
    globals: true,
    include: ['vite.config.test.ts', 'pwa.test.ts'],
  },
})
