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
  signature: string
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

// ---- 个人中心(只作用于当前登录者,入参无 id)----

export interface ProfileForm {
  nickname: string
  email: string
  phone: string
  signature: string
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

// ---- 用户管理 ----

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
  uploadAvatar: (id: number, formData: FormData) => request.put<ApiBody<{ avatar: string }>>(`/users/${id}/avatar`, formData),
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
  groupId: number
  isPublic: boolean
  uploader: string
  createdAt: string
}

// 取值由后端裁决(api/internal/service/file.go normalizeCategory):
// image 来自上传白名单的图片归类,video 按 mime 前缀判定(当前白名单不含音视频,该档恒空),
// file 是前两者的补集。document/archive/other 是旧的上传归类,已不再对外。
export type FileCategory = 'all' | 'image' | 'video' | 'file'

export interface FileListParams {
  page: number
  pageSize: number
  originName?: string
  category?: FileCategory
  // 不传=全部分组;传 0=未分组。两者语义不同,所以这里必须是可选参数而不是默认 0。
  groupId?: number
}

export interface FileUploadResult {
  id: number
  originName: string
  storePath: string
  url: string
  size: number
  mime: string
}

/** 分组行。列表接口附带 count(按 category 限定),写接口只回传分组本身。 */
export interface FileGroup {
  id: number
  name: string
  count?: number
}

/** 左栏需要的完整计数:分组各自的数量 + 未分组 + 全部。三者都限定在查询所用的 category 内。 */
export interface FileGroupTree {
  groups: FileGroup[]
  unfiled: number
  total: number
}

export const fileApi = {
  list: (params: FileListParams) => request.get<ApiBody<Paged<FileRow>>>('/files', { params }),
  remove: (id: number) => request.delete<ApiBody<null>>(`/files/${id}`),
  move: (ids: number[], groupId: number) => request.put<ApiBody<null>>('/files/group', { ids, groupId }),
  // 批量删除用 POST:带请求体的 DELETE 在网关与访问日志里都不友好
  batchRemove: (ids: number[]) => request.post<ApiBody<null>>('/files/batch-delete', { ids }),
  upload: (file: File, isPublic = false, groupId = 0) => {
    const form = new FormData()
    form.append('file', file)
    form.append('isPublic', String(isPublic))
    form.append('groupId', String(groupId))
    return request.post<ApiBody<FileUploadResult>>('/files', form)
  },
  // 私有文件取字节:<img src> 与 <a download> 都无法携带 Authorization,只能经鉴权层下载再用 objectURL。
  // silent=true 用于缩略图(失败降级为图标,不打扰);用户主动点下载时传 false,失败要提示。
  fetchBlob: (id: number, silent = true) =>
    request.get<Blob>(`/files/${id}/download`, { responseType: 'blob', silentError: silent })
}

export const fileGroupApi = {
  // category 是文件中心的最高层导航,左栏各组计数按它统计;不传=全部类型。
  list: (category?: FileCategory) =>
    request.get<ApiBody<FileGroupTree>>('/file-groups', { params: { category } }),
  create: (name: string) => request.post<ApiBody<FileGroup>>('/file-groups', { name }),
  update: (id: number, name: string) => request.put<ApiBody<null>>(`/file-groups/${id}`, { name }),
  remove: (id: number) => request.delete<ApiBody<null>>(`/file-groups/${id}`)
}

// ---- 文章资讯 ----

export interface ArticleCategory {
  id: number
  name: string
  sort: number
  /** 分类下文章数(含草稿),列表接口附带 */
  count?: number
  createdAt: string
}

export const articleCategoryApi = {
  list: () => request.get<ApiBody<ArticleCategory[]>>('/article-categories'),
  create: (data: { name: string; sort: number }) =>
    request.post<ApiBody<ArticleCategory>>('/article-categories', data),
  update: (id: number, data: { name: string; sort: number }) =>
    request.put<ApiBody<null>>(`/article-categories/${id}`, data),
  remove: (id: number) => request.delete<ApiBody<null>>(`/article-categories/${id}`)
}

/** 列表条目不含正文,编辑时通过 get 拉详情回填富文本。 */
export interface ArticleRow {
  id: number
  categoryId: number
  categoryName: string
  title: string
  summary: string
  status: number
  author: string
  publishedAt: string | null
  createdAt: string
}

export interface ArticleDetail extends ArticleRow {
  content: string
}

export interface ArticleSaveForm {
  categoryId: number
  title: string
  summary: string
  content: string
  status: number
}

export const articleApi = {
  list: (params: Record<string, unknown>) => request.get<ApiBody<Paged<ArticleRow>>>('/articles', { params }),
  get: (id: number) => request.get<ApiBody<ArticleDetail>>(`/articles/${id}`),
  create: (data: ArticleSaveForm) => request.post<ApiBody<ArticleDetail>>('/articles', data),
  update: (id: number, data: ArticleSaveForm) => request.put<ApiBody<null>>(`/articles/${id}`, data),
  remove: (id: number) => request.delete<ApiBody<null>>(`/articles/${id}`),
  // 富文本配图:服务端强制公开可访问(正文 <img> 无法携带 Authorization)
  uploadImage: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return request.post<ApiBody<{ url: string }>>('/article-images', form)
  }
}

// ---- 系统参数 ----

export interface SysConfig {
  id: number
  name: string
  /** 参数键,业务模块按它读取;内置参数不可改键、不可删 */
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

// ---- 字典 ----

/** 字典类型:一组可枚举子项的容器,业务按 key 读取。 */
export interface DictType {
  id: number
  name: string
  key: string
  remark: string
  builtin: boolean
  /** 类型下子项数(含停用),列表接口附带 */
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
  /** 前端标签配色:空/primary/success/warning/danger/info */
  tagType: string
  /** 1=启用 2=停用 */
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

/** 业务读取用:展示文本、存储值与标签配色。 */
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

/** 业务模块按类型键取启用子项(仅登录即可,无需字典管理权限)。 */
export const dictDataApi = {
  byKey: (key: string) => request.get<ApiBody<DictOption[]>>('/dict-data', { params: { key } })
}
