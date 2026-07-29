package kops

import (
	"context"
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

type permissionAuditCredentialsStub struct{}

func (permissionAuditCredentialsStub) Put(context.Context, uint64, string, time.Duration) error {
	return nil
}
func (permissionAuditCredentialsStub) Get(context.Context, uint64) (string, bool, error) {
	return "", false, nil
}
func (permissionAuditCredentialsStub) Delete(context.Context, uint64) {}

func TestPermissionAuditEngineKeepsCredentialsInjectable(t *testing.T) {
	credentials := permissionAuditCredentialsStub{}
	engine := NewPermissionAuditEngine(nil, nil, nil, nil, credentials, 0)
	if engine == nil || engine.creds == nil {
		t.Fatal("engine did not retain injected credential store")
	}
	if engine.credentialTTL != 2*time.Hour {
		t.Fatalf("credential TTL = %s, want 2h", engine.credentialTTL)
	}
}

func TestPermissionAuditKubeconfigErrorsStayAtKopsBoundary(t *testing.T) {
	if err := permissionAuditKubeconfigError(kopsclient.ErrKubeconfigEmpty); !errors.Is(err, kopsapp.ErrInvalidParams) {
		t.Fatalf("empty credential error = %v, want invalid params", err)
	}
}

func TestPermissionAuditCandidateGVRsIncludeLegacyIngressVersion(t *testing.T) {
	candidates := candidateGVRs(schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"})
	for _, candidate := range candidates {
		if candidate.Group == "extensions" && candidate.Version == "v1beta1" && candidate.Resource == "ingresses" {
			return
		}
	}
	t.Fatalf("ingress candidates = %#v, want extensions/v1beta1 fallback", candidates)
}
