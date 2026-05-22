package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/auth"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// GetClaims 从 gin.Context 中读取鉴权中间件注入的 token Claims。
// 返回值说明：
// - (*auth.Claims, true)：存在且类型正确
// - (nil, false)：未登录/中间件未执行/类型不匹配
func GetClaims(c *gin.Context) (*auth.Claims, bool) {
	// GetClaims 从 gin.Context 里取出鉴权中间件写入的 Claims。
	// 适用于需要获取当前用户信息/权限点的场景。
	v, ok := c.Get("auth_claims")
	if !ok {
		return nil, false
	}
	claims, ok := v.(*auth.Claims)
	if !ok || claims == nil {
		return nil, false
	}
	return claims, true
}

// AuthRequired 为鉴权中间件（不刷新 RBAC 快照）。
// 适用于：仅需要识别登录用户，不需要实时权限变化的场景。
func AuthRequired(mgr *auth.Manager) gin.HandlerFunc {
	return AuthRequiredWithRBAC(mgr, nil)
}

// AuthRequiredWithRBAC 为鉴权中间件（可选刷新 RBAC 快照）。
//
// 认证流程：
// 1) 从 Authorization: Bearer <token> 解析 token（HTTP 请求）
// 2) 对 WebSocket 请求做兼容：允许从 query string 的 token=... 读取（浏览器限制 header 场景）
// 3) 解析 JWT 并写入 gin.Context，供后续 handler 使用
// 4) 若传入 rbacSvc，则从 DB 读取用户最新 roles/perms 并覆盖到 Claims（降低“旧 token 权限快照”问题）
func AuthRequiredWithRBAC(mgr *auth.Manager, rbacSvc *service.RbacService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mgr == nil {
			resp.Fail(c, 5000, "内部错误")
			c.Abort()
			return
		}
		authHeader := c.GetHeader("Authorization")
		var token string
		if strings.HasPrefix(authHeader, "Bearer ") {
			// 常规 HTTP 请求：从 Authorization 取 Bearer token。
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		} else {
			// WebSocket 兼容：浏览器/代理在 upgrade 时可能无法携带自定义 header，
			// 这里允许通过 query 参数 token 传递。
			connHdr := strings.ToLower(c.GetHeader("Connection"))
			upHdr := strings.ToLower(c.GetHeader("Upgrade"))
			if strings.Contains(connHdr, "upgrade") && upHdr == "websocket" {
				token = strings.TrimSpace(c.Query("token"))
			}
		}
		if token == "" {
			resp.Fail(c, 1002, "未登录或登录已过期")
			c.Abort()
			return
		}
		claims, err := mgr.ParseToken(token)
		if err != nil {
			resp.Fail(c, 1002, "未登录或登录已过期")
			c.Abort()
			return
		}

		if rbacSvc != nil && claims.UserID > 0 {
			// 从 DB 刷新角色与权限点：当后台调整了角色权限后，新请求能立即生效。
			// 解析失败不阻断请求（仍沿用 token 中的快照），避免 DB 临时抖动导致系统不可用。
			roles, perms, err := rbacSvc.GetUserRolesPerms(c.Request.Context(), uint64(claims.UserID))
			if err == nil {
				claims.Roles = roles
				claims.Perms = perms
			}
		}

		// 兼容写入：
		// - user_id：方便部分 handler 直接取用户 ID
		// - username：供审计与展示场景直接读取用户名
		// - auth_claims：提供更完整的 Claims（用户名、角色、权限点等）
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("auth_claims", claims)
		c.Next()
	}
}

func RequirePerm(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// RequirePerm 基于 token 中的权限点快照进行鉴权。
		// 依赖 AuthRequired 先执行，确保 Context 内有 Claims。
		claims, ok := GetClaims(c)
		if !ok {
			resp.Fail(c, 1002, "未登录或登录已过期")
			c.Abort()
			return
		}
		if perm == "" {
			c.Next()
			return
		}
		// 线性扫描即可满足当前规模；如权限点数量变大可改为 map 结构。
		for _, p := range claims.Perms {
			if p == perm {
				c.Next()
				return
			}
		}
		// 1003：无权限（前端可按需展示“无权限访问”）。
		resp.Fail(c, 1003, "权限不足")
		c.Abort()
	}
}

func RequireAnyPerm(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			resp.Fail(c, 1002, "未登录或登录已过期")
			c.Abort()
			return
		}
		if len(perms) == 0 {
			c.Next()
			return
		}
		for _, need := range perms {
			if need == "" {
				c.Next()
				return
			}
			for _, p := range claims.Perms {
				if p == need {
					c.Next()
					return
				}
			}
		}
		resp.Fail(c, 1003, "权限不足")
		c.Abort()
	}
}
