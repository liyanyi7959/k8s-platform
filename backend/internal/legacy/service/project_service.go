package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/legacy/model"
)

// ProjectService 项目（多租户/命名空间分组）服务，负责项目的 CRUD。
type ProjectService struct {
	db *gorm.DB
}

// NewProjectService 创建项目服务实例。
func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

// CreateProject 创建项目。会校验名称非空且不与已存在项目重名。
// 创建成功后 p.ID 会被 GORM 回填。
func (s *ProjectService) CreateProject(ctx context.Context, p *model.Project) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if p == nil {
		return ErrWithMessage(ErrInvalidParams, "项目参数不能为空")
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return ErrWithMessage(ErrInvalidParams, "项目名称不能为空")
	}
	p.Name = name

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Project
		if err := tx.Where("deleted_at IS NULL AND name = ?", name).First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(p).Error
	})
}

// ListProjects 分页查询项目列表，按 id 倒序排列。
func (s *ProjectService) ListProjects(ctx context.Context, page, pageSize int) (PageResult[model.Project], error) {
	if s.db == nil {
		return PageResult[model.Project]{}, errors.New("db is required")
	}
	page, pageSize = normalizePage(page, pageSize)

	q := s.db.WithContext(ctx).Model(&model.Project{}).Where("deleted_at IS NULL")

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[model.Project]{}, err
	}

	var rows []model.Project
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[model.Project]{}, err
	}

	return PageResult[model.Project]{List: rows, Total: int(total), Page: page, PageSize: pageSize}, nil
}

// GetProject 根据 ID 查询项目详情。
func (s *ProjectService) GetProject(ctx context.Context, id uint64) (*model.Project, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	if id == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "项目ID无效")
	}
	var p model.Project
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// UpdateProject 根据 ID 更新项目字段。若更新名称需保证不与其他项目重名。
func (s *ProjectService) UpdateProject(ctx context.Context, id uint64, updates map[string]any) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "项目ID无效")
	}
	if len(updates) == 0 {
		return ErrWithMessage(ErrInvalidParams, "更新内容不能为空")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.Project
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		// 若更新了名称，需保证名称不与其他项目冲突。
		if name, ok := updates["name"].(string); ok {
			name = strings.TrimSpace(name)
			if name == "" {
				return ErrWithMessage(ErrInvalidParams, "项目名称不能为空")
			}
			updates["name"] = name
			if name != row.Name {
				var existing model.Project
				if err := tx.Select("id").Where("deleted_at IS NULL AND name = ? AND id <> ?", name, id).First(&existing).Error; err == nil {
					return ErrConflict
				} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
		}

		return tx.Model(&model.Project{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
	})
}

// DeleteProject 软删除项目（设置 deleted_at）。
func (s *ProjectService) DeleteProject(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "项目ID无效")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.Project
		if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&model.Project{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", &now).Error
	})
}
