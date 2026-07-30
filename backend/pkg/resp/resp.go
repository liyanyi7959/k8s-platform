package resp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/pkg/problem"
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

const (
	// CodePlatformSessionExpired is reserved for a missing or expired AIOPS login session.
	// Frontend clients use it to clear the platform token and return to the login page.
	CodePlatformSessionExpired = 1002

	// CodeClusterCredentialInvalid identifies authentication failures returned by a
	// managed Kubernetes cluster. It must not be confused with a platform session
	// failure: callers can keep using AIOPS while repairing the affected cluster.
	CodeClusterCredentialInvalid = 2001
)

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
	if isV2(c) {
		c.JSON(http.StatusOK, data)
		return
	}
	c.Set("resp_code", 0)
	c.Set("resp_message", "ok")
	c.JSON(http.StatusOK, ApiResponse[T]{Code: 0, Message: "ok", Data: data})
}

func Fail(c *gin.Context, code int, msg string) {
	FailWithData(c, code, msg, any(nil))
}

func FailWithData[T any](c *gin.Context, code int, msg string, data T) {
	msg = normalizeMessage(msg)
	if isV2(c) {
		problem.WriteKind(c, legacyProblemKind(code), msg)
		return
	}
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
		if isV2(c) {
			problem.Write(c, http.StatusGatewayTimeout, problem.TypeBaseURI+"timeout", "请求超时", "请求超时")
			c.Abort()
			return true
		}
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

func isV2(c *gin.Context) bool {
	return c != nil && c.GetString("api_version") == "v2"
}

func legacyProblemKind(code int) problem.Kind {
	switch code {
	case 1002, 4010:
		return problem.KindUnauthorized
	case 1003:
		return problem.KindForbidden
	case 2001:
		return problem.KindClusterCredentialInvalid
	case 4000:
		return problem.KindInvalidRequest
	case 4040:
		return problem.KindNotFound
	case 4090:
		return problem.KindConflict
	default:
		return problem.KindInternal
	}
}
