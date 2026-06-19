package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// DeployConfigService 部署配置服务
type DeployConfigService struct {
	db *gorm.DB
}

func NewDeployConfigService(db *gorm.DB) *DeployConfigService {
	return &DeployConfigService{db: db}
}

// ─── DeployConfig CRUD ───

// ListConfigs 获取部署配置列表，支持按 os_type 筛选
func (s *DeployConfigService) ListConfigs(ctx context.Context, osType string) ([]model.DeployConfig, error) {
	var configs []model.DeployConfig
	q := s.db.WithContext(ctx).Order("step_order ASC, os_type ASC")
	if osType != "" {
		q = q.Where("os_type = ?", osType)
	}
	if err := q.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetConfig 根据ID获取配置
func (s *DeployConfigService) GetConfig(ctx context.Context, id uint64) (*model.DeployConfig, error) {
	var config model.DeployConfig
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &config, nil
}

// GetConfigByKey 根据step_key和os_type获取配置
func (s *DeployConfigService) GetConfigByKey(ctx context.Context, stepKey, osType string) (*model.DeployConfig, error) {
	var config model.DeployConfig
	q := s.db.WithContext(ctx).Where("step_key = ?", stepKey)
	if osType != "" {
		q = q.Where("os_type = ?", osType)
	}
	if err := q.First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &config, nil
}

// ListSupportedOSTypes 获取所有已配置的操作系统类型
func (s *DeployConfigService) ListSupportedOSTypes(ctx context.Context) ([]string, error) {
	var osTypes []string
	if err := s.db.WithContext(ctx).Model(&model.DeployConfig{}).Distinct().Pluck("os_type", &osTypes).Error; err != nil {
		return nil, err
	}
	return osTypes, nil
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	CommandTemplate string  `json:"command_template"`
	Description     *string `json:"description"`
	Enabled         *bool   `json:"enabled"`
	TimeoutSeconds  *int    `json:"timeout_seconds"`
	RetryCount      *int    `json:"retry_count"`
	ChangeSummary   string  `json:"change_summary"`
}

// UpdateConfig 更新部署配置（带版本记录）
func (s *DeployConfigService) UpdateConfig(ctx context.Context, id uint64, req UpdateConfigRequest, userID uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var config model.DeployConfig
		if err := tx.Where("id = ?", id).First(&config).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		// 保存版本历史
		version := model.DeployConfigVersion{
			ConfigID:        config.ID,
			StepKey:         config.StepKey,
			CommandTemplate: config.CommandTemplate,
			Description:     config.Description,
			ChangeType:      "update",
			ChangedBy:       userID,
		}
		if req.ChangeSummary != "" {
			version.ChangeSummary = &req.ChangeSummary
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}

		// 更新配置
		updates := map[string]any{
			"command_template": req.CommandTemplate,
		}
		if req.Description != nil {
			updates["description"] = *req.Description
		}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if req.TimeoutSeconds != nil {
			updates["timeout_seconds"] = *req.TimeoutSeconds
		}
		if req.RetryCount != nil {
			updates["retry_count"] = *req.RetryCount
		}

		return tx.Model(&model.DeployConfig{}).Where("id = ?", id).Updates(updates).Error
	})
}

// GetConfigVersions 获取配置的版本历史
func (s *DeployConfigService) GetConfigVersions(ctx context.Context, configID uint64) ([]model.DeployConfigVersion, error) {
	var versions []model.DeployConfigVersion
	if err := s.db.WithContext(ctx).
		Where("config_id = ?", configID).
		Order("changed_at DESC").
		Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// GetConfigVersionsByKey 根据step_key获取配置的版本历史
func (s *DeployConfigService) GetConfigVersionsByKey(ctx context.Context, stepKey string) ([]model.DeployConfigVersion, error) {
	var versions []model.DeployConfigVersion
	if err := s.db.WithContext(ctx).
		Where("step_key = ?", stepKey).
		Order("changed_at DESC").
		Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// ─── DeployRepository CRUD ───

// ListRepositories 获取所有仓库配置
func (s *DeployConfigService) ListRepositories(ctx context.Context, repoType string) ([]model.DeployRepository, error) {
	var repos []model.DeployRepository
	query := s.db.WithContext(ctx).Where("deleted_at IS NULL")
	if repoType != "" {
		query = query.Where("repo_type = ?", repoType)
	}
	if err := query.Order("priority ASC, id ASC").Find(&repos).Error; err != nil {
		return nil, err
	}
	return repos, nil
}

// GetRepository 根据ID获取仓库配置
func (s *DeployConfigService) GetRepository(ctx context.Context, id uint64) (*model.DeployRepository, error) {
	var repo model.DeployRepository
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&repo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &repo, nil
}

// CreateRepositoryRequest 创建仓库配置请求
type CreateRepositoryRequest struct {
	RepoType    string  `json:"repo_type" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	URL         string  `json:"url" binding:"required"`
	Description *string `json:"description"`
	AuthType    string  `json:"auth_type"`
	AuthConfig  *string `json:"auth_config"`
	Priority    int     `json:"priority"`
	IsDefault   bool    `json:"is_default"`
	MirrorOf    *string `json:"mirror_of"`
}

// CreateRepository 创建仓库配置
func (s *DeployConfigService) CreateRepository(ctx context.Context, req CreateRepositoryRequest, userID uint64) (uint64, error) {
	repo := model.DeployRepository{
		RepoType:    req.RepoType,
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		AuthType:    req.AuthType,
		Priority:    req.Priority,
		Enabled:     true,
		IsDefault:   req.IsDefault,
		MirrorOf:    req.MirrorOf,
		CreatedBy:   userID,
	}

	if req.AuthType == "" {
		repo.AuthType = "none"
	}
	if req.Priority == 0 {
		repo.Priority = 100
	}

	if err := s.db.WithContext(ctx).Create(&repo).Error; err != nil {
		return 0, err
	}
	return repo.ID, nil
}

// UpdateRepositoryRequest 更新仓库配置请求
type UpdateRepositoryRequest struct {
	Name        *string `json:"name"`
	URL         *string `json:"url"`
	Description *string `json:"description"`
	AuthType    *string `json:"auth_type"`
	AuthConfig  *string `json:"auth_config"`
	Priority    *int    `json:"priority"`
	Enabled     *bool   `json:"enabled"`
	IsDefault   *bool   `json:"is_default"`
	MirrorOf    *string `json:"mirror_of"`
}

// UpdateRepository 更新仓库配置
func (s *DeployConfigService) UpdateRepository(ctx context.Context, id uint64, req UpdateRepositoryRequest) error {
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.AuthType != nil {
		updates["auth_type"] = *req.AuthType
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}
	if req.MirrorOf != nil {
		updates["mirror_of"] = *req.MirrorOf
	}

	if len(updates) == 0 {
		return nil
	}

	result := s.db.WithContext(ctx).Model(&model.DeployRepository{}).
		Where("deleted_at IS NULL AND id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteRepository 软删除仓库配置
func (s *DeployConfigService) DeleteRepository(ctx context.Context, id uint64) error {
	result := s.db.WithContext(ctx).Model(&model.DeployRepository{}).
		Where("deleted_at IS NULL AND id = ?", id).
		Update("deleted_at", gorm.Expr("NOW(3)"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
