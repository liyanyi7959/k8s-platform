package mysql

import (
	"context"
	"errors"
	"time"

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

func (repository *Repository) AppendRevision(ctx context.Context, revision domain.Revision) error {
	row := revisionRow{
		ProposalID: revision.ProposalID, FromStatus: string(revision.FromStatus), ToStatus: string(revision.ToStatus),
		ActorID: revision.ActorID, ActorName: revision.ActorName, Decision: revision.Decision, CreatedAt: revision.CreatedAt,
	}
	return repository.db.WithContext(ctx).Create(&row).Error
}

func (repository *Repository) CreateExecution(ctx context.Context, execution domain.Execution) (domain.Execution, error) {
	row := toExecutionRow(execution)
	query := repository.db.WithContext(ctx).Table("change_executions")
	// 幂等键唯一约束兜底：同键并发登记时静默放弃，由 Application 改查已有记录。
	if execution.IdempotencyKey != "" {
		query = query.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "idempotency_key"}}, DoNothing: true})
	}
	result := query.Create(&row)
	if result.Error != nil {
		return domain.Execution{}, result.Error
	}
	if execution.IdempotencyKey != "" && result.RowsAffected == 0 {
		return domain.Execution{}, ports.ErrIdempotencyConflict
	}
	return fromExecutionRow(row), nil
}

func (repository *Repository) FindExecutionByKey(ctx context.Context, proposalID uint64, idempotencyKey string) (domain.Execution, error) {
	var row executionRow
	err := repository.db.WithContext(ctx).
		Where("proposal_id = ? AND idempotency_key = ?", proposalID, idempotencyKey).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Execution{}, ports.ErrExecutionNotFound
	}
	if err != nil {
		return domain.Execution{}, err
	}
	return fromExecutionRow(row), nil
}

func (repository *Repository) FindActiveExecution(ctx context.Context, proposalID uint64) (domain.Execution, error) {
	var row executionRow
	err := repository.db.WithContext(ctx).
		Where("proposal_id = ? AND status IN ?", proposalID, []string{string(domain.ExecutionRunning), string(domain.ExecutionPending)}).
		Order("id DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Execution{}, ports.ErrExecutionNotFound
	}
	if err != nil {
		return domain.Execution{}, err
	}
	return fromExecutionRow(row), nil
}

func (repository *Repository) CompleteExecution(ctx context.Context, execution domain.Execution, resultJSON string) error {
	row := executionRow{
		ID: execution.ID, Status: string(execution.Status), FinishedAt: execution.FinishedAt,
	}
	if resultJSON != "" {
		row.ResultJSON = &resultJSON
	}
	result := repository.db.WithContext(ctx).Table("change_executions").
		Where("id = ? AND status IN ?", execution.ID, []string{string(domain.ExecutionRunning), string(domain.ExecutionPending)}).
		Updates(map[string]any{
			"status": string(execution.Status), "result_json": row.ResultJSON,
			"finished_at": execution.FinishedAt,
		})
	return changedResult(result)
}

func (repository *Repository) FailExecution(ctx context.Context, execution domain.Execution, message string) error {
	result := repository.db.WithContext(ctx).Table("change_executions").
		Where("id = ? AND status IN ?", execution.ID, []string{string(domain.ExecutionRunning), string(domain.ExecutionPending)}).
		Updates(map[string]any{
			"status": string(execution.Status), "error_message": message,
			"finished_at": execution.FinishedAt,
		})
	return changedResult(result)
}

func (repository *Repository) ListTimedOutExecutions(ctx context.Context, before time.Time, limit int) ([]domain.Execution, error) {
	var rows []executionRow
	err := repository.db.WithContext(ctx).
		Where("status = ? AND started_at < ?", string(domain.ExecutionRunning), before).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	executions := make([]domain.Execution, 0, len(rows))
	for _, row := range rows {
		executions = append(executions, fromExecutionRow(row))
	}
	return executions, nil
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

func toExecutionRow(execution domain.Execution) executionRow {
	row := executionRow{
		ProposalID: execution.ProposalID, ClusterID: execution.ClusterID, ExecutionNo: execution.ExecutionNo,
		Status: string(execution.Status), OperatorID: execution.OperatorID, OperatorName: execution.OperatorName,
		CommandSnapshot: execution.CommandSnapshot, StartedAt: execution.StartedAt,
	}
	if execution.IdempotencyKey != "" {
		key := execution.IdempotencyKey
		row.IdempotencyKey = &key
	}
	return row
}

func fromExecutionRow(row executionRow) domain.Execution {
	execution := domain.Execution{
		ID: row.ID, ProposalID: row.ProposalID, ClusterID: row.ClusterID, ExecutionNo: row.ExecutionNo,
		Status: domain.ExecutionStatus(row.Status), OperatorID: row.OperatorID, OperatorName: row.OperatorName,
		CommandSnapshot: row.CommandSnapshot, StartedAt: row.StartedAt, FinishedAt: row.FinishedAt,
	}
	if row.IdempotencyKey != nil {
		execution.IdempotencyKey = *row.IdempotencyKey
	}
	return execution
}

type proposalRow struct {
	ID           uint64  `gorm:"column:id;primaryKey"`
	ClusterID    uint64  `gorm:"column:cluster_id"`
	Status       string  `gorm:"column:status"`
	ConfirmLevel string  `gorm:"column:confirm_level"`
	ApprovedBy   *uint64 `gorm:"column:approved_by"`
}

func (proposalRow) TableName() string { return "ai_action_proposals" }

type executionRow struct {
	ID              uint64     `gorm:"column:id;primaryKey"`
	ProposalID      uint64     `gorm:"column:proposal_id"`
	ClusterID       uint64     `gorm:"column:cluster_id"`
	ExecutionNo     int        `gorm:"column:execution_no"`
	Status          string     `gorm:"column:status"`
	IdempotencyKey  *string    `gorm:"column:idempotency_key"`
	OperatorID      uint64     `gorm:"column:operator_id"`
	OperatorName    string     `gorm:"column:operator_name"`
	CommandSnapshot string     `gorm:"column:command_snapshot"`
	ResultJSON      *string    `gorm:"column:result_json"`
	ErrorMessage    *string    `gorm:"column:error_message"`
	StartedAt       *time.Time `gorm:"column:started_at"`
	FinishedAt      *time.Time `gorm:"column:finished_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (executionRow) TableName() string { return "change_executions" }

type revisionRow struct {
	ID         uint64    `gorm:"column:id;primaryKey"`
	ProposalID uint64    `gorm:"column:proposal_id"`
	ClusterID  uint64    `gorm:"column:cluster_id"`
	FromStatus string    `gorm:"column:from_status"`
	ToStatus   string    `gorm:"column:to_status"`
	ActorID    uint64    `gorm:"column:actor_id"`
	ActorName  string    `gorm:"column:actor_name"`
	Decision   string    `gorm:"column:decision"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (revisionRow) TableName() string { return "change_proposal_revisions" }
