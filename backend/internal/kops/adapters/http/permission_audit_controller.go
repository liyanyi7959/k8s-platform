package http

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

// PermissionAuditController is the HTTP adapter for Kops permission-audit
// commands and read queries. The scanner itself is reached only through the
// application runtime port.
type PermissionAuditController struct {
	service *kopsapp.PermissionAuditService
}

func NewPermissionAuditController(service *kopsapp.PermissionAuditService) *PermissionAuditController {
	return &PermissionAuditController{service: service}
}

type permissionAuditCreateRequest struct {
	Mode                      string   `json:"mode"`
	IncludeRuntimeRBAC        bool     `json:"include_runtime_rbac"`
	IncludeOwnershipDetection bool     `json:"include_ownership_detection"`
	Namespaces                []string `json:"namespaces"`
	LabelSelector             string   `json:"label_selector"`
	ResourceAllowlist         []string `json:"resource_allowlist"`
}

type permissionAuditAdhocRequest struct {
	DisplayName string `json:"display_name"`
	Kubeconfig  string `json:"kubeconfig"`
	permissionAuditCreateRequest
}

func (ctl *PermissionAuditController) CreateManaged(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	clusterID, ok := permissionAuditPathID(c, "id")
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var request permissionAuditCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	result, err := ctl.service.CreateManaged(c.Request.Context(), clusterID, permissionAuditInput(request), permissionAuditCurrentUserID(c))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *PermissionAuditController) CreateAdhoc(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	var request permissionAuditAdhocRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	result, err := ctl.service.CreateAdhoc(c.Request.Context(), kopsapp.PermissionAuditAdhocInput{
		DisplayName: request.DisplayName, Kubeconfig: request.Kubeconfig, PermissionAuditCreateInput: permissionAuditInput(request.permissionAuditCreateRequest),
	}, permissionAuditCurrentUserID(c))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *PermissionAuditController) List(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.List(c.Request.Context(), kopsapp.PermissionAuditListQuery{
		Page: permissionAuditQueryInt(c, "page", 1), PageSize: permissionAuditQueryInt(c, "page_size", 10),
		SourceType: c.Query("source_type"), Status: c.Query("status"), RiskLevel: c.Query("risk_level"),
		ClusterID: permissionAuditQueryUint(c, "cluster_id"), Keyword: c.Query("keyword"), SortBy: c.Query("sort_by"), Order: c.Query("order"),
	})
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *PermissionAuditController) Get(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.Get(c.Request.Context(), permissionAuditPathIDValue(c, "id"))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *PermissionAuditController) Logs(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.Logs(c.Request.Context(), permissionAuditPathIDValue(c, "id"), permissionAuditQueryInt(c, "offset", 0), permissionAuditQueryInt(c, "limit", 200))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *PermissionAuditController) Cancel(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	if err := ctl.service.Cancel(c.Request.Context(), permissionAuditPathIDValue(c, "id")); err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func (ctl *PermissionAuditController) Compare(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.Compare(c.Request.Context(), permissionAuditPathIDValue(c, "id"), permissionAuditQueryUint(c, "baseline_id"))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *PermissionAuditController) ListFindings(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.ListFindings(c.Request.Context(), permissionAuditPathIDValue(c, "id"), kopsapp.PermissionAuditFindingsQuery{
		Page: permissionAuditQueryInt(c, "page", 1), PageSize: permissionAuditQueryInt(c, "page_size", 20),
		FindingType: c.Query("finding_type"), RiskLevel: c.Query("risk_level"), OwnershipClass: c.Query("ownership_class"),
		PrivilegeClass: c.Query("privilege_class"), Namespace: c.Query("namespace"), Kind: c.Query("kind"),
		DeploymentBlocker: permissionAuditOptionalBool(c, "deployment_blocker"), Keyword: c.Query("keyword"),
		SortBy: c.Query("sort_by"), Order: c.Query("order"),
	})
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *PermissionAuditController) LatestForCluster(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.LatestForCluster(c.Request.Context(), permissionAuditPathIDValue(c, "id"))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *PermissionAuditController) RecommendRBAC(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.RecommendRBAC(c.Request.Context(), permissionAuditPathIDValue(c, "id"), permissionAuditCSV(c.Query("namespaces")))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *PermissionAuditController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}

func permissionAuditInput(request permissionAuditCreateRequest) kopsapp.PermissionAuditCreateInput {
	return kopsapp.PermissionAuditCreateInput{
		Mode: request.Mode, IncludeRuntimeRBAC: request.IncludeRuntimeRBAC, IncludeOwnershipDetection: request.IncludeOwnershipDetection,
		Namespaces: request.Namespaces, LabelSelector: request.LabelSelector, ResourceAllowlist: request.ResourceAllowlist,
	}
}

func permissionAuditPathID(c *gin.Context, name string) (uint64, bool) {
	value, err := strconv.ParseUint(strings.TrimSpace(c.Param(name)), 10, 64)
	return value, err == nil && value > 0
}

func permissionAuditPathIDValue(c *gin.Context, name string) uint64 {
	value, _ := permissionAuditPathID(c, name)
	return value
}

func permissionAuditQueryInt(c *gin.Context, name string, fallback int) int {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func permissionAuditQueryUint(c *gin.Context, name string) uint64 {
	value, err := strconv.ParseUint(strings.TrimSpace(c.Query(name)), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func permissionAuditOptionalBool(c *gin.Context, name string) *bool {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil
	}
	value := strings.EqualFold(raw, "true") || raw == "1"
	return &value
}

func permissionAuditCSV(raw string) []string {
	return strings.Split(raw, ",")
}

func permissionAuditCurrentUserID(c *gin.Context) uint64 {
	claims, _ := middleware.GetClaims(c)
	if claims == nil || claims.UserID <= 0 {
		return 0
	}
	return uint64(claims.UserID)
}
