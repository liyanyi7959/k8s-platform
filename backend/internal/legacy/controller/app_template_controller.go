package controller

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"k8s-platform-backend/internal/legacy/model"
	"k8s-platform-backend/internal/legacy/service"
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
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	Description      string `json:"description"`
	Category         string `json:"category"`
	Icon             string `json:"icon"`
	Template         string `json:"template"`
	Variables        string `json:"variables"`
	DeployType       string `json:"deploy_type"`
	HelmRepoName     string `json:"helm_repo_name"`
	HelmRepoURL      string `json:"helm_repo_url"`
	HelmChartVersion string `json:"helm_chart_version"`
	HelmValuesYAML   string `json:"helm_values_yaml"`
}

func (r *appTemplateReq) normalizeAndValidate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.DeployType = strings.ToLower(strings.TrimSpace(r.DeployType))
	if r.DeployType == "" {
		r.DeployType = "yaml"
	}
	if r.DeployType != "yaml" && r.DeployType != "helm" {
		return fmt.Errorf("deploy_type 仅支持 yaml 或 helm")
	}
	if r.DeployType == "helm" {
		r.Template = strings.TrimSpace(r.Template)
		r.HelmRepoName = strings.TrimSpace(r.HelmRepoName)
		r.HelmRepoURL = strings.TrimSpace(r.HelmRepoURL)
		if r.Template == "" {
			return fmt.Errorf("Helm Chart 不能为空")
		}
		if (r.HelmRepoName == "") != (r.HelmRepoURL == "") {
			return fmt.Errorf("Helm 仓库名称和地址需同时填写")
		}
		if r.HelmRepoName != "" {
			if len(r.HelmRepoName) > 63 || !helmKubernetesName.MatchString(r.HelmRepoName) {
				return fmt.Errorf("Helm 仓库名称必须是 1-63 位小写 DNS 名称")
			}
			parsedURL, err := url.ParseRequestURI(r.HelmRepoURL)
			if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
				return fmt.Errorf("Helm 仓库地址必须是有效的 HTTPS URL")
			}
			if !strings.HasPrefix(r.Template, r.HelmRepoName+"/") {
				return fmt.Errorf("Chart 必须以仓库名称 %s/ 开头", r.HelmRepoName)
			}
		}
		if strings.TrimSpace(r.HelmValuesYAML) != "" {
			values := map[string]any{}
			if err := yaml.Unmarshal([]byte(r.HelmValuesYAML), &values); err != nil {
				return fmt.Errorf("默认 values.yaml 解析失败: %w", err)
			}
		}
	}
	return nil
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
	if err := req.normalizeAndValidate(); err != nil {
		resp.Fail(c, 4000, err.Error())
		return
	}

	t := &model.AppTemplate{
		Name:             req.Name,
		DisplayName:      req.DisplayName,
		Description:      req.Description,
		Category:         req.Category,
		Icon:             req.Icon,
		Template:         req.Template,
		Variables:        req.Variables,
		DeployType:       req.DeployType,
		HelmRepoName:     req.HelmRepoName,
		HelmRepoURL:      req.HelmRepoURL,
		HelmChartVersion: req.HelmChartVersion,
		HelmValuesYAML:   req.HelmValuesYAML,
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
	if err := req.normalizeAndValidate(); err != nil {
		resp.Fail(c, 4000, err.Error())
		return
	}

	updates := map[string]any{
		"name":               req.Name,
		"display_name":       req.DisplayName,
		"description":        req.Description,
		"category":           req.Category,
		"icon":               req.Icon,
		"template":           req.Template,
		"variables":          req.Variables,
		"deploy_type":        req.DeployType,
		"helm_repo_name":     req.HelmRepoName,
		"helm_repo_url":      req.HelmRepoURL,
		"helm_chart_version": req.HelmChartVersion,
		"helm_values_yaml":   req.HelmValuesYAML,
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
