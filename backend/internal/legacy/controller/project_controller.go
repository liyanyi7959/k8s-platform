package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/legacy/model"
	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/pkg/resp"
)

// ProjectController 项目管理控制器。
type ProjectController struct {
	svc    *service.ProjectService
	k8sSvc *service.K8sService
}

// NewProjectController 创建项目管理控制器实例。
func NewProjectController(svc *service.ProjectService, k8sSvc *service.K8sService) *ProjectController {
	return &ProjectController{svc: svc, k8sSvc: k8sSvc}
}

// projectReq 为创建/更新项目时的请求体。
type projectReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ClusterID   uint64 `json:"cluster_id"`
	Namespaces  string `json:"namespaces"`
	QuotaCPU    string `json:"quota_cpu"`
	QuotaMemory string `json:"quota_memory"`
	QuotaPods   string `json:"quota_pods"`
}

// ListProjects 分页查询项目列表。
// GET /projects?page=1&page_size=20
func (pc *ProjectController) ListProjects(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)
	data, err := pc.svc.ListProjects(c.Request.Context(), page, pageSize)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// GetProject 查询单个项目详情。
// GET /projects/:id
func (pc *ProjectController) GetProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := pc.svc.GetProject(c.Request.Context(), uint64(id))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// CreateProject 创建项目。
// POST /projects
func (pc *ProjectController) CreateProject(c *gin.Context) {
	var req projectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	// 从鉴权信息中获取创建者 ID。
	creatorID := uint64(0)
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		creatorID = uint64(claims.UserID)
	}

	p := &model.Project{
		Name:        req.Name,
		Description: req.Description,
		ClusterID:   req.ClusterID,
		Namespaces:  req.Namespaces,
		QuotaCPU:    req.QuotaCPU,
		QuotaMemory: req.QuotaMemory,
		QuotaPods:   req.QuotaPods,
		CreatorID:   creatorID,
	}
	if err := pc.svc.CreateProject(c.Request.Context(), p); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": p.ID})
}

// UpdateProject 更新项目。
// PUT /projects/:id
func (pc *ProjectController) UpdateProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req projectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	updates := map[string]any{
		"name":         req.Name,
		"description":  req.Description,
		"cluster_id":   req.ClusterID,
		"namespaces":   req.Namespaces,
		"quota_cpu":    req.QuotaCPU,
		"quota_memory": req.QuotaMemory,
		"quota_pods":   req.QuotaPods,
	}
	if err := pc.svc.UpdateProject(c.Request.Context(), uint64(id), updates); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// DeleteProject 删除项目（软删除）。
// DELETE /projects/:id
func (pc *ProjectController) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := pc.svc.DeleteProject(c.Request.Context(), uint64(id)); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// namespaceResourcesResp 单个命名空间的资源统计。
type namespaceResourcesResp struct {
	Pods        int `json:"pods"`
	Deployments int `json:"deployments"`
	Services    int `json:"services"`
	ConfigMaps  int `json:"configmaps"`
	Total       int `json:"total"`
}

// GetProjectResources 获取项目下所有命名空间的资源统计。
// GET /projects/:id/resources
func (pc *ProjectController) GetProjectResources(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	// 获取项目信息
	project, err := pc.svc.GetProject(c.Request.Context(), uint64(id))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}

	// 未绑定集群或未配置命名空间时返回空结果
	if project.ClusterID == 0 || pc.k8sSvc == nil {
		resp.OK(c, gin.H{"cluster_id": project.ClusterID, "namespaces": gin.H{}})
		return
	}

	// 拆分命名空间
	nsList := splitAndTrim(project.Namespaces)
	if len(nsList) == 0 {
		resp.OK(c, gin.H{"cluster_id": project.ClusterID, "namespaces": gin.H{}})
		return
	}

	// 遍历每个命名空间，获取资源统计
	result := make(map[string]namespaceResourcesResp, len(nsList))
	for _, ns := range nsList {
		items, total, err := pc.k8sSvc.GetNamespaceResourcesSummary(c.Request.Context(), project.ClusterID, ns)
		if err != nil {
			// 单个命名空间查询失败时记录零值，不影响整体返回
			result[ns] = namespaceResourcesResp{}
			continue
		}
		r := namespaceResourcesResp{Total: total}
		for _, item := range items {
			switch item.Key {
			case "pods|pod":
				r.Pods = item.Count
			case "deployments|deployment":
				r.Deployments = item.Count
			case "services|service":
				r.Services = item.Count
			case "configmaps|configmap":
				r.ConfigMaps = item.Count
			}
		}
		result[ns] = r
	}

	resp.OK(c, gin.H{"cluster_id": project.ClusterID, "namespaces": result})
}

// assignNamespacesReq 分配命名空间请求体。
type assignNamespacesReq struct {
	Namespaces []string `json:"namespaces"`
}

// AssignNamespaces 将命名空间分配到项目。
// PUT /projects/:id/namespaces
func (pc *ProjectController) AssignNamespaces(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req assignNamespacesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	// 命名空间数组转逗号分隔字符串
	namespaces := joinNamespaces(req.Namespaces)
	updates := map[string]any{"namespaces": namespaces}
	if err := pc.svc.UpdateProject(c.Request.Context(), uint64(id), updates); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// splitAndTrim 将逗号分隔的字符串拆分并去除空白项。
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// joinNamespaces 将命名空间数组拼接为逗号分隔字符串。
func joinNamespaces(ns []string) string {
	parts := make([]string, 0, len(ns))
	for _, n := range ns {
		n = strings.TrimSpace(n)
		if n != "" {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, ",")
}
