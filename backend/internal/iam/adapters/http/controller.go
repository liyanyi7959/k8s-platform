package http

import (
	"errors"
	"net/mail"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/iam/application"
	"k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/pkg/resp"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

type Controller struct {
	users *application.UserManagement
	roles *application.RoleManagement
}

func NewController(users *application.UserManagement, roles *application.RoleManagement) *Controller {
	return &Controller{users: users, roles: roles}
}

func (ctl *Controller) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	roleID, _ := strconv.ParseUint(c.Query("role_id"), 10, 64)
	result, err := ctl.users.List(c.Request.Context(), application.UserListParams{Page: page, PageSize: pageSize, Keyword: c.Query("keyword"), Status: c.Query("status"), RoleID: roleID})
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

func (ctl *Controller) CreateUser(c *gin.Context) {
	var request createUserRequest
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	request.Username, request.Nickname, request.Email = strings.TrimSpace(request.Username), strings.TrimSpace(request.Nickname), strings.TrimSpace(request.Email)
	if !validUsername(request.Username) {
		resp.Fail(c, 4000, "用户名必须以字母开头，3-32 位字母/数字/下划线")
		return
	}
	if !validEmail(request.Email) {
		resp.Fail(c, 4000, "邮箱格式不正确")
		return
	}
	if !validPassword(request.Password) {
		resp.Fail(c, 4000, "密码需 8-64 位且包含大小写字母、数字和特殊字符")
		return
	}
	if len(request.RoleIDs) == 0 {
		resp.Fail(c, 4000, "至少选择一个角色")
		return
	}
	id, err := ctl.users.Create(c.Request.Context(), application.CreateUserRequest{Username: request.Username, Nickname: request.Nickname, Email: request.Email, Password: request.Password, RoleIDs: request.RoleIDs})
	if err != nil {
		writeError(c, err)
		return
	}
	if !request.Enabled {
		status := "disabled"
		if err := ctl.users.Update(c.Request.Context(), id, application.UpdateUserRequest{Status: &status}); err != nil {
			writeError(c, err)
			return
		}
	}
	resp.OK(c, gin.H{"id": id})
}

type updateUserRequest struct {
	Nickname string   `json:"nickname"`
	Email    string   `json:"email"`
	Enabled  *bool    `json:"enabled"`
	RoleIDs  []uint64 `json:"roleIds"`
}

func (ctl *Controller) UpdateUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request updateUserRequest
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	request.Nickname, request.Email = strings.TrimSpace(request.Nickname), strings.TrimSpace(request.Email)
	if request.Email != "" && !validEmail(request.Email) {
		resp.Fail(c, 4000, "邮箱格式不正确")
		return
	}
	if len(request.RoleIDs) == 0 {
		resp.Fail(c, 4000, "至少选择一个角色")
		return
	}
	serviceRequest := application.UpdateUserRequest{Nickname: &request.Nickname, Email: &request.Email, RoleIDs: request.RoleIDs}
	if request.Enabled != nil {
		status := "active"
		if !*request.Enabled {
			status = "disabled"
		}
		serviceRequest.Status = &status
	}
	if err := ctl.users.Update(c.Request.Context(), id, serviceRequest); err != nil {
		writeError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (ctl *Controller) DeleteUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := ctl.users.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (ctl *Controller) ResetPassword(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request struct {
		Password string `json:"password"`
	}
	if c.ShouldBindJSON(&request) != nil || !validPassword(request.Password) {
		resp.Fail(c, 4000, "密码需 8-64 位且包含大小写字母、数字和特殊字符")
		return
	}
	if err := ctl.users.ResetPassword(c.Request.Context(), id, request.Password); err != nil {
		writeError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (ctl *Controller) ListRoles(c *gin.Context) {
	items, err := ctl.roles.List(c.Request.Context())
	if err != nil {
		resp.Fail(c, 5000, "查询失败")
		return
	}
	resp.OK(c, items)
}
func (ctl *Controller) CreateRole(c *gin.Context) {
	var request application.CreateRoleRequest
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	request.Name, request.Code = strings.TrimSpace(request.Name), strings.TrimSpace(request.Code)
	if request.Name == "" {
		resp.Fail(c, 4000, "角色名称不能为空")
		return
	}
	if !validRoleCode(request.Code) {
		resp.Fail(c, 4000, "角色编码必须以小写字母开头，2-32 位小写字母/数字/下划线")
		return
	}
	id, err := ctl.roles.Create(c.Request.Context(), request)
	if err != nil {
		writeError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}
func (ctl *Controller) UpdateRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request application.UpdateRoleRequest
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if request.Name != nil && strings.TrimSpace(*request.Name) == "" {
		resp.Fail(c, 4000, "角色名称不能为空")
		return
	}
	if request.Code != nil && !validRoleCode(strings.TrimSpace(*request.Code)) {
		resp.Fail(c, 4000, "角色编码必须以小写字母开头，2-32 位小写字母/数字/下划线")
		return
	}
	if err := ctl.roles.Update(c.Request.Context(), id, request); err != nil {
		writeError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *Controller) DeleteRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := ctl.roles.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *Controller) ListPermissions(c *gin.Context) {
	items, err := ctl.roles.ListPermissions(c.Request.Context())
	if err != nil {
		resp.Fail(c, 5000, "查询失败")
		return
	}
	resp.OK(c, items)
}

func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidParams):
		resp.Fail(c, 4000, "参数错误")
	case errors.Is(err, domain.ErrNotFound):
		resp.Fail(c, 4040, "未找到")
	case errors.Is(err, domain.ErrConflict):
		resp.Fail(c, 4090, "资源冲突")
	default:
		resp.Fail(c, 5000, "内部错误")
	}
}
func validUsername(value string) bool {
	return usernamePattern.MatchString(value) && len(value) >= 3 && len(value) <= 32
}
func validRoleCode(value string) bool {
	return roleCodePattern.MatchString(value) && len(value) >= 2 && len(value) <= 32
}
func validEmail(value string) bool {
	if value == "" {
		return true
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}
func validPassword(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}
	var lower, upper, digit, special bool
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
			lower = true
		case character >= 'A' && character <= 'Z':
			upper = true
		case character >= '0' && character <= '9':
			digit = true
		case strings.ContainsRune("!@#$%^&*", character):
			special = true
		}
	}
	return lower && upper && digit && special
}
