// Package user 用户管理域:CRUD、启停、重置密码、角色分配、Excel 导出。
// 超级管理员保护规则:
//   - 内置账号 admin(id=1)不可删除、不可停用;
//   - 内置角色 super_admin 不可删除、不可停用;
//   - 保证系统永远存在可管理账号。
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/page"
)

type UserService struct{ DB *gorm.DB }

func NewUserService(db *gorm.DB) *UserService { return &UserService{DB: db} }

// ---- DTO ----

type UserListInput struct {
	page.Query
	Username  string ``
	Nickname  string ``
	Status    int    ``
	StartTime string ``
	EndTime   string ``
}

type UserItem struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Nickname    string     `json:"nickname"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Avatar      string     `json:"avatar"`
	Remark      string     `json:"remark"`
	Status      int        `json:"status"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	RoleIDs     []int64    `json:"roleIds"`
	Super       bool       `json:"super"`
}

type UserSaveInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

type UserSetStatusInput struct {
	Status int `json:"status"`
}

type UserResetPasswordInput struct {
	Password string `json:"password"`
}

type UserAssignRolesInput struct {
	RoleIDs []int64 `json:"roleIds"`
}

// ---- UserService ----

func (s *UserService) List(ctx context.Context, req *UserListInput) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	q := s.DB.WithContext(ctx).Model(&model.SysUser{})
	if req.Username != "" {
		q = q.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.Nickname != "" {
		q = q.Where("nickname LIKE ?", "%"+req.Nickname+"%")
	}
	if req.Status != 0 {
		q = q.Where("status = ?", req.Status)
	}
	if req.StartTime != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", req.StartTime, time.Local); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if req.EndTime != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", req.EndTime, time.Local); err == nil {
			q = q.Where("created_at <= ?", t)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errs.Internal("查询用户失败").WithCause(err)
	}
	var users []model.SysUser
	if err := q.Order("id ASC").Offset(req.Offset()).Limit(req.PageSize).Find(&users).Error; err != nil {
		return nil, errs.Internal("查询用户失败").WithCause(err)
	}
	items := make([]UserItem, 0, len(users))
	for i := range users {
		item := s.toItem(ctx, &users[i])
		items = append(items, item)
	}
	return &page.Result{List: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

func (s *UserService) toItem(ctx context.Context, u *model.SysUser) UserItem {
	var roleIDs []int64
	_ = s.DB.WithContext(ctx).Model(&model.SysUserRole{}).Where("user_id = ?", u.ID).Pluck("role_id", &roleIDs).Error
	if roleIDs == nil {
		roleIDs = []int64{}
	}
	return UserItem{
		ID: u.ID, Username: u.Username, Nickname: u.Nickname,
		Email: u.Email, Phone: u.Phone, Avatar: u.Avatar, Remark: u.Remark, Status: u.Status,
		LastLoginAt: u.LastLoginAt, CreatedAt: u.CreatedAt,
		RoleIDs: roleIDs, Super: u.IsSuperAdmin(),
	}
}

func (s *UserService) Create(ctx context.Context, req *UserSaveInput) (*UserItem, error) {
	if req.Password == "" {
		return nil, errs.InvalidParam("密码不能为空")
	}
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}
	var count int64
	if err := s.DB.WithContext(ctx).Model(&model.SysUser{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return nil, errs.Internal("查询失败").WithCause(err)
	}
	if count > 0 {
		return nil, errs.Conflict("用户名已存在")
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, errs.Internal("密码处理失败").WithCause(err)
	}
	status := req.Status
	if status == 0 {
		status = model.StatusEnabled
	}
	u := &model.SysUser{
		Username: req.Username, Password: hash, Nickname: req.Nickname,
		Email: req.Email, Phone: req.Phone, Status: status, Remark: req.Remark,
	}
	if err := s.DB.WithContext(ctx).Create(u).Error; err != nil {
		return nil, errs.Internal("创建用户失败").WithCause(err)
	}
	item := s.toItem(ctx, u)
	return &item, nil
}

func (s *UserService) Update(ctx context.Context, id int64, req *UserSaveInput) (*UserItem, error) {
	var u model.SysUser
	if err := s.DB.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, errs.NotFound("用户不存在")
	}
	if req.Password != "" {
		return nil, errs.InvalidParam("请使用重置密码接口修改密码")
	}
	updates := map[string]interface{}{
		"nickname": req.Nickname,
		"email":    req.Email,
		"phone":    req.Phone,
		"remark":   req.Remark,
	}
	if req.Status != 0 {
		if err := checkStatusChange(&u, req.Status); err != nil {
			return nil, err
		}
		updates["status"] = req.Status
		// 停用立即生效:使已签发凭据失效
		if req.Status == model.StatusDisabled {
			updates["token_version"] = gorm.Expr("token_version + 1")
		}
	}
	if err := s.DB.WithContext(ctx).Model(&u).Updates(updates).Error; err != nil {
		return nil, errs.Internal("更新用户失败").WithCause(err)
	}
	_ = s.DB.WithContext(ctx).First(&u, id).Error
	item := s.toItem(ctx, &u)
	return &item, nil
}

// checkStatusChange 超管保护:内置账号不可停用。
func checkStatusChange(u *model.SysUser, target int) error {
	if u.IsSuperAdmin() && target == model.StatusDisabled {
		return errs.InvalidParam("内置超级管理员账号不允许停用")
	}
	return nil
}

func (s *UserService) Delete(ctx context.Context, id int64) error {
	if id == 1 {
		return errs.InvalidParam("内置超级管理员账号不允许删除")
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u model.SysUser
		if err := tx.First(&u, id).Error; err != nil {
			return errs.NotFound("用户不存在")
		}
		if err := tx.Delete(&u).Error; err != nil {
			return errs.Internal("删除用户失败").WithCause(err)
		}
		if err := tx.Where("user_id = ?", id).Delete(&model.SysUserRole{}).Error; err != nil {
			return errs.Internal("清理用户角色关联失败").WithCause(err)
		}
		if err := tx.Where("user_id = ?", id).Delete(&model.SysRefreshToken{}).Error; err != nil {
			return errs.Internal("清理刷新令牌失败").WithCause(err)
		}
		return nil
	})
}

func (s *UserService) SetStatus(ctx context.Context, id int64, req *UserSetStatusInput) error {
	var u model.SysUser
	if err := s.DB.WithContext(ctx).First(&u, id).Error; err != nil {
		return errs.NotFound("用户不存在")
	}
	if err := checkStatusChange(&u, req.Status); err != nil {
		return err
	}
	updates := map[string]interface{}{"status": req.Status}
	if req.Status == model.StatusDisabled {
		updates["token_version"] = gorm.Expr("token_version + 1")
	}
	if err := s.DB.WithContext(ctx).Model(&u).Updates(updates).Error; err != nil {
		return errs.Internal("更新状态失败").WithCause(err)
	}
	return nil
}

func (s *UserService) ResetPassword(ctx context.Context, id int64, req *UserResetPasswordInput) error {
	if err := ValidatePassword(req.Password); err != nil {
		return err
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return errs.Internal("密码处理失败").WithCause(err)
	}
	res := s.DB.WithContext(ctx).Model(&model.SysUser{ID: id}).Updates(map[string]interface{}{
		"password":      hash,
		"token_version": gorm.Expr("token_version + 1"),
	})
	if res.Error != nil {
		return errs.Internal("重置密码失败").WithCause(res.Error)
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("用户不存在")
	}
	return nil
}

func (s *UserService) AssignRoles(ctx context.Context, id int64, req *UserAssignRolesInput) error {
	// 校验角色均存在且启用
	var validIDs []int64
	if err := s.DB.WithContext(ctx).Model(&model.SysRole{}).
		Where("id IN ? AND status = ?", req.RoleIDs, model.StatusEnabled).
		Pluck("id", &validIDs).Error; err != nil {
		return errs.Internal("查询角色失败").WithCause(err)
	}
	if len(validIDs) != len(req.RoleIDs) {
		return errs.InvalidParam("存在无效或停用的角色")
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.SysUser
		if err := tx.First(&user, id).Error; err != nil {
			return errs.NotFound("用户不存在")
		}
		if err := tx.Where("user_id = ?", id).Delete(&model.SysUserRole{}).Error; err != nil {
			return errs.Internal("清理旧角色失败").WithCause(err)
		}
		for _, rid := range req.RoleIDs {
			if err := tx.Create(&model.SysUserRole{UserID: id, RoleID: rid}).Error; err != nil {
				return errs.Internal("分配角色失败").WithCause(err)
			}
		}
		// 角色变化后使旧凭据失效,重新登录生效
		if err := tx.Model(&model.SysUser{ID: id}).
			UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
			return errs.Internal("更新凭据版本失败").WithCause(err)
		}
		return nil
	})
}

// ExportUser 导出列显式定义,不通过反射暴露所有模型字段。
type ExportUser struct {
	ID        int64
	Username  string
	Nickname  string
	Email     string
	Phone     string
	Status    string
	CreatedAt string
}

// Export 导出用户列表 xlsx,上限 page.MaxExportRows() 行。
func (s *UserService) Export(ctx context.Context) (*excelize.File, string, error) {
	q := s.DB.WithContext(ctx).Model(&model.SysUser{}).Limit(page.MaxExportRows())
	var users []model.SysUser
	if err := q.Order("id ASC").Find(&users).Error; err != nil {
		return nil, "", errs.Internal("查询用户失败").WithCause(err)
	}
	f := excelize.NewFile()
	sheet := "用户列表"
	f.SetSheetName("Sheet1", sheet)
	headers := []string{"ID", "用户名", "昵称", "邮箱", "手机号", "状态", "创建时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for rIdx, u := range users {
		status := "启用"
		if u.Status != model.StatusEnabled {
			status = "停用"
		}
		vals := []interface{}{u.ID, u.Username, u.Nickname, u.Email, u.Phone, status, u.CreatedAt.Format("2006-01-02 15:04:05")}
		for i, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, rIdx+2)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	filename := fmt.Sprintf("users_%s.xlsx", time.Now().Format("20060102150405"))
	return f, filename, nil
}
