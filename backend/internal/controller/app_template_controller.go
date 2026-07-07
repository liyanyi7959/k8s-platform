package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/model"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// AppTemplateController 应用商店模板控制器。
type AppTemplateController struct {
	svc *service.AppTemplateService
}

// NewAppTemplateController 创建应用商店模板控制器实例。
func NewAppTemplateController(svc *service.AppTemplateService) *AppTemplateController {
	return &AppTemplateController{svc: svc}
}

// appTemplateReq 为创建/更新应用模板时的请求体。
type appTemplateReq struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Icon        string `json:"icon"`
	Template    string `json:"template"`
	Variables   string `json:"variables"`
}

// ListAppTemplates 分页查询应用模板列表。
// GET /app-templates?category=xxx&page=1&page_size=100
func (ac *AppTemplateController) ListAppTemplates(c *gin.Context) {
	category := c.Query("category")
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 100)
	data, err := ac.svc.ListAppTemplates(c.Request.Context(), category, page, pageSize)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// GetAppTemplate 查询单个应用模板详情。
// GET /app-templates/:id
func (ac *AppTemplateController) GetAppTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ac.svc.GetAppTemplate(c.Request.Context(), uint64(id))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// CreateAppTemplate 创建应用模板。
// POST /app-templates
func (ac *AppTemplateController) CreateAppTemplate(c *gin.Context) {
	var req appTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	t := &model.AppTemplate{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Category:    req.Category,
		Icon:        req.Icon,
		Template:    req.Template,
		Variables:   req.Variables,
	}
	if err := ac.svc.CreateAppTemplate(c.Request.Context(), t); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": t.ID})
}

// UpdateAppTemplate 更新应用模板。
// PUT /app-templates/:id
func (ac *AppTemplateController) UpdateAppTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req appTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	updates := map[string]any{
		"name":         req.Name,
		"display_name": req.DisplayName,
		"description":  req.Description,
		"category":     req.Category,
		"icon":         req.Icon,
		"template":     req.Template,
		"variables":    req.Variables,
	}
	if err := ac.svc.UpdateAppTemplate(c.Request.Context(), uint64(id), updates); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// DeleteAppTemplate 删除应用模板（软删除，内置模板不可删除）。
// DELETE /app-templates/:id
func (ac *AppTemplateController) DeleteAppTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ac.svc.DeleteAppTemplate(c.Request.Context(), uint64(id)); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
