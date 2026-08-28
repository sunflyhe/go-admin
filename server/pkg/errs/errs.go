// Package errs 定义全局业务错误码与错误类型。
// 统一约定:code 为 0 表示成功,非 0 表示业务失败,HTTP 状态码与 code 分离。
package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// 业务错误码。按 HTTP 语义分段,便于前端统一处理。
const (
	CodeOK                 = 0
	CodeInvalidParam       = 40000
	CodeUnauthorized       = 40100
	CodeTokenExpired       = 40101
	CodeForbidden          = 40300
	CodeNotFound           = 40400
	CodeConflict           = 40900
	CodeTooManyReqs        = 42900
	CodeInternal           = 50000
	CodeServiceUnavailable = 50300
)

// AppError 携带业务错误码、对外消息与 HTTP 状态码。
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	HTTP    int    `json:"-"`
	cause   error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.cause }

// WithCause 附加底层错误用于日志排查,不改变对外消息。
func (e *AppError) WithCause(err error) *AppError {
	return &AppError{Code: e.Code, Message: e.Message, HTTP: e.HTTP, cause: err}
}

// New 创建自定义业务错误。
func New(code int, message string, httpStatus int) *AppError {
	return &AppError{Code: code, Message: message, HTTP: httpStatus}
}

func InvalidParam(msg string) *AppError {
	return &AppError{Code: CodeInvalidParam, Message: msg, HTTP: http.StatusBadRequest}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: CodeUnauthorized, Message: msg, HTTP: http.StatusUnauthorized}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: CodeForbidden, Message: msg, HTTP: http.StatusForbidden}
}

func NotFound(msg string) *AppError {
	return &AppError{Code: CodeNotFound, Message: msg, HTTP: http.StatusNotFound}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: CodeConflict, Message: msg, HTTP: http.StatusConflict}
}

func TooManyRequests(msg string) *AppError {
	return &AppError{Code: CodeTooManyReqs, Message: msg, HTTP: http.StatusTooManyRequests}
}

func Internal(msg string) *AppError {
	return &AppError{Code: CodeInternal, Message: msg, HTTP: http.StatusInternalServerError}
}

// From 将任意 error 归一化为 *AppError;未知的错误按内部错误处理,避免内部细节外泄。
func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return &AppError{Code: CodeInternal, Message: "internal server error", HTTP: http.StatusInternalServerError, cause: err}
}
