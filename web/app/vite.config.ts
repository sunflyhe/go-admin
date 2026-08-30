import { defineConfig } from 'vitest/config'
import { loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

// 开发模式代理目标由 .env.development 的 VITE_PROXY_TARGET 控制,默认 8080
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://localhost:8080'
  return {
    plugins: [vue()],
    server: {
      // 5175:避开 admin 端默认的 5173,两端可同时起 dev server
      port: 5175,
      proxy: {
        '/api': proxyTarget,
        '/files': proxyTarget
      }
    },
    test: {
      environment: 'jsdom',
      include: ['src/**/*.spec.ts']
    }
  }
})
