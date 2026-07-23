package service

import (
	"context"
	"fmt"
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
	retryFromStep, _ := task.Meta["retry_from_step"].(string)
	startIdx := 0
	if retryFromStep != "" {
		for i, step := range ansibleSteps {
			if step.Key == retryFromStep {
				startIdx = i
				break
			}
		}
	}
	now := time.Now().UTC()
	for i, step := range ansibleSteps {
		status := StepPending
		if retryFromStep != "" && i < startIdx {
			status = StepSuccess
		}
		task.Steps[i] = TaskStep{
			Key:        step.Key,
			Title:      step.Title,
			Status:     status,
			StartedAt:  nil,
			FinishedAt: nil,
		}
		if status == StepSuccess {
			task.Steps[i].StartedAt = &now
			task.Steps[i].FinishedAt = &now
		}
	}
	percent := 0
	task.Percent = &percent
	// Runner 准备、SSH 连接和依赖安装都属于首个待执行步骤。先将其标记为
	// running，确保 Ansible 输出 PLAY 之前产生的日志和错误也能归属到步骤。
	for i := range task.Steps {
		if task.Steps[i].Status != StepSuccess {
			task.Steps[i].Status = StepRunning
			task.Steps[i].StartedAt = &now
			break
		}
	}
	_ = s.taskStore.Put(task)

	// 获取部署计划和节点
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		task.AppendLog(fmt.Sprintf("[error] 获取部署计划失败: %v", err), activeDeployStepKey(task))
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

	stepKey := activeDeployStepKey(task)
	task.AppendLog("[info] 正在准备 Master 临时 Runner", stepKey)
	task.AppendLog(fmt.Sprintf("[info] 集群: %s, K8s 版本: %s, CNI: %s", plan.ClusterName, plan.K8sVersion, plan.CNIType), stepKey)
	_ = s.taskStore.Put(task)

	kubeconfig, err := s.runAnsibleOnMaster(ctx, plan, nodes, task)

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
	completedAt := time.Now().UTC()
	for i := range task.Steps {
		if task.Steps[i].Status != StepSuccess {
			task.Steps[i].Status = StepSuccess
		}
		finishUnresolvedSubSteps(&task.Steps[i], completedAt)
	}
	_ = s.taskStore.Put(task)

	// 注册集群到平台
	clusterID, regErr := s.registerClusterAfterDeploy(ctx, plan, kubeconfig)
	if regErr != nil {
		if len(task.Steps) > 0 {
			last := len(task.Steps) - 1
			task.Steps[last].Status = StepFailed
			message := fmt.Sprintf("集群已安装但注册平台失败: %v", regErr)
			task.Steps[last].Message = &message
		}
		s.markTaskFailed(task, fmt.Sprintf("集群已安装但注册平台失败: %v，请检查日志后重试注册流程", regErr))
		s.updatePlanStatusDirect(ctx, planID, "failed")
		return
	}
	task.AppendLog(fmt.Sprintf("[info] 集群 %s 注册成功，ID: %d", plan.ClusterName, clusterID), "")
	_ = s.taskStore.Put(task)

	// 全部成功
	percent = 100
	task.Percent = &percent
	task.Status = TaskSuccess
	successMsg := "集群部署成功"
	task.Message = &successMsg
	_ = s.taskStore.Put(task)
	s.updatePlanStatusDirect(ctx, planID, "success")
}

// activeDeployStepKey 返回当前部署日志应归属的步骤。
// Ansible 在输出第一个 PLAY 之前可能已因 Runner/SSH/依赖问题退出，此时仍需
// 将诊断日志记到用户看到的失败步骤，而不是不可见的空 step_key。
func activeDeployStepKey(task *Task) string {
	if task == nil {
		return ""
	}
	for _, step := range task.Steps {
		if step.Status == StepRunning {
			return step.Key
		}
	}
	for _, step := range task.Steps {
		if step.Status != StepSuccess {
			return step.Key
		}
	}
	return ""
}

// registerClusterAfterDeploy 使用从 Master SSH 回收的 kubeconfig 注册集群。
func (s *DeployService) registerClusterAfterDeploy(ctx context.Context, plan model.DeployPlan, kubeconfig string) (uint64, error) {
	if strings.TrimSpace(kubeconfig) == "" {
		return 0, fmt.Errorf("kubeconfig 内容为空")
	}

	clusterID, err := s.clusterRegistry.ImportCluster(ctx, plan.ClusterName, kubeconfig, "")
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
		finishUnresolvedSubSteps(&task.Steps[i], now)
	}
	_ = s.taskStore.Put(task)
}

// finishUnresolvedSubSteps ensures that a parent step reaching a terminal state
// cannot leave its final Ansible task displayed as running in the UI.
func finishUnresolvedSubSteps(step *TaskStep, finishedAt time.Time) {
	if step == nil {
		return
	}
	for i := range step.SubSteps {
		if step.SubSteps[i].Status != StepRunning {
			continue
		}
		step.SubSteps[i].FinishedAt = &finishedAt
		if step.Status == StepFailed {
			step.SubSteps[i].Status = StepFailed
		} else {
			step.SubSteps[i].Status = StepSuccess
		}
	}
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
