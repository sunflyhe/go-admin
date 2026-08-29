// 系统参数的 HTTP 绑定 DTO 与到 Service 输入的显式转换。
package handler

import (
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/page"
)

// ConfigListQuery GET /api/v1/configs 查询参数。
type ConfigListQuery struct {
	page.Query
	Keyword string `form:"keyword"`
}

func (q *ConfigListQuery) toInput() *service.ConfigListInput {
	return &service.ConfigListInput{Query: q.Query, Keyword: q.Keyword}
}

// ConfigSaveRequest POST/PUT /api/v1/configs 请求体。
// 键的格式与唯一性、内置参数的保护规则由 Service 裁决,binding 只挡明显乱传。
type ConfigSaveRequest struct {
	Name   string `json:"name" binding:"required,max=64"`
	Key    string `json:"key" binding:"required,max=64"`
	Value  string `json:"value" binding:"max=512"`
	Remark string `json:"remark" binding:"max=255"`
}

func (r *ConfigSaveRequest) toInput() service.ConfigSaveInput {
	return service.ConfigSaveInput{Name: r.Name, Key: r.Key, Value: r.Value, Remark: r.Remark}
}
