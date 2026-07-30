package provisioning

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	fleetapp "k8s-platform-backend/internal/fleet/application"
	platformapp "k8s-platform-backend/internal/platform/application"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
)

// DeploymentExecutor owns the operational state machine for a deployment
// plan. It is the Provisioning runtime's single owner for task persistence,
// readiness gating, Ansible execution, and the post-deployment fleet import.
//
// The platform task store and fleet registry are cross-context ports. Their
// concrete implementations are wired at the composition root; no legacy
// service coordinator participates in deployment execution.
type DeploymentExecutor struct {
	db              *gorm.DB
	taskStore       *platformapp.TaskStore
	clusterRegistry *fleetapp.Registry
	preflight       *PreflightRuntime
	ansibleRunner   *AnsibleRunner
}

func NewDeploymentExecutor(
	db *gorm.DB,
	taskStore *platformapp.TaskStore,
	clusterRegistry *fleetapp.Registry,
	preflight *PreflightRuntime,
	ansibleRunner *AnsibleRunner,
) *DeploymentExecutor {
	return &DeploymentExecutor{
		db:              db,
		taskStore:       taskStore,
		clusterRegistry: clusterRegistry,
		preflight:       preflight,
		ansibleRunner:   ansibleRunner,
	}
}

func (e *DeploymentExecutor) Execute(ctx context.Context, planID, userID uint64) (uint64, error) {
	return e.executeFromStep(ctx, planID, userID, "")
}

func (e *DeploymentExecutor) executeFromStep(ctx context.Context, planID, userID uint64, retryFromStep string) (uint64, error) {
	if planID == 0 {
		return 0, provisionapp.ErrInvalidParams
	}
	if err := e.readyForExecution(); err != nil {
		return 0, err
	}

	preflight, err := e.preflight.Preflight(ctx, planID)
	if err != nil {
		return 0, err
	}
	if !preflight.Ready {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrInvalidParams, preflightFailureMessage(preflight))
	}

	var taskID uint64
	err = e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var plan model.DeployPlan
		if err := tx.Where("deleted_at IS NULL AND id = ?", planID).First(&plan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return provisionapp.ErrNotFound
			}
			return err
		}
		if plan.Status != "draft" && plan.Status != "failed" && plan.Status != "cancelled" {
			return provisionapp.ErrWithMessage(provisionapp.ErrConflict, "当前状态不允许执行部署")
		}

		title := "部署集群 " + plan.ClusterName
		if retryFromStep != "" {
			title += "（从步骤重试）"
		}
		percent := 0
		message := "部署任务已创建，正在执行"
		meta := map[string]any{"deploy_plan_id": plan.ID}
		if retryFromStep != "" {
			meta["retry_from_step"] = retryFromStep
		}
		task := &platformapp.Task{
			Type:      "deploy_cluster",
			Status:    platformapp.TaskPending,
			Title:     &title,
			CreatedBy: int64(userID),
			Percent:   &percent,
			Message:   &message,
			Meta:      meta,
		}
		if err := e.taskStore.Put(task); err != nil {
			return err
		}
		taskID = uint64(task.ID)
		return tx.Model(&model.DeployPlan{}).Where("id = ?", planID).Updates(map[string]any{
			"status": "running", "task_id": taskID,
		}).Error
	})
	if err != nil {
		return 0, err
	}
	go e.ansiblePipeline(context.Background(), planID, int64(taskID))
	return taskID, nil
}

func (e *DeploymentExecutor) RetryStep(ctx context.Context, planID uint64, stepKey string, userID uint64) (uint64, error) {
	stepKey = strings.TrimSpace(stepKey)
	if planID == 0 || stepKey == "" {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrInvalidParams, "步骤 key 不能为空")
	}
	if !isKnownAnsibleStep(stepKey) {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrInvalidParams, "未知的步骤 key")
	}
	plan, _, err := e.planWithNodes(ctx, planID)
	if err != nil {
		return 0, err
	}
	if plan.Status != "failed" && plan.Status != "cancelled" {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrConflict, "仅失败或已取消的计划可重试步骤")
	}
	return e.executeFromStep(ctx, planID, userID, stepKey)
}

func (e *DeploymentExecutor) Cancel(ctx context.Context, planID uint64) error {
	if planID == 0 {
		return provisionapp.ErrInvalidParams
	}
	if e == nil || e.db == nil || e.taskStore == nil {
		return provisionapp.ErrConflict
	}
	var plan model.DeployPlan
	if err := e.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", planID).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return provisionapp.ErrNotFound
		}
		return err
	}
	if plan.Status != "running" {
		return provisionapp.ErrWithMessage(provisionapp.ErrConflict, "仅运行中的计划可取消")
	}
	if plan.TaskID != nil && *plan.TaskID > 0 {
		e.taskStore.CancelExecution(int64(*plan.TaskID))
	}
	return e.updatePlanStatus(ctx, planID, "running", "cancelled")
}

func (e *DeploymentExecutor) Retry(ctx context.Context, planID, userID uint64) (uint64, error) {
	if planID == 0 {
		return 0, provisionapp.ErrInvalidParams
	}
	if e == nil || e.db == nil || e.taskStore == nil {
		return 0, provisionapp.ErrConflict
	}
	var plan model.DeployPlan
	if err := e.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", planID).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, provisionapp.ErrNotFound
		}
		return 0, err
	}
	if plan.Status != "failed" && plan.Status != "cancelled" {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrConflict, "仅失败或已取消的计划可重试")
	}

	retryFromStep := ""
	if plan.TaskID != nil && *plan.TaskID > 0 {
		if task, ok := e.taskStore.Get(int64(*plan.TaskID)); ok {
			for _, step := range task.Steps {
				if step.Status == platformapp.StepFailed && isKnownAnsibleStep(step.Key) {
					retryFromStep = step.Key
					break
				}
			}
		}
	}
	return e.executeFromStep(ctx, planID, userID, retryFromStep)
}

func (e *DeploymentExecutor) InstallAddons(ctx context.Context, planID uint64, requested []string, userID uint64) (uint64, error) {
	if planID == 0 {
		return 0, provisionapp.ErrInvalidParams
	}
	if e == nil || e.db == nil || e.taskStore == nil {
		return 0, provisionapp.ErrConflict
	}
	addons, err := normalizeClusterAddons(requested)
	if err != nil {
		return 0, err
	}

	plan, _, err := e.planWithNodes(ctx, planID)
	if err != nil {
		return 0, err
	}
	if plan.Status != "success" || plan.ClusterID == nil || *plan.ClusterID == 0 {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrConflict, "仅已成功并已纳管的集群可以补充安装组件")
	}
	if installed := selectedInstalledAddons([]string(plan.Addons), plan.HelmInstall, addons); len(installed) > 0 {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrConflict, "以下组件已安装，不能重复安装："+strings.Join(installed, "、"))
	}
	for _, task := range e.taskStore.List() {
		if task.Type != "install_cluster_addons" || (task.Status != platformapp.TaskPending && task.Status != platformapp.TaskRunning) {
			continue
		}
		if fmt.Sprint(task.Meta["deploy_plan_id"]) == fmt.Sprint(planID) {
			return 0, provisionapp.ErrWithMessage(provisionapp.ErrConflict, "该集群已有附加组件安装任务正在执行，请等待其完成")
		}
	}

	title := "补充安装集群组件 " + plan.ClusterName
	if isHelmOnlyAddonSelection(addons) {
		title = "补充安装 Helm " + plan.ClusterName
	}
	message := "组件安装任务已创建，正在准备 Master Ansible Runner"
	percent := 0
	task := &platformapp.Task{
		Type:      "install_cluster_addons",
		Status:    platformapp.TaskPending,
		Title:     &title,
		CreatedBy: int64(userID),
		Percent:   &percent,
		Message:   &message,
		Meta: map[string]any{
			"deploy_plan_id": planID,
			"addons":         addons,
			"enabled_steps":  addonInstallSteps(addons),
		},
	}
	if err := e.taskStore.Put(task); err != nil {
		return 0, err
	}
	go e.addonInstallPipeline(context.Background(), int64(planID), task.ID, addons)
	return uint64(task.ID), nil
}

func (e *DeploymentExecutor) LatestAddonTask(ctx context.Context, planID uint64) (*platformapp.Task, error) {
	if planID == 0 {
		return nil, provisionapp.ErrInvalidParams
	}
	if e == nil || e.taskStore == nil {
		return nil, provisionapp.ErrConflict
	}
	if _, _, err := e.planWithNodes(ctx, planID); err != nil {
		return nil, err
	}
	var latest *platformapp.Task
	for _, task := range e.taskStore.List() {
		if task.Type != "install_cluster_addons" || fmt.Sprint(task.Meta["deploy_plan_id"]) != fmt.Sprint(planID) {
			continue
		}
		if latest == nil || task.ID > latest.ID {
			latest = task
		}
	}
	return latest, nil
}

func (e *DeploymentExecutor) RetryAddons(ctx context.Context, planID, userID uint64) (uint64, error) {
	task, err := e.LatestAddonTask(ctx, planID)
	if err != nil {
		return 0, err
	}
	if task == nil {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrNotFound, "未找到可重试的附加组件安装任务")
	}
	if task.Status != platformapp.TaskFailed && task.Status != platformapp.TaskCanceled {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrConflict, "当前附加组件任务无需重试")
	}
	addons := taskMetaStringSlice(task.Meta, "addons")
	if len(addons) == 0 {
		return 0, provisionapp.ErrWithMessage(provisionapp.ErrConflict, "原附加组件任务未记录安装组件，无法重试")
	}
	return e.InstallAddons(ctx, planID, addons, userID)
}

func (e *DeploymentExecutor) GetTask(taskID int64) (*platformapp.Task, bool) {
	if e == nil || e.taskStore == nil {
		return nil, false
	}
	return e.taskStore.Get(taskID)
}

func (e *DeploymentExecutor) TaskLogs(taskID int64, offset, limit int, stepKey string) ([]platformapp.TaskLogEntry, bool) {
	task, ok := e.GetTask(taskID)
	if !ok {
		return nil, false
	}
	return task.LogEntries(offset, limit, stepKey), true
}

func (e *DeploymentExecutor) readyForExecution() error {
	if e == nil || e.db == nil || e.taskStore == nil || e.preflight == nil {
		return provisionapp.ErrWithMessage(provisionapp.ErrConflict, "部署运行时未初始化")
	}
	return nil
}

func (e *DeploymentExecutor) planWithNodes(ctx context.Context, planID uint64) (model.DeployPlan, []model.DeployPlanNode, error) {
	if e == nil || e.db == nil {
		return model.DeployPlan{}, nil, provisionapp.ErrConflict
	}
	var plan model.DeployPlan
	if err := e.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", planID).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DeployPlan{}, nil, provisionapp.ErrNotFound
		}
		return model.DeployPlan{}, nil, err
	}
	var nodes []model.DeployPlanNode
	if err := e.db.WithContext(ctx).Where("plan_id = ?", planID).Order("sort_order asc, id asc").Find(&nodes).Error; err != nil {
		return model.DeployPlan{}, nil, err
	}
	return plan, nodes, nil
}

func (e *DeploymentExecutor) updatePlanStatus(ctx context.Context, planID uint64, from, to string) error {
	result := e.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("deleted_at IS NULL AND id = ? AND status = ?", planID, from).Update("status", to)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return provisionapp.ErrWithMessage(provisionapp.ErrConflict, "部署计划状态不允许执行该操作")
	}
	return nil
}

func (e *DeploymentExecutor) ansiblePipeline(ctx context.Context, planID uint64, taskID int64) {
	task, ok := e.GetTask(taskID)
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	e.taskStore.RegisterCancel(taskID, cancel)
	defer e.taskStore.UnregisterCancel(taskID)
	defer cancel()

	task.Status = platformapp.TaskRunning
	message := "正在执行 Ansible 部署流水线"
	task.Message = &message
	_ = e.taskStore.Put(task)

	e.initializeAnsibleSteps(task)
	plan, nodes, err := e.planWithNodes(ctx, planID)
	if err != nil {
		task.AppendLog(fmt.Sprintf("[error] 获取部署计划失败: %v", err), activeTaskStepKey(task))
		e.markTaskFailed(task, fmt.Sprintf("获取部署计划失败: %v", err))
		e.updatePlanStatusDirect(ctx, planID, "failed")
		return
	}
	if ctx.Err() != nil {
		e.markTaskCanceled(task)
		e.updatePlanStatusDirect(ctx, planID, "cancelled")
		return
	}

	stepKey := activeTaskStepKey(task)
	task.AppendLog("[info] 正在准备 Master 临时 Runner", stepKey)
	task.AppendLog(fmt.Sprintf("[info] 集群: %s, K8s 版本: %s, CNI: %s", plan.ClusterName, plan.K8sVersion, plan.CNIType), stepKey)
	_ = e.taskStore.Put(task)

	var kubeconfig string
	if e.ansibleRunner == nil {
		err = fmt.Errorf("Ansible Runner 未注入")
	} else {
		kubeconfig, err = e.ansibleRunner.Run(ctx, plan, nodes, task)
	}
	if ctx.Err() != nil {
		e.markTaskCanceled(task)
		e.updatePlanStatusDirect(ctx, planID, "cancelled")
		return
	}
	if err != nil {
		for index := range task.Steps {
			if task.Steps[index].Status == platformapp.StepRunning {
				task.Steps[index].Status = platformapp.StepFailed
				errMessage := err.Error()
				task.Steps[index].Message = &errMessage
			}
		}
		e.markTaskFailed(task, fmt.Sprintf("部署失败: %v", err))
		e.updatePlanStatusDirect(ctx, planID, "failed")
		return
	}

	completedAt := time.Now().UTC()
	for index := range task.Steps {
		if task.Steps[index].Status != platformapp.StepSuccess {
			task.Steps[index].Status = platformapp.StepSuccess
		}
		finishUnresolvedSubSteps(&task.Steps[index], completedAt)
	}
	_ = e.taskStore.Put(task)

	clusterID, registerErr := e.registerClusterAfterDeploy(ctx, plan, kubeconfig)
	if registerErr != nil {
		if len(task.Steps) > 0 {
			last := len(task.Steps) - 1
			task.Steps[last].Status = platformapp.StepFailed
			stepMessage := fmt.Sprintf("集群已安装但注册平台失败: %v", registerErr)
			task.Steps[last].Message = &stepMessage
		}
		e.markTaskFailed(task, fmt.Sprintf("集群已安装但注册平台失败: %v，请检查日志后重试注册流程", registerErr))
		e.updatePlanStatusDirect(ctx, planID, "failed")
		return
	}
	task.AppendLog(fmt.Sprintf("[info] 集群 %s 注册成功，ID: %d", plan.ClusterName, clusterID), "")
	percent := 100
	task.Percent = &percent
	task.Status = platformapp.TaskSuccess
	message = "集群部署成功"
	task.Message = &message
	_ = e.taskStore.Put(task)
	e.updatePlanStatusDirect(ctx, planID, "success")
}

func (e *DeploymentExecutor) initializeAnsibleSteps(task *platformapp.Task) {
	if task == nil {
		return
	}
	steps := provisionapp.DefaultAnsibleSteps()
	task.Steps = make([]platformapp.TaskStep, len(steps))
	retryFromStep, _ := task.Meta["retry_from_step"].(string)
	startIndex := 0
	for index, step := range steps {
		if retryFromStep == step.Key {
			startIndex = index
			break
		}
	}
	now := time.Now().UTC()
	for index, step := range steps {
		status := platformapp.StepPending
		if retryFromStep != "" && index < startIndex {
			status = platformapp.StepSuccess
		}
		task.Steps[index] = platformapp.TaskStep{Key: step.Key, Title: step.Title, Status: status}
		if status == platformapp.StepSuccess {
			task.Steps[index].StartedAt = &now
			task.Steps[index].FinishedAt = &now
		}
	}
	percent := 0
	task.Percent = &percent
	for index := range task.Steps {
		if task.Steps[index].Status == platformapp.StepSuccess {
			continue
		}
		task.Steps[index].Status = platformapp.StepRunning
		task.Steps[index].StartedAt = &now
		break
	}
	_ = e.taskStore.Put(task)
}

func (e *DeploymentExecutor) registerClusterAfterDeploy(ctx context.Context, plan model.DeployPlan, kubeconfig string) (uint64, error) {
	if strings.TrimSpace(kubeconfig) == "" {
		return 0, fmt.Errorf("kubeconfig 内容为空")
	}
	if e.clusterRegistry == nil {
		return 0, fmt.Errorf("集群注册运行时未初始化")
	}
	clusterID, err := e.clusterRegistry.Import(ctx, plan.ClusterName, kubeconfig, "")
	if err != nil {
		return 0, fmt.Errorf("注册集群失败: %w", err)
	}
	_ = e.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("id = ?", plan.ID).Update("cluster_id", clusterID).Error
	return clusterID, nil
}

func (e *DeploymentExecutor) addonInstallPipeline(ctx context.Context, planID, taskID int64, addons []string) {
	task, ok := e.GetTask(taskID)
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	e.taskStore.RegisterCancel(taskID, cancel)
	defer e.taskStore.UnregisterCancel(taskID)
	defer cancel()

	now := time.Now().UTC()
	task.Status = platformapp.TaskRunning
	task.Steps = []platformapp.TaskStep{newAddonInstallTaskStep(addons, now)}
	message := "正在通过 Master 安装所选附加组件"
	task.Message = &message
	_ = e.taskStore.Put(task)

	plan, nodes, err := e.planWithNodes(ctx, uint64(planID))
	if err != nil {
		e.markTaskFailed(task, "读取部署方案失败："+err.Error())
		return
	}
	runPlan := plan
	runPlan.Addons = model.JSONStringSlice(runtimeClusterAddons(addons))
	runPlan.HelmInstall = containsClusterAddon(addons, "helm")
	task.AppendLog("[info] 将仅执行组件安装角色："+strings.Join(addons, ", "), activeTaskStepKey(task))
	_ = e.taskStore.Put(task)

	if e.ansibleRunner == nil {
		err = fmt.Errorf("Ansible Runner 未注入")
	} else {
		_, err = e.ansibleRunner.Run(ctx, runPlan, nodes, task)
	}
	if ctx.Err() != nil {
		e.markTaskCanceled(task)
		return
	}
	if err != nil {
		e.markTaskFailed(task, "附加组件安装失败："+err.Error())
		return
	}

	merged := mergeClusterAddons([]string(plan.Addons), runtimeClusterAddons(addons))
	updates := map[string]any{"addons": model.JSONStringSlice(merged)}
	if containsClusterAddon(addons, "helm") {
		updates["helm_install"] = true
	}
	if err := e.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("id = ?", planID).Updates(updates).Error; err != nil {
		e.markTaskFailed(task, "组件已完成安装，但更新部署方案记录失败："+err.Error())
		return
	}
	finished := time.Now().UTC()
	task.Steps[0].Status = platformapp.StepSuccess
	task.Steps[0].FinishedAt = &finished
	percent := 100
	task.Percent = &percent
	task.Status = platformapp.TaskSuccess
	message = "附加组件安装完成：" + strings.Join(addons, ", ")
	task.Message = &message
	task.AppendLog("[info] "+message, "install_addons")
	_ = e.taskStore.Put(task)
}

func (e *DeploymentExecutor) markTaskFailed(task *platformapp.Task, message string) {
	if task == nil {
		return
	}
	task.Status = platformapp.TaskFailed
	task.Message = &message
	now := time.Now().UTC()
	for index := range task.Steps {
		if task.Steps[index].Status == platformapp.StepRunning {
			task.Steps[index].Status = platformapp.StepFailed
			task.Steps[index].FinishedAt = &now
			stepMessage := message
			task.Steps[index].Message = &stepMessage
		}
		finishUnresolvedSubSteps(&task.Steps[index], now)
	}
	_ = e.taskStore.Put(task)
}

func (e *DeploymentExecutor) markTaskCanceled(task *platformapp.Task) {
	if task == nil {
		return
	}
	task.Status = platformapp.TaskCanceled
	message := "任务已取消"
	task.Message = &message
	_ = e.taskStore.Put(task)
}

func (e *DeploymentExecutor) updatePlanStatusDirect(ctx context.Context, planID uint64, status string) {
	if e != nil && e.db != nil {
		_ = e.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("id = ?", planID).Update("status", status).Error
	}
}

func preflightFailureMessage(result provisionapp.PreflightResult) string {
	messages := make([]string, 0, 3)
	for _, check := range result.Checks {
		if check.Status == provisionapp.PreflightError {
			messages = append(messages, check.Message)
		}
		if len(messages) == 3 {
			break
		}
	}
	if len(messages) == 0 {
		return "部署预检未通过"
	}
	return "部署预检未通过：" + strings.Join(messages, "；")
}

func finishUnresolvedSubSteps(step *platformapp.TaskStep, finishedAt time.Time) {
	if step == nil {
		return
	}
	for index := range step.SubSteps {
		if step.SubSteps[index].Status != platformapp.StepRunning {
			continue
		}
		step.SubSteps[index].FinishedAt = &finishedAt
		if step.Status == platformapp.StepFailed {
			step.SubSteps[index].Status = platformapp.StepFailed
		} else {
			step.SubSteps[index].Status = platformapp.StepSuccess
		}
	}
}

func isKnownAnsibleStep(key string) bool {
	for _, step := range provisionapp.DefaultAnsibleSteps() {
		if step.Key == key {
			return true
		}
	}
	return false
}

var supportedClusterAddons = map[string]struct{}{
	"metrics-server": {},
	"ingress-nginx":  {},
	"local-storage":  {},
	"helm":           {},
}

func normalizeClusterAddons(input []string) ([]string, error) {
	addons := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, addon := range input {
		addon = strings.TrimSpace(addon)
		if _, ok := supportedClusterAddons[addon]; !ok {
			return nil, provisionapp.ErrWithMessage(provisionapp.ErrInvalidParams, "包含不支持的附加组件")
		}
		if _, exists := seen[addon]; exists {
			continue
		}
		seen[addon] = struct{}{}
		addons = append(addons, addon)
	}
	if len(addons) == 0 {
		return nil, provisionapp.ErrWithMessage(provisionapp.ErrInvalidParams, "请至少选择一个附加组件")
	}
	return addons, nil
}

func mergeClusterAddons(existing, added []string) []string {
	result := make([]string, 0, len(existing)+len(added))
	seen := make(map[string]struct{}, len(existing)+len(added))
	for _, addon := range append(existing, added...) {
		if _, exists := seen[addon]; exists {
			continue
		}
		seen[addon] = struct{}{}
		result = append(result, addon)
	}
	return result
}

func selectedInstalledAddons(existing []string, helmInstalled bool, requested []string) []string {
	installed := make(map[string]struct{}, len(existing)+1)
	for _, addon := range existing {
		installed[addon] = struct{}{}
	}
	if helmInstalled {
		installed["helm"] = struct{}{}
	}
	duplicates := make([]string, 0, len(requested))
	for _, addon := range requested {
		if _, exists := installed[addon]; exists {
			duplicates = append(duplicates, addon)
		}
	}
	return duplicates
}

func isHelmOnlyAddonSelection(addons []string) bool { return len(addons) == 1 && addons[0] == "helm" }

func newAddonInstallTaskStep(addons []string, startedAt time.Time) platformapp.TaskStep {
	if isHelmOnlyAddonSelection(addons) {
		return platformapp.TaskStep{Key: "install_helm", Title: "补充安装 Helm", Status: platformapp.StepRunning, StartedAt: &startedAt}
	}
	return platformapp.TaskStep{Key: "install_addons", Title: "补充安装集群组件", Status: platformapp.StepRunning, StartedAt: &startedAt}
}

func addonInstallSteps(addons []string) []string {
	steps := make([]string, 0, 2)
	if containsClusterAddon(addons, "helm") {
		steps = append(steps, "install_helm")
	}
	if len(runtimeClusterAddons(addons)) > 0 {
		steps = append(steps, "install_addons")
	}
	return steps
}

func runtimeClusterAddons(addons []string) []string {
	result := make([]string, 0, len(addons))
	for _, addon := range addons {
		if addon != "helm" {
			result = append(result, addon)
		}
	}
	return result
}

func containsClusterAddon(addons []string, target string) bool {
	for _, addon := range addons {
		if addon == target {
			return true
		}
	}
	return false
}
