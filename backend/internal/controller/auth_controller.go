package controller

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/auth"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type AuthController struct {
	// jwtMgr 负责签发/解析 JWT。
	jwtMgr *auth.Manager
	// rbacSvc 负责用户校验、查询角色与权限点、修改密码等业务逻辑。
	rbacSvc *service.RbacService
	// auditSvc 负责记录登录/退出等认证类审计日志。
	auditSvc *service.AuditService
	// tokenTTL 为 access token 的有效期（由配置注入）。
	tokenTTL time.Duration
}

func NewAuthController(jwtMgr *auth.Manager, rbacSvc *service.RbacService, auditSvc *service.AuditService, tokenTTL time.Duration) *AuthController {
	// NewAuthController 通过依赖注入方式组装鉴权控制器，避免使用包级全局变量。
	if tokenTTL <= 0 {
		tokenTTL = 7 * 24 * time.Hour
	}
	return &AuthController{jwtMgr: jwtMgr, rbacSvc: rbacSvc, auditSvc: auditSvc, tokenTTL: tokenTTL}
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginUser struct {
	ID          uint64   `json:"id"`
	Username    string   `json:"username"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type LoginResp struct {
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	User        LoginUser `json:"user"`
}

func (ac *AuthController) recordAuthAudit(c *gin.Context, userID uint64, username, resourceName string, code int, detail string) {
	if ac == nil || ac.auditSvc == nil {
		return
	}
	path := c.Request.URL.Path
	clientIP := c.ClientIP()
	rid := ""
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			rid = s
		}
	}
	auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	go func() {
		defer cancel()
		ac.auditSvc.Record(auditCtx, service.AuditEntry{
			UserID:       userID,
			Username:     username,
			Action:       "auth",
			Resource:     "session",
			ResourceName: resourceName,
			Path:         path,
			StatusCode:   code,
			Detail:       detail,
			ClientIP:     clientIP,
			RequestID:    rid,
		})
	}()
}

// @Summary 账号密码登录
// @Description 传入用户名与密码，返回 access_token、有效期与用户角色/权限快照
// @Tags 认证接口
// @Accept json
// @Produce json
// @Param body body loginReq true "登录参数"
// @Success 200 {object} resp.Result{data=LoginResp} "登录成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 401 {object} resp.Result "未授权"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /auth/login [post]
func (ac *AuthController) Login(c *gin.Context) {
	// Login 用于账号密码登录。
	// 成功时返回 access_token、expires_in 与 user 信息（含角色/权限点）。
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	username := strings.TrimSpace(req.Username)
	password := req.Password
	if username == "" || password == "" {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	if ac.jwtMgr == nil || ac.rbacSvc == nil {
		// 依赖未正确注入属于服务端错误。
		resp.Fail(c, 5000, "内部错误")
		return
	}

	// 账号密码校验由 service 统一处理：
	// - 用户不存在/密码不匹配/用户被禁用时都返回错误（对外统一表现为 unauthorized）
	// - 成功时返回用户信息及其 roles/perms（用于写入 token）
	u, err := ac.rbacSvc.Authenticate(c.Request.Context(), username, password)
	if err != nil {
		switch err {
		case service.ErrInvalidParams:
			ac.recordAuthAudit(c, 0, username, "login", 4000, "参数错误")
			resp.Fail(c, 4000, "参数错误")
		case service.ErrAuthUserNotFound:
			ac.recordAuthAudit(c, 0, username, "login", 4000, "用户不存在")
			resp.Fail(c, 4000, "用户不存在")
		case service.ErrAuthUserDisabled:
			ac.recordAuthAudit(c, 0, username, "login", 4000, "账号已被禁用")
			resp.Fail(c, 4000, "账号已被禁用")
		case service.ErrAuthPasswordIncorrect:
			ac.recordAuthAudit(c, 0, username, "login", 4000, "密码不正确")
			resp.Fail(c, 4000, "密码不正确")
		default:
			ac.recordAuthAudit(c, 0, username, "login", 5000, "内部错误")
			resp.Fail(c, 5000, "内部错误")
		}
		return
	}

	// 将当前 roles/perms 作为“快照”写入 token，后续请求无需每次查库即可鉴权。
	token, err := ac.jwtMgr.IssueToken(auth.Claims{
		UserID:   u.ID,
		Username: u.Username,
		Roles:    u.Roles,
		Perms:    u.Perms,
	}, ac.tokenTTL)
	if err != nil {
		ac.recordAuthAudit(c, uint64(u.ID), u.Username, "login", 5000, "签发令牌失败")
		resp.Fail(c, 5000, "内部错误")
		return
	}
	ac.recordAuthAudit(c, uint64(u.ID), u.Username, "login", 0, "登录成功")

	resp.OK(c, gin.H{
		"access_token": token,
		"expires_in":   int(ac.tokenTTL.Seconds()),
		"user": gin.H{
			"id":          u.ID,
			"username":    u.Username,
			"status":      u.Status,
			"roles":       u.Roles,
			"permissions": u.Perms,
		},
	})
}

// @Summary 获取当前用户信息
// @Description 返回 token 中携带的用户信息与角色/权限快照
// @Tags 认证接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} resp.Result{data=LoginUser} "查询成功"
// @Failure 401 {object} resp.Result "未授权"
// @Router /auth/me [get]
func (ac *AuthController) Me(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	token := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if token == "" || ac.jwtMgr == nil {
		resp.OK(c, gin.H{
			"id":          uint64(0),
			"username":    "",
			"status":      "",
			"roles":       []string{},
			"permissions": []string{},
		})
		return
	}
	claims, err := ac.jwtMgr.ParseToken(token)
	if err != nil || claims == nil {
		resp.OK(c, gin.H{
			"id":          uint64(0),
			"username":    "",
			"status":      "",
			"roles":       []string{},
			"permissions": []string{},
		})
		return
	}
	resp.OK(c, gin.H{
		"id":          claims.UserID,
		"username":    claims.Username,
		"status":      "active",
		"roles":       claims.Roles,
		"permissions": claims.Perms,
	})
}

// @Summary 退出登录
// @Description JWT 为无状态，前端清理 token 即可
// @Tags 认证接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} resp.Result "退出成功"
// @Failure 401 {object} resp.Result "未授权"
// @Router /auth/logout [post]
func (ac *AuthController) Logout(c *gin.Context) {
	// Logout 当前为无状态接口（JWT 无服务端会话）；前端清理 token 即可。
	userID := uint64(0)
	username := ""
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") && ac.jwtMgr != nil {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token != "" {
			if claims, err := ac.jwtMgr.ParseToken(token); err == nil && claims != nil {
				userID = uint64(claims.UserID)
				username = claims.Username
			}
		}
	}
	ac.recordAuthAudit(c, userID, username, "logout", 0, "退出登录")
	resp.OK(c, gin.H{})
}

type changePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// @Summary 修改当前用户密码
// @Description 传入旧密码与新密码，校验通过后更新密码
// @Tags 认证接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body changePasswordReq true "修改密码参数"
// @Success 200 {object} resp.Result "修改成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 401 {object} resp.Result "未授权"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /auth/change-password [post]
func (ac *AuthController) ChangePassword(c *gin.Context) {
	// ChangePassword 修改当前登录用户密码：
	// - 通过 token 识别 user_id
	// - 校验 old_password，成功后更新为 new_password（bcrypt hash）
	if ac.rbacSvc == nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if strings.TrimSpace(req.OldPassword) == "" || strings.TrimSpace(req.NewPassword) == "" {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	v, ok := c.Get("auth_claims")
	if !ok {
		resp.Fail(c, 1002, "未登录或登录已过期")
		return
	}
	claims, ok := v.(*auth.Claims)
	if !ok || claims == nil || claims.UserID <= 0 {
		resp.Fail(c, 1002, "未登录或登录已过期")
		return
	}
	if err := ac.rbacSvc.ChangePassword(c.Request.Context(), uint64(claims.UserID), req.OldPassword, req.NewPassword); err != nil {
		switch err {
		case service.ErrInvalidParams:
			resp.Fail(c, 4000, "参数错误")
		case service.ErrAuthOldPasswordIncorrect:
			resp.Fail(c, 1002, "旧密码不正确")
		case service.ErrNotFound:
			resp.Fail(c, 1002, "未认证")
		default:
			resp.Fail(c, 5000, "内部错误")
		}
		return
	}
	resp.OK(c, gin.H{})
}
