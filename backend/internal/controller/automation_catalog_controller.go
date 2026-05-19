package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type AutomationCatalogController struct {
	svc *service.AutomationCatalogService
}

func NewAutomationCatalogController(svc *service.AutomationCatalogService) *AutomationCatalogController {
	return &AutomationCatalogController{svc: svc}
}

type upsertAutomationCatalogItemReq struct {
	Name               string   `json:"name"`
	Category           string   `json:"category"`
	Description        *string  `json:"description"`
	RecommendedVersion string   `json:"recommended_version"`
	Tags               []string `json:"tags"`
	Published          bool     `json:"published"`
	Visibility         string   `json:"visibility"`
	IconURL            *string  `json:"icon_url"`
	TemplateID         uint64   `json:"template_id"`
	TemplateVersion    string   `json:"template_version"`
	SortOrder          int      `json:"sort_order"`
}

func (ctl *AutomationCatalogController) List(c *gin.Context) {
	var published *bool
	if raw := c.Query("published"); raw != "" {
		value := raw == "1" || raw == "true"
		published = &value
	}
	data, err := ctl.svc.List(c.Request.Context(), service.ListAutomationCatalogItemsRequest{
		Page:       parseInt(c.Query("page"), 1),
		PageSize:   parseInt(c.Query("page_size"), 20),
		Keyword:    c.Query("keyword"),
		Category:   c.Query("category"),
		Published:  published,
		Visibility: c.Query("visibility"),
		SortBy:     c.Query("sort_by"),
		Order:      c.Query("order"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AutomationCatalogController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.Get(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (ctl *AutomationCatalogController) Create(c *gin.Context) {
	var req upsertAutomationCatalogItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := ctl.svc.Create(c.Request.Context(), service.UpsertAutomationCatalogItemRequest{
		Name:               req.Name,
		Category:           req.Category,
		Description:        req.Description,
		RecommendedVersion: req.RecommendedVersion,
		Tags:               req.Tags,
		Published:          req.Published,
		Visibility:         req.Visibility,
		IconURL:            req.IconURL,
		TemplateID:         req.TemplateID,
		TemplateVersion:    req.TemplateVersion,
		SortOrder:          req.SortOrder,
		OperatorID:         automationCatalogCurrentUserID(c),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (ctl *AutomationCatalogController) Patch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req upsertAutomationCatalogItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.svc.Patch(c.Request.Context(), id, service.UpsertAutomationCatalogItemRequest{
		Name:               req.Name,
		Category:           req.Category,
		Description:        req.Description,
		RecommendedVersion: req.RecommendedVersion,
		Tags:               req.Tags,
		Published:          req.Published,
		Visibility:         req.Visibility,
		IconURL:            req.IconURL,
		TemplateID:         req.TemplateID,
		TemplateVersion:    req.TemplateVersion,
		SortOrder:          req.SortOrder,
		OperatorID:         automationCatalogCurrentUserID(c),
	}); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func (ctl *AutomationCatalogController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.svc.Delete(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func automationCatalogCurrentUserID(c *gin.Context) uint64 {
	if claims, ok := middleware.GetClaims(c); ok && claims != nil && claims.UserID > 0 {
		return uint64(claims.UserID)
	}
	if v, ok := c.Get("user_id"); ok {
		if id, ok2 := v.(int64); ok2 && id > 0 {
			return uint64(id)
		}
	}
	return 1
}
