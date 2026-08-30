// 统一请求封装:携带 token、自动刷新 401、刷新失败统一退出。
import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'

// API 基础地址:默认空(同源相对路径);前后端分域部署时在 .env 设置 VITE_API_BASE_URL 覆盖
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

const request = axios.create({ baseURL: API_BASE_URL, timeout: 30000 })

let refreshing: Promise<string | null> | null = null

function clearAndRedirect() {
  localStorage.removeItem('accessToken')
  localStorage.removeItem('refreshToken')
  if (!location.pathname.startsWith('/login')) {
    location.href = '/login'
  }
}

async function doRefresh(): Promise<string | null> {
  const refreshToken = localStorage.getItem('refreshToken')
  if (!refreshToken) return null
  try {
    const { data } = await axios.post(`${API_BASE_URL}/auth/refresh`, { refreshToken })
    if (data.code === 0) {
      const { accessToken, refreshToken: newRefresh } = data.data
      localStorage.setItem('accessToken', accessToken)
      localStorage.setItem('refreshToken', newRefresh)
      return accessToken
    }
  } catch {
    /* 刷新失败 */
  }
  return null
}

request.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('accessToken')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

declare module 'axios' {
  export interface AxiosRequestConfig {
    /** 静默失败:不弹全局 toast,由调用方自行兜底(用于缩略图等后台批量请求) */
    silentError?: boolean
  }
}

request.interceptors.response.use(
  (res) => res, // 保留 axios 响应结构,调用方以 res.data 取统一响应体
  async (err: AxiosError<{ code?: number; message?: string }>) => {
    const resp = err.response
    if (resp && resp.status === 401) {
      const original = err.config as InternalAxiosRequestConfig & { _retried?: boolean }
      // 登录接口本身的 401 直接提示,不刷新
      if (original.url?.includes('/auth/login') || original._retried) {
        clearAndRedirect()
        return Promise.reject(err)
      }
      original._retried = true
      if (!refreshing) {
        refreshing = doRefresh().finally(() => {
          refreshing = null
        })
      }
      const newToken = await refreshing
      if (newToken) {
        original.headers.Authorization = `Bearer ${newToken}`
        return request(original)
      }
      clearAndRedirect()
      return Promise.reject(err)
    }
    if (!err.config?.silentError) {
      ElMessage.error(resp?.data?.message || '网络异常,请稍后再试')
    }
    return Promise.reject(err)
  }
)

// 统一响应体
export interface ApiBody<T = unknown> {
  code: number
  message: string
  data: T
}

export default request
