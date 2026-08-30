import request from './request'
import type { ApiBody, Paged } from './types'

export interface ArticleCategory {
  id: number
  name: string
  sort: number
  count?: number
  createdAt: string
}

export const articleCategoryApi = {
  list: () => request.get<ApiBody<ArticleCategory[]>>('/article-categories'),
  create: (data: { name: string; sort: number }) => request.post<ApiBody<ArticleCategory>>('/article-categories', data),
  update: (id: number, data: { name: string; sort: number }) =>
    request.put<ApiBody<null>>(`/article-categories/${id}`, data),
  remove: (id: number) => request.delete<ApiBody<null>>(`/article-categories/${id}`)
}

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
  uploadImage: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return request.post<ApiBody<{ url: string }>>('/article-images', form)
  }
}
