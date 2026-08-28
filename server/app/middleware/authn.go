// 认证与权限校验中间件。
// 安全边界:API 权限一律在服务端强制校验;前端隐藏按钮只是体验优化,不作为安全措施。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/app/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
)

// Authn 负责登录态校验与当前用户注入。
type Authn struct {
	DB  *gorm.DB
	JWT *auth.Manager
}

// Require 签名校验 + 加载当前用户 + 状态/版本校验。
// 每请求查询一次用户,保证禁用、删除、登出后的旧凭据立即失效。
func (a *Authn) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			resp.Fail(c, errs.Unauthorized("未登录或凭据缺失"))
			return
		}
		claims, err := a.JWT.Parse(strings.TrimPrefix(header, "Bearer "), auth.TokenTypeAccess)
		if err != nil {
			resp.Fail(c, err)
			return
		}
		var user model.SysUser
		if err := a.DB.First(&user, claims.UserID).Error; err != nil {
			resp.Fail(c, errs.Unauthorized("账号不存在或已删除"))
			return
		}
		if user.Status != model.StatusEnabled {
			resp.Fail(c, errs.Forbidden("账号已被停用"))
			return
		}
		if user.TokenVersion != claims.TokenVersion {
			resp.Fail(c, errs.Unauthorized("凭据已失效,请重新登录"))
			return
		}
		c.Set(CtxClaims, claims)
		c.Set(CtxUser, &user)
		c.Next()
	}
}

// CurrentUser 从上下文取当前登录用户(Authn.Require 之后可用)。
func CurrentUser(c *gin.Context) (*model.SysUser, bool) {
	v, ok := c.Get(CtxUser)
	if !ok {
		return nil, false
	}
	u, ok := v.(*model.SysUser)
	return u, ok
}

// RequirePerm 按钮级权限校验:超级管理员全量放行,其余按角色-菜单 permission code 匹配。
func (a *Authn) RequirePerm(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			resp.Fail(c, errs.Unauthorized("未登录"))
			return
		}
		if user.IsSuperAdmin() {
			c.Next()
			return
		}
		has, err := a.hasPermission(c, user.ID, permission)
		if err != nil {
			resp.Fail(c, err)
			return
		}
		if !has {
			resp.Fail(c, errs.Forbidden("没有操作权限: "+permission))
			return
		}
		c.Next()
	}
}

// Permissions 加载用户的按钮权限集合;超级管理员返回通配标记。
func (a *Authn) Permissions(c *gin.Context, userID int64) ([]string, error) {
	var user model.SysUser
	if err := a.DB.First(&user, userID).Error; err != nil {
		return nil, errs.Unauthorized("账号不存在")
	}
	if user.IsSuperAdmin() {
		return []string{"*"}, nil
	}
	return a.loadPerms(c, userID)
}

func (a *Authn) hasPermission(c *gin.Context, userID int64, permission string) (bool, error) {
	perms, err := a.loadPerms(c, userID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}

// loadPerms 通过 用户→角色→菜单 关联查询权限码(页面与按钮上挂载的 permission code 均参与校验)。
func (a *Authn) loadPerms(c *gin.Context, userID int64) ([]string, error) {
	var perms []string
	err := a.DB.WithContext(c).
		Table("sys_menu").
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
		Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role_menu.role_id").
		Joins("JOIN sys_role ON sys_role.id = sys_role_menu.role_id").
		Where("sys_user_role.user_id = ?", userID).
		Where("sys_menu.status = ? AND sys_menu.permission <> ''", model.StatusEnabled).
		Where("sys_role.status = ?", model.StatusEnabled).
		Distinct().
		Pluck("sys_menu.permission", &perms).Error
	if err != nil {
		return nil, errs.Internal("加载权限失败").WithCause(err)
	}
	return perms, nil
}

// LoadRoleIDs 返回用户拥有的角色 ID 列表。
func (a *Authn) LoadRoleIDs(c *gin.Context, userID int64) ([]int64, error) {
	var ids []int64
	err := a.DB.WithContext(c).Model(&model.SysUserRole{}).
		Where("user_id = ?", userID).Pluck("role_id", &ids).Error
	if err != nil {
		return nil, errs.Internal("加载角色失败").WithCause(err)
	}
	return ids, nil
}
