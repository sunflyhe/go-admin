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
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/page"
)

// fileKind 上传白名单的内部归类:只回答"这个扩展名允不允许上传、属于哪一类"。
type fileKind string

const (
	kindImage    fileKind = "image"
	kindDocument fileKind = "document"
	kindArchive  fileKind = "archive"
)

// FileCategory 文件中心的类型标签,对外出现在 GET /api/v1/files?category= 上。
type FileCategory string

const (
	FileCategoryAll   FileCategory = "all"   // 不按类型筛选(也接受空值)
	FileCategoryImage FileCategory = "image" // 图片
	FileCategoryVideo FileCategory = "video" // 视频
	FileCategoryFile  FileCategory = "file"  // 文件:图片与视频之外的全部
)

// fileExtKind 是上传扩展名白名单的唯一事实来源:key 为小写带点扩展名,value 为其归类。
// 新增 key 必须显式给出归类,否则它既进不了白名单也进不了类型标签。
// 归类依据只能是 ext:真实 MIME 嗅探会把 .docx/.xlsx 识别成 application/zip,按 mime 分会把文档归进压缩包。
var fileExtKind = map[string]fileKind{
	".jpg": kindImage, ".jpeg": kindImage, ".png": kindImage,
	".gif": kindImage, ".webp": kindImage,
	".pdf": kindDocument, ".txt": kindDocument, ".csv": kindDocument,
	".xlsx": kindDocument, ".xls": kindDocument, ".docx": kindDocument,
	".zip": kindArchive,
}

// allowedMIMEs 基于文件内容嗅探的独立第二道闸门,不因分类而放宽。
var allowedMIMEs = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
	"application/pdf": true, "text/plain": true, "text/csv": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/zip": true,
}

// videoMIMEPrefix 是「视频」标签唯一的判定依据:白名单里没有视频扩展名,拿不到 ext 集合。
// 将来放开视频上传后这一档会自动出现内容,不需要再改查询条件。
const videoMIMEPrefix = "video/%"

// normalizeCategory 校验标签取值:空值或 all 表示不筛选;未知值直接报错,
// 绝不静默退化成"全部",否则前端拼错的参数会返回一个看起来正确但其实没筛选的结果。
func normalizeCategory(raw string) (FileCategory, error) {
	c := FileCategory(strings.ToLower(strings.TrimSpace(raw)))
	switch c {
	case "", FileCategoryAll:
		return FileCategoryAll, nil
	case FileCategoryImage, FileCategoryVideo, FileCategoryFile:
		return c, nil
	}
	return "", errs.InvalidParam("不支持的文件分类: " + raw)
}

// extsInKind 返回该归类下的扩展名(升序,保证生成的 SQL 文本稳定可断言)。
func extsInKind(kind fileKind) []string {
	var exts []string
	for ext, k := range fileExtKind {
		if k == kind {
			exts = append(exts, ext)
		}
	}
	sort.Strings(exts)
	return exts
}

// knownExts 返回全部白名单扩展名(升序)。
func knownExts() []string {
	exts := make([]string, 0, len(fileExtKind))
	for ext := range fileExtKind {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

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
	GroupID  int64     // 目标分组,0=未分组
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
	if _, ok := fileExtKind[ext]; !ok {
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
		GroupID:    input.GroupID,
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
	GroupID    *int64       // nil=全部分组;0=未分组
	Category   FileCategory // 取值由 normalizeCategory 裁决;all/空 表示不筛选
}

// applyCategory 把类型标签翻译成 SQL 谓词。
// 文件列表与分组计数必须共用它:标签是最外层导航,左栏数字与右侧内容口径不一致会直接误导操作。
// 只用 IN / NOT IN / LIKE / LOWER:MySQL 与测试用的 glebarez/sqlite 行为一致。
// LOWER 是抵消 SQLite 的大小写敏感,历史数据里可能存着 .PNG。
func applyCategory(q *gorm.DB, category FileCategory) (*gorm.DB, error) {
	switch category {
	case FileCategoryAll:
		return q, nil
	case FileCategoryImage:
		exts := extsInKind(kindImage)
		if len(exts) == 0 {
			return nil, errs.Internal("图片分类配置缺失")
		}
		return q.Where("LOWER(ext) IN ?", exts).Where("mime NOT LIKE ?", videoMIMEPrefix), nil
	case FileCategoryVideo:
		return q.Where("mime LIKE ?", videoMIMEPrefix), nil
	case FileCategoryFile:
		// 「文件」= 图片与视频之外的全部,压缩包与文档都在这
		return q.Where("LOWER(ext) NOT IN ?", extsInKind(kindImage)).Where("mime NOT LIKE ?", videoMIMEPrefix), nil
	}
	return nil, errs.InvalidParam("不支持的文件分类: " + string(category))
}

func (s *FileService) List(ctx context.Context, req *FileListInput) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	category, err := normalizeCategory(string(req.Category))
	if err != nil {
		return nil, err
	}
	q := s.DB.WithContext(ctx).Model(&model.SysFile{})
	if req.OriginName != "" {
		q = q.Where("origin_name LIKE ?", "%"+req.OriginName+"%")
	}
	if req.IsPublic != nil {
		q = q.Where("is_public = ?", *req.IsPublic)
	}
	if req.GroupID != nil {
		q = q.Where("group_id = ?", *req.GroupID)
	}
	if q, err = applyCategory(q, category); err != nil {
		return nil, err
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

// maxBatchIDs 单次批量操作的 id 上限:避免一次请求把整库文件带走。
const maxBatchIDs = 200

// normalizeIDs 去重、剔除非法 id 并限制批量规模。空集合直接报错,
// 而不是"什么都没做却返回成功"。
func normalizeIDs(ids []int64) ([]int64, error) {
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, errs.InvalidParam("请至少选择一个文件")
	}
	if len(out) > maxBatchIDs {
		return nil, errs.InvalidParam("单次最多处理 200 个文件")
	}
	return out, nil
}

// existsCheck 确认这批 id 全部存在。不用 RowsAffected 判定:
// MySQL 在"更新成相同值"时返回 0 行,跨库语义不可靠。
func (s *FileService) existsCheck(ctx context.Context, ids []int64) error {
	var found int64
	if err := s.DB.WithContext(ctx).Model(&model.SysFile{}).Where("id IN ?", ids).Count(&found).Error; err != nil {
		return errs.Internal("查询文件失败").WithCause(err)
	}
	if found != int64(len(ids)) {
		return errs.NotFound("部分文件不存在,未做任何变更")
	}
	return nil
}

// Move 把选中的文件移动到目标分组,groupID=0 表示「未分组」。
// 目标分组必须存在:否则文件会被移进一个谁也看不见的桶里。
func (s *FileService) Move(ctx context.Context, ids []int64, groupID int64) error {
	target, err := normalizeIDs(ids)
	if err != nil {
		return err
	}
	if groupID < 0 {
		return errs.InvalidParam("分组不存在")
	}
	if groupID > 0 {
		var count int64
		if err := s.DB.WithContext(ctx).Model(&model.SysFileGroup{}).
			Where("id = ?", groupID).Count(&count).Error; err != nil {
			return errs.Internal("查询分组失败").WithCause(err)
		}
		if count == 0 {
			return errs.NotFound("分组不存在")
		}
	}
	if err := s.existsCheck(ctx, target); err != nil {
		return err
	}
	if err := s.DB.WithContext(ctx).Model(&model.SysFile{}).
		Where("id IN ?", target).UpdateColumn("group_id", groupID).Error; err != nil {
		return errs.Internal("移动文件失败").WithCause(err)
	}
	return nil
}

// BatchDelete 批量删除。先全量校验再动手,避免"删了一半才发现有 id 不存在";
// 删行与删文件放在同一个事务里,与单条 Delete 保持同一行为模型
// (文件系统不是事务参与者,提交前的删除失败会回滚元数据,已删的磁盘文件无法回滚 —— 这是既有取舍)。
func (s *FileService) BatchDelete(ctx context.Context, ids []int64) error {
	target, err := normalizeIDs(ids)
	if err != nil {
		return err
	}
	var entries []model.SysFile
	if err := s.DB.WithContext(ctx).Where("id IN ?", target).Find(&entries).Error; err != nil {
		return errs.Internal("查询文件失败").WithCause(err)
	}
	if len(entries) != len(target) {
		return errs.NotFound("部分文件不存在,未做任何删除")
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.SysFile{}, "id IN ?", target).Error; err != nil {
			return errs.Internal("删除文件记录失败").WithCause(err)
		}
		for i := range entries {
			if err := s.Storage.Delete(entries[i].StorePath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return errs.Internal("删除文件失败").WithCause(err)
			}
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
