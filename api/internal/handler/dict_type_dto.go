// 字典模块的 HTTP 绑定 DTO 与到 Service 输入的显式转换。
package handler

import (
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
)

// DictTypeSaveRequest POST/PUT /admin-api/dict-types 请求体。
type DictTypeSaveRequest struct {
	Name   string `json:"name" binding:"required,max=64"`
	Key    string `json:"key" binding:"required,max=64"`
	Remark string `json:"remark" binding:"max=255"`
}

func (r *DictTypeSaveRequest) toInput() service.DictTypeSaveInput {
	return service.DictTypeSaveInput{Name: r.Name, Key: r.Key, Remark: r.Remark}
}

// DictItemSaveRequest POST /admin-api/dict-types/:id/items 与 PUT /admin-api/dict-items/:id 请求体。
// status 取值合法性由 Service 裁决(binding 只挡明显乱传),0 视为未传,默认启用;
// tagType 对应前端 el-tag 配色,合法值由 Service 校验。
type DictItemSaveRequest struct {
	Label       string `json:"label" binding:"required,max=64"`
	Description string `json:"description" binding:"max=255"`
	Value       string `json:"value" binding:"required,max=128"`
	Sort        int    `json:"sort"`
	TagType     string `json:"tagType" binding:"max=32"`
	Status      int    `json:"status" binding:"omitempty,oneof=1 2"`
	Remark      string `json:"remark" binding:"max=255"`
}

func (r *DictItemSaveRequest) toInput() service.DictItemSaveInput {
	return service.DictItemSaveInput{
		Label: r.Label, Description: r.Description, Value: r.Value,
		Sort: r.Sort, TagType: r.TagType, Status: r.Status, Remark: r.Remark,
	}
}
