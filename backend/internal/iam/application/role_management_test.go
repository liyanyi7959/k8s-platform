package application

import (
	"context"
	"errors"
	"testing"

	"k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/internal/iam/ports"
)

type roleRepositoryStub struct {
	createData ports.CreateRoleData
	updateData ports.UpdateRoleData
}

func (stub *roleRepositoryStub) ListRoles(context.Context) ([]ports.RoleRecord, error) { return nil, nil }
func (stub *roleRepositoryStub) CreateRole(_ context.Context, data ports.CreateRoleData) (uint64, error) { stub.createData = data; return 7, nil }
func (stub *roleRepositoryStub) UpdateRole(_ context.Context, _ uint64, data ports.UpdateRoleData) ([]uint64, error) { stub.updateData = data; return nil, nil }
func (stub *roleRepositoryStub) DeleteRole(context.Context, uint64) error { return nil }
func (stub *roleRepositoryStub) ListPermissions(context.Context) ([]ports.PermissionRecord, error) { return nil, nil }

func TestRoleManagementCreateNormalizesAggregate(t *testing.T) {
	repository := &roleRepositoryStub{}
	service := NewRoleManagement(repository, &AuthService{})
	id, err := service.Create(context.Background(), CreateRoleRequest{Name: " operator ", Code: " ops ", Desc: " team ", Permissions: []string{"namespace:read", "namespace:read"}, NamespaceScope: &NamespaceScopeRequest{ClusterID: 3, Namespaces: []string{" prod ", "prod"}}})
	if err != nil { t.Fatalf("Create() error = %v", err) }
	if id != 7 || repository.createData.Name != "operator" || repository.createData.Code != "ops" { t.Fatalf("Create() did not normalize identity: %#v", repository.createData) }
	if repository.createData.Scope == nil || len(repository.createData.Scope.Namespaces) != 1 || repository.createData.Scope.Namespaces[0] != "prod" { t.Fatalf("Create() scope = %#v", repository.createData.Scope) }
}

func TestRoleManagementRejectsScopeWithoutNamespacePermission(t *testing.T) {
	service := NewRoleManagement(&roleRepositoryStub{}, &AuthService{})
	_, err := service.Create(context.Background(), CreateRoleRequest{Name: "operator", Code: "ops", Permissions: []string{"cluster:read"}, NamespaceScope: &NamespaceScopeRequest{ClusterID: 3, Namespaces: []string{"prod"}}})
	if !errors.Is(err, domain.ErrInvalidParams) { t.Fatalf("Create() error = %v, want invalid params", err) }
}

func TestRoleManagementUpdatePreservesScopeForRepositoryValidation(t *testing.T) {
	repository := &roleRepositoryStub{}
	service := NewRoleManagement(repository, &AuthService{})
	err := service.Update(context.Background(), 9, UpdateRoleRequest{NamespaceScope: &NamespaceScopeRequest{ClusterID: 2, Namespaces: []string{"staging"}}})
	if err != nil { t.Fatalf("Update() error = %v", err) }
	if !repository.updateData.SetScope || repository.updateData.Scope == nil || repository.updateData.Scope.ClusterID != 2 { t.Fatalf("Update() scope = %#v", repository.updateData) }
}
