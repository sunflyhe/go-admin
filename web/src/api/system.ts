import request from './request'
import type { ApiBody, MenuNode, Paged } from './types'

export interface UserItem {
  id: number
  username: string
  nickname: string
  email: string
  phone: string
  avatar: string
  remark: string
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
  uploadAvatar: (id: number, formData: FormData) =>
    request.put<ApiBody<{ avatar: string }>>(`/users/${id}/avatar`, formData),
  setStatus: (id: number, status: number) => request.put<ApiBody<null>>(`/users/${id}/status`, { status }),
  resetPassword: (id: number, password: string) => request.put<ApiBody<null>>(`/users/${id}/password`, { password }),
  assignRoles: (id: number, roleIds: number[]) => request.put<ApiBody<null>>(`/users/${id}/roles`, { roleIds }),
  exportUrl: '/api/v1/users/export'
}

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

export interface SysConfig {
  id: number
  name: string
  key: string
  value: string
  remark: string
  builtin: boolean
  createdAt: string
  updatedAt: string
}
export interface ConfigSaveForm {
  name: string
  key: string
  value: string
  remark: string
}
export const configApi = {
  list: (params: Record<string, unknown>) => request.get<ApiBody<Paged<SysConfig>>>('/configs', { params }),
  create: (data: ConfigSaveForm) => request.post<ApiBody<SysConfig>>('/configs', data),
  update: (id: number, data: ConfigSaveForm) => request.put<ApiBody<null>>(`/configs/${id}`, data),
  remove: (id: number) => request.delete<ApiBody<null>>(`/configs/${id}`)
}

export interface DictType {
  id: number
  name: string
  key: string
  remark: string
  builtin: boolean
  itemCount?: number
  createdAt: string
  updatedAt: string
}
export interface DictTypeSaveForm {
  name: string
  key: string
  remark: string
}
export interface DictItem {
  id: number
  typeId: number
  label: string
  description: string
  value: string
  sort: number
  tagType: string
  status: number
  remark: string
  createdAt: string
  updatedAt: string
}
export interface DictItemSaveForm {
  label: string
  description: string
  value: string
  sort: number
  tagType: string
  status: number
  remark: string
}
export interface DictOption {
  label: string
  value: string
  tagType: string
}
export const dictTypeApi = {
  list: () => request.get<ApiBody<DictType[]>>('/dict-types'),
  create: (data: DictTypeSaveForm) => request.post<ApiBody<DictType>>('/dict-types', data),
  update: (id: number, data: DictTypeSaveForm) => request.put<ApiBody<null>>(`/dict-types/${id}`, data),
  remove: (id: number) => request.delete<ApiBody<null>>(`/dict-types/${id}`),
  items: (typeId: number) => request.get<ApiBody<DictItem[]>>(`/dict-types/${typeId}/items`),
  createItem: (typeId: number, data: DictItemSaveForm) =>
    request.post<ApiBody<DictItem>>(`/dict-types/${typeId}/items`, data),
  updateItem: (id: number, data: DictItemSaveForm) => request.put<ApiBody<null>>(`/dict-items/${id}`, data),
  removeItem: (id: number) => request.delete<ApiBody<null>>(`/dict-items/${id}`)
}
export const dictDataApi = {
  byKey: (key: string) => request.get<ApiBody<DictOption[]>>('/dict-data', { params: { key } })
}
