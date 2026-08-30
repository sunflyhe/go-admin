import request from './request'
import type { ApiBody, Paged } from './types'

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
export type FileCategory = 'all' | 'image' | 'video' | 'file'
export interface FileListParams {
  page: number
  pageSize: number
  originName?: string
  category?: FileCategory
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
export interface FileGroup {
  id: number
  name: string
  count?: number
}
export interface FileGroupTree {
  groups: FileGroup[]
  unfiled: number
  total: number
}
export const fileApi = {
  list: (params: FileListParams) => request.get<ApiBody<Paged<FileRow>>>('/files', { params }),
  remove: (id: number) => request.delete<ApiBody<null>>(`/files/${id}`),
  move: (ids: number[], groupId: number) => request.put<ApiBody<null>>('/files/group', { ids, groupId }),
  batchRemove: (ids: number[]) => request.post<ApiBody<null>>('/files/batch-delete', { ids }),
  upload: (file: File, isPublic = false, groupId = 0) => {
    const form = new FormData()
    form.append('file', file)
    form.append('isPublic', String(isPublic))
    form.append('groupId', String(groupId))
    return request.post<ApiBody<FileUploadResult>>('/files', form)
  },
  fetchBlob: (id: number, silent = true) =>
    request.get<Blob>(`/files/${id}/download`, { responseType: 'blob', silentError: silent })
}
export const fileGroupApi = {
  list: (category?: FileCategory) => request.get<ApiBody<FileGroupTree>>('/file-groups', { params: { category } }),
  create: (name: string) => request.post<ApiBody<FileGroup>>('/file-groups', { name }),
  update: (id: number, name: string) => request.put<ApiBody<null>>(`/file-groups/${id}`, { name }),
  remove: (id: number) => request.delete<ApiBody<null>>(`/file-groups/${id}`)
}
