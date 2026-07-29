package application

import (
	"context"
	"testing"
	"time"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
)

func TestDeployPlanToItemPreservesExecutionConfiguration(t *testing.T) {
	row := provisiondomain.DeployPlan{
		ID:          42,
		Name:        "platform-cluster",
		ClusterName: "platform-k8s",
		HelmInstall: true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	item := deployPlanToItem(row, nil)
	if !item.HelmInstall {
		t.Fatal("helm_install should be returned with deploy plan detail")
	}
}

type runtimeSpy struct{ called bool }

func (r *runtimeSpy) DryRun(context.Context, uint64) (any, error)                    { r.called = true; return "ok", nil }
func (r *runtimeSpy) Preflight(context.Context, uint64) (any, error)                 { return nil, nil }
func (r *runtimeSpy) SetPreflightIgnore(context.Context, uint64, string, bool) error { return nil }
func (r *runtimeSpy) AnsibleConfig(context.Context, uint64) (any, error)             { return nil, nil }
func (r *runtimeSpy) Execute(context.Context, uint64, uint64) (uint64, error)        { return 0, nil }
func (r *runtimeSpy) Cancel(context.Context, uint64) error                           { return nil }
func (r *runtimeSpy) Retry(context.Context, uint64, uint64) (uint64, error)          { return 0, nil }
func (r *runtimeSpy) RetryStep(context.Context, uint64, string, uint64) (uint64, error) {
	return 0, nil
}
func (r *runtimeSpy) InstallAddons(context.Context, uint64, []string, uint64) (uint64, error) {
	return 0, nil
}
func (r *runtimeSpy) LatestAddonTask(context.Context, uint64) (any, error)        { return nil, nil }
func (r *runtimeSpy) RetryAddons(context.Context, uint64, uint64) (uint64, error) { return 0, nil }

func TestRuntimeServiceRejectsInvalidPlanBeforeRuntime(t *testing.T) {
	spy := &runtimeSpy{}
	service := NewRuntimeService(spy)
	if _, err := service.DryRun(context.Background(), 0); err == nil || spy.called {
		t.Fatalf("DryRun invalid plan = (%t, %v), want no runtime call and validation error", spy.called, err)
	}
	if value, err := service.DryRun(context.Background(), 1); err != nil || value != "ok" || !spy.called {
		t.Fatalf("DryRun valid plan = (%v, %v), called=%t", value, err, spy.called)
	}
}

func TestNormalizedPlanRetainsNodeOrder(t *testing.T) {
	plan, nodes, err := normalizedPlan(CreateDeployPlanRequest{
		Name: "platform", ClusterName: "platform-k8s", K8sVersion: "v1.31.0",
		Nodes: []DeployPlanNodeRequest{{ServerID: 1, Role: "master", SortOrder: 10}, {ServerID: 2, Role: "worker", SortOrder: 20}},
	}, 9)
	if err != nil {
		t.Fatalf("normalizedPlan() error = %v", err)
	}
	if plan.CreatedBy != 9 || len(nodes) != 2 || nodes[0].SortOrder != 10 || nodes[1].SortOrder != 20 {
		t.Fatalf("normalized plan = %#v, nodes = %#v", plan, nodes)
	}
}

func TestProvisioningSecretIsEncryptedAndPortIsNormalized(t *testing.T) {
	ciphertext, err := encryptProvisioningSecret("test-key", "plain-secret")
	if err != nil {
		t.Fatalf("encryptProvisioningSecret() error = %v", err)
	}
	if ciphertext == "plain-secret" || ciphertext == "" {
		t.Fatalf("encrypted secret = %q", ciphertext)
	}
	port, err := validSSHPort(0)
	if err != nil || port != 22 {
		t.Fatalf("validSSHPort(0) = (%d, %v), want (22, nil)", port, err)
	}
	if _, err := validSSHPort(65536); err == nil {
		t.Fatal("validSSHPort should reject out-of-range ports")
	}
}
