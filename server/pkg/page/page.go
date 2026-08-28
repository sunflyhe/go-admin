// Package page 统一分页参数与分页响应结构。
package page

import (
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// Query 前端传入的分页参数,page 从 1 开始。
type Query struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

// Normalize 校验并纠正分页参数,越界值回退到默认值。
func (q *Query) Normalize() error {
	if q.Page <= 0 {
		q.Page = defaultPage
	}
	if q.PageSize <= 0 {
		q.PageSize = defaultPageSize
	}
	if q.PageSize > maxPageSize {
		return errs.InvalidParam("pageSize 不能超过 100")
	}
	return nil
}

// Offset 计算 SQL 偏移量。
func (q *Query) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// MaxExportRows 导出接口允许的最大行数上限。
func MaxExportRows() int { return 10000 }

// Result 统一分页响应格式:
//
//	{"list":[],"total":0,"page":1,"pageSize":20}
type Result struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// NewResult 构造分页响应。
func NewResult(list interface{}, total int64, q Query) Result {
	return Result{List: list, Total: total, Page: q.Page, PageSize: q.PageSize}
}
