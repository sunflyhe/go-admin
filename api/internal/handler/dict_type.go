// 字典控制器:字典类型与子项的维护、业务按键读取。
// 类型与子项的写权限共用 system:dict:create/update/delete(一个页面统一管理);
// 业务按键读取 /dict-data 只要求登录 —— 业务页面对下拉数据的使用不该要求字典管理权限。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/validate"
)

type DictTypeHandler struct {
	Svc *service.DictTypeService
}

func NewDictTypeHandler(svc *service.DictTypeService) *DictTypeHandler {
	return &DictTypeHandler{Svc: svc}
}

// List GET /api/v1/dict-types
func (h *DictTypeHandler) List(c *gin.Context) {
	items, err := h.Svc.List(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, items)
}

// Create POST /api/v1/dict-types
func (h *DictTypeHandler) Create(c *gin.Context) {
	var req DictTypeSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 字典名与字典键必填"))
		return
	}
	dictType, err := h.Svc.Create(c.Request.Context(), req.toInput())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, dictType)
}

// Update PUT /api/v1/dict-types/:id
func (h *DictTypeHandler) Update(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req DictTypeSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 字典名与字典键必填"))
		return
	}
	if err := h.Svc.Update(c.Request.Context(), id, req.toInput()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Delete DELETE /api/v1/dict-types/:id
func (h *DictTypeHandler) Delete(c *gin.Context) {
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

// ListItems GET /api/v1/dict-types/:id/items
func (h *DictTypeHandler) ListItems(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	items, err := h.Svc.ListItems(c.Request.Context(), id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, items)
}

// CreateItem POST /api/v1/dict-types/:id/items
func (h *DictTypeHandler) CreateItem(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req DictItemSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 子项显示名与存储值必填"))
		return
	}
	item, err := h.Svc.CreateItem(c.Request.Context(), id, req.toInput())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, item)
}

// UpdateItem PUT /api/v1/dict-items/:id
func (h *DictTypeHandler) UpdateItem(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req DictItemSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 子项显示名与存储值必填"))
		return
	}
	if err := h.Svc.UpdateItem(c.Request.Context(), id, req.toInput()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// DeleteItem DELETE /api/v1/dict-items/:id
func (h *DictTypeHandler) DeleteItem(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if err := h.Svc.DeleteItem(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// DictData GET /api/v1/dict-data?key=xxx
// 业务模块按类型键取启用子项;只需登录,不需要字典管理权限。
func (h *DictTypeHandler) DictData(c *gin.Context) {
	options, err := h.Svc.EnabledByKey(c.Request.Context(), c.Query("key"))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, options)
}
