package application

import (
	"context"
	"errors"
	"testing"
)

type permissionAuditRuntimeSpy struct {
	managedInput PermissionAuditCreateInput
	managedID    uint64
	managedUser  uint64
	listQuery    PermissionAuditListQuery
	logsOffset   int
	logsLimit    int
	findings     PermissionAuditFindingsQuery
}

func (spy *permissionAuditRuntimeSpy) CreateManaged(_ context.Context, clusterID uint64, input PermissionAuditCreateInput, createdBy uint64) (any, error) {
	spy.managedID, spy.managedInput, spy.managedUser = clusterID, input, createdBy
	return map[string]uint64{"audit_id": 3}, nil
}
func (spy *permissionAuditRuntimeSpy) CreateAdhoc(context.Context, PermissionAuditAdhocInput, uint64) (any, error) {
	return nil, nil
}
func (spy *permissionAuditRuntimeSpy) List(_ context.Context, input PermissionAuditListQuery) (any, error) {
	spy.listQuery = input
	return input, nil
}
func (spy *permissionAuditRuntimeSpy) Get(context.Context, uint64) (any, error) {
	return map[string]any{}, nil
}
func (spy *permissionAuditRuntimeSpy) Logs(_ context.Context, _ uint64, offset, limit int) (any, error) {
	spy.logsOffset, spy.logsLimit = offset, limit
	return map[string]any{}, nil
}
func (spy *permissionAuditRuntimeSpy) Cancel(context.Context, uint64) error { return nil }
func (spy *permissionAuditRuntimeSpy) Compare(context.Context, uint64, uint64) (any, error) {
	return map[string]any{}, nil
}
func (spy *permissionAuditRuntimeSpy) ListFindings(_ context.Context, _ uint64, input PermissionAuditFindingsQuery) (any, error) {
	spy.findings = input
	return map[string]any{}, nil
}
func (spy *permissionAuditRuntimeSpy) LatestForCluster(context.Context, uint64) (any, error) {
	return map[string]any{}, nil
}
func (spy *permissionAuditRuntimeSpy) RecommendRBAC(context.Context, uint64, []string) any {
	return map[string]any{}
}

func TestPermissionAuditServiceNormalizesCommandsAndQueries(t *testing.T) {
	spy := &permissionAuditRuntimeSpy{}
	service := NewPermissionAuditService(spy)
	if _, err := service.CreateManaged(context.Background(), 8, PermissionAuditCreateInput{
		Mode: " FULL ", Namespaces: []string{" dev ", "dev", ""}, ResourceAllowlist: []string{"pods", "pods", " configmaps "},
	}, 21); err != nil {
		t.Fatalf("CreateManaged() error = %v", err)
	}
	if spy.managedID != 8 || spy.managedUser != 21 || spy.managedInput.Mode != "full" {
		t.Fatalf("managed input = %#v, cluster=%d user=%d", spy.managedInput, spy.managedID, spy.managedUser)
	}
	if len(spy.managedInput.Namespaces) != 1 || len(spy.managedInput.ResourceAllowlist) != 2 {
		t.Fatalf("normalized managed input = %#v", spy.managedInput)
	}
	if _, err := service.List(context.Background(), PermissionAuditListQuery{Page: 0, PageSize: 201, Keyword: " ops "}); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if spy.listQuery.Page != 1 || spy.listQuery.PageSize != 200 || spy.listQuery.Keyword != "ops" {
		t.Fatalf("list query = %#v", spy.listQuery)
	}
	if _, err := service.Logs(context.Background(), 7, -1, 2000); err != nil {
		t.Fatalf("Logs() error = %v", err)
	}
	if spy.logsOffset != 0 || spy.logsLimit != 200 {
		t.Fatalf("logs offset/limit = %d/%d", spy.logsOffset, spy.logsLimit)
	}
	if _, err := service.ListFindings(context.Background(), 7, PermissionAuditFindingsQuery{PageSize: 201, Namespace: " dev "}); err != nil {
		t.Fatalf("ListFindings() error = %v", err)
	}
	if spy.findings.Page != 1 || spy.findings.PageSize != 200 || spy.findings.Namespace != "dev" {
		t.Fatalf("findings query = %#v", spy.findings)
	}
	if _, err := service.CreateManaged(context.Background(), 0, PermissionAuditCreateInput{}, 0); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid managed audit error = %v", err)
	}
	if _, err := service.CreateAdhoc(context.Background(), PermissionAuditAdhocInput{}, 0); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid adhoc audit error = %v", err)
	}
}
