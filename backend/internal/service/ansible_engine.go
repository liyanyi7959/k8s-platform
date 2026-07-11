package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"k8s-platform-backend/internal/model"
)

// ansiblePipeline 执行基于 Ansible Playbook 的 K8s 集群部署流水线
// 替代原有的 deployPipeline 方法
func (s *DeployService) ansiblePipeline(ctx context.Context, planID uint64, taskID int64) {
	task, ok := s.taskStore.Get(taskID)
	if !ok {
		return
	}

	// 注册取消函数
	ctx, cancel := context.WithCancel(ctx)
	s.taskStore.RegisterCancel(taskID, cancel)
	defer s.taskStore.UnregisterCancel(taskID)

	// 更新任务状态为运行中
	task.Status = TaskRunning
	msg := "正在执行 Ansible 部署流水线"
	task.Message = &msg
	_ = s.taskStore.Put(task)

	// 初始化步骤状态
	task.Steps = make([]TaskStep, len(ansibleSteps))
	for i, step := range ansibleSteps {
		task.Steps[i] = TaskStep{
			Key:   step.Key,
			Title: step.Title,
			Status: StepPending,
		}
	}
	percent := 0
	task.Percent = &percent
	_ = s.taskStore.Put(task)

	// 获取部署计划和节点
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		s.markTaskFailed(task, fmt.Sprintf("获取部署计划失败: %v", err))
		s.updatePlanStatusDirect(ctx, planID, "failed")
		return
	}

	// 检查是否被取消
	if ctx.Err() != nil {
		s.markTaskCanceled(task)
		s.updatePlanStatusDirect(ctx, planID, "cancelled")
		return
	}

	// 生成 inventory 文件
	task.AppendLog("[info] 正在生成 Ansible inventory...")
	_ = s.taskStore.Put(task)
	inventoryPath, cleanup, err := s.generateInventoryFile(ctx, plan, nodes)
	if err != nil {
		s.markTaskFailed(task, fmt.Sprintf("生成 inventory 失败: %v", err))
		s.updatePlanStatusDirect(ctx, planID, "failed")
		return
	}
	defer cleanup()

	task.AppendLog(fmt.Sprintf("[info] Inventory 文件已生成: %s", inventoryPath))
	_ = s.taskStore.Put(task)

	// 构建 extra vars
	extraVars := map[string]interface{}{
		"k8s_version":      plan.K8sVersion,
		"k8s_minor_version": extractMinorVersion(plan.K8sVersion),
		"pod_cidr":         plan.PodCIDR,
		"svc_cidr":         plan.SvcCIDR,
		"cni_type":         plan.CNIType,
		"cluster_name":     plan.ClusterName,
	}

	// 执行 playbook
	playbookPath := s.ansiblePlaybookPath()
	cmdRunDir := s.ansiblePlaybookDir()

	task.AppendLog(fmt.Sprintf("[info] 开始执行部署，Playbook: %s", playbookPath))
	task.AppendLog(fmt.Sprintf("[info] 集群: %s, K8s 版本: %s, CNI: %s", plan.ClusterName, plan.K8sVersion, plan.CNIType))
	_ = s.taskStore.Put(task)

	// 标记第一个步骤为运行中
	if len(task.Steps) > 0 {
		task.Steps[0].Status = StepRunning
		_ = s.taskStore.Put(task)
	}

	err = s.runAnsiblePlaybook(ctx, ansibleExecuteOptions{
		PlaybookPath: playbookPath,
		Inventory:    inventoryPath,
		ExtraVars:    extraVars,
		CmdRunDir:    cmdRunDir,
	}, task)

	// 检查取消
	if ctx.Err() != nil {
		s.markTaskCanceled(task)
		s.updatePlanStatusDirect(ctx, planID, "cancelled")
		return
	}

	if err != nil {
		// 标记失败步骤
		for i := range task.Steps {
			if task.Steps[i].Status == StepRunning {
				task.Steps[i].Status = StepFailed
				errMsg := err.Error()
				task.Steps[i].Message = &errMsg
			}
		}
		s.markTaskFailed(task, fmt.Sprintf("部署失败: %v", err))
		s.updatePlanStatusDirect(ctx, planID, "failed")
		return
	}

	// 标记所有步骤为成功
	for i := range task.Steps {
		if task.Steps[i].Status != StepSuccess {
			task.Steps[i].Status = StepSuccess
		}
	}
	_ = s.taskStore.Put(task)

	// 注册集群到平台
	clusterID, regErr := s.registerClusterAfterDeploy(ctx, plan)
	if regErr != nil {
		task.AppendLog(fmt.Sprintf("[warn] 集群注册失败: %v（部署已完成，可手动导入集群）", regErr))
		_ = s.taskStore.Put(task)
	} else {
		task.AppendLog(fmt.Sprintf("[info] 集群 %s 注册成功，ID: %d", plan.ClusterName, clusterID))
		_ = s.taskStore.Put(task)
	}

	// 全部成功
	percent = 100
	task.Percent = &percent
	task.Status = TaskSuccess
	successMsg := "集群部署成功"
	task.Message = &successMsg
	_ = s.taskStore.Put(task)
	s.updatePlanStatusDirect(ctx, planID, "success")
}

// registerClusterAfterDeploy 从 Ansible 输出的 kubeconfig 文件注册集群
func (s *DeployService) registerClusterAfterDeploy(ctx context.Context, plan model.DeployPlan) (uint64, error) {
	// register role 将 kubeconfig 写到 /tmp/k8s-deploy-{cluster_name}-kubeconfig.yml
	kubeconfigPath := fmt.Sprintf("/tmp/k8s-deploy-%s-kubeconfig.yml", plan.ClusterName)
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return 0, fmt.Errorf("读取 kubeconfig 文件失败: %w", err)
	}
	defer os.Remove(kubeconfigPath)

	kubeconfig := string(data)
	if strings.TrimSpace(kubeconfig) == "" {
		return 0, fmt.Errorf("kubeconfig 内容为空")
	}

	clusterID, err := s.clusterRegistry.ImportCluster(ctx, plan.ClusterName, kubeconfig)
	if err != nil {
		return 0, fmt.Errorf("注册集群失败: %w", err)
	}

	// 更新部署计划的 cluster_id
	_ = s.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("id = ?", plan.ID).Update("cluster_id", clusterID).Error

	return clusterID, nil
}

// extractMinorVersion 从 K8s 版本号中提取次版本号
// 例如: "v1.31.0" -> "1.31"
func extractMinorVersion(version string) string {
	v := strings.TrimPrefix(version, "v")
	parts := strings.Split(v, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return v
}

// markTaskFailed 标记任务失败
func (s *DeployService) markTaskFailed(task *Task, msg string) {
	task.Status = TaskFailed
	task.Message = &msg
	now := time.Now().UTC()
	for i := range task.Steps {
		if task.Steps[i].Status == StepRunning {
			task.Steps[i].Status = StepFailed
			task.Steps[i].FinishedAt = &now
			m := msg
			task.Steps[i].Message = &m
		}
	}
	_ = s.taskStore.Put(task)
}

// markTaskCanceled 标记任务取消
func (s *DeployService) markTaskCanceled(task *Task) {
	task.Status = TaskCanceled
	msg := "任务已取消"
	task.Message = &msg
	_ = s.taskStore.Put(task)
}

// updatePlanStatusDirect 直接更新计划状态
func (s *DeployService) updatePlanStatusDirect(ctx context.Context, planID uint64, status string) {
	_ = s.db.WithContext(ctx).Model(&model.DeployPlan{}).Where("id = ?", planID).Update("status", status).Error
}
