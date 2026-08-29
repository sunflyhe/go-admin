// 文件列表 DTO 绑定与到 Service 输入的转换测试。
package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
)

func bindFileListQuery(t *testing.T, raw string) *FileListQuery {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", raw, nil)
	var q FileListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		t.Fatalf("绑定查询参数失败: %v", err)
	}
	return &q
}

func TestFileListQueryCarriesCategoryToInput(t *testing.T) {
	q := bindFileListQuery(t, "/files?page=2&pageSize=20&category=image&originName=logo")
	input := q.toInput()
	if input.Category != service.FileCategoryImage {
		t.Fatalf("category 应透传给 Service: %+v", input)
	}
	if input.Page != 2 || input.OriginName != "logo" {
		t.Fatalf("其余参数应照常透传: %+v", input)
	}
}

func TestFileListQueryWithoutCategoryStaysUnfiltered(t *testing.T) {
	// 省略 category 时必须保持零值,由 Service 归一为"不筛选";
	// 若在 Handler 里填默认值,旧调用方的行为会被悄悄改变。
	input := bindFileListQuery(t, "/files").toInput()
	if input.Category != "" {
		t.Fatalf("未传 category 应为零值: %q", input.Category)
	}
	if input.IsPublic != nil {
		t.Fatalf("未传 isPublic 应为 nil(不筛选): %v", *input.IsPublic)
	}
}

// TestFileListQueryDistinguishesUnfiledFromAll groupId 的"没传"与"传 0"是两回事:
// 前者是全部分组,后者是未分组。若塌成同一个零值,左栏点"未分组"会列出所有文件。
func TestFileListQueryDistinguishesUnfiledFromAll(t *testing.T) {
	if got := bindFileListQuery(t, "/files").toInput(); got.GroupID != nil {
		t.Fatalf("未传 groupId 应为 nil(全部分组): %v", *got.GroupID)
	}
	got := bindFileListQuery(t, "/files?groupId=0").toInput()
	if got.GroupID == nil || *got.GroupID != 0 {
		t.Fatalf("groupId=0 应解析为指向 0 的指针: %+v", got.GroupID)
	}
	got = bindFileListQuery(t, "/files?groupId=7").toInput()
	if got.GroupID == nil || *got.GroupID != 7 {
		t.Fatalf("groupId=7 应原样透传: %+v", got.GroupID)
	}
}

// bindJSON 复用生产 Handler 的绑定路径,让 binding 标签本身被测试覆盖。
func bindJSON[T any](t *testing.T, raw string) (T, error) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("PUT", "/files/group", strings.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	var req T
	err := c.ShouldBindJSON(&req)
	return req, err
}

func TestFileMoveRequestBinding(t *testing.T) {
	// groupId=0(未分组)是合法目标:指针 + required 只要求字段出现
	req, err := bindJSON[FileMoveRequest](t, `{"ids":[1,2],"groupId":0}`)
	if err != nil {
		t.Fatalf("移动到未分组应通过绑定: %v", err)
	}
	if req.GroupID == nil || *req.GroupID != 0 || len(req.IDs) != 2 {
		t.Fatalf("绑定结果不符: %+v %v", req.GroupID, req.IDs)
	}

	for _, raw := range []string{
		`{"ids":[1,2]}`,           // 缺目标分组:会误删归属,必须拒
		`{"ids":[],"groupId":3}`,  // 空集合
		`{"ids":[0],"groupId":3}`, // 非法 id
		`{"groupId":3}`,
	} {
		if _, err := bindJSON[FileMoveRequest](t, raw); err == nil {
			t.Fatalf("请求体 %s 应被绑定拒绝", raw)
		}
	}
}

func TestFileBatchDeleteRequestBinding(t *testing.T) {
	if _, err := bindJSON[FileBatchDeleteRequest](t, `{"ids":[1]}`); err != nil {
		t.Fatalf("单个 id 应通过: %v", err)
	}
	for _, raw := range []string{`{}`, `{"ids":[]}`, `{"ids":[-1]}`} {
		if _, err := bindJSON[FileBatchDeleteRequest](t, raw); err == nil {
			t.Fatalf("请求体 %s 应被绑定拒绝", raw)
		}
	}
}
