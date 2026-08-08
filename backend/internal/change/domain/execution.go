package domain

import (
	"errors"
	"time"
)

var ErrNotExecutable = errors.New("change execution is not in an executable state")

// ExecutionStatus 是提案执行登记的生命周期状态。
type ExecutionStatus string

const (
	ExecutionPending   ExecutionStatus = "pending"
	ExecutionRunning   ExecutionStatus = "running"
	ExecutionSucceeded ExecutionStatus = "succeeded"
	ExecutionFailed    ExecutionStatus = "failed"
	ExecutionCancelled ExecutionStatus = "cancelled"
)

// Execution 记录一次已确认提案的执行登记。持久化由 Change 执行端口负责，
// 状态转换必须通过 Complete/Fail 保持终态不可逆。
type Execution struct {
	ID              uint64
	ProposalID      uint64
	ClusterID       uint64
	ExecutionNo     int
	Status          ExecutionStatus
	IdempotencyKey  string
	OperatorID      uint64
	OperatorName    string
	CommandSnapshot string
	StartedAt       *time.Time
	FinishedAt      *time.Time
}

// Complete 将执行推进到成功终态，仅允许从进行中状态转换。
func (execution Execution) Complete(finishedAt time.Time) (Execution, error) {
	switch execution.Status {
	case ExecutionPending, ExecutionRunning:
	default:
		return execution, ErrNotExecutable
	}
	execution.Status = ExecutionSucceeded
	execution.FinishedAt = &finishedAt
	return execution, nil
}

// Fail 将执行推进到失败终态，仅允许从进行中状态转换。
func (execution Execution) Fail(finishedAt time.Time) (Execution, error) {
	switch execution.Status {
	case ExecutionPending, ExecutionRunning:
	default:
		return execution, ErrNotExecutable
	}
	execution.Status = ExecutionFailed
	execution.FinishedAt = &finishedAt
	return execution, nil
}
