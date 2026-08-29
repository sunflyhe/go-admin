// Package resp 提供统一 JSON 响应格式:
//
//	{"code":0,"message":"success","data":{}}
package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// OK 输出成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: errs.CodeOK, Message: "success", Data: data})
}

// Created 输出 201 成功响应。
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Body{Code: errs.CodeOK, Message: "success", Data: data})
}

// Fail 输出失败响应;未知错误统一降级为 internal,防止内部细节泄露。
// 原始错误由调用方或访问日志中间件负责记录。
func Fail(c *gin.Context, err error) {
	appErr := errs.From(err)
	status := appErr.HTTP
	if status == 0 {
		status = http.StatusInternalServerError
	}
	c.AbortWithStatusJSON(status, Body{Code: appErr.Code, Message: appErr.Message})
}
