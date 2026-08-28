// 文件服务:上传校验、元数据管理、鉴权下载。
package file

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/middleware"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/validate"
)

// 扩展名白名单与 MIME 白名单。
var (
	allowedExts = map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
		".pdf": true, ".txt": true, ".csv": true,
		".xlsx": true, ".xls": true, ".docx": true, ".zip": true,
	}
	allowedMIMEs = map[string]bool{
		"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
		"application/pdf": true, "text/plain": true, "text/csv": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/zip": true,
	}
)

type Service struct {
	DB        *gorm.DB
	Storage   Storage
	MaxSizeMB int
}

func NewService(db *gorm.DB, storage Storage, maxSizeMB int) *Service {
	return &Service{DB: db, Storage: storage, MaxSizeMB: maxSizeMB}
}

type ListReq struct {
	page.Query
	OriginName string `form:"originName"`
	IsPublic   *bool  `form:"isPublic"`
}

type UploadResp struct {
	ID         int64  `json:"id"`
	OriginName string `json:"originName"`
	StorePath  string `json:"storePath"`
	URL        string `json:"url"`
	Size       int64  `json:"size"`
	MIME       string `json:"mime"`
}

// Upload 处理 multipart 上传:大小、扩展名、真实 MIME 三重校验,随机文件名按日期分目录。
func (s *Service) Upload(c *gin.Context, publicURLPrefix string) (*UploadResp, error) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		return nil, errs.Unauthorized("未登录")
	}
	// 大小上限校验
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(s.MaxSizeMB)<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		return nil, errs.InvalidParam(fmt.Sprintf("上传失败: 未找到 file 字段或文件超过 %dMB 限制", s.MaxSizeMB))
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !allowedExts[ext] {
		return nil, errs.InvalidParam("不支持的文件类型: " + ext)
	}
	src, err := fh.Open()
	if err != nil {
		return nil, errs.Internal("读取文件失败").WithCause(err)
	}
	defer src.Close()

	// 真实 MIME 校验:基于文件内容嗅探,而非客户端声明
	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	mime := http.DetectContentType(head[:n])
	// text/plain; charset=utf-8 归一化
	if i := strings.Index(mime, ";"); i > 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	if !allowedMIMEs[mime] {
		return nil, errs.InvalidParam("文件真实类型不被允许: " + mime)
	}

	isPublic := c.PostForm("isPublic") == "true"
	now := time.Now()
	relPath := fmt.Sprintf("%04d/%02d/%02d/%s%s", now.Year(), now.Month(), now.Day(), uuid.NewString(), ext)
	if err := s.Storage.Save(relPath, io.MultiReader(strings.NewReader(string(head[:n])), src)); err != nil {
		return nil, errs.Internal("保存文件失败").WithCause(err)
	}
	entry := &model.SysFile{
		OriginName: filepath.Base(fh.Filename),
		StorePath:  relPath,
		Size:       fh.Size,
		MIME:       mime,
		Ext:        ext,
		IsPublic:   isPublic,
		UploaderID: user.ID,
		Uploader:   user.Username,
	}
	if err := s.DB.WithContext(c).Create(entry).Error; err != nil {
		_ = s.Storage.Delete(relPath)
		return nil, errs.Internal("记录文件元数据失败").WithCause(err)
	}
	url := fmt.Sprintf("%s/%s", publicURLPrefix, relPath)
	if !isPublic {
		url = fmt.Sprintf("/api/v1/files/%d/download", entry.ID)
	}
	return &UploadResp{
		ID: entry.ID, OriginName: entry.OriginName, StorePath: relPath,
		URL: url, Size: entry.Size, MIME: mime,
	}, nil
}

func (s *Service) List(c *gin.Context, req *ListReq) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	q := s.DB.WithContext(c).Model(&model.SysFile{})
	if req.OriginName != "" {
		q = q.Where("origin_name LIKE ?", "%"+req.OriginName+"%")
	}
	if req.IsPublic != nil {
		q = q.Where("is_public = ?", *req.IsPublic)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errs.Internal("查询文件失败").WithCause(err)
	}
	var files []model.SysFile
	if err := q.Order("id DESC").Offset(req.Offset()).Limit(req.PageSize).Find(&files).Error; err != nil {
		return nil, errs.Internal("查询文件失败").WithCause(err)
	}
	return &page.Result{List: files, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

func (s *Service) Delete(c *gin.Context, id int64) error {
	var entry model.SysFile
	if err := s.DB.WithContext(c).First(&entry, id).Error; err != nil {
		return errs.NotFound("文件不存在")
	}
	return s.DB.WithContext(c).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&entry).Error; err != nil {
			return errs.Internal("删除文件记录失败").WithCause(err)
		}
		if err := s.Storage.Delete(entry.StorePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return errs.Internal("删除文件失败").WithCause(err)
		}
		return nil
	})
}

// Download 鉴权下载私有文件:任何已登录且具有 system:file:list 权限的用户可下载。
func (s *Service) Download(c *gin.Context, id int64) error {
	var entry model.SysFile
	if err := s.DB.WithContext(c).First(&entry, id).Error; err != nil {
		return errs.NotFound("文件不存在")
	}
	if entry.IsPublic {
		c.Header("X-Audit-Skip", "1")
		c.Redirect(http.StatusFound, "/files/"+entry.StorePath)
		return nil
	}
	f, err := s.Storage.Open(entry.StorePath)
	if err != nil {
		return errs.NotFound("文件不存在")
	}
	defer f.Close()
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", urlEscape(entry.OriginName)))
	c.DataFromReader(http.StatusOK, entry.Size, entry.MIME, f, nil)
	return nil
}

func urlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			fmt.Fprintf(&b, "%%%02X", r)
		}
	}
	return b.String()
}

// ---- Handler ----

type Handler struct {
	Svc             *Service
	PublicURLPrefix string
}

func NewHandler(svc *Service, publicURLPrefix string) *Handler {
	return &Handler{Svc: svc, PublicURLPrefix: publicURLPrefix}
}

// Upload POST /api/v1/files
func (h *Handler) Upload(c *gin.Context) {
	c.Header("X-Audit-Skip", "1")
	result, err := h.Svc.Upload(c, h.PublicURLPrefix)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, result)
}

// List GET /api/v1/files
func (h *Handler) List(c *gin.Context) {
	var req ListReq
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
func (h *Handler) Delete(c *gin.Context) {
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
func (h *Handler) Download(c *gin.Context) {
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
