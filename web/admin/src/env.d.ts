/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** 开发模式接口代理目标(vite dev server 使用) */
  readonly VITE_PROXY_TARGET?: string
  /** API 基础地址,默认 /api/v1(同源);前后端分域部署时设置完整地址 */
  readonly VITE_API_BASE_URL?: string
}
