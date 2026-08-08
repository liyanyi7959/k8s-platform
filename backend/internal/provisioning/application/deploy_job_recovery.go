package application

import (
	"context"
	"time"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
	"k8s-platform-backend/internal/provisioning/ports"
)

// DeployJobRecoveryService 回收租约过期的部署执行：标记 Durable Job 失败，
// 并联动把仍在运行中的部署计划恢复为失败态，避免进程崩溃后计划永久卡在
// 运行中无法重试。
type DeployJobRecoveryService struct {
	repository ports.Repository
}

func NewDeployJobRecoveryService(repository ports.Repository) *DeployJobRecoveryService {
	return &DeployJobRecoveryService{repository: repository}
}

// Recover 将租约已过期的运行中部署执行标记为失败，并联动恢复部署计划。
func (service *DeployJobRecoveryService) Recover(ctx context.Context) error {
	if service == nil || service.repository == nil {
		return ErrInvalidParams
	}
	jobs, err := service.repository.ListExpiredDeployJobs(ctx, time.Now().UTC(), 50)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := service.recoverJob(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

func (service *DeployJobRecoveryService) recoverJob(ctx context.Context, job provisiondomain.DeployJob) error {
	failed, err := job.Fail(time.Now().UTC())
	if err != nil {
		return nil // 已离开运行中状态，跳过
	}
	if _, err := service.repository.CompleteDeployJob(ctx, job.PlanID, failed.Status, "execution timed out"); err != nil {
		return err
	}
	plan, found, err := service.repository.FindDeployPlan(ctx, job.PlanID)
	if err != nil || !found {
		return err
	}
	if plan.Status == "running" {
		return service.repository.UpdateDeployPlan(ctx, job.PlanID, map[string]any{"status": "failed"})
	}
	return nil
}

// RunDeployJobRecoveryLoop 进程内轮询回收过期部署执行；单次回收带独立
// 超时，避免阻塞应用退出。
func RunDeployJobRecoveryLoop(ctx context.Context, service *DeployJobRecoveryService, interval time.Duration) {
	if service == nil {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			recoveryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = service.Recover(recoveryCtx)
			cancel()
		}
	}
}
