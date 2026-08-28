// 统一请求封装:携带 token、自动刷新 401、刷新失败统一退出。
import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'

const request = axios.create({ baseURL: '/api/v1', timeout: 30000 })

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
    const { data } = await axios.post('/api/v1/auth/refresh', { refreshToken })
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
    const msg = resp?.data?.message || '网络异常,请稍后再试'
    ElMessage.error(msg)
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
