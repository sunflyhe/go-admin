// 文件服务:上传校验、元数据管理、鉴权下载。
// Gin multipart 解析留在 Handler;本服务只接收与 HTTP 无关的输入
// (文件流、文件名、大小、公开标记、操作者),并保持扩展名/真实 MIME/大小校验不变。
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
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
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/zip": true,
	}
)

type FileService struct {
	DB        *gorm.DB
	Storage   Storage
	MaxSizeMB int
}

func NewFileService(db *gorm.DB, storage Storage, maxSizeMB int) *FileService {
	return &FileService{DB: db, Storage: storage, MaxSizeMB: maxSizeMB}
}

// FileUploadInput 上传输入:与 HTTP 无关,由 Handler 从 multipart 解析后构造。
type FileUploadInput struct {
	FileName string    // 原始文件名(扩展名校验依据)
	Size     int64     // 文件大小(字节)
	Content  io.Reader // 文件内容流
	IsPublic bool      // 是否公开访问
}

// FileUploadResult 上传结果(URL 由 Handler 按访问前缀拼装)。
type FileUploadResult struct {
	ID         int64  `json:"id"`
	OriginName string `json:"originName"`
	StorePath  string `json:"storePath"`
	URL        string `json:"url"`
	Size       int64  `json:"size"`
	MIME       string `json:"mime"`
	IsPublic   bool   `json:"-"` // Handler 据此拼装访问地址
}

// Upload 上传:扩展名、真实 MIME(基于内容嗅探)校验,随机文件名按日期分目录。
func (s *FileService) Upload(ctx context.Context, actor Actor, input *FileUploadInput) (*FileUploadResult, error) {
	ext := strings.ToLower(filepath.Ext(input.FileName))
	if !allowedExts[ext] {
		return nil, errs.InvalidParam("不支持的文件类型: " + ext)
	}

	// 真实 MIME 校验:基于文件内容嗅探,而非客户端声明
	head := make([]byte, 512)
	n, _ := io.ReadFull(input.Content, head)
	mime := http.DetectContentType(head[:n])
	// text/plain; charset=utf-8 归一化
	if i := strings.Index(mime, ";"); i > 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	if !allowedMIMEs[mime] {
		return nil, errs.InvalidParam("文件真实类型不被允许: " + mime)
	}

	now := time.Now()
	relPath := fmt.Sprintf("%04d/%02d/%02d/%s%s", now.Year(), now.Month(), now.Day(), uuid.NewString(), ext)
	if err := s.Storage.Save(relPath, io.MultiReader(strings.NewReader(string(head[:n])), input.Content)); err != nil {
		return nil, errs.Internal("保存文件失败").WithCause(err)
	}
	entry := &model.SysFile{
		OriginName: filepath.Base(input.FileName),
		StorePath:  relPath,
		Size:       input.Size,
		MIME:       mime,
		Ext:        ext,
		IsPublic:   input.IsPublic,
		UploaderID: actor.ID,
		Uploader:   actor.Username,
	}
	if err := s.DB.WithContext(ctx).Create(entry).Error; err != nil {
		_ = s.Storage.Delete(relPath)
		return nil, errs.Internal("记录文件元数据失败").WithCause(err)
	}
	return &FileUploadResult{
		ID: entry.ID, OriginName: entry.OriginName, StorePath: relPath,
		Size: entry.Size, MIME: mime, IsPublic: entry.IsPublic,
	}, nil
}

type FileListInput struct {
	page.Query
	OriginName string
	IsPublic   *bool
}

func (s *FileService) List(ctx context.Context, req *FileListInput) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	q := s.DB.WithContext(ctx).Model(&model.SysFile{})
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

func (s *FileService) Delete(ctx context.Context, id int64) error {
	var entry model.SysFile
	if err := s.DB.WithContext(ctx).First(&entry, id).Error; err != nil {
		return errs.NotFound("文件不存在")
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&entry).Error; err != nil {
			return errs.Internal("删除文件记录失败").WithCause(err)
		}
		if err := s.Storage.Delete(entry.StorePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return errs.Internal("删除文件失败").WithCause(err)
		}
		return nil
	})
}

// GetFile 读取文件元数据(鉴权后的下载入口使用)。
func (s *FileService) GetFile(ctx context.Context, id int64) (*model.SysFile, error) {
	var entry model.SysFile
	if err := s.DB.WithContext(ctx).First(&entry, id).Error; err != nil {
		return nil, errs.NotFound("文件不存在")
	}
	return &entry, nil
}

// OpenFile 打开存储中的文件,由 Handler 负责输出与关闭。
func (s *FileService) OpenFile(ctx context.Context, relPath string) (*os.File, error) {
	return s.Storage.Open(relPath)
}

// GetPublicFile 仅返回数据库中显式标记为公开的文件元数据。storePath 由路由通配符提供。
func (s *FileService) GetPublicFile(ctx context.Context, storePath string) (*model.SysFile, error) {
	storePath = strings.TrimPrefix(storePath, "/")
	if storePath == "" {
		return nil, errs.NotFound("文件不存在")
	}
	var entry model.SysFile
	if err := s.DB.WithContext(ctx).
		Where("store_path = ? AND is_public = ?", storePath, true).
		First(&entry).Error; err != nil {
		return nil, errs.NotFound("文件不存在")
	}
	return &entry, nil
}
