package provisioning

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	platformapp "k8s-platform-backend/internal/platform/application"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
)

// Runtime adapts Provisioning persistence and execution to the application
// runtime port. DeploymentExecutor owns the asynchronous plan state machine.
type Runtime struct {
	db        *gorm.DB
	executor  *DeploymentExecutor
	preflight *PreflightRuntime
}

func NewRuntime(db *gorm.DB, executor *DeploymentExecutor, preflight *PreflightRuntime) *Runtime {
	return &Runtime{db: db, executor: executor, preflight: preflight}
}

func (r *Runtime) DryRun(ctx context.Context, id uint64) (any, error) {
	if id == 0 {
		return nil, provisionapp.ErrInvalidParams
	}
	if r == nil || r.db == nil {
		return nil, provisionapp.ErrConflict
	}

	var plan model.DeployPlan
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, provisionapp.ErrNotFound
		}
		return nil, err
	}
	var nodes []model.DeployPlanNode
	if err := r.db.WithContext(ctx).Where("plan_id = ?", id).Order("sort_order asc, id asc").Find(&nodes).Error; err != nil {
		return nil, err
	}

	serverIDs := make([]uint64, 0, len(nodes))
	for _, node := range nodes {
		serverIDs = append(serverIDs, node.ServerID)
	}
	var servers []model.DeployServer
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id IN ?", serverIDs).Find(&servers).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return provisionapp.BuildDryRun(dryRunPlanInput(plan, nodes, servers))
}

func dryRunPlanInput(plan model.DeployPlan, nodes []model.DeployPlanNode, servers []model.DeployServer) provisionapp.DryRunPlanInput {
	serverByID := make(map[uint64]model.DeployServer, len(servers))
	for _, server := range servers {
		serverByID[server.ID] = server
	}
	input := provisionapp.DryRunPlanInput{
		PlanID: plan.ID, PlanName: plan.Name, ClusterName: plan.ClusterName, K8sVersion: plan.K8sVersion, CNIType: plan.CNIType,
		Nodes: make([]provisionapp.DryRunNodeInput, 0, len(nodes)),
	}
	for _, node := range nodes {
		server := serverByID[node.ServerID]
		input.Nodes = append(input.Nodes, provisionapp.DryRunNodeInput{
			ServerID: node.ServerID, ServerName: server.Name, IP: server.IP, Role: node.Role, SortOrder: node.SortOrder,
		})
	}
	return input
}
func (r *Runtime) Preflight(ctx context.Context, id uint64) (any, error) {
	if r == nil || r.preflight == nil {
		return nil, provisionapp.ErrConflict
	}
	return r.preflight.Preflight(ctx, id)
}
func (r *Runtime) SetPreflightIgnore(ctx context.Context, id uint64, key string, ignored bool) error {
	if r == nil || r.preflight == nil {
		return provisionapp.ErrConflict
	}
	return r.preflight.SetPreflightIgnore(ctx, id, key, ignored)
}
func (r *Runtime) AnsibleConfig(ctx context.Context, id uint64) (any, error) {
	if id == 0 {
		return nil, provisionapp.ErrInvalidParams
	}
	if r == nil || r.db == nil {
		return nil, provisionapp.ErrConflict
	}
	plan, nodes, err := r.planWithNodes(ctx, id)
	if err != nil {
		return nil, err
	}
	inventory, err := r.maskedInventory(ctx, nodes)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"playbook_path": provisioningPlaybookPath(), "inventory": inventory,
		"extra_vars": map[string]any{
			"k8s_version": plan.K8sVersion, "k8s_minor_version": provisionapp.KubernetesMinorVersion(plan.K8sVersion),
			"pod_cidr": plan.PodCIDR, "svc_cidr": plan.SvcCIDR, "cni_type": plan.CNIType, "cluster_name": plan.ClusterName,
		},
	}, nil
}

func (r *Runtime) planWithNodes(ctx context.Context, id uint64) (model.DeployPlan, []model.DeployPlanNode, error) {
	var plan model.DeployPlan
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DeployPlan{}, nil, provisionapp.ErrNotFound
		}
		return model.DeployPlan{}, nil, err
	}
	var nodes []model.DeployPlanNode
	if err := r.db.WithContext(ctx).Where("plan_id = ?", id).Order("sort_order asc, id asc").Find(&nodes).Error; err != nil {
		return model.DeployPlan{}, nil, err
	}
	return plan, nodes, nil
}

func (r *Runtime) maskedInventory(ctx context.Context, nodes []model.DeployPlanNode) (string, error) {
	masters, workers := make([]provisionapp.InventoryHost, 0), make([]provisionapp.InventoryHost, 0)
	masterIndex, workerIndex := 0, 0
	for _, node := range nodes {
		var server model.DeployServer
		if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", node.ServerID).First(&server).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", provisionapp.ErrWithMessage(provisionapp.ErrNotFound, fmt.Sprintf("部署节点关联的服务器 %d 不存在", node.ServerID))
			}
			return "", err
		}
		if server.CredentialID != nil && *server.CredentialID > 0 {
			var credential model.SSHCredential
			if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", *server.CredentialID).First(&credential).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return "", provisionapp.ErrWithMessage(provisionapp.ErrNotFound, "部署节点关联的 SSH 凭据不存在")
				}
				return "", err
			}
			server.AuthType = credential.AuthType
		}
		role := strings.TrimSpace(node.Role)
		index := workerIndex + 1
		if role == "master" {
			masterIndex++
			index = masterIndex
		} else {
			workerIndex++
		}
		host := provisionapp.InventoryHost{Alias: provisionapp.InventoryNodeAlias(role, index, server.IP), IP: server.IP, SSHPort: server.SSHPort, User: server.User, AuthType: server.AuthType}
		if server.AuthType == "key" {
			host.KeyFile = "~/.ssh/id_rsa"
		} else {
			host.Password = "***"
		}
		if role == "master" {
			masters = append(masters, host)
		} else {
			workers = append(workers, host)
		}
	}
	return provisionapp.MarshalAnsibleInventory(masters, workers, true)
}

func provisioningPlaybookPath() string {
	candidates := []string{"ansible", filepath.Join("backend", "ansible")}
	if executable, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(executable), "ansible")}, candidates...)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Join(candidate, "site.yml")
		}
	}
	return filepath.Join("ansible", "site.yml")
}
func (r *Runtime) Execute(ctx context.Context, id, userID uint64) (uint64, error) {
	if r == nil || r.executor == nil {
		return 0, provisionapp.ErrConflict
	}
	return r.executor.Execute(ctx, id, userID)
}
func (r *Runtime) Cancel(ctx context.Context, id uint64) error {
	if r == nil || r.executor == nil {
		return provisionapp.ErrConflict
	}
	return r.executor.Cancel(ctx, id)
}
func (r *Runtime) Retry(ctx context.Context, id, userID uint64) (uint64, error) {
	if r == nil || r.executor == nil {
		return 0, provisionapp.ErrConflict
	}
	return r.executor.Retry(ctx, id, userID)
}
func (r *Runtime) RetryStep(ctx context.Context, id uint64, step string, userID uint64) (uint64, error) {
	if r == nil || r.executor == nil {
		return 0, provisionapp.ErrConflict
	}
	return r.executor.RetryStep(ctx, id, step, userID)
}
func (r *Runtime) InstallAddons(ctx context.Context, id uint64, addons []string, userID uint64) (uint64, error) {
	if r == nil || r.executor == nil {
		return 0, provisionapp.ErrConflict
	}
	return r.executor.InstallAddons(ctx, id, addons, userID)
}
func (r *Runtime) LatestAddonTask(ctx context.Context, id uint64) (any, error) {
	if r == nil || r.executor == nil {
		return nil, provisionapp.ErrConflict
	}
	return r.executor.LatestAddonTask(ctx, id)
}
func (r *Runtime) RetryAddons(ctx context.Context, id, userID uint64) (uint64, error) {
	if r == nil || r.executor == nil {
		return 0, provisionapp.ErrConflict
	}
	return r.executor.RetryAddons(ctx, id, userID)
}

func (r *Runtime) GetTask(ctx context.Context, id int64) (provisionapp.DeploymentTask, error) {
	if r == nil || r.executor == nil {
		return provisionapp.DeploymentTask{}, provisionapp.ErrConflict
	}
	task, ok := r.executor.GetTask(id)
	if !ok {
		return provisionapp.DeploymentTask{}, provisionapp.ErrNotFound
	}
	return deploymentTask(task), nil
}

func (r *Runtime) TaskLogs(ctx context.Context, id int64, offset, limit int, stepKey string) ([]provisionapp.DeploymentTaskLog, error) {
	if r == nil || r.executor == nil {
		return nil, provisionapp.ErrConflict
	}
	entries, ok := r.executor.TaskLogs(id, offset, limit, stepKey)
	if !ok {
		return nil, provisionapp.ErrNotFound
	}
	logs := make([]provisionapp.DeploymentTaskLog, 0, len(entries))
	for _, entry := range entries {
		logs = append(logs, provisionapp.DeploymentTaskLog{Content: entry.Content, CreatedAt: entry.CreatedAt})
	}
	return logs, nil
}

func deploymentTask(task *platformapp.Task) provisionapp.DeploymentTask {
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
