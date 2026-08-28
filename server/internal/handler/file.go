// 文件控制器:上传、列表、删除、下载。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/validate"
)

type FileHandler struct {
	Svc             *service.FileService
	PublicURLPrefix string
}

func NewFileHandler(svc *service.FileService, publicURLPrefix string) *FileHandler {
	return &FileHandler{Svc: svc, PublicURLPrefix: publicURLPrefix}
}

// Upload POST /api/v1/files
func (h *FileHandler) Upload(c *gin.Context) {
	result, err := h.Svc.Upload(c, h.PublicURLPrefix)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, result)
}

// PublicDownload GET /files/*storePath
// 公开下载无需登录，但服务端必须确认对应文件元数据仍标记为公开。
func (h *FileHandler) PublicDownload(c *gin.Context) {
	if err := h.Svc.PublicDownload(c, c.Param("storePath")); err != nil {
		resp.Fail(c, err)
	}
}

// List GET /api/v1/files
func (h *FileHandler) List(c *gin.Context) {
	var req service.FileListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("分页参数错误"))
		return
	}
	result, err := h.Svc.List(c, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Delete DELETE /api/v1/files/:id
func (h *FileHandler) Delete(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if err := h.Svc.Delete(c, id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Download GET /api/v1/files/:id/download
func (h *FileHandler) Download(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if err := h.Svc.Download(c, id); err != nil {
		resp.Fail(c, err)
		return
	}
}
