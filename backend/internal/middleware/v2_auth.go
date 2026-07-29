package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/auth"
	"k8s-platform-backend/pkg/problem"
)

// AuthRequiredV2 keeps authentication behavior aligned with v1 while emitting
// RFC 9457 Problem Details for the v2 API contract.
func AuthRequiredV2(mgr *auth.Manager, authorizationReader RolesPermissionsReader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mgr == nil {
			problem.Write(c, 500, "https://aiops.local/problems/internal", "内部错误", "认证服务未初始化")
			c.Abort()
			return
		}
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			problem.Unauthorized(c, "未提供有效的 Bearer Token")
			c.Abort()
			return
		}
		claims, err := mgr.ParseToken(strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")))
		if err != nil {
			problem.Unauthorized(c, "登录已过期，请重新登录")
			c.Abort()
			return
		}
		if authorizationReader != nil && claims.UserID > 0 {
			if roles, perms, refreshErr := authorizationReader.RolesPermissions(c.Request.Context(), uint64(claims.UserID)); refreshErr == nil {
				claims.Roles, claims.Perms = roles, perms
			}
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("auth_claims", claims)
		c.Next()
	}
}

func RequirePermV2(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			problem.Unauthorized(c, "登录已过期，请重新登录")
			c.Abort()
			return
		}
		for _, value := range claims.Perms {
			if value == permission {
				c.Next()
				return
			}
		}
		problem.Forbidden(c, "当前用户缺少权限："+permission)
		c.Abort()
	}
}
