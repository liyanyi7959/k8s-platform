package provisioning

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformapp "k8s-platform-backend/internal/platform/application"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
	provisionports "k8s-platform-backend/internal/provisioning/ports"
)

// 部署执行租约参数：长步骤期间由心跳持续续期，进程崩溃后租约过期交由
// 进程内 worker 兜底回收。
const (
	deployJobLeaseDuration     = 30 * time.Minute
	deployJobHeartbeatInterval = 2 * time.Minute
)

// DeploymentExecutor owns the operational state machine for a deployment
// plan. It is the Provisioning runtime's single owner for task persistence,
// readiness gating, Ansible execution, and the post-deployment fleet import.
//
// The platform task store and fleet registry are cross-context ports. Their
// concrete implementations are wired at the composition root; no legacy
// service coordinator participates in deployment execution.
type DeploymentExecutor struct {
	repository      provisionports.Repository
	taskStore       DeploymentTaskStore
	clusterRegistry ClusterRegistrar
	preflight       *PreflightRuntime
	ansibleRunner   *AnsibleRunner
}

func NewDeploymentExecutor(
	repository provisionports.Repository,
	taskStore DeploymentTaskStore,
	clusterRegistry ClusterRegistrar,
	preflight *PreflightRuntime,
	ansibleRunner *AnsibleRunner,
) *DeploymentExecutor {
	return &DeploymentExecutor{
		repository:      repository,
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
	err = e.repository.Transaction(ctx, func(tx provisionports.Repository) error {
		plan, found, err := tx.FindDeployPlan(ctx, planID)
		if err != nil {
			return err
		}
		if !found {
			return provisionapp.ErrNotFound
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
		leaseExpiresAt := time.Now().UTC().Add(deployJobLeaseDuration)
		job := &model.DeployJob{
			PlanID: planID, TaskID: taskID, JobType: "deploy_cluster",
			Status: model.DeployJobRunning, LeaseExpiresAt: &leaseExpiresAt,
		}
		if err := tx.CreateDeployJob(ctx, job); err != nil {
			return err
		}
		return tx.UpdateDeployPlan(ctx, planID, map[string]any{
			"status": "running", "task_id": taskID,
		})
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
	if e == nil || e.repository == nil || e.taskStore == nil {
		return provisionapp.ErrConflict
	}
	plan, found, err := e.repository.FindDeployPlan(ctx, planID)
	if err != nil {
		return err
	}
	if !found {
		return provisionapp.ErrNotFound
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
	if e == nil || e.repository == nil || e.taskStore == nil {
		return 0, provisionapp.ErrConflict
	}
	plan, found, err := e.repository.FindDeployPlan(ctx, planID)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, provisionapp.ErrNotFound
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
	if e == nil || e.repository == nil || e.taskStore == nil {
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
	leaseExpiresAt := time.Now().UTC().Add(deployJobLeaseDuration)
	job := &model.DeployJob{
		PlanID: planID, TaskID: uint64(task.ID), JobType: "install_cluster_addons",
		Status: model.DeployJobRunning, LeaseExpiresAt: &leaseExpiresAt,
	}
	if err := e.repository.CreateDeployJob(ctx, job); err != nil {
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
	if e == nil || e.repository == nil || e.taskStore == nil || e.preflight == nil {
		return provisionapp.ErrWithMessage(provisionapp.ErrConflict, "部署运行时未初始化")
	}
	return nil
}

func (e *DeploymentExecutor) planWithNodes(ctx context.Context, planID uint64) (model.DeployPlan, []model.DeployPlanNode, error) {
	if e == nil || e.repository == nil {
		return model.DeployPlan{}, nil, provisionapp.ErrConflict
	}
	plan, found, err := e.repository.FindDeployPlan(ctx, planID)
	if err != nil {
		return model.DeployPlan{}, nil, err
	}
	if !found {
		return model.DeployPlan{}, nil, provisionapp.ErrNotFound
	}
	nodes, err := e.repository.ListDeployPlanNodes(ctx, planID)
	if err != nil {
		return model.DeployPlan{}, nil, err
	}
	return plan, nodes, nil
}

func (e *DeploymentExecutor) updatePlanStatus(ctx context.Context, planID uint64, from, to string) error {
	return e.repository.Transaction(ctx, func(tx provisionports.Repository) error {
		plan, found, err := tx.FindDeployPlan(ctx, planID)
		if err != nil {
			return err
		}
		if !found {
			return provisionapp.ErrNotFound
		}
		if plan.Status != from {
			return provisionapp.ErrWithMessage(provisionapp.ErrConflict, "部署计划状态不允许执行该操作")
		}
		return tx.UpdateDeployPlan(ctx, planID, map[string]any{"status": to})
	})
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

	heartbeatStop := make(chan struct{})
	defer close(heartbeatStop)
	go e.deployJobHeartbeatLoop(planID, heartbeatStop)

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
		e.finishDeployJob(ctx, planID, model.DeployJobFailed, fmt.Sprintf("获取部署计划失败: %v", err))
		return
	}
	if ctx.Err() != nil {
		e.markTaskCanceled(task)
		e.updatePlanStatusDirect(ctx, planID, "cancelled")
		e.finishDeployJob(ctx, planID, model.DeployJobCancelled, "任务已取消")
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
		e.finishDeployJob(ctx, planID, model.DeployJobCancelled, "任务已取消")
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
		e.finishDeployJob(ctx, planID, model.DeployJobFailed, fmt.Sprintf("部署失败: %v", err))
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
		e.finishDeployJob(ctx, planID, model.DeployJobFailed, fmt.Sprintf("集群已安装但注册平台失败: %v", registerErr))
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
	e.finishDeployJob(ctx, planID, model.DeployJobSucceeded, "集群部署成功")
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
	_ = e.repository.UpdateDeployPlan(ctx, plan.ID, map[string]any{"cluster_id": clusterID})
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

	heartbeatStop := make(chan struct{})
	defer close(heartbeatStop)
	go e.deployJobHeartbeatLoop(uint64(planID), heartbeatStop)

	now := time.Now().UTC()
	task.Status = platformapp.TaskRunning
	task.Steps = []platformapp.TaskStep{newAddonInstallTaskStep(addons, now)}
	message := "正在通过 Master 安装所选附加组件"
	task.Message = &message
	_ = e.taskStore.Put(task)

	plan, nodes, err := e.planWithNodes(ctx, uint64(planID))
	if err != nil {
		e.markTaskFailed(task, "读取部署方案失败："+err.Error())
		e.finishDeployJob(ctx, uint64(planID), model.DeployJobFailed, "读取部署方案失败："+err.Error())
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
		e.finishDeployJob(ctx, uint64(planID), model.DeployJobCancelled, "任务已取消")
		return
	}
	if err != nil {
		e.markTaskFailed(task, "附加组件安装失败："+err.Error())
		e.finishDeployJob(ctx, uint64(planID), model.DeployJobFailed, "附加组件安装失败："+err.Error())
		return
	}

	merged := mergeClusterAddons([]string(plan.Addons), runtimeClusterAddons(addons))
	updates := map[string]any{"addons": model.JSONStringSlice(merged)}
	if containsClusterAddon(addons, "helm") {
		updates["helm_install"] = true
	}
	if err := e.repository.UpdateDeployPlan(ctx, uint64(planID), updates); err != nil {
		e.markTaskFailed(task, "组件已完成安装，但更新部署方案记录失败："+err.Error())
		e.finishDeployJob(ctx, uint64(planID), model.DeployJobFailed, "组件已完成安装，但更新部署方案记录失败："+err.Error())
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
	e.finishDeployJob(ctx, uint64(planID), model.DeployJobSucceeded, message)
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

// deployJobHeartbeatLoop 在流水线执行期间定期续期 Durable Job 租约，保证
// 长步骤不被 worker 误判为僵尸；流水线退出时通过 stop 通道停止。
func (e *DeploymentExecutor) deployJobHeartbeatLoop(planID uint64, stop <-chan struct{}) {
	if e == nil || e.repository == nil {
		return
	}
	ticker := time.NewTicker(deployJobHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			expiresAt := time.Now().UTC().Add(deployJobLeaseDuration)
			_, _ = e.repository.HeartbeatDeployJob(context.Background(), planID, expiresAt)
		}
	}
}

// finishDeployJob 通过领域状态机校验后将 Durable Job 推进到终态。
func (e *DeploymentExecutor) finishDeployJob(ctx context.Context, planID uint64, status model.DeployJobStatus, message string) {
	if e == nil || e.repository == nil {
		return
	}
	job, found, err := e.repository.FindDeployJobByPlan(ctx, planID)
	if err != nil || !found {
		return
	}
	var updated model.DeployJob
	switch status {
	case model.DeployJobSucceeded:
		updated, err = job.Complete(time.Now().UTC())
	case model.DeployJobCancelled:
		updated, err = job.Cancel(time.Now().UTC())
	default:
		updated, err = job.Fail(time.Now().UTC())
	}
	if err != nil {
		return
	}
	_, _ = e.repository.CompleteDeployJob(ctx, planID, updated.Status, message)
}

func (e *DeploymentExecutor) updatePlanStatusDirect(ctx context.Context, planID uint64, status string) {
	if e != nil && e.repository != nil {
		_ = e.repository.UpdateDeployPlan(ctx, planID, map[string]any{"status": status})
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
