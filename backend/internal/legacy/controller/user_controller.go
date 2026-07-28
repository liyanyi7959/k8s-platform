package controller

import (
	"net/mail"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/pkg/resp"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	roleCodeRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

func validateUsername(username string) bool {
	return usernameRegex.MatchString(username) && len(username) >= 3 && len(username) <= 32
}

func validateEmail(email string) bool {
	if strings.TrimSpace(email) == "" {
		return true
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func validatePassword(password string) bool {
	if len(password) < 8 || len(password) > 64 {
		return false
	}
	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*", ch):
			hasSpecial = true
		}
	}
	return hasLower && hasUpper && hasDigit && hasSpecial
}

func validateRoleCode(code string) bool {
	return roleCodeRegex.MatchString(code) && len(code) >= 2 && len(code) <= 32
}

type UserController struct {
	rbacSvc *service.RbacService
}

func NewUserController(rbacSvc *service.RbacService) *UserController {
	return &UserController{rbacSvc: rbacSvc}
}

// ListUsers 用户列表。
func (uc *UserController) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	roleID, _ := strconv.ParseUint(c.Query("role_id"), 10, 64)
	result, err := uc.rbacSvc.ListUsers(c.Request.Context(), service.UserListParams{
		Page:     page,
		PageSize: pageSize,
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		RoleID:   roleID,
	})
	if err != nil {
		resp.Fail(c, 5000, "查询失败")
		return
	}
	resp.OK(c, result)
}

type createUserRequest struct {
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	RoleIDs  []uint64 `json:"roleIds"`
	Enabled  bool     `json:"enabled"`
}

// CreateUser 创建用户。
func (uc *UserController) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if !validateUsername(strings.TrimSpace(req.Username)) {
		resp.Fail(c, 4000, "用户名必须以字母开头，3-32 位字母/数字/下划线")
		return
	}
	if !validateEmail(strings.TrimSpace(req.Email)) {
		resp.Fail(c, 4000, "邮箱格式不正确")
		return
	}
	if !validatePassword(req.Password) {
		resp.Fail(c, 4000, "密码需 8-64 位且包含大小写字母、数字和特殊字符")
		return
	}
	if len(req.RoleIDs) == 0 {
		resp.Fail(c, 4000, "至少选择一个角色")
		return
	}

	status := "active"
	if !req.Enabled {
		status = "disabled"
	}
	svcReq := service.CreateUserReq{
		Username: strings.TrimSpace(req.Username),
		Nickname: strings.TrimSpace(req.Nickname),
		Email:    strings.TrimSpace(req.Email),
		Password: req.Password,
		RoleIDs:  req.RoleIDs,
	}
	id, err := uc.rbacSvc.CreateUser(c.Request.Context(), svcReq)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}

	// 如果前端指定禁用，创建后更新状态
	if status == "disabled" {
		_ = uc.rbacSvc.UpdateUser(c.Request.Context(), id, service.UpdateUserReq{Status: &status})
	}
	resp.OK(c, gin.H{"id": id})
}

type updateUserRequest struct {
	Nickname string   `json:"nickname"`
	Email    string   `json:"email"`
	Enabled  *bool    `json:"enabled"`
	RoleIDs  []uint64 `json:"roleIds"`
}

// UpdateUser 更新用户（状态/角色）。
func (uc *UserController) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if strings.TrimSpace(req.Email) != "" && !validateEmail(strings.TrimSpace(req.Email)) {
		resp.Fail(c, 4000, "邮箱格式不正确")
		return
	}
	if len(req.RoleIDs) == 0 {
		resp.Fail(c, 4000, "至少选择一个角色")
		return
	}

	svcReq := service.UpdateUserReq{
		Nickname: &req.Nickname,
		Email:    &req.Email,
		RoleIDs:  req.RoleIDs,
	}
	if req.Enabled != nil {
		status := "active"
		if !*req.Enabled {
			status = "disabled"
		}
		svcReq.Status = &status
	}
	if err := uc.rbacSvc.UpdateUser(c.Request.Context(), id, svcReq); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// DeleteUser 删除用户。
func (uc *UserController) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := uc.rbacSvc.DeleteUser(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// ResetPassword 管理员重置密码。
func (uc *UserController) ResetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Password == "" {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if !validatePassword(body.Password) {
		resp.Fail(c, 4000, "密码需 8-64 位且包含大小写字母、数字和特殊字符")
		return
	}
	if err := uc.rbacSvc.ResetPassword(c.Request.Context(), id, body.Password); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// ListRoles 角色列表。
func (uc *UserController) ListRoles(c *gin.Context) {
	items, err := uc.rbacSvc.ListRoles(c.Request.Context())
	if err != nil {
		resp.Fail(c, 5000, "查询失败")
		return
	}
	resp.OK(c, items)
}

// CreateRole 创建角色。
func (uc *UserController) CreateRole(c *gin.Context) {
	var req service.CreateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	if req.Name == "" {
		resp.Fail(c, 4000, "角色名称不能为空")
		return
	}
	if !validateRoleCode(req.Code) {
		resp.Fail(c, 4000, "角色编码必须以小写字母开头，2-32 位小写字母/数字/下划线")
		return
	}
	id, err := uc.rbacSvc.CreateRole(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// UpdateRole 更新角色。
func (uc *UserController) UpdateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req service.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		resp.Fail(c, 4000, "角色名称不能为空")
		return
	}
	if req.Code != nil && !validateRoleCode(strings.TrimSpace(*req.Code)) {
		resp.Fail(c, 4000, "角色编码必须以小写字母开头，2-32 位小写字母/数字/下划线")
		return
	}
	if err := uc.rbacSvc.UpdateRole(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// DeleteRole 删除角色。
func (uc *UserController) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := uc.rbacSvc.DeleteRole(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// ListPermissions 列出所有权限点。
func (uc *UserController) ListPermissions(c *gin.Context) {
	items, err := uc.rbacSvc.ListPermissions(c.Request.Context())
	if err != nil {
		resp.Fail(c, 5000, "查询失败")
		return
	}
	resp.OK(c, items)
}
