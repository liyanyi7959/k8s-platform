package kubernetes

import (
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s-platform-backend/internal/fleet/domain"
)

func TestFleetTransportErrorPreservesFleetVocabulary(t *testing.T) {
	if err := fleetTransportError(apierrors.NewForbidden(schema.GroupResource{Resource: "nodes"}, "worker-01", nil)); !errors.Is(err, domain.ErrRuntimeForbidden) {
		t.Fatalf("forbidden error = %v", err)
	}
	if _, err := normalizeFleetKubeconfig(""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("validation error = %v", err)
	}
}
