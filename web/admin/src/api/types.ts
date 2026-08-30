export interface ApiBody<T> {
  code: number
  message: string
  data: T
}

export interface Paged<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

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
