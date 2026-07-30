package router

import (
	"context"
	"errors"

	orchestrationprovision "k8s-platform-backend/internal/orchestration/provisioning"
	platformapp "k8s-platform-backend/internal/platform/application"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
	provisionmemory "k8s-platform-backend/internal/provisioning/adapters/memory"
	provisionmysql "k8s-platform-backend/internal/provisioning/adapters/mysql"
	provisionssh "k8s-platform-backend/internal/provisioning/adapters/ssh"
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
	sshRuntime := provisionssh.NewSSHRuntime(d.DB, d.EncryptionKey)
	terminalSessions := provisionmemory.NewTerminalSessionStore(0)
	preflightRuntime := orchestrationprovision.NewPreflightRuntime(d.DB, d.EncryptionKey)
	ansibleRunner := orchestrationprovision.NewAnsibleRunner(d.DB, d.EncryptionKey, runtime.taskStore)
	deploymentExecutor := orchestrationprovision.NewDeploymentExecutor(provisionRepository, runtime.taskStore, runtime.clusterRegistry, preflightRuntime, ansibleRunner)
	deploymentRuntime := orchestrationprovision.NewRuntime(d.DB, deploymentExecutor, preflightRuntime)
	_ = appTemplateService.SeedBuiltinAppTemplates(context.Background())
	return provisioningModule{
		serverAccess: provisionhttp.NewServerAccessController(sshRuntime, terminalSessions, serverService),
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
