package controller

import (
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type K8sPermissionAuditController struct {
	svc *service.K8sPermissionAuditService
}

func NewK8sPermissionAuditController(svc *service.K8sPermissionAuditService) *K8sPermissionAuditController {
	return &K8sPermissionAuditController{svc: svc}
}

type createPermissionAuditReq struct {
	Mode                      string   `json:"mode"`
	IncludeRuntimeRBAC        bool     `json:"include_runtime_rbac"`
	IncludeOwnershipDetection bool     `json:"include_ownership_detection"`
	Namespaces                []string `json:"namespaces"`
	LabelSelector             string   `json:"label_selector"`
	ResourceAllowlist         []string `json:"resource_allowlist"`
}

type createAdhocPermissionAuditReq struct {
	DisplayName string `json:"display_name"`
	Kubeconfig  string `json:"kubeconfig"`
	createPermissionAuditReq
}

// CreateManaged godoc
// @Summary  发起已纳管集群权限分析
// @Tags     k8s-permission-audit
// @Security BearerAuth
// @Param    id path int true "集群 ID"
// @Param    body body createPermissionAuditReq true "分析参数"
// @Success  200 {object} resp.Result
// @Router   /api/v1/clusters/{id}/permission-audits [post]
func (ctl *K8sPermissionAuditController) CreateManaged(c *gin.Context) {
	if ctl == nil || ctl.svc == nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	clusterID, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req createPermissionAuditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.CreateManagedAudit(c.Request.Context(), clusterID, service.PermissionAuditCreateRequest{
		Mode:                      req.Mode,
		IncludeRuntimeRBAC:        req.IncludeRuntimeRBAC,
		IncludeOwnershipDetection: req.IncludeOwnershipDetection,
		Namespaces:                req.Namespaces,
		LabelSelector:             req.LabelSelector,
		ResourceAllowlist:         req.ResourceAllowlist,
	}, permissionAuditCurrentUserID(c))
	if err != nil {
		WriteServiceErr(c, err, K8sErrMappings...)
		return
	}
	resp.OK(c, data)
}

// CreateAdhoc godoc
// @Summary  发起临时 kubeconfig 权限分析
// @Tags     k8s-permission-audit
// @Security BearerAuth
// @Param    body body createAdhocPermissionAuditReq true "临时凭据与分析参数"
// @Success  200 {object} resp.Result
// @Router   /api/v1/permission-audits/adhoc [post]
func (ctl *K8sPermissionAuditController) CreateAdhoc(c *gin.Context) {
	if ctl == nil || ctl.svc == nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	var req createAdhocPermissionAuditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.CreateAdhocAudit(c.Request.Context(), service.PermissionAuditAdhocCreateRequest{
		DisplayName: req.DisplayName,
		Kubeconfig:  req.Kubeconfig,
		PermissionAuditCreateRequest: service.PermissionAuditCreateRequest{
			Mode:                      req.Mode,
			IncludeRuntimeRBAC:        req.IncludeRuntimeRBAC,
			IncludeOwnershipDetection: req.IncludeOwnershipDetection,
			Namespaces:                req.Namespaces,
			LabelSelector:             req.LabelSelector,
			ResourceAllowlist:         req.ResourceAllowlist,
		},
	}, permissionAuditCurrentUserID(c))
	if err != nil {
		WriteServiceErr(c, err, K8sErrMappings...)
		return
	}
	resp.OK(c, data)
}

// List godoc
// @Summary  权限分析任务列表
// @Tags     k8s-permission-audit
// @Security BearerAuth
// @Param    page query int false "页码"
// @Param    page_size query int false "每页数量"
// @Param    source_type query string false "来源类型"
// @Param    status query string false "任务状态"
// @Param    risk_level query string false "风险等级"
// @Param    cluster_id query int false "集群 ID"
// @Param    keyword query string false "关键字"
// @Param    sort_by query string false "排序字段"
// @Param    order query string false "排序方向"
// @Success  200 {object} resp.Result
// @Router   /api/v1/permission-audits [get]
func (ctl *K8sPermissionAuditController) List(c *gin.Context) {
	data, err := ctl.svc.ListAudits(c.Request.Context(), service.ListPermissionAuditsRequest{
		Page:       parseInt(c.Query("page"), 1),
		PageSize:   parseInt(c.Query("page_size"), 10),
		SourceType: strings.TrimSpace(c.Query("source_type")),
		Status:     strings.TrimSpace(c.Query("status")),
		RiskLevel:  strings.TrimSpace(c.Query("risk_level")),
		ClusterID:  uint64(parseInt64(c.Query("cluster_id"), 0)),
		Keyword:    strings.TrimSpace(c.Query("keyword")),
		SortBy:     strings.TrimSpace(c.Query("sort_by")),
		Order:      strings.TrimSpace(c.Query("order")),
	})
	if err != nil {
		WriteServiceErr(c, err, K8sErrMappings...)
		return
	}
	resp.OK(c, data)
}

// Get godoc
// @Summary  权限分析详情
// @Tags     k8s-permission-audit
// @Security BearerAuth
// @Param    id path int true "分析 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/permission-audits/{id} [get]
func (ctl *K8sPermissionAuditController) Get(c *gin.Context) {
	auditID := uint64(parseInt64(c.Param("id"), 0))
	if auditID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.GetAudit(c.Request.Context(), auditID)
	if err != nil {
		WriteServiceErr(c, err, K8sErrMappings...)
		return
	}
	resp.OK(c, data)
}

func (ctl *K8sPermissionAuditController) Logs(c *gin.Context) {
	auditID := uint64(parseInt64(c.Param("id"), 0))
	if auditID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.AuditTaskLogs(c.Request.Context(), auditID, parseInt(c.Query("offset"), 0), parseInt(c.Query("limit"), 200))
	if err != nil {
		WriteServiceErr(c, err, K8sErrMappings...)
		return
	}
	resp.OK(c, data)
}

func (ctl *K8sPermissionAuditController) Cancel(c *gin.Context) {
	auditID := uint64(parseInt64(c.Param("id"), 0))
	if auditID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.svc.CancelAudit(c.Request.Context(), auditID); err != nil {
		WriteServiceErr(c, err, K8sErrMappings...)
		return
	}
	resp.OK(c, gin.H{})
}

func (ctl *K8sPermissionAuditController) Compare(c *gin.Context) {
	auditID := uint64(parseInt64(c.Param("id"), 0))
	if auditID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	baselineID := uint64(parseInt64(c.Query("baseline_id"), 0))
	data, err := ctl.svc.CompareAudits(c.Request.Context(), auditID, baselineID)
	if err != nil {
		WriteServiceErr(c, err, K8sErrMappings...)
		return
	}
	resp.OK(c, data)
}

// ListFindings godoc
// @Summary  权限分析 findings 列表
// @Tags     k8s-permission-audit
// @Security BearerAuth
// @Param    id path int true "分析 ID"
// @Param    page query int false "页码"
// @Param    page_size query int false "每页数量"
// @Param    finding_type query string false "结论类型"
// @Param    risk_level query string false "风险等级"
// @Param    ownership_class query string false "归属分类"
// @Param    privilege_class query string false "权限分类"
// @Param    namespace query string false "命名空间"
// @Param    kind query string false "资源类型"
// @Param    deployment_blocker query bool false "是否部署阻塞"
// @Param    keyword query string false "关键字"
// @Param    sort_by query string false "排序字段"
// @Param    order query string false "排序方向"
// @Success  200 {object} resp.Result
// @Router   /api/v1/permission-audits/{id}/findings [get]
func (ctl *K8sPermissionAuditController) ListFindings(c *gin.Context) {
	auditID := uint64(parseInt64(c.Param("id"), 0))
	if auditID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var deploymentBlocker *bool
	if value := strings.TrimSpace(c.Query("deployment_blocker")); value != "" {
		parsed := strings.EqualFold(value, "true") || value == "1"
		deploymentBlocker = &parsed
	}
	data, err := ctl.svc.ListFindings(c.Request.Context(), auditID, service.ListPermissionAuditFindingsRequest{
		Page:              parseInt(c.Query("page"), 1),
		PageSize:          parseInt(c.Query("page_size"), 20),
		FindingType:       strings.TrimSpace(c.Query("finding_type")),
		RiskLevel:         strings.TrimSpace(c.Query("risk_level")),
		OwnershipClass:    strings.TrimSpace(c.Query("ownership_class")),
		PrivilegeClass:    strings.TrimSpace(c.Query("privilege_class")),
		Namespace:         strings.TrimSpace(c.Query("namespace")),
		Kind:              strings.TrimSpace(c.Query("kind")),
		DeploymentBlocker: deploymentBlocker,
		Keyword:           strings.TrimSpace(c.Query("keyword")),
		SortBy:            strings.TrimSpace(c.Query("sort_by")),
		Order:             strings.TrimSpace(c.Query("order")),
	})
	if err != nil {
		WriteServiceErr(c, err, K8sErrMappings...)
		return
	}
	resp.OK(c, data)
}

// LatestForCluster godoc
// @Summary  获取集群最近一次权限分析结果
// @Tags     k8s-permission-audit
// @Security BearerAuth
// @Param    id path int true "集群 ID"
// @Success  200 {object} resp.Result
// @Router   /api/v1/clusters/{id}/permission-audits/latest [get]
func (ctl *K8sPermissionAuditController) LatestForCluster(c *gin.Context) {
	clusterID, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.svc.GetLatestClusterAudit(c.Request.Context(), clusterID)
	if err != nil {
		WriteServiceErr(c, err, K8sErrMappings...)
		return
	}
	resp.OK(c, data)
}

func permissionAuditCurrentUserID(c *gin.Context) uint64 {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims == nil || claims.UserID <= 0 {
		return 0
	}
	return uint64(claims.UserID)
}

// RecommendRBAC godoc
// @Summary  生成最小权限 RBAC YAML
// @Tags     k8s-permission-audit
// @Security BearerAuth
// @Param    id         path  int    true  "集群 ID"
// @Param    namespaces query string false "目标命名空间，逗号分隔，例如 blueking,devops,coding"
// @Success  200 {object} resp.Result
// @Router   /api/v1/clusters/{id}/permission-audits/recommend-rbac [get]
func (ctl *K8sPermissionAuditController) RecommendRBAC(c *gin.Context) {
	if ctl == nil || ctl.svc == nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	clusterID, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var namespaces []string
	for _, ns := range strings.Split(c.Query("namespaces"), ",") {
		if n := strings.TrimSpace(ns); n != "" {
			namespaces = append(namespaces, n)
		}
	}
	result := ctl.svc.GenerateRBACRecommendation(c.Request.Context(), clusterID, namespaces)
	resp.OK(c, result)
}

// DefaultRBACMatrix godoc
// @Summary  获取推荐权限矩阵默认值
// @Tags     k8s-permission-audit
// @Security BearerAuth
// @Param    id         path  int    true  "集群 ID"
// @Param    namespaces query string false "目标命名空间，逗号分隔"
// @Success  200 {object} resp.Result
// @Router   /api/v1/clusters/{id}/permission-audits/rbac-matrix/default [get]
func (ctl *K8sPermissionAuditController) DefaultRBACMatrix(c *gin.Context) {
	if ctl == nil || ctl.svc == nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	var namespaces []string
	for _, ns := range strings.Split(c.Query("namespaces"), ",") {
		if n := strings.TrimSpace(ns); n != "" {
			namespaces = append(namespaces, n)
		}
	}
	result := service.DefaultRBACMatrix(namespaces)
	resp.OK(c, result)
}

// RBACFromMatrix godoc
// @Summary  根据权限矩阵生成 RBAC YAML
// @Tags     k8s-permission-audit
// @Security BearerAuth
// @Param    id   path int true "集群 ID"
// @Param    body body service.RBACMatrixRequest true "权限矩阵"
// @Success  200 {object} resp.Result
// @Router   /api/v1/clusters/{id}/permission-audits/rbac-matrix/yaml [post]
func (ctl *K8sPermissionAuditController) RBACFromMatrix(c *gin.Context) {
	if ctl == nil || ctl.svc == nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	var req service.RBACMatrixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	yaml := service.BuildRBACFromMatrix(req)
	resp.OK(c, map[string]string{"yaml_content": yaml})
}
