// 用户管理接口的 HTTP 绑定 DTO 与到 Service 输入的显式转换。
package handler

import (
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/page"
)

// UserListQuery GET /api/v1/users 查询参数。
type UserListQuery struct {
	page.Query
	Username  string `form:"username"`
	Nickname  string `form:"nickname"`
	Status    int    `form:"status" binding:"omitempty,oneof=1 2"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
}

// toInput 转换为 Service 输入(显式拷贝,避免 HTTP 标签泄漏到业务层)。
func (q *UserListQuery) toInput() *service.UserListInput {
	return &service.UserListInput{
		Query:     q.Query,
		Username:  q.Username,
		Nickname:  q.Nickname,
		Status:    q.Status,
		StartTime: q.StartTime,
		EndTime:   q.EndTime,
	}
}

// UserCreateRequest POST /api/v1/users 请求体。
type UserCreateRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	Nickname string `json:"nickname" binding:"max=64"`
	Email    string `json:"email" binding:"omitempty,email,max=128"`
	Phone    string `json:"phone" binding:"max=32"`
	Status   int    `json:"status" binding:"omitempty,oneof=1 2"`
	Remark   string `json:"remark" binding:"max=255"`
}

// UserUpdateRequest PUT /api/v1/users/:id 请求体(密码走重置接口,头像走个人中心上传)。
type UserUpdateRequest struct {
	Nickname string `json:"nickname" binding:"max=64"`
	Email    string `json:"email" binding:"omitempty,email,max=128"`
	Phone    string `json:"phone" binding:"max=32"`
	Status   int    `json:"status" binding:"omitempty,oneof=1 2"`
	Remark   string `json:"remark" binding:"max=255"`
}

// UserSetStatusRequest PUT /api/v1/users/:id/status 请求体。
type UserSetStatusRequest struct {
	Status int `json:"status" binding:"required,oneof=1 2"`
}

// UserResetPasswordRequest PUT /api/v1/users/:id/password 请求体。
type UserResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8,max=128"`
}

// UserAssignRolesRequest PUT /api/v1/users/:id/roles 请求体。
type UserAssignRolesRequest struct {
	RoleIDs []int64 `json:"roleIds" binding:"required"`
}

// toInput 创建:转换并补齐业务默认值(创建时密码必填、状态默认启用)。
func (r *UserCreateRequest) toInput() *service.UserSaveInput {
	return &service.UserSaveInput{
		Username: r.Username,
		Password: r.Password,
		Nickname: r.Nickname,
		Email:    r.Email,
		Phone:    r.Phone,
		Status:   r.Status,
		Remark:   r.Remark,
	}
}

// toInput 更新:Service 不接受密码字段,显式置空。
func (r *UserUpdateRequest) toInput() *service.UserSaveInput {
	return &service.UserSaveInput{
		Nickname: r.Nickname,
		Email:    r.Email,
		Phone:    r.Phone,
		Status:   r.Status,
		Remark:   r.Remark,
	}
}
