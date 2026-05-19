// middleware 存放 Gin 中间件实现。
//
// RequestID 中间件的职责：
// - 为每个请求分配一个 request_id
// - 将 request_id 写入 gin context，供服务端链路追踪
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"k8s-platform-backend/pkg/resp"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get("request_id"); ok {
			c.Next()
			return
		}

		// 如果上游（例如网关/前端）已传入 X-Request-Id，则沿用；
		// 否则服务端生成一个随机 ID。
		rid := c.GetHeader("X-Request-Id")
		if rid == "" {
			rid = newRequestID()
		}
		c.Set("request_id", rid)
		c.Next()
	}
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func AccessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		query := sanitizeQuery(c.Request.URL.RawQuery)
		clientIP := c.ClientIP()

		var rid string
		if v, ok := c.Get("request_id"); ok {
			if s, ok := v.(string); ok {
				rid = s
			}
		}

		respCode := 0
		if v, ok := c.Get("resp_code"); ok {
			switch t := v.(type) {
			case int:
				respCode = t
			case int32:
				respCode = int(t)
			case int64:
				respCode = int(t)
			}
		}

		fields := []zap.Field{
			zap.Int("status", status),
			zap.Int("code", respCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", clientIP),
			zap.Duration("latency", latency),
		}
		if rid != "" {
			fields = append(fields, zap.String("request_id", rid))
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		switch {
		case status >= 500 || respCode >= 5000:
			zap.L().Error("http_request", fields...)
		case status >= 400 || respCode > 0:
			zap.L().Warn("http_request", fields...)
		default:
			zap.L().Info("http_request", fields...)
		}
	}
}

func sanitizeQuery(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return raw
	}
	for _, key := range []string{"token", "access_token", "authorization", "auth"} {
		if _, ok := values[key]; ok {
			values.Set(key, "***")
		}
	}
	return values.Encode()
}

func RecoveryWithZap() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		var rid string
		if v, ok := c.Get("request_id"); ok {
			if s, ok := v.(string); ok {
				rid = s
			}
		}
		if rid == "" {
			rid = c.GetHeader("X-Request-Id")
			if rid == "" {
				rid = newRequestID()
			}
			c.Set("request_id", rid)
		}

		fields := []zap.Field{
			zap.Any("panic", recovered),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.ByteString("stack", debug.Stack()),
		}
		if rid != "" {
			fields = append(fields, zap.String("request_id", rid))
		}
		zap.L().Error("panic_recovered", fields...)

		resp.Fail(c, 5000, "内部错误")
		c.Abort()
	})
}
