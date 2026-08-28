import request from './request'
import type { ApiBody } from './request'
export type { ApiBody }

export interface MenuNode {
  id: number
  parentId: number
  name: string
  type: number
  path: string
  component: string
  icon: string
  sort: number
  children?: MenuNode[]
}

export interface UserProfile {
  id: number
  username: string
  nickname: string
  email: string
  phone: string
  avatar: string
  roles: string[]
  super: boolean
}

export interface MeResp {
  user: UserProfile
  permissions: string[]
  menus: MenuNode[]
}

export interface Paged<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

export function login(username: string, password: string) {
  return request.post<ApiBody<{ accessToken: string; refreshToken: string; user: UserProfile }>>('/auth/login', { username, password })
}

export function logout() {
  return request.post<ApiBody<null>>('/auth/logout')
}

export function getMe() {
  return request.get<ApiBody<MeResp>>('/auth/me')
}

// ---- 用户管理 ----

export interface UserItem {
  id: number
  username: string
  nickname: string
  email: string
  phone: string
  status: number
  lastLoginAt: string | null
  createdAt: string
  roleIds: number[]
  super: boolean
}

export const userApi = {
  list: (params: Record<string, unknown>) => request.get<ApiBody<Paged<UserItem>>>('/users', { params }),
  create: (data: Record<string, unknown>) => request.post<ApiBody<UserItem>>('/users', data),
  update: (id: number, data: Record<string, unknown>) => request.put<ApiBody<UserItem>>(`/users/${id}`, data),
  remove: (id: number) => request.delete<ApiBody<null>>(`/users/${id}`),
  setStatus: (id: number, status: number) => request.put<ApiBody<null>>(`/users/${id}/status`, { status }),
  resetPassword: (id: number, password: string) => request.put<ApiBody<null>>(`/users/${id}/password`, { password }),
  assignRoles: (id: number, roleIds: number[]) => request.put<ApiBody<null>>(`/users/${id}/roles`, { roleIds }),
  exportUrl: '/api/v1/users/export'
}

// ---- 角色管理 ----

export interface RoleItem {
  id: number
  name: string
  code: string
  description: string
  builtin: boolean
  status: number
  userCount: number
  menuIds?: number[]
}

export const roleApi = {
  list: (params: Record<string, unknown>) => request.get<ApiBody<Paged<RoleItem>>>('/roles', { params }),
  create: (data: Record<string, unknown>) => request.post<ApiBody<RoleItem>>('/roles', data),
  update: (id: number, data: Record<string, unknown>) => request.put<ApiBody<RoleItem>>(`/roles/${id}`, data),
  remove: (id: number) => request.delete<ApiBody<null>>(`/roles/${id}`),
  menus: (id: number) => request.get<ApiBody<number[]>>(`/roles/${id}/menus`),
  assignMenus: (id: number, menuIds: number[]) => request.put<ApiBody<null>>(`/roles/${id}/menus`, { menuIds })
}

// ---- 菜单管理 ----

export interface MenuRow {
  id: number
  parentId: number
  name: string
  type: number
  path: string
  component: string
  permission: string
  icon: string
  sort: number
  status: number
  createdAt: string
  updatedAt: string
}

export const menuApi = {
  list: () => request.get<ApiBody<MenuRow[]>>('/menus'),
  tree: () => request.get<ApiBody<MenuNode[]>>('/menus/tree'),
  create: (data: Record<string, unknown>) => request.post<ApiBody<MenuRow>>('/menus', data),
  update: (id: number, data: Record<string, unknown>) => request.put<ApiBody<MenuRow>>(`/menus/${id}`, data),
  remove: (id: number) => request.delete<ApiBody<null>>(`/menus/${id}`)
}

// ---- 日志 ----

export interface LoginLog {
  id: number
  username: string
  success: boolean
  failReason: string
  ip: string
  userAgent: string
  createdAt: string
}

export interface AuditLog {
  id: number
  userId: number
  username: string
  method: string
  path: string
  status: number
  latencyMs: number
  ip: string
  userAgent: string
  requestSummary: string
  responseSummary: string
  createdAt: string
}

export const logApi = {
  loginLogs: (params: Record<string, unknown>) => request.get<ApiBody<Paged<LoginLog>>>('/login-logs', { params }),
  auditLogs: (params: Record<string, unknown>) => request.get<ApiBody<Paged<AuditLog>>>('/audit-logs', { params })
}

// ---- 文件 ----

export interface FileRow {
  id: number
  originName: string
  storePath: string
  size: number
  mime: string
  ext: string
  isPublic: boolean
  uploader: string
  createdAt: string
}

export const fileApi = {
  list: (params: Record<string, unknown>) => request.get<ApiBody<Paged<FileRow>>>('/files', { params }),
  remove: (id: number) => request.delete<ApiBody<null>>(`/files/${id}`)
}
