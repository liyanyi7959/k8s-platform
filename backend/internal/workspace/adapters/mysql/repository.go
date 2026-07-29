package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/workspace/domain"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, project *domain.Project) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&projectRow{}).Where("deleted_at IS NULL AND name = ?", project.Name).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return domain.ErrConflict
		}
		row := fromDomain(*project)
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		project.ID, project.CreatedAt, project.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
		return nil
	})
}

func (r *Repository) List(ctx context.Context, page, pageSize int) ([]domain.Project, int64, error) {
	query := r.db.WithContext(ctx).Model(&projectRow{}).Where("deleted_at IS NULL")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []projectRow
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	projects := make([]domain.Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, toDomain(row))
	}
	return projects, total, nil
}

func (r *Repository) Get(ctx context.Context, id uint64) (domain.Project, error) {
	if id == 0 {
		return domain.Project{}, domain.ErrValidation
	}
	var row projectRow
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Project{}, domain.ErrNotFound
		}
		return domain.Project{}, err
	}
	return toDomain(row), nil
}

func (r *Repository) Update(ctx context.Context, project domain.Project) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&projectRow{}).Where("deleted_at IS NULL AND name = ? AND id <> ?", project.Name, project.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return domain.ErrConflict
		}
		result := tx.Model(&projectRow{}).Where("id = ? AND deleted_at IS NULL", project.ID).Updates(map[string]any{"name": project.Name, "description": project.Description, "cluster_id": project.ClusterID, "namespaces": project.Namespaces, "quota_cpu": project.QuotaCPU, "quota_memory": project.QuotaMemory, "quota_pods": project.QuotaPods})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&projectRow{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type projectRow struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name        string     `gorm:"column:name"`
	Description string     `gorm:"column:description"`
	ClusterID   uint64     `gorm:"column:cluster_id"`
	Namespaces  string     `gorm:"column:namespaces"`
	QuotaCPU    string     `gorm:"column:quota_cpu"`
	QuotaMemory string     `gorm:"column:quota_memory"`
	QuotaPods   string     `gorm:"column:quota_pods"`
	CreatorID   uint64     `gorm:"column:creator_id"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

func (projectRow) TableName() string { return "projects" }
func toDomain(row projectRow) domain.Project {
	return domain.Project{ID: row.ID, Name: row.Name, Description: row.Description, ClusterID: row.ClusterID, Namespaces: row.Namespaces, QuotaCPU: row.QuotaCPU, QuotaMemory: row.QuotaMemory, QuotaPods: row.QuotaPods, CreatorID: row.CreatorID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: row.DeletedAt}
}
func fromDomain(project domain.Project) projectRow {
	return projectRow{Name: project.Name, Description: project.Description, ClusterID: project.ClusterID, Namespaces: project.Namespaces, QuotaCPU: project.QuotaCPU, QuotaMemory: project.QuotaMemory, QuotaPods: project.QuotaPods, CreatorID: project.CreatorID}
}
