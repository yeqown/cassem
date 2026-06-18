import react from '@vitejs/plugin-react'
import { loadEnv, type UserConfig } from 'vite'
import { configDefaults, defineConfig } from 'vitest/config'

const defaultAdmTarget = 'http://127.0.0.1:20218'

const coarseManualChunks = (id: string) => {
  if (!id.includes('/node_modules/')) return undefined
  if (id.includes('/@codemirror/') || id.includes('/@lezer/') || id.includes('/@uiw/react-codemirror/')) return 'vendor-editor'
  if (id.includes('/@mui/') || id.includes('/@emotion/')) return 'vendor-mui'
  if (id.includes('/react/') || id.includes('/react-dom/') || id.includes('/react-router-dom/')) return 'vendor-react'
  return undefined
}

export function buildAppConfig(command: 'serve' | 'build', env: Record<string, string | undefined>): UserConfig {
  const config: UserConfig = {
    base: command === 'serve' ? '/' : '/ui/',
    plugins: [react()],
    build: {
      outDir: '../internal/cassemadm/dist',
      emptyOutDir: true,
      assetsDir: 'assets',
      rollupOptions: {
        output: {
          manualChunks: coarseManualChunks,
        },
      },
    },
    test: {
      environment: 'jsdom',
      setupFiles: './src/test/setup.ts',
      globals: true,
      testTimeout: 10_000,
      exclude: configDefaults.exclude,
    }
  }

  if (command === 'serve') {
    config.server = {
      host: '0.0.0.0',
      proxy: {
        '/api': {
          target: env.CASSEMADM_API_TARGET || defaultAdmTarget,
          changeOrigin: true,
        },
      },
    }
  }

  return config
}

export default defineConfig(({ command, mode }) => buildAppConfig(command, loadEnv(mode, process.cwd(), '')))
