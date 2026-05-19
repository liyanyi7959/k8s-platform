// cors.go 提供 CORS 跨域中间件。
//
// 设计说明：
// - 生产环境建议通过 Nginx/网关处理 CORS；此中间件主要用于开发/测试。
// - 对 OPTIONS 预检请求直接返回 204，避免进入后续业务逻辑。
// - Access-Control-Max-Age 设为 10 分钟，减少浏览器预检频次。
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORS 返回一个处理 CORS 响应头的中间件。
// 它会回显请求中的 Origin 头（而非 `*`），以兼容带凭据的场景。
func CORS() gin.HandlerFunc {
	maxAge := strconv.Itoa(int((10 * time.Minute).Seconds()))
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Request-Id")
			c.Header("Access-Control-Max-Age", maxAge)
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
