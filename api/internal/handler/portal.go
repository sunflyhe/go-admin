// portal 开放接口控制器:面向 app/C 端的公开只读接口,无需登录。
// 与 admin 端 handler 的区别:不接权限码,响应字段按对外口径裁剪。
package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/pkg/resp"
)

type PortalHandler struct{}

func NewPortalHandler() *PortalHandler {
	return &PortalHandler{}
}

// DemoInfo GET /api/portal/demo 的响应体。
type DemoInfo struct {
	Name       string   `json:"name"`
	Tagline    string   `json:"tagline"`
	Stack      []string `json:"stack"`
	ServerTime string   `json:"serverTime"`
}

// Demo GET /api/portal/demo
// 演示 portal 组的公开接口形态:无鉴权、统一响应体,供 app 端联调与模板参考。
func (h *PortalHandler) Demo(c *gin.Context) {
	resp.OK(c, DemoInfo{
		Name:       "Go Admin",
		Tagline:    "开箱即用的企业后台开发底座",
		Stack:      []string{"Go", "Gin", "GORM", "Vue3", "TypeScript", "Vite"},
		ServerTime: time.Now().Format(time.DateTime),
	})
}
