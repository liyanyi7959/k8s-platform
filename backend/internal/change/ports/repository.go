package ports

import (
	"context"
	"errors"
	"time"

	"k8s-platform-backend/internal/change/domain"
)

var (
	ErrProposalNotFound    = errors.New("change proposal not found")
	ErrConcurrentChange    = errors.New("change proposal changed concurrently")
	ErrExecutionNotFound   = errors.New("change execution not found")
	ErrIdempotencyConflict = errors.New("change execution idempotency key already used")
)

// Approval records the identity attached to a proposal state transition.
type Approval struct {
	ActorID   uint64
	ActorName string
	At        time.Time
}

// ProposalRepository owns persistence for the Change proposal aggregate.
// Transaction supplies a transaction-bound repository to the callback.
type ProposalRepository interface {
	Transaction(ctx context.Context, fn func(ProposalRepository) error) error
	FindForUpdate(ctx context.Context, clusterID, proposalID uint64) (domain.Proposal, error)
	RecordFirstApproval(ctx context.Context, proposal domain.Proposal, approval Approval) error
	ReserveExecution(ctx context.Context, proposal domain.Proposal, approval Approval) error

	// 执行登记与审计：确认通过后登记持久化执行记录，AI 同步执行完成后回填终态。
	AppendRevision(ctx context.Context, revision domain.Revision) error
	CreateExecution(ctx context.Context, execution domain.Execution) (domain.Execution, error)
	FindExecutionByKey(ctx context.Context, proposalID uint64, idempotencyKey string) (domain.Execution, error)
	FindActiveExecution(ctx context.Context, proposalID uint64) (domain.Execution, error)
	CompleteExecution(ctx context.Context, execution domain.Execution, resultJSON string) error
	FailExecution(ctx context.Context, execution domain.Execution, message string) error
	ListTimedOutExecutions(ctx context.Context, before time.Time, limit int) ([]domain.Execution, error)
}
