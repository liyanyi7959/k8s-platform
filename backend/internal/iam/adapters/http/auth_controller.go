package http

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/auth"
	iamapp "k8s-platform-backend/internal/iam/application"
	iamdomain "k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/pkg/resp"
)

type AuthController struct {
	jwtMgr          *auth.Manager
	authSvc         *iamapp.AuthService
	auditSvc        AuthAuditRecorder
	captchaSvc      *iamapp.CaptchaService
	loginAttemptSvc *iamapp.LoginAttemptService
	pwdResetSvc     *iamapp.PasswordResetService
	tokenTTL        time.Duration
}

// AuthAuditEntry captures an authentication event without coupling IAM's HTTP
// adapter to the audit bounded context.
type AuthAuditEntry struct {
	UserID, ClusterID      uint64
	Username, Action       string
	Resource, ResourceName string
	Path, Detail, ClientIP string
	RequestID              string
	StatusCode             int
}

// AuthAuditRecorder is supplied by the composition root and translates an IAM
// authentication event into the platform's audit implementation.
type AuthAuditRecorder func(context.Context, AuthAuditEntry)

func NewAuthController(
	jwtMgr *auth.Manager,
	authSvc *iamapp.AuthService,
	auditSvc AuthAuditRecorder,
	captchaSvc *iamapp.CaptchaService,
	loginAttemptSvc *iamapp.LoginAttemptService,
	pwdResetSvc *iamapp.PasswordResetService,
	tokenTTL time.Duration,
) *AuthController {
	if tokenTTL <= 0 {
		tokenTTL = 7 * 24 * time.Hour
	}
	return &AuthController{
		jwtMgr:          jwtMgr,
		authSvc:         authSvc,
		auditSvc:        auditSvc,
		captchaSvc:      captchaSvc,
		loginAttemptSvc: loginAttemptSvc,
		pwdResetSvc:     pwdResetSvc,
		tokenTTL:        tokenTTL,
	}
}

type loginReq struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token"`
	CaptchaX     int    `json:"captcha_x"`
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

const (
	loginCodeInvalidCredentials = 4101
	loginCodeAccountDisabled    = 4102
	loginCodeAccountLocked      = 4103
	loginCodeCaptchaInvalid     = 4104
)

type loginFailureData struct {
	Reason               string   `json:"reason"`
	FailedAttempts       int      `json:"failed_attempts,omitempty"`
	RemainingAttempts    int      `json:"remaining_attempts,omitempty"`
	MaxAttempts          int      `json:"max_attempts,omitempty"`
	Locked               bool     `json:"locked,omitempty"`
	LockRemainingSeconds int      `json:"lock_remaining_seconds,omitempty"`
	LockDurationSeconds  int      `json:"lock_duration_seconds,omitempty"`
	CanResetPassword     bool     `json:"can_reset_password,omitempty"`
	Suggestions          []string `json:"suggestions,omitempty"`
}

func (ac *AuthController) recordAuthAudit(c *gin.Context, userID uint64, username, resourceName string, code int, detail string) {
	if ac == nil || ac.auditSvc == nil {
		return
	}
	path := c.Request.URL.Path
	clientIP := c.ClientIP()
	requestID := ""
	if value, ok := c.Get("request_id"); ok {
		if id, ok := value.(string); ok {
			requestID = id
		}
	}
	auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	go func() {
		defer cancel()
		ac.auditSvc(auditCtx, AuthAuditEntry{
			UserID:       userID,
			Username:     username,
			Action:       "auth",
			Resource:     "session",
			ResourceName: resourceName,
			Path:         path,
			StatusCode:   code,
			Detail:       detail,
			ClientIP:     clientIP,
			RequestID:    requestID,
		})
	}()
}

func normalizeSuggestions(items ...string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func buildLoginFailureData(reason string, status *iamapp.LoginAttemptStatus, canResetPassword bool, suggestions ...string) loginFailureData {
	data := loginFailureData{
		Reason:           reason,
		CanResetPassword: canResetPassword,
		Suggestions:      normalizeSuggestions(suggestions...),
	}
	if status != nil {
		data.FailedAttempts = status.FailedAttempts
		data.RemainingAttempts = status.RemainingAttempts
		data.MaxAttempts = status.MaxAttempts
		data.Locked = status.Locked
		data.LockRemainingSeconds = status.LockRemainingSeconds
		data.LockDurationSeconds = status.LockDurationSeconds
	}
	return data
}

func (ac *AuthController) failLogin(c *gin.Context, code int, msg string, data loginFailureData) {
	resp.FailWithData(c, code, msg, data)
}

func (ac *AuthController) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ac.recordAuthAudit(c, 0, "", "login", 4000, "参数错误")
		ac.failLogin(c, 4000, "参数错误", buildLoginFailureData(
			"invalid_params",
			nil,
			false,
			"请填写用户名和密码后重试",
		))
		return
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password
	if username == "" || password == "" {
		ac.recordAuthAudit(c, 0, username, "login", 4000, "参数错误")
		ac.failLogin(c, 4000, "参数错误", buildLoginFailureData(
			"invalid_params",
			nil,
			false,
			"请填写用户名和密码后重试",
		))
		return
	}

	if ac.jwtMgr == nil || ac.authSvc == nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}

	if ac.loginAttemptSvc != nil {
		lockStatus := ac.loginAttemptSvc.GetStatus(c.Request.Context(), username)
		if lockStatus.Locked {
			ac.recordAuthAudit(c, 0, username, "login", loginCodeAccountLocked, "账号已锁定")
			ac.failLogin(c, loginCodeAccountLocked, iamapp.FormatLockMessage(lockStatus.LockRemainingSeconds), buildLoginFailureData(
				"account_locked",
				&lockStatus,
				true,
				"请等待锁定结束后再试",
				"如已忘记密码，可使用忘记密码功能重置",
			))
			return
		}
	}

	if ac.captchaSvc != nil && ac.captchaSvc.CacheEnabled() {
		if strings.TrimSpace(req.CaptchaToken) == "" {
			ac.recordAuthAudit(c, 0, username, "login", loginCodeCaptchaInvalid, "缺少验证码")
			ac.failLogin(c, loginCodeCaptchaInvalid, "请先完成安全验证后再登录", buildLoginFailureData(
				"captcha_invalid",
				nil,
				false,
				"请先完成滑块验证后重新提交",
			))
			return
		}
		if err := ac.captchaSvc.Verify(c.Request.Context(), req.CaptchaToken, req.CaptchaX); err != nil {
			ac.recordAuthAudit(c, 0, username, "login", loginCodeCaptchaInvalid, "验证码校验失败")
			ac.failLogin(c, loginCodeCaptchaInvalid, "安全验证失败，请重新完成验证", buildLoginFailureData(
				"captcha_invalid",
				nil,
				false,
				"请重新完成安全验证后再试",
			))
			return
		}
	}

	user, err := ac.authSvc.Authenticate(c.Request.Context(), username, password)
	if err != nil {
		var attemptStatus *iamapp.LoginAttemptStatus
		if ac.loginAttemptSvc != nil && (err == iamdomain.ErrPasswordIncorrect || err == iamdomain.ErrUserNotFound) {
			status := ac.loginAttemptSvc.RecordFailureDetail(c.Request.Context(), username)
			attemptStatus = &status
			if status.Locked {
				ac.recordAuthAudit(c, 0, username, "login", loginCodeAccountLocked, "账号已锁定")
				ac.failLogin(c, loginCodeAccountLocked, iamapp.FormatLockMessage(status.LockRemainingSeconds), buildLoginFailureData(
					"account_locked",
					&status,
					true,
					"请等待锁定结束后再试",
					"如已忘记密码，可使用忘记密码功能重置",
				))
				return
			}
		}

		switch err {
		case iamdomain.ErrInvalidParams:
			ac.recordAuthAudit(c, 0, username, "login", 4000, "参数错误")
			ac.failLogin(c, 4000, "参数错误", buildLoginFailureData(
				"invalid_params",
				nil,
				false,
				"请检查请求参数后重试",
			))
		case iamdomain.ErrUserNotFound:
			ac.recordAuthAudit(c, 0, username, "login", loginCodeInvalidCredentials, "用户不存在")
			ac.failLogin(c, loginCodeInvalidCredentials, "用户名或密码错误", buildLoginFailureData(
				"invalid_credentials",
				attemptStatus,
				true,
				"请检查用户名和密码是否输入正确",
				"连续失败过多会临时锁定账号",
				"如已忘记密码，可使用忘记密码功能重置",
			))
		case iamdomain.ErrUserDisabled:
			ac.recordAuthAudit(c, 0, username, "login", loginCodeAccountDisabled, "账号已被禁用")
			ac.failLogin(c, loginCodeAccountDisabled, "账号已被禁用，请联系管理员", buildLoginFailureData(
				"account_disabled",
				nil,
				false,
				"请联系管理员检查账号状态",
			))
		case iamdomain.ErrPasswordIncorrect:
			ac.recordAuthAudit(c, 0, username, "login", loginCodeInvalidCredentials, "密码不正确")
			ac.failLogin(c, loginCodeInvalidCredentials, "用户名或密码错误", buildLoginFailureData(
				"invalid_credentials",
				attemptStatus,
				true,
				"请检查用户名和密码是否输入正确",
				"连续失败过多会临时锁定账号",
				"如已忘记密码，可使用忘记密码功能重置",
			))
		default:
			ac.recordAuthAudit(c, 0, username, "login", 5000, "内部错误")
			resp.Fail(c, 5000, "内部错误")
		}
		return
	}

	if ac.loginAttemptSvc != nil {
		ac.loginAttemptSvc.ResetFailures(c.Request.Context(), username)
	}

	token, err := ac.jwtMgr.IssueToken(auth.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    user.Roles,
		Perms:    user.Permissions,
	}, ac.tokenTTL)
	if err != nil {
		ac.recordAuthAudit(c, uint64(user.ID), user.Username, "login", 5000, "签发令牌失败")
		resp.Fail(c, 5000, "内部错误")
		return
	}

	ac.recordAuthAudit(c, uint64(user.ID), user.Username, "login", 0, "登录成功")
	resp.OK(c, gin.H{
		"access_token": token,
		"expires_in":   int(ac.tokenTTL.Seconds()),
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"status":      user.Status,
			"roles":       user.Roles,
			"permissions": user.Permissions,
		},
	})
}

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

	roles := claims.Roles
	perms := claims.Perms
	if ac.authSvc != nil && claims.UserID > 0 {
		if latestRoles, latestPerms, err := ac.authSvc.RolesPermissions(c.Request.Context(), uint64(claims.UserID)); err == nil {
			roles = latestRoles
			perms = latestPerms
		}
	}

	resp.OK(c, gin.H{
		"id":          claims.UserID,
		"username":    claims.Username,
		"status":      "active",
		"roles":       roles,
		"permissions": perms,
	})
}

func (ac *AuthController) Logout(c *gin.Context) {
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

func (ac *AuthController) ChangePassword(c *gin.Context) {
	if ac.authSvc == nil {
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

	value, ok := c.Get("auth_claims")
	if !ok {
		resp.Fail(c, 1002, "未登录或登录已过期")
		return
	}
	claims, ok := value.(*auth.Claims)
	if !ok || claims == nil || claims.UserID <= 0 {
		resp.Fail(c, 1002, "未登录或登录已过期")
		return
	}

	if err := ac.authSvc.ChangePassword(c.Request.Context(), uint64(claims.UserID), req.OldPassword, req.NewPassword); err != nil {
		switch err {
		case iamdomain.ErrInvalidParams:
			resp.Fail(c, 4000, "参数错误")
		case iamdomain.ErrOldPasswordIncorrect:
			resp.Fail(c, 1002, "旧密码不正确")
		case iamdomain.ErrNotFound:
			resp.Fail(c, 1002, "未认证")
		default:
			resp.Fail(c, 5000, "内部错误")
		}
		return
	}

	resp.OK(c, gin.H{})
}

func (ac *AuthController) GetCaptcha(c *gin.Context) {
	if ac.captchaSvc == nil || !ac.captchaSvc.CacheEnabled() {
		resp.OK(c, gin.H{"enabled": false})
		return
	}

	puzzle, err := ac.captchaSvc.Generate(c.Request.Context())
	if err != nil {
		resp.Fail(c, 5000, "验证码生成失败")
		return
	}

	resp.OK(c, gin.H{
		"enabled":     true,
		"token":       puzzle.Token,
		"target_x":    puzzle.TargetX,
		"track_width": puzzle.TrackWidth,
	})
}

type requestResetReq struct {
	Identifier string `json:"identifier"`
}

type resetPasswordReq struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (ac *AuthController) RequestPasswordReset(c *gin.Context) {
	if ac.pwdResetSvc == nil {
		resp.Fail(c, 5000, "密码重置服务不可用")
		return
	}

	var req requestResetReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Identifier) == "" {
		resp.Fail(c, 4000, "请输入用户名或邮箱")
		return
	}

	token, username, err := ac.pwdResetSvc.RequestReset(c.Request.Context(), req.Identifier)
	if err != nil {
		switch err {
		case iamdomain.ErrNotFound:
			resp.OK(c, gin.H{"message": "如果该账号存在，重置链接已生成"})
		case iamdomain.ErrUserDisabled:
			resp.Fail(c, 4000, "账号已被禁用")
		default:
			resp.Fail(c, 5000, "内部错误")
		}
		return
	}

	ac.recordAuthAudit(c, 0, username, "password-reset-request", 0, "请求密码重置")
	resp.OK(c, gin.H{
		"token":    token,
		"username": username,
		"message":  "重置 token 已生成，请使用该 token 设置新密码",
	})
}

func (ac *AuthController) ConfirmPasswordReset(c *gin.Context) {
	if ac.pwdResetSvc == nil {
		resp.Fail(c, 5000, "密码重置服务不可用")
		return
	}

	var req resetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if strings.TrimSpace(req.Token) == "" || strings.TrimSpace(req.NewPassword) == "" {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if len(req.NewPassword) < 6 {
		resp.Fail(c, 4000, "密码长度至少 6 位")
		return
	}

	if err := ac.pwdResetSvc.ResetPasswordByToken(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		switch err {
		case iamdomain.ErrResetTokenNotFound:
			resp.Fail(c, 4004, "重置链接已过期或不存在，请重新申请")
		case iamdomain.ErrResetTokenUsed:
			resp.Fail(c, 4004, "该重置链接已被使用")
		default:
			resp.Fail(c, 5000, "内部错误")
		}
		return
	}

	ac.recordAuthAudit(c, 0, "", "password-reset-confirm", 0, "密码重置成功")
	resp.OK(c, gin.H{"message": "密码重置成功，请使用新密码登录"})
}
