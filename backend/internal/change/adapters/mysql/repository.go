package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"k8s-platform-backend/internal/change/domain"
	"k8s-platform-backend/internal/change/ports"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (repository *Repository) Transaction(ctx context.Context, fn func(ports.ProposalRepository) error) error {
	if repository == nil || repository.db == nil {
		return errors.New("change repository database is required")
	}
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Repository{db: tx})
	})
}

func (repository *Repository) FindForUpdate(ctx context.Context, clusterID, proposalID uint64) (domain.Proposal, error) {
	var row proposalRow
	err := repository.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND cluster_id = ?", proposalID, clusterID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Proposal{}, ports.ErrProposalNotFound
	}
	if err != nil {
		return domain.Proposal{}, err
	}
	return domain.Proposal{
		ID: row.ID, Status: domain.Status(row.Status), ConfirmLevel: domain.ConfirmLevel(row.ConfirmLevel), ApprovedBy: row.ApprovedBy,
	}, nil
}

func (repository *Repository) RecordFirstApproval(ctx context.Context, proposal domain.Proposal, approval ports.Approval) error {
	result := repository.db.WithContext(ctx).Table("ai_action_proposals").
		Where("id = ? AND status = ?", proposal.ID, string(proposal.Status)).
		Updates(map[string]any{
			"status": "approved", "approved_by": approval.ActorID,
			"approved_by_name": approval.ActorName, "approved_at": approval.At,
		})
	return changedResult(result)
}

func (repository *Repository) ReserveExecution(ctx context.Context, proposal domain.Proposal, approval ports.Approval) error {
	updates := map[string]any{"status": "executing"}
	if proposal.ConfirmLevel == domain.ConfirmDouble {
		updates["second_approved_by"] = approval.ActorID
		updates["second_approved_name"] = approval.ActorName
		updates["second_approved_at"] = approval.At
	} else if proposal.ApprovedBy == nil {
		updates["approved_by"] = approval.ActorID
		updates["approved_by_name"] = approval.ActorName
		updates["approved_at"] = approval.At
	}
	result := repository.db.WithContext(ctx).Table("ai_action_proposals").
		Where("id = ? AND status = ?", proposal.ID, string(proposal.Status)).
		Updates(updates)
	return changedResult(result)
}

func changedResult(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ports.ErrConcurrentChange
	}
	return nil
}

type proposalRow struct {
	ID           uint64  `gorm:"column:id;primaryKey"`
	ClusterID    uint64  `gorm:"column:cluster_id"`
	Status       string  `gorm:"column:status"`
	ConfirmLevel string  `gorm:"column:confirm_level"`
	ApprovedBy   *uint64 `gorm:"column:approved_by"`
}

func (proposalRow) TableName() string { return "ai_action_proposals" }
