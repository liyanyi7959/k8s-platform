package ports

import (
	"context"
	"errors"
	"time"

	"k8s-platform-backend/internal/change/domain"
)

var (
	ErrProposalNotFound = errors.New("change proposal not found")
	ErrConcurrentChange = errors.New("change proposal changed concurrently")
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
}
