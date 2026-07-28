package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s-platform-backend/internal/legacy/model"
)

var supportedClusterAddons = map[string]struct{}{
	"metrics-server": {},
	"ingress-nginx":  {},
	"local-storage":  {},
	"helm":           {},
}

// InstallPlanAddons starts a focused, idempotent add-on task for a cluster
// that has already completed its initial deployment. It deliberately runs
// only the install_addons role and never re-initializes Kubernetes nodes.
func (s *DeployService) InstallPlanAddons(ctx context.Context, planID uint64, requested []string, userID uint64) (uint64, error) {
	if planID == 0 {
		return 0, ErrInvalidParams
	}
	addons, err := normalizeClusterAddons(requested)
	if err != nil {
		return 0, err
	}

	plan, _, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		return 0, err
	}
	if plan.Status != "success" || plan.ClusterID == nil || *plan.ClusterID == 0 {
		return 0, ErrWithMessage(ErrConflict, "仅已成功并已纳管的集群可以补充安装组件")
	}
	if alreadyInstalled := selectedInstalledAddons([]string(plan.Addons), plan.HelmInstall, addons); len(alreadyInstalled) > 0 {
		return 0, ErrWithMessage(ErrConflict, "以下组件已安装，不能重复安装："+strings.Join(alreadyInstalled, "、"))
	}
	for _, task := range s.taskStore.List() {
		if task.Type != "install_cluster_addons" || (task.Status != TaskPending && task.Status != TaskRunning) {
			continue
		}
		if fmt.Sprint(task.Meta["deploy_plan_id"]) == fmt.Sprint(planID) {
			return 0, ErrWithMessage(ErrConflict, "该集群已有附加组件安装任务正在执行，请等待其完成")
		}
	}

	title := "补充安装集群组件 " + plan.ClusterName
	if isHelmOnlyAddonSelection(addons) {
		title = "补充安装 Helm " + plan.ClusterName
	}
	message := "组件安装任务已创建，正在准备 Master Ansible Runner"
	percent := 0
	task := &Task{
		Type:      "install_cluster_addons",
		Status:    TaskPending,
		Title:     &title,
		CreatedBy: int64(userID),
		Percent:   &percent,
		Message:   &message,
		Meta: map[string]any{
			"deploy_plan_id": planID,
			"addons":         addons,
			// The runner archive uses this to render only the selected roles.
			"enabled_steps": addonInstallSteps(addons),
		},
	}
	if err := s.taskStore.Put(task); err != nil {
		return 0, err
	}
	go s.addonInstallPipeline(context.Background(), planID, task.ID, addons)
	return uint64(task.ID), nil
}

// GetLatestPlanAddonTask returns the most recent supplementary component task
// so the deployment pipeline can show its status after a page refresh.
func (s *DeployService) GetLatestPlanAddonTask(ctx context.Context, planID uint64) (*Task, error) {
	if planID == 0 {
		return nil, ErrInvalidParams
	}
	if _, _, err := s.getPlanWithNodes(ctx, planID); err != nil {
		return nil, err
	}
	var latest *Task
	for _, task := range s.taskStore.List() {
		if task.Type != "install_cluster_addons" || fmt.Sprint(task.Meta["deploy_plan_id"]) != fmt.Sprint(planID) {
			continue
		}
		if latest == nil || task.ID > latest.ID {
			latest = task
		}
	}
	return latest, nil
}

// RetryPlanAddons retries the last failed supplementary component task using
// its original component selection. The original plan remains successful; the
// retry is intentionally isolated from the cluster bootstrap workflow.
func (s *DeployService) RetryPlanAddons(ctx context.Context, planID uint64, userID uint64) (uint64, error) {
	task, err := s.GetLatestPlanAddonTask(ctx, planID)
	if err != nil {
		return 0, err
	}
	if task == nil {
		return 0, ErrWithMessage(ErrNotFound, "未找到可重试的附加组件安装任务")
	}
	if task.Status != TaskFailed && task.Status != TaskCanceled {
		return 0, ErrWithMessage(ErrConflict, "当前附加组件任务无需重试")
	}
	addons := taskMetaStringSlice(task.Meta, "addons")
	if len(addons) == 0 {
		return 0, ErrWithMessage(ErrConflict, "原附加组件任务未记录安装组件，无法重试")
	}
	return s.InstallPlanAddons(ctx, planID, addons, userID)
}

func normalizeClusterAddons(input []string) ([]string, error) {
	addons := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, addon := range input {
		addon = strings.TrimSpace(addon)
		if _, ok := supportedClusterAddons[addon]; !ok {
			return nil, ErrWithMessage(ErrInvalidParams, "包含不支持的附加组件")
		}
		if _, ok := seen[addon]; ok {
			continue
		}
		seen[addon] = struct{}{}
		addons = append(addons, addon)
	}
	if len(addons) == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "请至少选择一个附加组件")
	}
	return addons, nil
}

func (s *DeployService) addonInstallPipeline(ctx context.Context, planID uint64, taskID int64, addons []string) {
	task, ok := s.taskStore.Get(taskID)
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	s.taskStore.RegisterCancel(taskID, cancel)
	defer s.taskStore.UnregisterCancel(taskID)
	defer cancel()

	now := time.Now().UTC()
	task.Status = TaskRunning
	task.Steps = []TaskStep{newAddonInstallTaskStep(addons, now)}
	message := "正在通过 Master 安装所选附加组件"
	task.Message = &message
	_ = s.taskStore.Put(task)

	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		s.markTaskFailed(task, "读取部署方案失败："+err.Error())
		return
	}
	// The original plan remains unchanged until the role has completed. This
	// makes the summary truthful even when a remote install fails midway.
	runPlan := plan
	runPlan.Addons = model.JSONStringSlice(runtimeClusterAddons(addons))
	runPlan.HelmInstall = containsClusterAddon(addons, "helm")
	task.AppendLog("[info] 将仅执行组件安装角色："+strings.Join(addons, ", "), activeDeployStepKey(task))
	_ = s.taskStore.Put(task)

	_, err = s.runAnsibleOnMaster(ctx, runPlan, nodes, task)
	if ctx.Err() != nil {
		s.markTaskCanceled(task)
		return
	}
	if err != nil {
		s.markTaskFailed(task, "附加组件安装失败："+err.Error())
		return
	}

	merged := mergeClusterAddons([]string(plan.Addons), runtimeClusterAddons(addons))
	updates := map[string]any{"addons": model.JSONStringSlice(merged)}
	if containsClusterAddon(addons, "helm") {
		updates["helm_install"] = true
	}
	if err := s.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("id = ?", planID).Updates(updates).Error; err != nil {
		s.markTaskFailed(task, "组件已完成安装，但更新部署方案记录失败："+err.Error())
		return
	}
	finished := time.Now().UTC()
	task.Steps[0].Status = StepSuccess
	task.Steps[0].FinishedAt = &finished
	percent := 100
	task.Percent = &percent
	task.Status = TaskSuccess
	message = "附加组件安装完成：" + strings.Join(addons, ", ")
	task.Message = &message
	task.AppendLog("[info] "+message, "install_addons")
	_ = s.taskStore.Put(task)
}

func mergeClusterAddons(existing, added []string) []string {
	result := make([]string, 0, len(existing)+len(added))
	seen := make(map[string]struct{}, len(existing)+len(added))
	for _, addon := range append(existing, added...) {
		if _, ok := seen[addon]; ok {
			continue
		}
		seen[addon] = struct{}{}
		result = append(result, addon)
	}
	return result
}

func isHelmOnlyAddonSelection(addons []string) bool {
	return len(addons) == 1 && addons[0] == "helm"
}

func newAddonInstallTaskStep(addons []string, startedAt time.Time) TaskStep {
	if isHelmOnlyAddonSelection(addons) {
		return TaskStep{Key: "install_helm", Title: "补充安装 Helm", Status: StepRunning, StartedAt: &startedAt}
	}
	return TaskStep{Key: "install_addons", Title: "补充安装集群组件", Status: StepRunning, StartedAt: &startedAt}
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
		if _, ok := installed[addon]; ok {
			duplicates = append(duplicates, addon)
		}
	}
	return duplicates
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
