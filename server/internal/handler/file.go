// 文件控制器:multipart 解析、大小上限、上传、列表、删除、下载。
// Gin multipart 与 MaxBytesReader 留在本层;Service 只接收文件流等与 HTTP 无关的输入。
package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/validate"
)

type FileHandler struct {
	Svc             *service.FileService
	PublicURLPrefix string
	MaxSizeMB       int
}

func NewFileHandler(svc *service.FileService, publicURLPrefix string, maxSizeMB int) *FileHandler {
	return &FileHandler{Svc: svc, PublicURLPrefix: publicURLPrefix, MaxSizeMB: maxSizeMB}
}

// Upload POST /api/v1/files
func (h *FileHandler) Upload(c *gin.Context) {
	actor, ok := middlewareActor(c)
	if !ok {
		resp.Fail(c, errs.Unauthorized("未登录"))
		return
	}
	// 大小上限校验
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(h.MaxSizeMB)<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, errs.InvalidParam(fmt.Sprintf("上传失败: 未找到 file 字段或文件超过 %dMB 限制", h.MaxSizeMB)))
		return
	}
	src, err := fh.Open()
	if err != nil {
		resp.Fail(c, errs.Internal("读取文件失败").WithCause(err))
		return
	}
	defer src.Close()

	groupID, err := parseGroupIDForm(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	input := &service.FileUploadInput{
		FileName: fh.Filename,
		Size:     fh.Size,
		Content:  src,
		IsPublic: c.PostForm("isPublic") == "true",
		GroupID:  groupID,
	}
	result, err := h.Svc.Upload(c.Request.Context(), actor, input)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, toUploadResponse(result, h.PublicURLPrefix))
}

// PublicDownload GET /files/*storePath
// 公开下载无需登录,但服务端必须确认对应文件元数据仍标记为公开。
func (h *FileHandler) PublicDownload(c *gin.Context) {
	ctx := c.Request.Context()
	entry, err := h.Svc.GetPublicFile(ctx, c.Param("storePath"))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	f, err := h.Svc.OpenFile(ctx, entry.StorePath)
	if err != nil {
		resp.Fail(c, errs.NotFound("文件不存在"))
		return
	}
	defer f.Close()
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename*=UTF-8''%s", urlEscape(entry.OriginName)))
	c.DataFromReader(http.StatusOK, entry.Size, entry.MIME, f, nil)
}

// List GET /api/v1/files
func (h *FileHandler) List(c *gin.Context) {
	var query FileListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		resp.Fail(c, errs.InvalidParam("分页参数错误"))
		return
	}
	result, err := h.Svc.List(c.Request.Context(), query.toInput())
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
	if err := h.Svc.Delete(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Move PUT /api/v1/files/group
// 批量把选中文件移动到目标分组,groupId=0 表示"未分组"。
func (h *FileHandler) Move(c *gin.Context) {
	var req FileMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 需选择至少一个文件并提供 groupId"))
		return
	}
	if err := h.Svc.Move(c.Request.Context(), req.IDs, *req.GroupID); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// BatchDelete POST /api/v1/files/batch-delete
// 用 POST 承载批量删除:带请求体的 DELETE 在网关与访问日志里都不友好。
func (h *FileHandler) BatchDelete(c *gin.Context) {
	var req FileBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 请选择要删除的文件"))
		return
	}
	if err := h.Svc.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// parseGroupIDForm 读取上传表单里的 groupId。
// 未传视为 0(未分组);传了却解析失败必须报错,否则文件会静默落进错误的分组。
func parseGroupIDForm(c *gin.Context) (int64, error) {
	raw := strings.TrimSpace(c.PostForm("groupId"))
	if raw == "" {
		return 0, nil
	}
	groupID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || groupID < 0 {
		return 0, errs.InvalidParam("参数错误: groupId 非法")
	}
	return groupID, nil
}

// Download GET /api/v1/files/:id/download
// 公开文件 302 到公开前缀;私有文件经本接口鉴权后流式输出。
func (h *FileHandler) Download(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	entry, err := h.Svc.GetFile(ctx, id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if entry.IsPublic {
		c.Redirect(http.StatusFound, h.PublicURLPrefix+"/"+entry.StorePath)
		return
	}
	f, err := h.Svc.OpenFile(ctx, entry.StorePath)
	if err != nil {
		resp.Fail(c, errs.NotFound("文件不存在"))
		return
	}
	defer f.Close()
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", urlEscape(entry.OriginName)))
	c.DataFromReader(http.StatusOK, entry.Size, entry.MIME, f, nil)
}

// urlEscape Content-Disposition 的文件名转义(非 HTTP 语义不进入 Service)。
func urlEscape(s string) string {
	var b []byte
	for _, ch := range []byte(s) {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == '_' {
			b = append(b, ch)
		} else {
			b = append(b, []byte(fmt.Sprintf("%%%02X", ch))...)
		}
	}
	return string(b)
}
