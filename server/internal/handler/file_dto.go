// 文件接口的 HTTP 绑定 DTO、上传响应与到 Service 输入的显式转换。
package handler

import (
	"strconv"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
)

// FileListQuery GET /api/v1/files 查询参数。
type FileListQuery struct {
	page.Query
	OriginName string `form:"originName"`
	IsPublic   *bool  `form:"isPublic"`
	// GroupID 用指针区分"没传"(全部分组)与"传了 0"(未分组)。
	GroupID *int64 `form:"groupId"`
	// category: all|image|video|file。合法取值与上传白名单同源,
	// 故不加 binding:"oneof",统一由 Service 裁决,避免两处枚举漂移。
	Category string `form:"category"`
}

func (q *FileListQuery) toInput() *service.FileListInput {
	return &service.FileListInput{
		Query:      q.Query,
		OriginName: q.OriginName,
		IsPublic:   q.IsPublic,
		GroupID:    q.GroupID,
		Category:   service.FileCategory(q.Category),
	}
}

// FileMoveRequest PUT /api/v1/files/group 请求体。
// groupId 用指针 + required:0(未分组)是合法目标,binding 只要求字段出现。
type FileMoveRequest struct {
	IDs     []int64 `json:"ids" binding:"required,min=1,max=200,dive,gt=0"`
	GroupID *int64  `json:"groupId" binding:"required"`
}

// FileBatchDeleteRequest POST /api/v1/files/batch-delete 请求体。
// 用 POST 而不是带请求体的 DELETE:后者在网关与访问日志里都不友好。
type FileBatchDeleteRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,max=200,dive,gt=0"`
}

// FileUploadResponse 上传响应(URL 字段由 Handler 按访问策略拼装)。
type FileUploadResponse struct {
	ID         int64  `json:"id"`
	OriginName string `json:"originName"`
	StorePath  string `json:"storePath"`
	URL        string `json:"url"`
	Size       int64  `json:"size"`
	MIME       string `json:"mime"`
}

// toUploadResponse 将 Service 结果转换为 HTTP 响应,并按公私策略生成访问地址。
func toUploadResponse(r *service.FileUploadResult, publicURLPrefix string) *FileUploadResponse {
	url := publicURLPrefix + "/" + r.StorePath
	if !r.IsPublic {
		url = "/api/v1/files/" + strconv.FormatInt(r.ID, 10) + "/download"
	}
	return &FileUploadResponse{
		ID:         r.ID,
		OriginName: r.OriginName,
		StorePath:  r.StorePath,
		URL:        url,
		Size:       r.Size,
		MIME:       r.MIME,
	}
}
