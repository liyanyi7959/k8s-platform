package application

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
	"k8s-platform-backend/internal/provisioning/ports"
)

type DeployConfigService struct{ repository ports.Repository }

func NewDeployConfigService(repository ports.Repository) *DeployConfigService {
	return &DeployConfigService{repository: repository}
}
func (s *DeployConfigService) ListConfigs(ctx context.Context, osType string) ([]provisiondomain.DeployConfig, error) {
	if osType != "" {
		return s.listConfigsWithFallback(ctx, osType)
	}
	return s.repository.ListDeployConfigs(ctx, nil)
}
func (s *DeployConfigService) listConfigsWithFallback(ctx context.Context, osType string) ([]provisiondomain.DeployConfig, error) {
	fallback := configFallbackTypes(osType)
	rows, err := s.repository.ListDeployConfigs(ctx, fallback)
	if err != nil {
		return nil, err
	}
	priority := map[string]int{}
	for i, item := range fallback {
		priority[item] = i
	}
	merged := map[string]provisiondomain.DeployConfig{}
	for _, row := range rows {
		existing, ok := merged[row.StepKey]
		if !ok || priority[row.OSType] < priority[existing.OSType] {
			merged[row.StepKey] = row
		}
	}
	configs := make([]provisiondomain.DeployConfig, 0, len(merged))
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
func (s *DeployConfigService) GetConfig(ctx context.Context, id uint64) (*provisiondomain.DeployConfig, error) {
	config, found, err := s.repository.FindDeployConfig(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return &config, nil
}
func (s *DeployConfigService) GetConfigByKey(ctx context.Context, stepKey, osType string) (*provisiondomain.DeployConfig, error) {
	types := []string(nil)
	if osType != "" {
		types = configFallbackTypes(osType)
	}
	rows, err := s.repository.FindDeployConfigByKey(ctx, stepKey, types)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	if osType == "" {
		return &rows[0], nil
	}
	priority := map[string]int{}
	for i, item := range types {
		priority[item] = i
	}
	best := rows[0]
	for _, row := range rows[1:] {
		if priority[row.OSType] < priority[best.OSType] {
			best = row
		}
	}
	return &best, nil
}
func (s *DeployConfigService) ListSupportedOSTypes(ctx context.Context) ([]string, error) {
	return s.repository.ListSupportedOSTypes(ctx)
}

type UpdateConfigRequest struct {
	CommandTemplate string  `json:"command_template"`
	Description     *string `json:"description"`
	Enabled         *bool   `json:"enabled"`
	TimeoutSeconds  *int    `json:"timeout_seconds"`
	RetryCount      *int    `json:"retry_count"`
	ChangeSummary   string  `json:"change_summary"`
}

func (s *DeployConfigService) UpdateConfig(ctx context.Context, id uint64, req UpdateConfigRequest, userID uint64) error {
	return s.repository.Transaction(ctx, func(tx ports.Repository) error {
		config, found, err := tx.FindDeployConfig(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return ErrNotFound
		}
		version := provisiondomain.DeployConfigVersion{ConfigID: config.ID, StepKey: config.StepKey, CommandTemplate: config.CommandTemplate, Description: config.Description, ChangeType: "update", ChangedBy: userID}
		if req.ChangeSummary != "" {
			version.ChangeSummary = &req.ChangeSummary
		}
		if err := tx.CreateDeployConfigVersion(ctx, &version); err != nil {
			return err
		}
		updates := map[string]any{"command_template": req.CommandTemplate}
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
		return tx.UpdateDeployConfig(ctx, id, updates)
	})
}
func (s *DeployConfigService) GetConfigVersions(ctx context.Context, configID uint64) ([]provisiondomain.DeployConfigVersion, error) {
	return s.repository.ListDeployConfigVersions(ctx, configID, "")
}
func (s *DeployConfigService) GetConfigVersionsByKey(ctx context.Context, stepKey string) ([]provisiondomain.DeployConfigVersion, error) {
	return s.repository.ListDeployConfigVersions(ctx, 0, stepKey)
}
func (s *DeployConfigService) ListRepositories(ctx context.Context, repoType string) ([]provisiondomain.DeployRepository, error) {
	return s.repository.ListDeployRepositories(ctx, repoType)
}
func (s *DeployConfigService) GetRepository(ctx context.Context, id uint64) (*provisiondomain.DeployRepository, error) {
	repo, found, err := s.repository.FindDeployRepository(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return &repo, nil
}

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

func (s *DeployConfigService) CreateRepository(ctx context.Context, req CreateRepositoryRequest, userID uint64) (uint64, error) {
	repo := provisiondomain.DeployRepository{RepoType: req.RepoType, Name: req.Name, URL: req.URL, Description: req.Description, AuthType: req.AuthType, Priority: req.Priority, Enabled: true, IsDefault: req.IsDefault, MirrorOf: req.MirrorOf, CreatedBy: userID}
	if repo.AuthType == "" {
		repo.AuthType = "none"
	}
	if repo.Priority == 0 {
		repo.Priority = 100
	}
	if err := s.repository.CreateDeployRepository(ctx, &repo); err != nil {
		return 0, err
	}
	return repo.ID, nil
}

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
	updated, err := s.repository.UpdateDeployRepository(ctx, id, updates)
	if err != nil {
		return err
	}
	if !updated {
		return ErrNotFound
	}
	return nil
}
func (s *DeployConfigService) DeleteRepository(ctx context.Context, id uint64) error {
	updated, err := s.repository.SoftDeleteDeployRepository(ctx, id, nowUTC())
	if err != nil {
		return err
	}
	if !updated {
		return ErrNotFound
	}
	return nil
}
func nowUTC() time.Time { return time.Now().UTC() }

func (s *DeployConfigService) ReadAnsiblePlaybook(ctx context.Context) (string, error) {
	data, err := os.ReadFile(s.ansiblePlaybookPath())
	if os.IsNotExist(err) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}
func (s *DeployConfigService) ReadAnsibleInventoryTemplate(ctx context.Context) (string, error) {
	data, err := os.ReadFile(filepath.Join(s.ansiblePlaybookDir(), "inventory.ini"))
	if os.IsNotExist(err) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}
func (s *DeployConfigService) CheckAnsibleEnv(ctx context.Context) (bool, string, error) {
	out, err := exec.CommandContext(ctx, "ansible-playbook", "--version").Output()
	if err != nil {
		return false, "", nil
	}
	lines := strings.Split(string(out), "\n")
	version := ""
	if len(lines) > 0 {
		version = strings.TrimSpace(lines[0])
	}
	return true, version, nil
}
func (s *DeployConfigService) ansiblePlaybookDir() string { return resolveAnsibleDir("") }
func (s *DeployConfigService) ansiblePlaybookPath() string {
	return filepath.Join(s.ansiblePlaybookDir(), "site.yml")
}

type AnsibleTreeNode struct {
	Name     string            `json:"name"`
	Path     string            `json:"path"`
	Type     string            `json:"type"`
	Content  string            `json:"content,omitempty"`
	Children []AnsibleTreeNode `json:"children,omitempty"`
}

func (s *DeployConfigService) ReadAnsibleTree(ctx context.Context) (*AnsibleTreeNode, error) {
	root := s.ansiblePlaybookDir()
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
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
	rel := strings.TrimPrefix(strings.TrimPrefix(fullPath, s.ansiblePlaybookDir()), string(filepath.Separator))
	if rel == "" {
		rel = s.ansiblePlaybookDir()
	}
	node := &AnsibleTreeNode{Name: info.Name(), Path: rel, Type: "file"}
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
			if err == nil {
				node.Children = append(node.Children, *child)
			}
		}
		return node, nil
	}
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
func resolveAnsibleDir(configured string) string {
	if configured != "" {
		return configured
	}
	candidates := []string{"ansible", filepath.Join("backend", "ansible")}
	if executable, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(executable), "ansible")}, candidates...)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "ansible"
}
