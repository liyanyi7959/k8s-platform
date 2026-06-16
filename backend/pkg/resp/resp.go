// resp 提供统一的 API 响应封装。
//
// 约定：
// - 成功/失败都使用 HTTP 200 返回（便于前端统一处理）
// - 业务错误通过 code/message 表达
// - context.Canceled 返回 HTTP 499（客户端主动断开，无需响应体）
// - context.DeadlineExceeded 返回 HTTP 504（网关超时）
package resp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ApiResponse[T any] struct {
	// Code 为业务状态码：0 表示成功，其它值表示失败。
	Code int `json:"code"`
	// Message 为可读的提示信息，通常用于前端 toast / 弹窗展示。
	Message string `json:"message"`
	// Data 为业务数据；失败时一般为 nil（或空结构）。
	Data T `json:"data"`
}

type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// OK 返回成功响应。
func OK[T any](c *gin.Context, data T) {
	c.Set("resp_code", 0)
	c.Set("resp_message", "ok")
	c.JSON(http.StatusOK, ApiResponse[T]{Code: 0, Message: "ok", Data: data})
}

// Fail 返回失败响应。
//
// 注意：这里仍返回 HTTP 200，业务错误码由 code 表达。
func Fail(c *gin.Context, code int, msg string) {
	switch strings.ToLower(strings.TrimSpace(msg)) {
	case "invalid params":
		msg = "参数错误"
	case "internal error":
		msg = "内部错误"
	}
	c.Set("resp_code", code)
	c.Set("resp_message", msg)
	c.JSON(http.StatusOK, ApiResponse[any]{Code: code, Message: msg, Data: nil})
}

// HandleK8sError 处理 K8s 操作中常见的 context 错误，返回 true 表示已处理。
//
// - context.Canceled → HTTP 499（客户端主动断开，Abort 不写响应体）
// - context.DeadlineExceeded → HTTP 504 + 超时提示
//
// 返回 false 时调用方应继续走常规 WriteServiceErr 逻辑。
func HandleK8sError(c *gin.Context, err error) bool {
	if errors.Is(err, context.Canceled) {
		c.AbortWithStatus(499) // 499 Client Closed Request（nginx 自定义）
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		c.Set("resp_code", 5000)
		c.Set("resp_message", "请求超时")
		c.JSON(http.StatusGatewayTimeout, ApiResponse[any]{ // 504
			Code:    5000,
			Message: "请求超时",
			Data:    nil,
		})
		c.Abort()
		return true
	}
	return false
}
