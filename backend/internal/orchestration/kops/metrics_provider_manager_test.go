package kops

import (
	"context"
	"errors"
	"testing"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

func TestMetricsProviderManagerRejectsUnknownSource(t *testing.T) {
	manager := NewMetricsProviderManager(nil, nil)
	err := manager.SwitchProvider(context.Background(), 1, kopsapp.MonitorSource("invalid"))
	if !errors.Is(err, kopsapp.ErrInvalidParams) {
		t.Fatalf("SwitchProvider invalid error = %v, want invalid params", err)
	}
}
