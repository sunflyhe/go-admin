// 文章分类控制器:文章资讯的分类枚举维护。
// 权限判定全部在路由上挂 RequirePerm,本层只做绑定与转调。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/validate"
)

type ArticleCategoryHandler struct {
	Svc *service.ArticleCategoryService
}

func NewArticleCategoryHandler(svc *service.ArticleCategoryService) *ArticleCategoryHandler {
	return &ArticleCategoryHandler{Svc: svc}
}

// List GET /api/v1/article-categories
// 分类是少量枚举,不分页;附带各分类文章数,供列表展示与删除提示。
func (h *ArticleCategoryHandler) List(c *gin.Context) {
	items, err := h.Svc.List(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, items)
}

// Create POST /api/v1/article-categories
func (h *ArticleCategoryHandler) Create(c *gin.Context) {
	var req ArticleCategorySaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 分类名必填且不超过 64 个字符"))
		return
	}
	category, err := h.Svc.Create(c.Request.Context(), req.Name, req.Sort)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, category)
}

// Update PUT /api/v1/article-categories/:id
func (h *ArticleCategoryHandler) Update(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req ArticleCategorySaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 分类名必填且不超过 64 个字符"))
		return
	}
	if err := h.Svc.Update(c.Request.Context(), id, req.Name, req.Sort); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Delete DELETE /api/v1/article-categories/:id
// 仅允许删除空分类;非空分类由 Service 返回冲突,避免删分类连带孤立文章。
func (h *ArticleCategoryHandler) Delete(c *gin.Context) {
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
