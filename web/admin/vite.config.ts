import { defineConfig } from 'vitest/config'
import { loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

// 开发模式代理目标由 .env.development 的 VITE_PROXY_TARGET 控制,默认 8080
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://localhost:8080'
  return {
    // 管理端部署在 /admin/ 路径下(与 app 端同域共存),开发模式同样从 /admin/ 访问
    base: '/admin/',
    plugins: [
      vue(),
      Components({
        dts: false,
        resolvers: [ElementPlusResolver({ importStyle: 'css' })]
      })
    ],
    server: {
      port: 5173,
      proxy: {
        '/admin-api': proxyTarget,
        '/files': proxyTarget
      }
    },
    test: {
      environment: 'jsdom',
      include: ['src/**/*.spec.ts']
    }
  }
})
