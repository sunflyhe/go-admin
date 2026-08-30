// 文件分组接口的 HTTP 绑定 DTO。
package handler

// FileGroupSaveRequest 分组创建/更新请求体。
// max=64 与 Service 的上界一致,按 rune 计:HTTP 层要给出比业务校验更早的拒绝。
// 本轮只做一级分组,parent/sort 不对外暴露,由 Service 维护。
type FileGroupSaveRequest struct {
	Name string `json:"name" binding:"required,max=64"`
}

// FileGroupListQuery GET /admin-api/file-groups 查询参数。
// category 是文件中心的最高层导航,左栏各组计数按它统计;取值集合与文件列表完全一致,
// 故不在这里加 oneof 校验,统一由 Service 裁决,避免两处枚举漂移。
type FileGroupListQuery struct {
	Category string `form:"category"`
}
