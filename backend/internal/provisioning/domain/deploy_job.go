package domain

import (
	"errors"
	"time"
)

var ErrDeployJobNotActive = errors.New("provisioning deploy job is not active")

// DeployJobStatus 是部署执行登记的生命周期状态。
type DeployJobStatus string

const (
	DeployJobRunning   DeployJobStatus = "running"
	DeployJobSucceeded DeployJobStatus = "succeeded"
	DeployJobFailed    DeployJobStatus = "failed"
	DeployJobCancelled DeployJobStatus = "cancelled"
)

// DeployJob 记录一次部署长任务的租约登记。进程崩溃后租约过期，由进程内
// worker 回收；执行期间通过 Heartbeat 续期，避免长步骤被误判为僵尸。
type DeployJob struct {
	ID             uint64
	PlanID         uint64
	TaskID         uint64
	JobType        string
	Status         DeployJobStatus
	LeaseExpiresAt *time.Time
	Message        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Complete 将执行推进到成功终态，仅允许从运行中转换。
func (job DeployJob) Complete(finishedAt time.Time) (DeployJob, error) {
	if job.Status != DeployJobRunning {
		return job, ErrDeployJobNotActive
	}
	job.Status = DeployJobSucceeded
	return job, nil
}

// Fail 将执行推进到失败终态，仅允许从运行中转换。
func (job DeployJob) Fail(finishedAt time.Time) (DeployJob, error) {
	if job.Status != DeployJobRunning {
		return job, ErrDeployJobNotActive
	}
	job.Status = DeployJobFailed
	return job, nil
}

// Cancel 将执行推进到取消终态，仅允许从运行中转换。
func (job DeployJob) Cancel(finishedAt time.Time) (DeployJob, error) {
	if job.Status != DeployJobRunning {
		return job, ErrDeployJobNotActive
	}
	job.Status = DeployJobCancelled
	return job, nil
}

// Heartbeat 续期租约，仅允许从运行中转换。
func (job DeployJob) Heartbeat(expiresAt time.Time) (DeployJob, error) {
	if job.Status != DeployJobRunning {
		return job, ErrDeployJobNotActive
	}
	job.LeaseExpiresAt = &expiresAt
	return job, nil
}

// IsExpired 判断运行中的执行是否已超过租约期限。
func (job DeployJob) IsExpired(now time.Time) bool {
	if job.Status != DeployJobRunning || job.LeaseExpiresAt == nil {
		return false
	}
	return now.After(*job.LeaseExpiresAt)
}
