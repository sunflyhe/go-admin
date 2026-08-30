// PaginatedTable 通过 defineExpose 暴露的模板 ref 句柄。
// 泛型 SFC 无法用 InstanceType<typeof Comp> 推导,故各列表页统一引用此类型。
export interface PaginatedTableHandle {
  load: () => Promise<void>
  search: () => Promise<void>
  page: number
  pageSize: number
}
