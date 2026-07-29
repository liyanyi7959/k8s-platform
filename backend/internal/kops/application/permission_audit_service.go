package application

import (
	"context"
	"strings"
)

// PermissionAuditCreateInput is the stable command at the Kops boundary.
// The runtime decides whether it is executed against a managed cluster or a
// supplied kubeconfig, but transport code never constructs legacy requests.
type PermissionAuditCreateInput struct {
	Mode                      string   `json:"mode"`
	IncludeRuntimeRBAC        bool     `json:"include_runtime_rbac"`
	IncludeOwnershipDetection bool     `json:"include_ownership_detection"`
	Namespaces                []string `json:"namespaces"`
	LabelSelector             string   `json:"label_selector"`
	ResourceAllowlist         []string `json:"resource_allowlist"`
}

type PermissionAuditAdhocInput struct {
	DisplayName string `json:"display_name"`
	Kubeconfig  string `json:"kubeconfig"`
	PermissionAuditCreateInput
}

type PermissionAuditListQuery struct {
	Page       int
	PageSize   int
	SourceType string
	Status     string
	RiskLevel  string
	ClusterID  uint64
	Keyword    string
	SortBy     string
	Order      string
}

type PermissionAuditFindingsQuery struct {
	Page              int
	PageSize          int
	FindingType       string
	RiskLevel         string
	OwnershipClass    string
	PrivilegeClass    string
	Namespace         string
	Kind              string
	DeploymentBlocker *bool
	Keyword           string
	SortBy            string
	Order             string
}

// PermissionAuditRuntime isolates the retained scan engine and audit store.
// Result values stay opaque at this boundary until the persistence/read-model
// migration is complete; commands and validation are fully Kops-owned now.
type PermissionAuditRuntime interface {
	CreateManaged(context.Context, uint64, PermissionAuditCreateInput, uint64) (any, error)
	CreateAdhoc(context.Context, PermissionAuditAdhocInput, uint64) (any, error)
	List(context.Context, PermissionAuditListQuery) (any, error)
	Get(context.Context, uint64) (any, error)
	Logs(context.Context, uint64, int, int) (any, error)
	Cancel(context.Context, uint64) error
	Compare(context.Context, uint64, uint64) (any, error)
	ListFindings(context.Context, uint64, PermissionAuditFindingsQuery) (any, error)
	LatestForCluster(context.Context, uint64) (any, error)
	RecommendRBAC(context.Context, uint64, []string) any
}

type PermissionAuditService struct{ runtime PermissionAuditRuntime }

func NewPermissionAuditService(runtime PermissionAuditRuntime) *PermissionAuditService {
	return &PermissionAuditService{runtime: runtime}
}

func (s *PermissionAuditService) CreateManaged(ctx context.Context, clusterID uint64, input PermissionAuditCreateInput, createdBy uint64) (any, error) {
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	input, err := normalizePermissionAuditCreate(input)
	if err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.CreateManaged(ctx, clusterID, input, createdBy)
}

func (s *PermissionAuditService) CreateAdhoc(ctx context.Context, input PermissionAuditAdhocInput, createdBy uint64) (any, error) {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Kubeconfig = strings.TrimSpace(input.Kubeconfig)
	if input.Kubeconfig == "" {
		return nil, ErrInvalidParams
	}
	value, err := normalizePermissionAuditCreate(input.PermissionAuditCreateInput)
	if err != nil {
		return nil, err
	}
	input.PermissionAuditCreateInput = value
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.CreateAdhoc(ctx, input, createdBy)
}

func (s *PermissionAuditService) List(ctx context.Context, query PermissionAuditListQuery) (any, error) {
	query = normalizePermissionAuditListQuery(query)
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.List(ctx, query)
}

func (s *PermissionAuditService) Get(ctx context.Context, auditID uint64) (any, error) {
	if auditID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Get(ctx, auditID)
}

func (s *PermissionAuditService) Logs(ctx context.Context, auditID uint64, offset, limit int) (any, error) {
	if auditID == 0 {
		return nil, ErrInvalidParams
	}
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > 1000 {
		limit = 200
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Logs(ctx, auditID, offset, limit)
}

func (s *PermissionAuditService) Cancel(ctx context.Context, auditID uint64) error {
	if auditID == 0 {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Cancel(ctx, auditID)
}

func (s *PermissionAuditService) Compare(ctx context.Context, auditID, baselineID uint64) (any, error) {
	if auditID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Compare(ctx, auditID, baselineID)
}

func (s *PermissionAuditService) ListFindings(ctx context.Context, auditID uint64, query PermissionAuditFindingsQuery) (any, error) {
	if auditID == 0 {
		return nil, ErrInvalidParams
	}
	query = normalizePermissionAuditFindingsQuery(query)
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.ListFindings(ctx, auditID, query)
}

func (s *PermissionAuditService) LatestForCluster(ctx context.Context, clusterID uint64) (any, error) {
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.LatestForCluster(ctx, clusterID)
}

func (s *PermissionAuditService) RecommendRBAC(ctx context.Context, clusterID uint64, namespaces []string) (any, error) {
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.RecommendRBAC(ctx, clusterID, normalizePermissionAuditStrings(namespaces)), nil
}

func normalizePermissionAuditCreate(input PermissionAuditCreateInput) (PermissionAuditCreateInput, error) {
	input.Mode = strings.ToLower(strings.TrimSpace(input.Mode))
	if input.Mode == "" {
		input.Mode = "full"
	}
	if input.Mode != "full" {
		return PermissionAuditCreateInput{}, ErrInvalidParams
	}
	input.Namespaces = normalizePermissionAuditStrings(input.Namespaces)
	input.LabelSelector = strings.TrimSpace(input.LabelSelector)
	input.ResourceAllowlist = normalizePermissionAuditStrings(input.ResourceAllowlist)
	return input, nil
}

func normalizePermissionAuditListQuery(query PermissionAuditListQuery) PermissionAuditListQuery {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 10
	}
	if query.PageSize > 200 {
		query.PageSize = 200
	}
	query.SourceType = strings.TrimSpace(query.SourceType)
	query.Status = strings.TrimSpace(query.Status)
	query.RiskLevel = strings.TrimSpace(query.RiskLevel)
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.SortBy = strings.TrimSpace(query.SortBy)
	query.Order = strings.TrimSpace(query.Order)
	return query
}

func normalizePermissionAuditFindingsQuery(query PermissionAuditFindingsQuery) PermissionAuditFindingsQuery {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 200 {
		query.PageSize = 200
	}
	query.FindingType = strings.TrimSpace(query.FindingType)
	query.RiskLevel = strings.TrimSpace(query.RiskLevel)
	query.OwnershipClass = strings.TrimSpace(query.OwnershipClass)
	query.PrivilegeClass = strings.TrimSpace(query.PrivilegeClass)
	query.Namespace = strings.TrimSpace(query.Namespace)
	query.Kind = strings.TrimSpace(query.Kind)
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.SortBy = strings.TrimSpace(query.SortBy)
	query.Order = strings.TrimSpace(query.Order)
	return query
}

func normalizePermissionAuditStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
