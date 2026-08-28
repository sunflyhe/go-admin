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
}

func (q *FileListQuery) toInput() *service.FileListInput {
	return &service.FileListInput{
		Query:      q.Query,
		OriginName: q.OriginName,
		IsPublic:   q.IsPublic,
	}
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
