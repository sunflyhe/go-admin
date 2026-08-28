import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

// 开发模式代理目标由 .env.development 的 VITE_PROXY_TARGET 控制,默认 8080
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://localhost:8080'
  return {
    plugins: [vue()],
    server: {
      port: 5173,
      proxy: {
        '/api': proxyTarget,
        '/files': proxyTarget
      }
    }
  }
})
