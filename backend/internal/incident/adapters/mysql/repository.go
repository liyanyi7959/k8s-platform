package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/incident/domain"
	"k8s-platform-backend/internal/incident/ports"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, filter ports.ListFilter) ([]domain.Incident, int64, error) {
	query := r.db.WithContext(ctx).Model(&incidentRow{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []incidentRow
	if err := query.Order("started_at DESC").Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.Incident, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDomain(row))
	}
	return items, total, nil
}

func (r *Repository) Get(ctx context.Context, id uint64) (domain.Incident, []domain.TimelineEntry, error) {
	var row incidentRow
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Incident{}, nil, domain.ErrNotFound
		}
		return domain.Incident{}, nil, err
	}
	var timeline []timelineRow
	if err := r.db.WithContext(ctx).Where("incident_id = ?", id).Order("created_at ASC").Find(&timeline).Error; err != nil {
		return domain.Incident{}, nil, err
	}
	entries := make([]domain.TimelineEntry, 0, len(timeline))
	for _, item := range timeline {
		entries = append(entries, domain.TimelineEntry{ID: item.ID, Type: item.Type, Title: item.Title, Detail: item.Detail, OperatorID: item.OperatorID, Operator: item.Operator, CreatedAt: item.CreatedAt})
	}
	return toDomain(row), entries, nil
}

func (r *Repository) SaveTransition(ctx context.Context, incident domain.Incident, expectedVersion uint64, entry domain.TimelineEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"status": incident.Status, "version": incident.Version,
			"acknowledged_at": incident.AcknowledgedAt, "resolved_at": incident.ResolvedAt,
			"verified_at": incident.VerifiedAt, "verification_note": incident.VerificationNote,
			"assignee_id": incident.AssigneeID, "assignee_name": incident.AssigneeName,
		}
		result := tx.Model(&incidentRow{}).Where("id = ? AND version = ?", incident.ID, expectedVersion).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			var count int64
			if err := tx.Model(&incidentRow{}).Where("id = ?", incident.ID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return domain.ErrNotFound
			}
			return domain.ErrVersionConflict
		}
		return tx.Create(&timelineRow{IncidentID: incident.ID, Type: entry.Type, Title: entry.Title, Detail: entry.Detail, OperatorID: entry.OperatorID, Operator: entry.Operator, CreatedAt: entry.CreatedAt}).Error
	})
}

func (r *Repository) Names(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	result := make(map[uint64]string, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var rows []clusterNameRow
	if err := r.db.WithContext(ctx).Select("id", "name").Where("id IN ? AND deleted_at IS NULL", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.Name
	}
	return result, nil
}

func toDomain(row incidentRow) domain.Incident {
	version := row.Version
	if version == 0 {
		version = 1
	}
	return domain.Incident{ID: row.ID, AlertName: row.AlertName, ClusterID: row.ClusterID, Namespace: row.Namespace, ResourceKind: row.ResourceKind, ResourceName: row.ResourceName, Severity: row.Severity, Status: domain.Status(row.Status), Summary: row.Summary, StartedAt: row.StartedAt, AcknowledgedAt: row.AcknowledgedAt, ResolvedAt: row.ResolvedAt, VerifiedAt: row.VerifiedAt, AIConversationID: row.AIConversationID, AIProposalID: row.AIProposalID, AssigneeID: row.AssigneeID, AssigneeName: row.AssigneeName, VerificationNote: row.VerificationNote, Version: version}
}

// Persistence rows are private to the Incident adapter. The global model
// package remains available only to legacy code during migration.
type incidentRow struct {
	ID               uint64     `gorm:"column:id;primaryKey"`
	AlertName        string     `gorm:"column:alert_name"`
	ClusterID        uint64     `gorm:"column:cluster_id"`
	Namespace        string     `gorm:"column:namespace"`
	ResourceKind     string     `gorm:"column:resource_kind"`
	ResourceName     string     `gorm:"column:resource_name"`
	Severity         string     `gorm:"column:severity"`
	Status           string     `gorm:"column:status"`
	Summary          string     `gorm:"column:summary"`
	AIConversationID *uint64    `gorm:"column:ai_conversation_id"`
	AIProposalID     *uint64    `gorm:"column:ai_proposal_id"`
	AssigneeID       *uint64    `gorm:"column:assignee_id"`
	AssigneeName     string     `gorm:"column:assignee_name"`
	StartedAt        time.Time  `gorm:"column:started_at"`
	AcknowledgedAt   *time.Time `gorm:"column:acknowledged_at"`
	ResolvedAt       *time.Time `gorm:"column:resolved_at"`
	VerifiedAt       *time.Time `gorm:"column:verified_at"`
	VerificationNote string     `gorm:"column:verification_note"`
	Version          uint64     `gorm:"column:version"`
}

func (incidentRow) TableName() string { return "monitor_incidents" }

type timelineRow struct {
	ID         uint64    `gorm:"column:id;primaryKey"`
	IncidentID uint64    `gorm:"column:incident_id"`
	Type       string    `gorm:"column:type"`
	Title      string    `gorm:"column:title"`
	Detail     string    `gorm:"column:detail"`
	OperatorID uint64    `gorm:"column:operator_id"`
	Operator   string    `gorm:"column:operator"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (timelineRow) TableName() string { return "monitor_incident_timelines" }

type clusterNameRow struct {
	ID   uint64 `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
}

func (clusterNameRow) TableName() string { return "clusters" }
