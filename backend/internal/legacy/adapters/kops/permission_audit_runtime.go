package kops

import (
	"context"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

// PermissionAuditRuntime keeps the existing asynchronous scanner behind the
// Kops application port while route and command ownership move out of legacy.
type PermissionAuditRuntime struct {
	service *service.K8sPermissionAuditService
}

func NewPermissionAuditRuntime(service *service.K8sPermissionAuditService) *PermissionAuditRuntime {
	return &PermissionAuditRuntime{service: service}
}

func (r *PermissionAuditRuntime) CreateManaged(ctx context.Context, clusterID uint64, input kopsapp.PermissionAuditCreateInput, createdBy uint64) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.CreateManagedAudit(ctx, clusterID, permissionAuditCreateRequest(input), createdBy)
	return value, translateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) CreateAdhoc(ctx context.Context, input kopsapp.PermissionAuditAdhocInput, createdBy uint64) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.CreateAdhocAudit(ctx, service.PermissionAuditAdhocCreateRequest{
		DisplayName: input.DisplayName, Kubeconfig: input.Kubeconfig, PermissionAuditCreateRequest: permissionAuditCreateRequest(input.PermissionAuditCreateInput),
	}, createdBy)
	return value, translateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) List(ctx context.Context, query kopsapp.PermissionAuditListQuery) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.ListAudits(ctx, service.ListPermissionAuditsRequest{
		Page: query.Page, PageSize: query.PageSize, SourceType: query.SourceType, Status: query.Status, RiskLevel: query.RiskLevel,
		ClusterID: query.ClusterID, Keyword: query.Keyword, SortBy: query.SortBy, Order: query.Order,
	})
	return value, translateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) Get(ctx context.Context, auditID uint64) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.GetAudit(ctx, auditID)
	return value, translateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) Logs(ctx context.Context, auditID uint64, offset, limit int) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.AuditTaskLogs(ctx, auditID, offset, limit)
	return value, translateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) Cancel(ctx context.Context, auditID uint64) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.CancelAudit(ctx, auditID))
}

func (r *PermissionAuditRuntime) Compare(ctx context.Context, auditID, baselineID uint64) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.CompareAudits(ctx, auditID, baselineID)
	return value, translateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) ListFindings(ctx context.Context, auditID uint64, query kopsapp.PermissionAuditFindingsQuery) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.ListFindings(ctx, auditID, service.ListPermissionAuditFindingsRequest{
		Page: query.Page, PageSize: query.PageSize, FindingType: query.FindingType, RiskLevel: query.RiskLevel,
		OwnershipClass: query.OwnershipClass, PrivilegeClass: query.PrivilegeClass, Namespace: query.Namespace, Kind: query.Kind,
		DeploymentBlocker: query.DeploymentBlocker, Keyword: query.Keyword, SortBy: query.SortBy, Order: query.Order,
	})
	return value, translateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) LatestForCluster(ctx context.Context, clusterID uint64) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.GetLatestClusterAudit(ctx, clusterID)
	return value, translateKopsRuntimeError(err)
}

func (r *PermissionAuditRuntime) RecommendRBAC(ctx context.Context, clusterID uint64, namespaces []string) any {
	if r == nil || r.service == nil {
		return nil
	}
	return r.service.GenerateRBACRecommendation(ctx, clusterID, namespaces)
}

func permissionAuditCreateRequest(input kopsapp.PermissionAuditCreateInput) service.PermissionAuditCreateRequest {
	return service.PermissionAuditCreateRequest{
		Mode: input.Mode, IncludeRuntimeRBAC: input.IncludeRuntimeRBAC, IncludeOwnershipDetection: input.IncludeOwnershipDetection,
		Namespaces: input.Namespaces, LabelSelector: input.LabelSelector, ResourceAllowlist: input.ResourceAllowlist,
	}
}
