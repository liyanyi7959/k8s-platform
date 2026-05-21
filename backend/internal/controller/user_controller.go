package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

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
	result, err := uc.rbacSvc.ListUsers(c.Request.Context(), service.UserListParams{
		Page:     page,
		PageSize: pageSize,
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
	})
	if err != nil {
		resp.Fail(c, 5000, "查询失败")
		return
	}
	resp.OK(c, result)
}

// CreateUser 创建用户。
func (uc *UserController) CreateUser(c *gin.Context) {
	var req service.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := uc.rbacSvc.CreateUser(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// UpdateUser 更新用户（状态/角色）。
func (uc *UserController) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req service.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := uc.rbacSvc.UpdateUser(c.Request.Context(), id, req); err != nil {
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
