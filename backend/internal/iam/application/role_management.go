package application

import (
	"context"
	"sort"
	"strings"

	"k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/internal/iam/ports"
)

type RoleManagement struct {
	repository ports.RoleManagementRepository
	auth       *AuthService
}

func NewRoleManagement(repository ports.RoleManagementRepository, auth *AuthService) *RoleManagement {
	return &RoleManagement{repository: repository, auth: auth}
}

type RoleNamespaceScope struct {
	ClusterID   uint64   `json:"cluster_id"`
	ClusterName string   `json:"cluster_name"`
	Namespaces  []string `json:"namespaces"`
}
type NamespaceScopeRequest struct {
	ClusterID  uint64   `json:"cluster_id"`
	Namespaces []string `json:"namespaces"`
}
type RoleListItem struct {
	ID             uint64              `json:"id"`
	Name           string              `json:"name"`
	Code           string              `json:"code"`
	Description    string              `json:"description"`
	Permissions    []string            `json:"permissions"`
	NamespaceScope *RoleNamespaceScope `json:"namespace_scope,omitempty"`
	UserCount      int64               `json:"user_count"`
	Builtin        bool                `json:"builtin"`
	CreatedAt      string              `json:"created_at"`
}
type CreateRoleRequest struct {
	Name           string                 `json:"name"`
	Code           string                 `json:"code"`
	Description    string                 `json:"description"`
	Desc           string                 `json:"desc"`
	Permissions    []string               `json:"permissions"`
	NamespaceScope *NamespaceScopeRequest `json:"namespace_scope"`
}
type UpdateRoleRequest struct {
	Name           *string                `json:"name"`
	Code           *string                `json:"code"`
	Description    *string                `json:"description"`
	Desc           *string                `json:"desc"`
	Permissions    []string               `json:"permissions"`
	NamespaceScope *NamespaceScopeRequest `json:"namespace_scope"`
}
type PermissionListItem struct {
	ID            uint64 `json:"id"`
	Code          string `json:"code"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	CategoryLabel string `json:"category_label"`
	Builtin       bool   `json:"builtin"`
}

func (s *RoleManagement) List(ctx context.Context) ([]RoleListItem, error) {
	records, err := s.repository.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]RoleListItem, 0, len(records))
	for _, record := range records {
		item := RoleListItem{ID: record.ID, Name: record.Name, Code: record.Code, Description: record.Description, Permissions: record.Permissions, UserCount: record.UserCount, Builtin: record.Name == "admin" || record.Code == "admin", CreatedAt: record.CreatedAt}
		if record.NamespaceScope != nil {
			item.NamespaceScope = &RoleNamespaceScope{ClusterID: record.NamespaceScope.ClusterID, ClusterName: record.NamespaceScope.ClusterName, Namespaces: record.NamespaceScope.Namespaces}
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *RoleManagement) Create(ctx context.Context, request CreateRoleRequest) (uint64, error) {
	name, code := strings.TrimSpace(request.Name), strings.TrimSpace(request.Code)
	if name == "" || code == "" {
		return 0, domain.ErrInvalidParams
	}
	permissions := normalizeStrings(request.Permissions)
	scope, err := normalizeScope(request.NamespaceScope, permissions)
	if err != nil {
		return 0, err
	}
	return s.repository.CreateRole(ctx, ports.CreateRoleData{Name: name, Code: code, Description: createDescription(request.Description, request.Desc), Permissions: permissions, Scope: scope})
}

func (s *RoleManagement) Update(ctx context.Context, id uint64, request UpdateRoleRequest) error {
	if id == 0 {
		return domain.ErrInvalidParams
	}
	data := ports.UpdateRoleData{SetPermissions: request.Permissions != nil, SetScope: request.NamespaceScope != nil}
	if request.Permissions != nil {
		data.Permissions = normalizeStrings(request.Permissions)
	}
	if request.Name != nil {
		value := strings.TrimSpace(*request.Name)
		if value == "" {
			return domain.ErrInvalidParams
		}
		data.Name = &value
	}
	if request.Code != nil {
		value := strings.TrimSpace(*request.Code)
		if value == "" {
			return domain.ErrInvalidParams
		}
		data.Code = &value
	}
	if request.Description != nil || request.Desc != nil {
		data.SetDescription = true
		data.Description = updateDescription(request.Description, request.Desc)
	}
	if request.NamespaceScope != nil {
		scope, err := normalizeScope(request.NamespaceScope, data.Permissions)
		if err != nil {
			return err
		}
		data.Scope = scope
	}
	userIDs, err := s.repository.UpdateRole(ctx, id, data)
	if err != nil {
		return err
	}
	for _, userID := range userIDs {
		s.auth.InvalidateUser(ctx, userID)
	}
	return nil
}

func (s *RoleManagement) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return domain.ErrInvalidParams
	}
	return s.repository.DeleteRole(ctx, id)
}

func (s *RoleManagement) ListPermissions(ctx context.Context) ([]PermissionListItem, error) {
	records, err := s.repository.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]PermissionListItem, 0, len(records))
	for _, record := range records {
		meta := domain.DescribePermission(record.Code, record.Description)
		items = append(items, PermissionListItem{ID: record.ID, Code: record.Code, Description: meta.Description, Category: meta.Category, CategoryLabel: meta.CategoryLabel, Builtin: meta.Builtin})
	}
	sort.Slice(items, func(i, j int) bool {
		left, right := domain.PermissionCategoryOrder(items[i].Category), domain.PermissionCategoryOrder(items[j].Category)
		if left != right {
			return left < right
		}
		if items[i].CategoryLabel != items[j].CategoryLabel {
			return items[i].CategoryLabel < items[j].CategoryLabel
		}
		return items[i].Code < items[j].Code
	})
	return items, nil
}

func normalizeScope(request *NamespaceScopeRequest, permissions []string) (*ports.RoleNamespaceScopeData, error) {
	if request == nil {
		return nil, nil
	}
	namespaces := normalizeStrings(request.Namespaces)
	if len(namespaces) == 0 {
		return nil, nil
	}
	if request.ClusterID == 0 || (permissions != nil && !domain.HasNamespacePermission(permissions)) {
		return nil, domain.ErrInvalidParams
	}
	return &ports.RoleNamespaceScopeData{ClusterID: request.ClusterID, Namespaces: namespaces}, nil
}
func createDescription(description, fallback string) *string {
	value := strings.TrimSpace(description)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if value == "" {
		return nil
	}
	return &value
}
func updateDescription(description, fallback *string) *string {
	value := ""
	if description != nil {
		value = strings.TrimSpace(*description)
	} else if fallback != nil {
		value = strings.TrimSpace(*fallback)
	}
	if value == "" {
		return nil
	}
	return &value
}
func normalizeStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
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
	sort.Strings(result)
	return result
}
