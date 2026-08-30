// 统一请求封装:统一解包后端响应体。
// 后端统一响应结构:{ code: 0 成功; 非 0 为业务错误码, message 为文案, data 为载荷 }
import axios, { AxiosError } from 'axios'

// API 基础地址:默认空(同源相对路径);前后端分域部署时在 .env 设置 VITE_API_BASE_URL 覆盖
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

export interface ApiEnvelope<T = unknown> {
  code: number
  message: string
  data: T
}

const request = axios.create({ baseURL: API_BASE_URL, timeout: 30000 })

request.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// 统一处理 401:清掉本地凭据后跳登录页。refresh token 轮换待接入认证时再补。
request.interceptors.response.use(
  (res) => res,
  (err: AxiosError) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('accessToken')
      localStorage.removeItem('refreshToken')
      if (!location.pathname.startsWith('/login')) {
        location.href = '/login'
      }
    }
    return Promise.reject(err)
  }
)

/** 发起请求并解包统一响应体;业务非 0 码抛出带 message 的错误 */
export async function api<T>(config: Parameters<typeof request.request>[0]): Promise<T> {
  const res = await request.request<ApiEnvelope<T>>(config)
  if (res.data.code !== 0) {
    throw new Error(res.data.message || '请求失败')
  }
  return res.data.data
}

export default request
