import react from '@vitejs/plugin-react'
import { loadEnv, type UserConfig } from 'vite'
import { configDefaults, defineConfig } from 'vitest/config'

const defaultAdmTarget = 'http://127.0.0.1:20218'

export function buildAppConfig(command: 'serve' | 'build', env: Record<string, string | undefined>): UserConfig {
  const config: UserConfig = {
    base: command === 'serve' ? '/' : '/ui/',
    plugins: [react()],
    build: {
      outDir: '../ui/dist',
      emptyOutDir: true,
      assetsDir: 'assets',
    },
    test: {
      environment: 'jsdom',
      setupFiles: './src/test/setup.ts',
      globals: true,
      testTimeout: 10_000,
      exclude: [...configDefaults.exclude, 'vite.config.test.ts'],
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
