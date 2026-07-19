package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// DeployConfigController 部署配置控制器
type DeployConfigController struct {
	svc *service.DeployConfigService
}

func NewDeployConfigController(svc *service.DeployConfigService) *DeployConfigController {
	return &DeployConfigController{svc: svc}
}

// ─── DeployConfig ───

// ListConfigs 获取部署配置列表，支持 ?os_type=ubuntu 筛选
func (dc *DeployConfigController) ListConfigs(c *gin.Context) {
	osType := c.Query("os_type")
	configs, err := dc.svc.ListConfigs(c.Request.Context(), osType)
	if err != nil {
		resp.Fail(c, 5000, "获取配置失败")
		return
	}
	resp.OK(c, configs)
}

// ListSupportedOSTypes 获取支持的操作系统类型列表
func (dc *DeployConfigController) ListSupportedOSTypes(c *gin.Context) {
	osTypes, err := dc.svc.ListSupportedOSTypes(c.Request.Context())
	if err != nil {
		resp.Fail(c, 5000, "获取操作系统类型失败")
		return
	}
	resp.OK(c, osTypes)
}

// GetConfig 获取单个配置详情
func (dc *DeployConfigController) GetConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	config, err := dc.svc.GetConfig(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, config)
}

// UpdateConfig 更新部署配置
func (dc *DeployConfigController) UpdateConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	var req service.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "请求参数错误")
		return
	}

	if req.CommandTemplate == "" {
		resp.Fail(c, 4000, "命令模板不能为空")
		return
	}

	claims, _ := middleware.GetClaims(c)
	userID := uint64(0)
	if claims != nil {
		userID = uint64(claims.UserID)
	}

	if err := dc.svc.UpdateConfig(c.Request.Context(), id, req, userID); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// GetConfigVersions 获取配置版本历史
func (dc *DeployConfigController) GetConfigVersions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	versions, err := dc.svc.GetConfigVersions(c.Request.Context(), id)
	if err != nil {
		resp.Fail(c, 5000, "获取版本历史失败")
		return
	}
	resp.OK(c, versions)
}

// ─── DeployRepository ───

// ListRepositories 获取仓库配置列表
func (dc *DeployConfigController) ListRepositories(c *gin.Context) {
	repoType := c.Query("type")
	repos, err := dc.svc.ListRepositories(c.Request.Context(), repoType)
	if err != nil {
		resp.Fail(c, 5000, "获取仓库列表失败")
		return
	}
	resp.OK(c, repos)
}

// GetRepository 获取单个仓库配置
func (dc *DeployConfigController) GetRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	repo, err := dc.svc.GetRepository(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, repo)
}

// CreateRepository 创建仓库配置
func (dc *DeployConfigController) CreateRepository(c *gin.Context) {
	var req service.CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "请求参数错误")
		return
	}

	claims, _ := middleware.GetClaims(c)
	userID := uint64(0)
	if claims != nil {
		userID = uint64(claims.UserID)
	}

	id, err := dc.svc.CreateRepository(c.Request.Context(), req, userID)
	if err != nil {
		resp.Fail(c, 5000, "创建仓库配置失败")
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// UpdateRepository 更新仓库配置
func (dc *DeployConfigController) UpdateRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	var req service.UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "请求参数错误")
		return
	}

	if err := dc.svc.UpdateRepository(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// GetAnsiblePlaybook 读取 Ansible site.yml 源码。
func (dc *DeployConfigController) GetAnsiblePlaybook(c *gin.Context) {
	content, err := dc.svc.ReadAnsiblePlaybook(c.Request.Context())
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"content": content, "path": "ansible/site.yml"})
}

// GetAnsibleInventoryTemplate 读取 Ansible inventory 模板。
func (dc *DeployConfigController) GetAnsibleInventoryTemplate(c *gin.Context) {
	content, err := dc.svc.ReadAnsibleInventoryTemplate(c.Request.Context())
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"content": content, "path": "ansible/inventory.ini"})
}

// CheckAnsibleEnv 检查当前环境是否已安装 ansible-playbook。
func (dc *DeployConfigController) CheckAnsibleEnv(c *gin.Context) {
	installed, version, err := dc.svc.CheckAnsibleEnv(c.Request.Context())
	if err != nil {
		resp.Fail(c, 5000, "检查 Ansible 环境失败")
		return
	}
	resp.OK(c, gin.H{"installed": installed, "version": version})
}

// DeleteRepository 删除仓库配置
func (dc *DeployConfigController) DeleteRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	if err := dc.svc.DeleteRepository(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
