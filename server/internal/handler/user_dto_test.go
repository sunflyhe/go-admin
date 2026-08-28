// 验证 Handler 能正确把 HTTP 绑定 DTO 转换为 Service 输入 DTO。
package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func bindListQuery(t *testing.T, raw string) *UserListQuery {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", raw, nil)
	var q UserListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		t.Fatalf("绑定查询参数失败: %v", err)
	}
	return &q
}

func TestUserListQueryToInput(t *testing.T) {
	q := bindListQuery(t, "/users?page=2&pageSize=50&username=ali&status=1&startTime=2026-01-01+00%3A00%3A00")
	input := q.toInput()
	if input.Page != 2 || input.PageSize != 50 {
		t.Fatalf("分页参数应透传: %+v", input.Query)
	}
	if input.Username != "ali" || input.Status != 1 {
		t.Fatalf("筛选参数应透传: %+v", input)
	}
	if input.StartTime != "2026-01-01 00:00:00" {
		t.Fatalf("时间参数应透传: %s", input.StartTime)
	}
}

func TestUserCreateRequestToInput(t *testing.T) {
	req := UserCreateRequest{
		Username: "carol", Password: "12345678",
		Nickname: "Carol", Email: "c@example.com", Status: 1,
	}
	input := req.toInput()
	if input.Username != "carol" || input.Password != "12345678" || input.Email != "c@example.com" {
		t.Fatalf("创建字段应完整透传: %+v", input)
	}
}

func TestUserUpdateRequestToInputDropsPassword(t *testing.T) {
	req := UserUpdateRequest{Nickname: "New", Status: 2}
	input := req.toInput()
	if input.Password != "" {
		t.Fatal("更新转换不得携带密码,修改密码必须走重置接口")
	}
	if input.Nickname != "New" || input.Status != 2 {
		t.Fatalf("更新字段应透传: %+v", input)
	}
}
