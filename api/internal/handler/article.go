// 文章控制器:文章资讯的列表、详情、增删改与富文本配图上传。
// 权限判定全部在路由上挂 RequirePerm,本层只做绑定、multipart 解析与转调。
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/validate"
)

type ArticleHandler struct {
	Svc             *service.ArticleService
	FileSvc         *service.FileService // 配图复用文件上传的白名单/MIME 校验与存储
	PublicURLPrefix string
	MaxSizeMB       int
}

func NewArticleHandler(svc *service.ArticleService, fileSvc *service.FileService, publicURLPrefix string, maxSizeMB int) *ArticleHandler {
	return &ArticleHandler{Svc: svc, FileSvc: fileSvc, PublicURLPrefix: publicURLPrefix, MaxSizeMB: maxSizeMB}
}

// List GET /api/v1/articles
func (h *ArticleHandler) List(c *gin.Context) {
	var query ArticleListQuery
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

// Get GET /api/v1/articles/:id — 详情含正文,编辑弹窗据此回填富文本。
func (h *ArticleHandler) Get(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	detail, err := h.Svc.Get(c.Request.Context(), id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, detail)
}

// Create POST /api/v1/articles
func (h *ArticleHandler) Create(c *gin.Context) {
	actor, ok := middlewareActor(c)
	if !ok {
		resp.Fail(c, errs.Unauthorized("未登录"))
		return
	}
	var req ArticleSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 标题与正文必填,状态只能为 1(草稿)或 2(已发布)"))
		return
	}
	article, err := h.Svc.Create(c.Request.Context(), actor, req.toInput())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, article)
}

// Update PUT /api/v1/articles/:id
func (h *ArticleHandler) Update(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req ArticleSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 标题与正文必填,状态只能为 1(草稿)或 2(已发布)"))
		return
	}
	if err := h.Svc.Update(c.Request.Context(), id, req.toInput()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Delete DELETE /api/v1/articles/:id
func (h *ArticleHandler) Delete(c *gin.Context) {
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

// UploadImage POST /api/v1/article-images
// 富文本配图上传:复用文件上传的白名单、大小与真实 MIME 校验,
// 但强制 is_public=true —— 正文 <img> 无法携带 Authorization,配图必须公开可访问。
func (h *ArticleHandler) UploadImage(c *gin.Context) {
	actor, ok := middlewareActor(c)
	if !ok {
		resp.Fail(c, errs.Unauthorized("未登录"))
		return
	}
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

	input := &service.FileUploadInput{
		FileName: fh.Filename,
		Size:     fh.Size,
		Content:  src,
		IsPublic: true,
	}
	result, err := h.FileSvc.Upload(c.Request.Context(), actor, input)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, toUploadResponse(result, h.PublicURLPrefix))
}
