// Package validate 提供参数解析辅助函数。
package validate

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
)

// PathID 解析路径参数 :id,非法值返回参数错误。
func PathID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errs.InvalidParam("无效的 ID 参数")
	}
	return id, nil
}
