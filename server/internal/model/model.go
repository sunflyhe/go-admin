// Package model 定义系统域的 GORM 模型。表结构以 SQL migration 为准,模型仅作查询映射。
package model

import (
	"time"
)

// 账号与角色状态。
const (
	StatusEnabled  = 1
	StatusDisabled = 2
)

// 菜单类型。
const (
	MenuTypeDirectory = 1 // 目录
	MenuTypePage      = 2 // 页面
	MenuTypeButton    = 3 // 按钮
)

// SysUser 系统用户。
type SysUser struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:64;uniqueIndex" json:"username"`
	Password     string     `gorm:"size:100" json:"-"` // bcrypt 哈希,禁止序列化
	Nickname     string     `gorm:"size:64" json:"nickname"`
	Email        string     `gorm:"size:128" json:"email"`
	Phone        string     `gorm:"size:32" json:"phone"`
	Avatar       string     `gorm:"size:255" json:"avatar"`
	Signature    string     `gorm:"size:255" json:"signature"`
	Status       int        `gorm:"default:1" json:"status"`
	TokenVersion int64      `json:"-"` // 登出/重置密码后自增,使已签发 token 全部失效
	LastLoginAt  *time.Time `json:"lastLoginAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Roles        []SysRole  `gorm:"many2many:sys_user_role" json:"roles,omitempty"`
}

func (SysUser) TableName() string { return "sys_user" }

// IsSuperAdmin 判断是否为内置超级管理员账号(id=1)。
func (u *SysUser) IsSuperAdmin() bool { return u.ID == 1 }

// SysRole 角色。
type SysRole struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64" json:"name"`
	Code        string    `gorm:"size:64;uniqueIndex" json:"code"`
	Description string    `gorm:"size:255" json:"description"`
	Builtin     bool      `json:"builtin"` // 内置角色不可删除,避免系统失去管理入口
	Status      int       `gorm:"default:1" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (SysRole) TableName() string { return "sys_role" }

// IsSuperAdminRole 判断是否为内置超级管理员角色(code=super_admin)。
func (r *SysRole) IsSuperAdminRole() bool { return r.Code == "super_admin" }

// SysMenu 菜单(目录/页面/按钮共用一张表,通过 type 区分)。
type SysMenu struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	ParentID   int64     `gorm:"default:0" json:"parentId"`
	Name       string    `gorm:"size:64" json:"name"`
	Type       int       `json:"type"`
	Path       string    `gorm:"size:255" json:"path"`
	Component  string    `gorm:"size:255" json:"component"`
	Permission string    `gorm:"size:128" json:"permission"`
	Icon       string    `gorm:"size:64" json:"icon"`
	Sort       int       `gorm:"default:0" json:"sort"`
	Status     int       `gorm:"default:1" json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (SysMenu) TableName() string { return "sys_menu" }

// SysUserRole 用户-角色关联。
type SysUserRole struct {
	UserID int64 `gorm:"primaryKey"`
	RoleID int64 `gorm:"primaryKey"`
}

func (SysUserRole) TableName() string { return "sys_user_role" }

// SysRoleMenu 角色-菜单关联。
type SysRoleMenu struct {
	RoleID int64 `gorm:"primaryKey"`
	MenuID int64 `gorm:"primaryKey"`
}

func (SysRoleMenu) TableName() string { return "sys_role_menu" }

// SysLoginLog 登录日志(成功与失败都记录)。
type SysLoginLog struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	Username   string    `gorm:"size:64;index" json:"username"`
	Success    bool      `json:"success"`
	FailReason string    `gorm:"size:255" json:"failReason"`
	IP         string    `gorm:"size:64" json:"ip"`
	UserAgent  string    `gorm:"size:255" json:"userAgent"`
	CreatedAt  time.Time `gorm:"index" json:"createdAt"`
}

func (SysLoginLog) TableName() string { return "sys_login_log" }

// SysAuditLog 操作审计日志。
type SysAuditLog struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	UserID          int64     `json:"userId"`
	Username        string    `gorm:"size:64;index" json:"username"`
	Method          string    `gorm:"size:8" json:"method"`
	Path            string    `gorm:"size:255" json:"path"`
	Status          int       `json:"status"`
	LatencyMs       int64     `json:"latencyMs"`
	IP              string    `gorm:"size:64" json:"ip"`
	UserAgent       string    `gorm:"size:255" json:"userAgent"`
	RequestSummary  string    `gorm:"type:text" json:"requestSummary"`  // 已脱敏的请求摘要
	ResponseSummary string    `gorm:"type:text" json:"responseSummary"` // 响应摘要/错误信息,不保存完整响应体
	CreatedAt       time.Time `gorm:"index" json:"createdAt"`
}

func (SysAuditLog) TableName() string { return "sys_audit_log" }

// SysFile 上传文件元数据。
type SysFile struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	OriginName string    `gorm:"size:255" json:"originName"`
	StorePath  string    `gorm:"size:255;uniqueIndex" json:"storePath"` // 相对存储根目录的路径
	Size       int64     `json:"size"`
	MIME       string    `gorm:"size:128" json:"mime"`
	Ext        string    `gorm:"size:16" json:"ext"`
	GroupID    int64     `gorm:"index" json:"groupId"` // 所属分组,0=未分组
	IsPublic   bool      `json:"isPublic"`
	UploaderID int64     `json:"uploaderId"`
	Uploader   string    `gorm:"size:64" json:"uploader"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (SysFile) TableName() string { return "sys_file" }

// SysFileGroup 文件分组。本轮只做一级分组,ParentID 先留位不开放嵌套。
type SysFileGroup struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	ParentID  int64     `gorm:"index" json:"parentId"`
	Name      string    `gorm:"size:64" json:"name"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
}

func (SysFileGroup) TableName() string { return "sys_file_group" }

// SysRefreshToken 刷新令牌登记表,用于轮换与吊销。
type SysRefreshToken struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	JTI       string     `gorm:"size:64;uniqueIndex" json:"jti"`
	UserID    int64      `gorm:"index" json:"userId"`
	ExpiresAt time.Time  `json:"expiresAt"`
	Revoked   bool       `json:"revoked"`
	RevokedAt *time.Time `json:"revokedAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (SysRefreshToken) TableName() string { return "sys_refresh_token" }
