package kops

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
	model "k8s-platform-backend/internal/kops/domain"
)

// PermissionAuditRuntime keeps the existing asynchronous scanner behind the
// Kops application port while route and command ownership move out of legacy.
type PermissionAuditRuntime struct {
	engine *PermissionAuditEngine
	db     *gorm.DB
}

// NewPermissionAuditRuntimeWithStore keeps the audit read-model dependency in
// the Kops runtime adapter. It lets recommendation queries share the same
// audit store while scan execution remains in the Kops adapter engine.
func NewPermissionAuditRuntimeWithStore(engine *PermissionAuditEngine, db *gorm.DB) *PermissionAuditRuntime {
	return &PermissionAuditRuntime{engine: engine, db: db}
}

func (r *PermissionAuditRuntime) CreateManaged(ctx context.Context, clusterID uint64, input kopsapp.PermissionAuditCreateInput, createdBy uint64) (any, error) {
	if r == nil || r.engine == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.engine.CreateManagedAudit(ctx, clusterID, permissionAuditCreateRequest(input), createdBy)
	return value, kopsruntime.TranslateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) CreateAdhoc(ctx context.Context, input kopsapp.PermissionAuditAdhocInput, createdBy uint64) (any, error) {
	if r == nil || r.engine == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.engine.CreateAdhocAudit(ctx, PermissionAuditAdhocCreateRequest{
		DisplayName: input.DisplayName, Kubeconfig: input.Kubeconfig, PermissionAuditCreateRequest: permissionAuditCreateRequest(input.PermissionAuditCreateInput),
	}, createdBy)
	return value, kopsruntime.TranslateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) List(ctx context.Context, query kopsapp.PermissionAuditListQuery) (any, error) {
	if r == nil || r.engine == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.engine.ListAudits(ctx, ListPermissionAuditsRequest{
		Page: query.Page, PageSize: query.PageSize, SourceType: query.SourceType, Status: query.Status, RiskLevel: query.RiskLevel,
		ClusterID: query.ClusterID, Keyword: query.Keyword, SortBy: query.SortBy, Order: query.Order,
	})
	return value, kopsruntime.TranslateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) Get(ctx context.Context, auditID uint64) (any, error) {
	if r == nil || r.engine == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.engine.GetAudit(ctx, auditID)
	return value, kopsruntime.TranslateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) Logs(ctx context.Context, auditID uint64, offset, limit int) (any, error) {
	if r == nil || r.engine == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.engine.AuditTaskLogs(ctx, auditID, offset, limit)
	return value, kopsruntime.TranslateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) Cancel(ctx context.Context, auditID uint64) error {
	if r == nil || r.engine == nil {
		return kopsapp.ErrConflict
	}
	return kopsruntime.TranslateKopsRuntimeError(r.engine.CancelAudit(ctx, auditID))
}

func (r *PermissionAuditRuntime) Compare(ctx context.Context, auditID, baselineID uint64) (any, error) {
	if r == nil || r.engine == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.engine.CompareAudits(ctx, auditID, baselineID)
	return value, kopsruntime.TranslateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) ListFindings(ctx context.Context, auditID uint64, query kopsapp.PermissionAuditFindingsQuery) (any, error) {
	if r == nil || r.engine == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.engine.ListFindings(ctx, auditID, ListPermissionAuditFindingsRequest{
		Page: query.Page, PageSize: query.PageSize, FindingType: query.FindingType, RiskLevel: query.RiskLevel,
		OwnershipClass: query.OwnershipClass, PrivilegeClass: query.PrivilegeClass, Namespace: query.Namespace, Kind: query.Kind,
		DeploymentBlocker: query.DeploymentBlocker, Keyword: query.Keyword, SortBy: query.SortBy, Order: query.Order,
	})
	return value, kopsruntime.TranslateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) LatestForCluster(ctx context.Context, clusterID uint64) (any, error) {
	if r == nil || r.engine == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.engine.GetLatestClusterAudit(ctx, clusterID)
	return value, kopsruntime.TranslateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) RecommendRBAC(ctx context.Context, clusterID uint64, namespaces []string) any {
	if r == nil {
		return nil
	}
	return r.recommendRBAC(ctx, clusterID, namespaces)
}

func permissionAuditCreateRequest(input kopsapp.PermissionAuditCreateInput) PermissionAuditCreateRequest {
	return PermissionAuditCreateRequest{
		Mode: input.Mode, IncludeRuntimeRBAC: input.IncludeRuntimeRBAC, IncludeOwnershipDetection: input.IncludeOwnershipDetection,
		Namespaces: input.Namespaces, LabelSelector: input.LabelSelector, ResourceAllowlist: input.ResourceAllowlist,
	}
}

type permissionAuditRBACRecommendation struct {
	YAMLContent      string   `json:"yaml_content"`
	ServiceAccount   string   `json:"service_account"`
	SANamespace      string   `json:"sa_namespace"`
	TargetNamespaces []string `json:"target_namespaces"`
}

func (r *PermissionAuditRuntime) recommendRBAC(ctx context.Context, clusterID uint64, namespaces []string) permissionAuditRBACRecommendation {
	namespaces = permissionAuditRuntimeStrings(namespaces, false)
	allowlist := []string(nil)
	if clusterID > 0 && r.db != nil {
		if row := r.latestSuccessfulAudit(ctx, clusterID); row != nil {
			allowlist = permissionAuditRuntimeRequestStrings(row.RequestJSON, "resource_allowlist", true)
			if len(namespaces) == 0 {
				namespaces = permissionAuditRuntimeRequestStrings(row.RequestJSON, "namespaces", false)
			}
		}
	}
	matrix := kopsapp.DefaultRBACMatrix(namespaces)
	if len(allowlist) > 0 {
		matrix = kopsapp.FilterRBACMatrixByResources(matrix, allowlist)
	}
	return permissionAuditRBACRecommendation{
		YAMLContent:      kopsapp.BuildRBACFromMatrix(matrix),
		ServiceAccount:   matrix.ServiceAccount,
		SANamespace:      matrix.SANamespace,
		TargetNamespaces: matrix.TargetNamespaces,
	}
}

func (r *PermissionAuditRuntime) latestSuccessfulAudit(ctx context.Context, clusterID uint64) *model.K8sPermissionAudit {
	if r == nil || r.db == nil || clusterID == 0 {
		return nil
	}
	var row model.K8sPermissionAudit
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND cluster_id = ? AND status IN ?", clusterID, []string{"success", "incomplete"}).
		Order("created_at desc").
		First(&row).Error; err != nil {
		return nil
	}
	return &row
}

func permissionAuditRuntimeRequestStrings(request model.JSONMap, key string, allowlist bool) []string {
	if request == nil {
		return nil
	}
	if direct, ok := request[key].([]string); ok {
		return permissionAuditRuntimeStrings(direct, allowlist)
	}
	items, ok := request[key].([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, fmt.Sprint(item))
	}
	return permissionAuditRuntimeStrings(values, allowlist)
}

func permissionAuditRuntimeStrings(values []string, allowlist bool) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if allowlist {
			value = kopsapp.NormalizePermissionAuditAllowlistItem(value)
		}
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
