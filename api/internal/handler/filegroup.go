// 文件分组控制器:文件中心左栏目录的增删改查。
// 权限判定全部在路由上挂 RequirePerm,本层只做绑定与转调。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/validate"
)

type FileGroupHandler struct {
	Svc *service.FileGroupService
}

func NewFileGroupHandler(svc *service.FileGroupService) *FileGroupHandler {
	return &FileGroupHandler{Svc: svc}
}

// List GET /api/v1/file-groups
// 返回分组及其文件数,另含"未分组"与"全部"两个伪节点的计数;计数按 category 限定。
func (h *FileGroupHandler) List(c *gin.Context) {
	var query FileGroupListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误"))
		return
	}
	tree, err := h.Svc.List(c.Request.Context(), query.Category)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, tree)
}

// Create POST /api/v1/file-groups
func (h *FileGroupHandler) Create(c *gin.Context) {
	var req FileGroupSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 分组名必填且不超过 64 个字符"))
		return
	}
	group, err := h.Svc.Create(c.Request.Context(), req.Name)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, group)
}

// Update PUT /api/v1/file-groups/:id
func (h *FileGroupHandler) Update(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req FileGroupSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 分组名必填且不超过 64 个字符"))
		return
	}
	if err := h.Svc.Update(c.Request.Context(), id, req.Name); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Delete DELETE /api/v1/file-groups/:id
// 仅允许删除空分组;非空分组由 Service 返回冲突,避免删目录连带删文件。
func (h *FileGroupHandler) Delete(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if err := h.Svc.Delete(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}
