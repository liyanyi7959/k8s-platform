package controller

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// ── AppTemplateController ────────────────────────────────────────────────────

type AppTemplateController struct {
	svc *service.AppTemplateService
}

func NewAppTemplateController(svc *service.AppTemplateService) *AppTemplateController {
	return &AppTemplateController{svc: svc}
}

// List godoc
// @Summary  应用模板列表
// @Tags     apps
// @Security BearerAuth
// @Param    page      query int    false "页码"
// @Param    page_size query int    false "每页数量"
// @Param    keyword   query string false "关键字"
// @Param    category  query string false "分类"
// @Param    engine    query string false "模板引擎"
// @Param    status    query string false "模板状态"
// @Param    source_type query string false "模板来源类型"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-templates [get]
func (ctl *AppTemplateController) List(c *gin.Context) {
	data, err := ctl.svc.List(c.Request.Context(), service.ListAppTemplatesRequest{
		Page:       parseInt(c.Query("page"), 1),
		PageSize:   parseInt(c.Query("page_size"), 10),
		Keyword:    c.Query("keyword"),
		Category:   c.Query("category"),
		Engine:     c.Query("engine"),
		Status:     c.Query("status"),
		SourceType: c.Query("source_type"),
		SortBy:     c.Query("sort_by"),
		Order:      c.Query("order"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// Get godoc
// @Summary  应用模板详情
// @Tags     apps
// @Security BearerAuth
// @Param    id path int true "模板 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-templates/{id} [get]
func (ctl *AppTemplateController) Get(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的模板 ID")
		return
	}
	data, err := ctl.svc.Get(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type createAppTemplateReq struct {
	Name               string                 `json:"name"`
	ChartName          string                 `json:"chart_name"`
	Category           string                 `json:"category"`
	Version            string                 `json:"version"`
	AppVersion         string                 `json:"app_version"`
	Engine             string                 `json:"engine"`
	Status             string                 `json:"status"`
	Summary            string                 `json:"summary"`
	Tags               []string               `json:"tags"`
	Manifest           string                 `json:"manifest"`
	ValuesSchema       map[string]interface{} `json:"values_schema"`
	DefaultValues      map[string]interface{} `json:"default_values"`
	Source             string                 `json:"source"`
	SourceType         string                 `json:"source_type"`
	SourceURL          string                 `json:"source_url"`
	SourceRef          map[string]interface{} `json:"source_ref"`
	Owner              string                 `json:"owner"`
	Readme             string                 `json:"readme"`
	EnvExample         string                 `json:"env_example"`
	ProjectNameDefault string                 `json:"project_name_default"`
	InstallDirDefault  string                 `json:"install_dir_default"`
	ExtraFiles         []string               `json:"extra_files"`
}

// Create godoc
// @Summary  新建应用模板
// @Tags     apps
// @Security BearerAuth
// @Param    body body createAppTemplateReq true "模板信息"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-templates [post]
func (ctl *AppTemplateController) Create(c *gin.Context) {
	var req createAppTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.Create(c.Request.Context(), service.CreateAppTemplateRequest{
		Name:               req.Name,
		ChartName:          req.ChartName,
		Category:           req.Category,
		Version:            req.Version,
		AppVersion:         req.AppVersion,
		Engine:             req.Engine,
		Status:             req.Status,
		Summary:            req.Summary,
		Tags:               req.Tags,
		Manifest:           req.Manifest,
		ValuesSchema:       req.ValuesSchema,
		DefaultValues:      req.DefaultValues,
		Source:             req.Source,
		SourceType:         req.SourceType,
		SourceURL:          req.SourceURL,
		SourceRef:          req.SourceRef,
		Owner:              req.Owner,
		Readme:             req.Readme,
		EnvExample:         req.EnvExample,
		ProjectNameDefault: req.ProjectNameDefault,
		InstallDirDefault:  req.InstallDirDefault,
		ExtraFiles:         req.ExtraFiles,
		CreatedBy:          uint64(currentUserID(c)),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type patchAppTemplateReq struct {
	Name               *string                `json:"name"`
	ChartName          *string                `json:"chart_name"`
	Category           *string                `json:"category"`
	Version            *string                `json:"version"`
	AppVersion         *string                `json:"app_version"`
	Engine             *string                `json:"engine"`
	Status             *string                `json:"status"`
	Summary            *string                `json:"summary"`
	Tags               []string               `json:"tags"`
	Manifest           *string                `json:"manifest"`
	ValuesSchema       map[string]interface{} `json:"values_schema"`
	DefaultValues      map[string]interface{} `json:"default_values"`
	Source             *string                `json:"source"`
	SourceType         *string                `json:"source_type"`
	SourceURL          *string                `json:"source_url"`
	SourceRef          map[string]interface{} `json:"source_ref"`
	Owner              *string                `json:"owner"`
	Readme             *string                `json:"readme"`
	EnvExample         *string                `json:"env_example"`
	ProjectNameDefault *string                `json:"project_name_default"`
	InstallDirDefault  *string                `json:"install_dir_default"`
	ExtraFiles         []string               `json:"extra_files"`
}

// Patch godoc
// @Summary  更新应用模板
// @Tags     apps
// @Security BearerAuth
// @Param    id   path int              true "模板 ID"
// @Param    body body patchAppTemplateReq true "更新字段"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-templates/{id} [patch]
func (ctl *AppTemplateController) Patch(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的模板 ID")
		return
	}
	var req patchAppTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.Patch(c.Request.Context(), id, service.PatchAppTemplateRequest{
		Name:               req.Name,
		ChartName:          req.ChartName,
		Category:           req.Category,
		Version:            req.Version,
		AppVersion:         req.AppVersion,
		Engine:             req.Engine,
		Status:             req.Status,
		Summary:            req.Summary,
		Tags:               req.Tags,
		Manifest:           req.Manifest,
		ValuesSchema:       req.ValuesSchema,
		DefaultValues:      req.DefaultValues,
		Source:             req.Source,
		SourceType:         req.SourceType,
		SourceURL:          req.SourceURL,
		SourceRef:          req.SourceRef,
		Owner:              req.Owner,
		Readme:             req.Readme,
		EnvExample:         req.EnvExample,
		ProjectNameDefault: req.ProjectNameDefault,
		InstallDirDefault:  req.InstallDirDefault,
		ExtraFiles:         req.ExtraFiles,
		UpdatedBy:          uint64(currentUserID(c)),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// Delete godoc
// @Summary  删除应用模板
// @Tags     apps
// @Security BearerAuth
// @Param    id path int true "模板 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-templates/{id} [delete]
func (ctl *AppTemplateController) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的模板 ID")
		return
	}
	if err := ctl.svc.Delete(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

type renderAppTemplateReq struct {
	Values map[string]interface{} `json:"values"`
}

// Render godoc
// @Summary  渲染应用模板预览
// @Tags     apps
// @Security BearerAuth
// @Param    id   path int               true "模板 ID"
// @Param    body body renderAppTemplateReq true "渲染值"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-templates/{id}/render [post]
func (ctl *AppTemplateController) Render(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的模板 ID")
		return
	}
	var req renderAppTemplateReq
	_ = c.ShouldBindJSON(&req)
	rendered, err := ctl.svc.Render(c.Request.Context(), id, req.Values)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"rendered_yaml": rendered})
}

// ImportPackage godoc
// @Summary  上传文件导入应用模板
// @Tags     apps
// @Security BearerAuth
// @Accept   mpfd
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-templates/import-package [post]
func (ctl *AppTemplateController) ImportPackage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, 4000, "请上传模板文件")
		return
	}
	opened, err := file.Open()
	if err != nil {
		resp.Fail(c, 4000, "读取上传文件失败")
		return
	}
	defer opened.Close()
	content, err := io.ReadAll(io.LimitReader(opened, 20<<20))
	if err != nil {
		resp.Fail(c, 4000, "读取上传文件内容失败")
		return
	}
	data, err := ctl.svc.ImportPackage(c.Request.Context(), service.ImportAppTemplateRequest{
		Name:       c.PostForm("name"),
		Category:   c.PostForm("category"),
		Engine:     c.PostForm("engine"),
		Tags:       parseCSV(c.PostForm("tags")),
		SourceType: firstNonEmpty(c.PostForm("source_type"), "upload"),
		Owner:      currentUsername(c),
		CreatedBy:  uint64(currentUserID(c)),
	}, file.Filename, content)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type importRemoteAppTemplateReq struct {
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Engine     string   `json:"engine"`
	Tags       []string `json:"tags"`
	SourceType string   `json:"source_type"`
	SourceURL  string   `json:"source_url"`
}

// ImportRemote godoc
// @Summary  从远程地址导入应用模板
// @Tags     apps
// @Security BearerAuth
// @Param    body body importRemoteAppTemplateReq true "导入信息"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-templates/import-remote [post]
func (ctl *AppTemplateController) ImportRemote(c *gin.Context) {
	var req importRemoteAppTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.ImportRemote(c.Request.Context(), service.ImportAppTemplateRequest{
		Name:       req.Name,
		Category:   req.Category,
		Engine:     req.Engine,
		Tags:       req.Tags,
		SourceType: firstNonEmpty(req.SourceType, "remote"),
		SourceURL:  req.SourceURL,
		Owner:      currentUsername(c),
		CreatedBy:  uint64(currentUserID(c)),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// ── AppReleaseController ─────────────────────────────────────────────────────

type AppReleaseController struct {
	svc *service.AppReleaseService
}

func NewAppReleaseController(svc *service.AppReleaseService) *AppReleaseController {
	return &AppReleaseController{svc: svc}
}

// List godoc
// @Summary  应用发布列表
// @Tags     apps
// @Security BearerAuth
// @Param    page        query int    false "页码"
// @Param    page_size   query int    false "每页数量"
// @Param    keyword     query string false "关键字"
// @Param    namespace   query string false "命名空间"
// @Param    cluster_id  query int    false "集群 ID"
// @Param    target_id   query int    false "目标服务器 ID"
// @Param    status      query string false "状态"
// @Param    template_id query int    false "模板 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases [get]
func (ctl *AppReleaseController) List(c *gin.Context) {
	data, err := ctl.svc.List(c.Request.Context(), service.ListAppReleasesRequest{
		Page:       parseInt(c.Query("page"), 1),
		PageSize:   parseInt(c.Query("page_size"), 10),
		Keyword:    c.Query("keyword"),
		Namespace:  c.Query("namespace"),
		ClusterID:  uint64(parseInt(c.Query("cluster_id"), 0)),
		TargetID:   uint64(parseInt(c.Query("target_id"), 0)),
		Status:     c.Query("status"),
		TemplateID: uint64(parseInt(c.Query("template_id"), 0)),
		SortBy:     c.Query("sort_by"),
		Order:      c.Query("order"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// Get godoc
// @Summary  应用发布详情
// @Tags     apps
// @Security BearerAuth
// @Param    id path int true "发布 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases/{id} [get]
func (ctl *AppReleaseController) Get(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的发布 ID")
		return
	}
	data, err := ctl.svc.Get(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// ListRevisions godoc
// @Summary  应用发布版本历史
// @Tags     apps
// @Security BearerAuth
// @Param    id path int true "发布 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases/{id}/revisions [get]
func (ctl *AppReleaseController) ListRevisions(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的发布 ID")
		return
	}
	data, err := ctl.svc.ListRevisions(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type createAppReleaseReq struct {
	Name               string                 `json:"name"`
	ClusterID          uint64                 `json:"cluster_id"`
	ClusterName        string                 `json:"cluster_name"`
	Namespace          string                 `json:"namespace"`
	TargetType         string                 `json:"target_type"`
	TargetID           uint64                 `json:"target_id"`
	TargetName         string                 `json:"target_name"`
	ProjectName        string                 `json:"project_name"`
	InstallDir         string                 `json:"install_dir"`
	EnvOverride        string                 `json:"env_override"`
	PullImages         *bool                  `json:"pull_images"`
	AutoInstallDocker  *bool                  `json:"auto_install_docker"`
	AutoInstallCompose *bool                  `json:"auto_install_compose"`
	TemplateID         uint64                 `json:"template_id"`
	Strategy           string                 `json:"strategy"`
	Values             map[string]interface{} `json:"values"`
}

// Create godoc
// @Summary  新建应用发布
// @Tags     apps
// @Security BearerAuth
// @Param    body body createAppReleaseReq true "发布信息"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases [post]
func (ctl *AppReleaseController) Create(c *gin.Context) {
	var req createAppReleaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.Create(c.Request.Context(), service.CreateAppReleaseRequest{
		Name:               req.Name,
		ClusterID:          req.ClusterID,
		ClusterName:        req.ClusterName,
		Namespace:          req.Namespace,
		TargetType:         req.TargetType,
		TargetID:           req.TargetID,
		TargetName:         req.TargetName,
		ProjectName:        req.ProjectName,
		InstallDir:         req.InstallDir,
		EnvOverride:        req.EnvOverride,
		PullImages:         req.PullImages == nil || *req.PullImages,
		AutoInstallDocker:  req.AutoInstallDocker == nil || *req.AutoInstallDocker,
		AutoInstallCompose: req.AutoInstallCompose == nil || *req.AutoInstallCompose,
		TemplateID:         req.TemplateID,
		Strategy:           req.Strategy,
		Values:             req.Values,
		CreatedBy:          uint64(currentUserID(c)),
		Operator:           currentUsername(c),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type upgradeAppReleaseReq struct {
	Values      map[string]interface{} `json:"values"`
	EnvOverride string                 `json:"env_override"`
	PullImages  *bool                  `json:"pull_images"`
}

type rollbackAppReleaseReq struct {
	Revision int `json:"revision"`
}

// Upgrade godoc
// @Summary  升级应用发布
// @Tags     apps
// @Security BearerAuth
// @Param    id   path int                true "发布 ID"
// @Param    body body upgradeAppReleaseReq true "升级参数"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases/{id}/upgrade [post]
func (ctl *AppReleaseController) Upgrade(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的发布 ID")
		return
	}
	var req upgradeAppReleaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.Upgrade(c.Request.Context(), id, service.UpgradeAppReleaseRequest{
		Values:      req.Values,
		EnvOverride: req.EnvOverride,
		PullImages:  req.PullImages,
		UpdatedBy:   uint64(currentUserID(c)),
		Operator:    currentUsername(c),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// Rollback godoc
// @Summary  回滚应用发布
// @Tags     apps
// @Security BearerAuth
// @Param    id path int true "发布 ID"
// @Param    body body rollbackAppReleaseReq false "回滚参数"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases/{id}/rollback [post]
func (ctl *AppReleaseController) Rollback(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的发布 ID")
		return
	}
	var req rollbackAppReleaseReq
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			resp.Fail(c, 4000, "参数错误")
			return
		}
	}
	data, err := ctl.svc.Rollback(c.Request.Context(), id, req.Revision, uint64(currentUserID(c)), currentUsername(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// Start godoc
// @Summary  启动应用发布
// @Tags     apps
// @Security BearerAuth
// @Param    id path int true "发布 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases/{id}/start [post]
func (ctl *AppReleaseController) Start(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的发布 ID")
		return
	}
	data, err := ctl.svc.Start(c.Request.Context(), id, uint64(currentUserID(c)), currentUsername(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// Stop godoc
// @Summary  停止应用发布
// @Tags     apps
// @Security BearerAuth
// @Param    id path int true "发布 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases/{id}/stop [post]
func (ctl *AppReleaseController) Stop(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的发布 ID")
		return
	}
	data, err := ctl.svc.Stop(c.Request.Context(), id, uint64(currentUserID(c)), currentUsername(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// TogglePause godoc
// @Summary  暂停/恢复应用发布
// @Tags     apps
// @Security BearerAuth
// @Param    id path int true "发布 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases/{id}/toggle-pause [post]
func (ctl *AppReleaseController) TogglePause(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的发布 ID")
		return
	}
	data, err := ctl.svc.TogglePause(c.Request.Context(), id, uint64(currentUserID(c)))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// Delete godoc
// @Summary  删除应用发布
// @Tags     apps
// @Security BearerAuth
// @Param    id path int true "发布 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/app-releases/{id} [delete]
func (ctl *AppReleaseController) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		resp.Fail(c, 4000, "无效的发布 ID")
		return
	}
	if err := ctl.svc.Delete(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────

func parseUint64Param(c *gin.Context, key string) (uint64, error) {
	return strconv.ParseUint(c.Param(key), 10, 64)
}

// currentUsername 从 JWT claims 中读取用户名（用于 operator 字段）。
func currentUsername(c *gin.Context) string {
	if claims, ok := middleware.GetClaims(c); ok && claims != nil && claims.Username != "" {
		return claims.Username
	}
	return "system"
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
