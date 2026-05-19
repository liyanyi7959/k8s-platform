package controller

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type RbacController struct {
	// svc 承载 RBAC 业务逻辑（用户/角色/权限点）。
	svc *service.RbacService
}

func NewRbacController(svc *service.RbacService) *RbacController {
	// 控制器只做：参数解析/校验、调用 service、返回统一响应结构。
	return &RbacController{svc: svc}
}

type PermissionListPage struct {
	List     []service.PermissionItem `json:"list"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

// @Summary 权限点列表
// @Description 分页列出权限点
// @Tags 用户与权限接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param page query int false "页码" example(1)
// @Param page_size query int false "每页条数" example(10)
// @Success 200 {object} resp.Result{data=PermissionListPage} "查询成功"
// @Failure 200 {object} resp.Result "查询失败"
// @Router /permissions [get]
func (rc *RbacController) ListPermissions(c *gin.Context) {
	// ListPermissions 分页列出权限点。
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 10)
	data, err := rc.svc.ListPermissions(c.Request.Context(), service.ListPermissionsRequest{Page: page, PageSize: pageSize})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type RoleListPage struct {
	List     []service.RoleItem `json:"list"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

// @Summary 角色列表
// @Description 分页列出角色，支持 keyword 搜索与排序
// @Tags 用户与权限接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param page query int false "页码" example(1)
// @Param page_size query int false "每页条数" example(10)
// @Param keyword query string false "关键字（按角色名模糊搜索）"
// @Param sort_by query string false "排序字段" example(created_at)
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=RoleListPage} "查询成功"
// @Failure 200 {object} resp.Result "查询失败"
// @Router /roles [get]
func (rc *RbacController) ListRoles(c *gin.Context) {
	// ListRoles 分页列出角色，支持 keyword 搜索与 created_at 排序。
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 10)
	data, err := rc.svc.ListRoles(c.Request.Context(), service.ListRolesRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  c.Query("keyword"),
		SortBy:   c.Query("sort_by"),
		Order:    c.Query("order"),
	})
	if err != nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	resp.OK(c, data)
}

type createRoleReq struct {
	Name string  `json:"name"`
	Desc *string `json:"desc"`
}

// @Summary 创建角色
// @Description 创建一个新角色
// @Tags 用户与权限接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createRoleReq true "创建参数"
// @Success 200 {object} resp.Result "创建成功"
// @Failure 200 {object} resp.Result "创建失败"
// @Router /roles [post]
func (rc *RbacController) CreateRole(c *gin.Context) {
	// CreateRole 创建角色。
	var req createRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := rc.svc.CreateRole(c.Request.Context(), service.CreateRoleRequest{Name: req.Name, Desc: req.Desc}); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

type updateRolePermReq struct {
	PermissionCodes []string `json:"permission_codes"`
}

// @Summary 更新角色权限点
// @Description 全量覆盖更新角色绑定的权限点（按 code）
// @Tags 用户与权限接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "角色ID" example(1001)
// @Param body body updateRolePermReq true "权限点 code 列表"
// @Success 200 {object} resp.Result "更新成功"
// @Failure 200 {object} resp.Result "更新失败"
// @Router /roles/{id}/permissions [put]
func (rc *RbacController) UpdateRolePermissions(c *gin.Context) {
	// UpdateRolePermissions 全量覆盖更新角色权限点：
	// body: { "permission_codes": ["sys:user_admin", ...] }
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req updateRolePermReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := rc.svc.UpdateRolePermissions(c.Request.Context(), id, service.UpdateRolePermissionsRequest{PermissionCodes: req.PermissionCodes}); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// @Summary 角色权限点
// @Description 获取指定角色绑定的权限点 code 列表
// @Tags 用户与权限接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "角色ID" example(1001)
// @Success 200 {object} resp.Result{data=map[string][]string} "查询成功"
// @Failure 200 {object} resp.Result "查询失败"
// @Router /roles/{id}/permissions [get]
func (rc *RbacController) GetRolePermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	codes, err := rc.svc.GetRolePermissionCodes(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"permission_codes": codes})
}

type UserListPage struct {
	List     []service.UserItem `json:"list"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

// @Summary 用户列表
// @Description 分页列出用户，支持 keyword/status 过滤与排序
// @Tags 用户与权限接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param page query int false "页码" example(1)
// @Param page_size query int false "每页条数" example(10)
// @Param keyword query string false "关键字（按用户名模糊搜索）"
// @Param status query string false "状态过滤" example(active)
// @Param sort_by query string false "排序字段" example(created_at)
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=UserListPage} "查询成功"
// @Failure 200 {object} resp.Result "查询失败"
// @Router /users [get]
func (rc *RbacController) ListUsers(c *gin.Context) {
	// ListUsers 分页列出用户，支持 keyword/status 过滤与 created_at 排序。
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 10)
	data, err := rc.svc.ListUsers(c.Request.Context(), service.ListUsersRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		SortBy:   c.Query("sort_by"),
		Order:    c.Query("order"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type createUserReq struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	RoleIDs  []uint64 `json:"role_ids"`
	Status   string   `json:"status"`
}

// @Summary 创建用户
// @Description 创建用户并绑定角色
// @Tags 用户与权限接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createUserReq true "创建参数"
// @Success 200 {object} resp.Result "创建成功"
// @Failure 200 {object} resp.Result "创建失败"
// @Router /users [post]
func (rc *RbacController) CreateUser(c *gin.Context) {
	// CreateUser 创建用户并绑定角色：
	// body: { "username":"xx", "password":"xx", "role_ids":[1,2], "status":"active" }
	var req createUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := rc.svc.CreateUser(c.Request.Context(), service.CreateUserRequest{
		Username: req.Username,
		Password: req.Password,
		RoleIDs:  req.RoleIDs,
		Status:   req.Status,
	}); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// @Summary 更新用户
// @Description 支持更新用户状态或角色绑定（部分字段更新）
// @Tags 用户与权限接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "用户ID" example(1001)
// @Success 200 {object} resp.Result "更新成功"
// @Failure 200 {object} resp.Result "更新失败"
// @Router /users/{id} [patch]
func (rc *RbacController) PatchUser(c *gin.Context) {
	// PatchUser 支持部分更新：
	// body 可包含 { "status": "disabled" } 或 { "role_ids": [1,2] } 或两者都有。
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	var req service.PatchUserRequest
	if b, ok := raw["status"]; ok {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		v = strings.TrimSpace(v)
		req.Status = &v
	}
	if b, ok := raw["role_ids"]; ok {
		if string(b) == "null" {
			// role_ids 传 null 视为清空角色绑定。
			empty := []uint64{}
			req.RoleIDs = &empty
		} else {
			var v []uint64
			if err := json.Unmarshal(b, &v); err != nil {
				resp.Fail(c, 4000, "参数错误")
				return
			}
			req.RoleIDs = &v
		}
	}

	if err := rc.svc.PatchUser(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

type resetPasswordReq struct {
	NewPassword string `json:"new_password"`
}

// @Summary 重置用户密码
// @Description 管理员重置指定用户密码
// @Tags 用户与权限接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "用户ID" example(1001)
// @Param body body resetPasswordReq true "重置参数"
// @Success 200 {object} resp.Result "重置成功"
// @Failure 200 {object} resp.Result "重置失败"
// @Router /users/{id}/reset-password [post]
func (rc *RbacController) ResetUserPassword(c *gin.Context) {
	// ResetUserPassword 管理员重置指定用户密码：
	// body: { "new_password": "xxx" }
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req resetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := rc.svc.ResetUserPassword(c.Request.Context(), id, req.NewPassword); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}
