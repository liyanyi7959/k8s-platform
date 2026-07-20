package service

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
	if osType != "" {
		return s.listConfigsWithFallback(ctx, osType)
	}

	var configs []model.DeployConfig
	q := s.db.WithContext(ctx).Order("step_order ASC, os_type ASC")
	if err := q.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *DeployConfigService) listConfigsWithFallback(ctx context.Context, osType string) ([]model.DeployConfig, error) {
	fallbackTypes := configFallbackTypes(osType)
	var rows []model.DeployConfig
	if err := s.db.WithContext(ctx).
		Where("os_type IN ?", fallbackTypes).
		Order("step_order ASC, os_type ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	priority := make(map[string]int, len(fallbackTypes))
	for index, item := range fallbackTypes {
		priority[item] = index
	}

	merged := make(map[string]model.DeployConfig)
	for _, row := range rows {
		existing, ok := merged[row.StepKey]
		if !ok || priority[row.OSType] < priority[existing.OSType] {
			merged[row.StepKey] = row
		}
	}

	configs := make([]model.DeployConfig, 0, len(merged))
	for _, item := range merged {
		configs = append(configs, item)
	}
	sort.SliceStable(configs, func(i, j int) bool {
		if configs[i].StepOrder != configs[j].StepOrder {
			return configs[i].StepOrder < configs[j].StepOrder
		}
		return configs[i].StepKey < configs[j].StepKey
	})
	return configs, nil
}

func configFallbackTypes(osType string) []string {
	switch osType {
	case "centos", "rocky", "rhel", "almalinux":
		return []string{osType, "centos", "rocky", "rhel", "almalinux", "ubuntu"}
	case "debian", "ubuntu":
		return []string{osType, "ubuntu", "debian"}
	default:
		return []string{osType, "ubuntu"}
	}
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
	if osType == "" {
		var config model.DeployConfig
		if err := s.db.WithContext(ctx).Where("step_key = ?", stepKey).First(&config).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		return &config, nil
	}

	fallbackTypes := configFallbackTypes(osType)
	var rows []model.DeployConfig
	if err := s.db.WithContext(ctx).
		Where("step_key = ? AND os_type IN ?", stepKey, fallbackTypes).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}

	priority := make(map[string]int, len(fallbackTypes))
	for index, item := range fallbackTypes {
		priority[item] = index
	}
	best := rows[0]
	for _, row := range rows[1:] {
		if priority[row.OSType] < priority[best.OSType] {
			best = row
		}
	}
	return &best, nil
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

// ReadAnsiblePlaybook 读取 Ansible playbook（site.yml）内容。
func (s *DeployConfigService) ReadAnsiblePlaybook(ctx context.Context) (string, error) {
	path := s.ansiblePlaybookPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	return string(data), nil
}

// ReadAnsibleInventoryTemplate 读取 inventory 模板内容。
func (s *DeployConfigService) ReadAnsibleInventoryTemplate(ctx context.Context) (string, error) {
	path := filepath.Join(s.ansiblePlaybookDir(), "inventory.ini")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	return string(data), nil
}

// CheckAnsibleEnv 检查当前环境是否已安装 ansible-playbook。
func (s *DeployConfigService) CheckAnsibleEnv(ctx context.Context) (installed bool, version string, err error) {
	cmd := exec.CommandContext(ctx, "ansible-playbook", "--version")
	out, err := cmd.Output()
	if err != nil {
		return false, "", nil
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		version = strings.TrimSpace(lines[0])
	}
	return true, version, nil
}

// ansiblePlaybookDir 获取 Ansible playbook 目录路径。
func (s *DeployConfigService) ansiblePlaybookDir() string {
	return resolveAnsibleDir("")
}

// ansiblePlaybookPath 获取 site.yml 完整路径。
func (s *DeployConfigService) ansiblePlaybookPath() string {
	return filepath.Join(s.ansiblePlaybookDir(), "site.yml")
}

// AnsibleTreeNode 表示 Ansible 目录树中的一个节点。
type AnsibleTreeNode struct {
	Name     string            `json:"name"`
	Path     string            `json:"path"`
	Type     string            `json:"type"` // file | dir
	Content  string            `json:"content,omitempty"`
	Children []AnsibleTreeNode `json:"children,omitempty"`
}

// ReadAnsibleTree 递归读取 Ansible 目录树及文件内容。
func (s *DeployConfigService) ReadAnsibleTree(ctx context.Context) (*AnsibleTreeNode, error) {
	root := s.ansiblePlaybookDir()
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	node, err := s.readAnsibleTreeNode(root, info)
	if err != nil {
		return nil, err
	}
	node.Name = "ansible"
	node.Path = "ansible"
	return node, nil
}

func (s *DeployConfigService) readAnsibleTreeNode(fullPath string, info os.FileInfo) (*AnsibleTreeNode, error) {
	relPath := strings.TrimPrefix(fullPath, s.ansiblePlaybookDir())
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	if relPath == "" {
		relPath = s.ansiblePlaybookDir()
	}

	node := &AnsibleTreeNode{
		Name: info.Name(),
		Path: relPath,
		Type: "file",
	}
	if info.IsDir() {
		node.Type = "dir"
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			childInfo, err := entry.Info()
			if err != nil {
				continue
			}
			child, err := s.readAnsibleTreeNode(filepath.Join(fullPath, entry.Name()), childInfo)
			if err != nil {
				continue
			}
			node.Children = append(node.Children, *child)
		}
		return node, nil
	}

	// 只读取文本文件（.yml/.yaml/.ini/.j2/.conf）。
	ext := strings.ToLower(filepath.Ext(info.Name()))
	if ext == ".yml" || ext == ".yaml" || ext == ".ini" || ext == ".j2" || ext == ".conf" {
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		node.Content = string(data)
	}
	return node, nil
}
