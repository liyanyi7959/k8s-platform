package router

import (
	"context"
	"errors"

	legacyprovision "k8s-platform-backend/internal/integration/provisioning"
	platformapp "k8s-platform-backend/internal/platform/application"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
	provisionmysql "k8s-platform-backend/internal/provisioning/adapters/mysql"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
)

// automationTaskRuntime bridges the platform-wide task centre into the
// Provisioning automation use case at the composition root.
type automationTaskRuntime struct{ tasks *platformapp.TaskService }

func (runtime automationTaskRuntime) ListAutomationTasks(request provisionapp.AutomationTaskListRequest) (any, error) {
	if runtime.tasks == nil {
		return nil, provisionapp.ErrConflict
	}
	return runtime.tasks.List(platformapp.ListTasksRequest{
		Page: request.Page, PageSize: request.PageSize, Type: request.Type, Status: request.Status,
		Keyword: request.Keyword, SortBy: request.SortBy, Order: request.Order,
	}), nil
}

func (runtime automationTaskRuntime) GetAutomationTask(taskID int64) (any, error) {
	if runtime.tasks == nil {
		return nil, provisionapp.ErrConflict
	}
	task, found := runtime.tasks.Get(taskID)
	if !found {
		return nil, provisionapp.ErrNotFound
	}
	return task, nil
}

func (runtime automationTaskRuntime) AutomationTaskLogs(taskID int64, offset, limit int) ([]string, error) {
	if runtime.tasks == nil {
		return nil, provisionapp.ErrConflict
	}
	logs, found := runtime.tasks.Logs(taskID, offset, limit)
	if !found {
		return nil, provisionapp.ErrNotFound
	}
	return logs, nil
}

func (runtime automationTaskRuntime) CancelAutomationTask(taskID int64) error {
	if runtime.tasks == nil {
		return provisionapp.ErrConflict
	}
	err := runtime.tasks.Cancel(taskID)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, platformapp.ErrTaskNotFound):
		return provisionapp.ErrNotFound
	case errors.Is(err, platformapp.ErrTaskCannotCancel):
		return provisionapp.ErrConflict
	default:
		return err
	}
}

func buildProvisioningModule(d Deps, runtime moduleRuntime) provisioningModule {
	provisionRepository := provisionmysql.NewRepository(d.DB)
	appTemplateService := provisionapp.NewAppTemplateService(provisionRepository)
	serverService := provisionapp.NewServerService(provisionRepository, d.EncryptionKey)
	credentialService := provisionapp.NewCredentialService(provisionRepository, d.EncryptionKey)
	sshRuntime := legacyprovision.NewSSHRuntime(d.DB, d.EncryptionKey)
	preflightRuntime := legacyprovision.NewPreflightRuntime(d.DB, d.EncryptionKey)
	ansibleRunner := legacyprovision.NewAnsibleRunner(d.DB, d.EncryptionKey, runtime.taskStore)
	deploymentExecutor := legacyprovision.NewDeploymentExecutor(d.DB, runtime.taskStore, runtime.clusterRegistry, preflightRuntime, ansibleRunner)
	deploymentRuntime := legacyprovision.NewRuntime(d.DB, deploymentExecutor, preflightRuntime)
	_ = appTemplateService.SeedBuiltinAppTemplates(context.Background())
	return provisioningModule{
		serverAccess: legacyprovision.NewServerAccessController(sshRuntime, runtime.execSessions, serverService),
		servers:      provisionhttp.NewServerController(serverService),
		credentials:  provisionhttp.NewCredentialController(credentialService),
		plans:        provisionhttp.NewDeployPlanController(provisionapp.NewDeployPlanService(provisionRepository)),
		runtime:      provisionhttp.NewRuntimeController(provisionapp.NewRuntimeService(deploymentRuntime)),
		tasks:        provisionhttp.NewTaskController(provisionapp.NewTaskService(deploymentRuntime)),
		config:       provisionhttp.NewDeployConfigController(provisionapp.NewDeployConfigService(provisionRepository)),
		automation:   provisionhttp.NewAutomationTaskController(provisionapp.NewAutomationTaskService(automationTaskRuntime{tasks: platformapp.NewTaskService(runtime.taskStore)})),
		appTemplate:  provisionhttp.NewAppTemplateController(appTemplateService),
	}
}
