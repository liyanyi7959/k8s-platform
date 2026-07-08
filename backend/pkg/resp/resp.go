package resp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ApiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func normalizeMessage(msg string) string {
	switch strings.ToLower(strings.TrimSpace(msg)) {
	case "invalid params":
		return "参数错误"
	case "internal error":
		return "内部错误"
	default:
		return msg
	}
}

func OK[T any](c *gin.Context, data T) {
	c.Set("resp_code", 0)
	c.Set("resp_message", "ok")
	c.JSON(http.StatusOK, ApiResponse[T]{Code: 0, Message: "ok", Data: data})
}

func Fail(c *gin.Context, code int, msg string) {
	FailWithData(c, code, msg, any(nil))
}

func FailWithData[T any](c *gin.Context, code int, msg string, data T) {
	msg = normalizeMessage(msg)
	c.Set("resp_code", code)
	c.Set("resp_message", msg)
	c.JSON(http.StatusOK, ApiResponse[T]{Code: code, Message: msg, Data: data})
}

func HandleK8sError(c *gin.Context, err error) bool {
	if errors.Is(err, context.Canceled) {
		c.AbortWithStatus(499)
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		c.Set("resp_code", 5000)
		c.Set("resp_message", "请求超时")
		c.JSON(http.StatusGatewayTimeout, ApiResponse[any]{
			Code:    5000,
			Message: "请求超时",
			Data:    nil,
		})
		c.Abort()
		return true
	}
	return false
}
