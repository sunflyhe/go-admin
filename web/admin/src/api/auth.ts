import request from './request'
import type { ApiBody, MenuNode } from './types'

export interface UserProfile {
  id: number
  username: string
  nickname: string
  email: string
  phone: string
  avatar: string
  signature: string
  roles: string[]
  /** 角色显示名,与 roles 一一对应 */
  roleNames: string[]
  super: boolean
}

export interface MeResp {
  user: UserProfile
  permissions: string[]
  menus: MenuNode[]
}

export interface ProfileForm {
  nickname: string
  email: string
  phone: string
  signature: string
}

export function login(username: string, password: string) {
  // silentError:登录失败的展示交给登录页的常驻错误条,不走全局 toast
  return request.post<ApiBody<{ accessToken: string; refreshToken: string; user: UserProfile }>>(
    '/auth/login',
    { username, password },
    { silentError: true }
  )
}

export function logout() {
  return request.post<ApiBody<null>>('/auth/logout')
}

export function getMe() {
  return request.get<ApiBody<MeResp>>('/auth/me')
}

export function updateProfile(data: ProfileForm) {
  return request.put<ApiBody<null>>('/auth/profile', data)
}

export function changePassword(oldPassword: string, newPassword: string) {
  return request.post<ApiBody<null>>('/auth/password', { oldPassword, newPassword })
}

export function uploadAvatar(file: File) {
  const form = new FormData()
  form.append('file', file)
  return request.post<ApiBody<{ avatar: string }>>('/auth/avatar', form)
}
