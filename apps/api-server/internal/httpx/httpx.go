package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AppError is the structured error type all handlers should emit through Fail.
type AppError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

func NewError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

var (
	ErrUnauthorized   = NewError(http.StatusUnauthorized, "unauthorized", "认证失败")
	ErrForbidden      = NewError(http.StatusForbidden, "forbidden", "权限不足")
	ErrConflict       = NewError(http.StatusConflict, "conflict", "状态冲突")
	ErrBadRequest     = NewError(http.StatusBadRequest, "bad_request", "请求参数错误")
	ErrNotFound       = NewError(http.StatusNotFound, "not_found", "资源不存在")
	ErrServerInternal = NewError(http.StatusInternalServerError, "internal", "服务内部错误")
)

// OK writes a 200 JSON response with the given payload.
func OK(c *gin.Context, data any) { c.JSON(http.StatusOK, data) }

// Created writes a 201 JSON response with the given payload.
func Created(c *gin.Context, data any) { c.JSON(http.StatusCreated, data) }

// Fail emits an AppError. Non-AppError errors become 500.
func Fail(c *gin.Context, err error) {
	var ae *AppError
	if errors.As(err, &ae) {
		c.AbortWithStatusJSON(ae.Status, ae)
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, &AppError{
		Code:    "internal",
		Message: "服务内部错误",
	})
}
