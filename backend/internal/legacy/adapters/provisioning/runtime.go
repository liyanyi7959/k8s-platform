package provisioning

import (
	"context"
	"errors"

	"k8s-platform-backend/internal/legacy/service"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
)

// Runtime adapts the retained Ansible executor to the provisioning runtime
// port. Error translation keeps the new HTTP adapter API-compatible.
type Runtime struct{ deploy *service.DeployService }

func NewRuntime(deploy *service.DeployService) *Runtime { return &Runtime{deploy: deploy} }

func (r *Runtime) DryRun(ctx context.Context, id uint64) (any, error) {
	value, err := r.deploy.DryRunPlan(ctx, id)
	return value, translate(err)
}
func (r *Runtime) Preflight(ctx context.Context, id uint64) (any, error) {
	value, err := r.deploy.PreflightPlan(ctx, id)
	return value, translate(err)
}
func (r *Runtime) SetPreflightIgnore(ctx context.Context, id uint64, key string, ignored bool) error {
	return translate(r.deploy.SetPreflightIgnore(ctx, id, key, ignored))
}
func (r *Runtime) AnsibleConfig(ctx context.Context, id uint64) (any, error) {
	value, err := r.deploy.GetPlanAnsibleConfig(ctx, id)
	return value, translate(err)
}
func (r *Runtime) Execute(ctx context.Context, id, userID uint64) (uint64, error) {
	value, err := r.deploy.ExecutePlan(ctx, id, userID)
	return value, translate(err)
}
func (r *Runtime) Cancel(ctx context.Context, id uint64) error {
	return translate(r.deploy.CancelPlan(ctx, id))
}
func (r *Runtime) Retry(ctx context.Context, id, userID uint64) (uint64, error) {
	value, err := r.deploy.RetryPlan(ctx, id, userID)
	return value, translate(err)
}
func (r *Runtime) RetryStep(ctx context.Context, id uint64, step string, userID uint64) (uint64, error) {
	value, err := r.deploy.RetryStep(ctx, id, step, userID)
	return value, translate(err)
}
func (r *Runtime) InstallAddons(ctx context.Context, id uint64, addons []string, userID uint64) (uint64, error) {
	value, err := r.deploy.InstallPlanAddons(ctx, id, addons, userID)
	return value, translate(err)
}
func (r *Runtime) LatestAddonTask(ctx context.Context, id uint64) (any, error) {
	value, err := r.deploy.GetLatestPlanAddonTask(ctx, id)
	return value, translate(err)
}
func (r *Runtime) RetryAddons(ctx context.Context, id, userID uint64) (uint64, error) {
	value, err := r.deploy.RetryPlanAddons(ctx, id, userID)
	return value, translate(err)
}

func (r *Runtime) GetTask(ctx context.Context, id int64) (provisionapp.DeploymentTask, error) {
	task, ok := r.deploy.GetTaskStore().Get(id)
	if !ok {
		return provisionapp.DeploymentTask{}, provisionapp.ErrNotFound
	}
	return deploymentTask(task), nil
}

func (r *Runtime) TaskLogs(ctx context.Context, id int64, offset, limit int, stepKey string) ([]provisionapp.DeploymentTaskLog, error) {
	task, ok := r.deploy.GetTaskStore().Get(id)
	if !ok {
		return nil, provisionapp.ErrNotFound
	}
	entries := task.LogEntries(offset, limit, stepKey)
	logs := make([]provisionapp.DeploymentTaskLog, 0, len(entries))
	for _, entry := range entries {
		logs = append(logs, provisionapp.DeploymentTaskLog{Content: entry.Content, CreatedAt: entry.CreatedAt})
	}
	return logs, nil
}

func deploymentTask(task *service.Task) provisionapp.DeploymentTask {
	result := provisionapp.DeploymentTask{
		ID: task.ID, Type: task.Type, Status: string(task.Status), Title: task.Title, CreatedAt: task.CreatedAt,
		CreatedBy: task.CreatedBy, Percent: task.Percent, Message: task.Message, Meta: task.Meta,
	}
	if task.Steps != nil {
		result.Steps = make([]provisionapp.DeploymentTaskStep, 0, len(task.Steps))
		for _, step := range task.Steps {
			item := provisionapp.DeploymentTaskStep{
				Key: step.Key, Title: step.Title, Status: string(step.Status), StartedAt: step.StartedAt,
				FinishedAt: step.FinishedAt, Message: step.Message,
			}
			for _, sub := range step.SubSteps {
				item.SubSteps = append(item.SubSteps, provisionapp.DeploymentTaskSubStep{
					Key: sub.Key, Title: sub.Title, Status: string(sub.Status), StartedAt: sub.StartedAt, FinishedAt: sub.FinishedAt,
				})
			}
			result.Steps = append(result.Steps, item)
		}
	}
	return result
}

func translate(err error) error {
	if err == nil {
		return nil
	}
	for _, candidate := range []struct{ legacy, target error }{
		{service.ErrInvalidParams, provisionapp.ErrInvalidParams}, {service.ErrNotFound, provisionapp.ErrNotFound}, {service.ErrConflict, provisionapp.ErrConflict},
	} {
		if errors.Is(err, candidate.legacy) {
			if message, ok := service.UserMessage(err); ok {
				return provisionapp.ErrWithMessage(candidate.target, message)
			}
			return candidate.target
		}
	}
	return err
}
