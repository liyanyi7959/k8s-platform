package application

import (
	"context"
	"strings"
)

// DeploymentRuntime is the anti-corruption port for Ansible/task execution.
// Its current implementation lives in the legacy runtime adapter; callers in
// the provisioning HTTP layer never depend on that implementation directly.
type DeploymentRuntime interface {
	DryRun(context.Context, uint64) (any, error)
	Preflight(context.Context, uint64) (any, error)
	SetPreflightIgnore(context.Context, uint64, string, bool) error
	AnsibleConfig(context.Context, uint64) (any, error)
	Execute(context.Context, uint64, uint64) (uint64, error)
	Cancel(context.Context, uint64) error
	Retry(context.Context, uint64, uint64) (uint64, error)
	RetryStep(context.Context, uint64, string, uint64) (uint64, error)
	InstallAddons(context.Context, uint64, []string, uint64) (uint64, error)
	LatestAddonTask(context.Context, uint64) (any, error)
	RetryAddons(context.Context, uint64, uint64) (uint64, error)
}

type RuntimeService struct{ runtime DeploymentRuntime }

func NewRuntimeService(runtime DeploymentRuntime) *RuntimeService {
	return &RuntimeService{runtime: runtime}
}

func (s *RuntimeService) DryRun(ctx context.Context, planID uint64) (any, error) {
	if planID == 0 {
		return nil, ErrInvalidParams
	}
	return s.runtime.DryRun(ctx, planID)
}

func (s *RuntimeService) Preflight(ctx context.Context, planID uint64) (any, error) {
	if planID == 0 {
		return nil, ErrInvalidParams
	}
	return s.runtime.Preflight(ctx, planID)
}

func (s *RuntimeService) SetPreflightIgnore(ctx context.Context, planID uint64, key string, ignored bool) error {
	if planID == 0 || strings.TrimSpace(key) == "" {
		return ErrInvalidParams
	}
	return s.runtime.SetPreflightIgnore(ctx, planID, strings.TrimSpace(key), ignored)
}

func (s *RuntimeService) AnsibleConfig(ctx context.Context, planID uint64) (any, error) {
	if planID == 0 {
		return nil, ErrInvalidParams
	}
	return s.runtime.AnsibleConfig(ctx, planID)
}

func (s *RuntimeService) Execute(ctx context.Context, planID, userID uint64) (uint64, error) {
	if planID == 0 {
		return 0, ErrInvalidParams
	}
	return s.runtime.Execute(ctx, planID, userID)
}

func (s *RuntimeService) Cancel(ctx context.Context, planID uint64) error {
	if planID == 0 {
		return ErrInvalidParams
	}
	return s.runtime.Cancel(ctx, planID)
}

func (s *RuntimeService) Retry(ctx context.Context, planID, userID uint64) (uint64, error) {
	if planID == 0 {
		return 0, ErrInvalidParams
	}
	return s.runtime.Retry(ctx, planID, userID)
}

func (s *RuntimeService) RetryStep(ctx context.Context, planID uint64, stepKey string, userID uint64) (uint64, error) {
	if planID == 0 || strings.TrimSpace(stepKey) == "" {
		return 0, ErrInvalidParams
	}
	return s.runtime.RetryStep(ctx, planID, strings.TrimSpace(stepKey), userID)
}

func (s *RuntimeService) InstallAddons(ctx context.Context, planID uint64, addons []string, userID uint64) (uint64, error) {
	if planID == 0 || len(addons) == 0 {
		return 0, ErrInvalidParams
	}
	return s.runtime.InstallAddons(ctx, planID, addons, userID)
}

func (s *RuntimeService) LatestAddonTask(ctx context.Context, planID uint64) (any, error) {
	if planID == 0 {
		return nil, ErrInvalidParams
	}
	return s.runtime.LatestAddonTask(ctx, planID)
}

func (s *RuntimeService) RetryAddons(ctx context.Context, planID, userID uint64) (uint64, error) {
	if planID == 0 {
		return 0, ErrInvalidParams
	}
	return s.runtime.RetryAddons(ctx, planID, userID)
}
