package mysql

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
	"k8s-platform-backend/internal/provisioning/ports"
)

// Repository is the MySQL/GORM implementation of the provisioning persistence
// boundary.  GORM is deliberately contained in this adapter package.
type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Transaction(ctx context.Context, fn func(ports.Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Repository{db: tx})
	})
}

func (r *Repository) ListAppTemplates(ctx context.Context, category string, offset, limit int) ([]provisiondomain.AppTemplate, int, error) {
	q := r.db.WithContext(ctx).Model(&provisiondomain.AppTemplate{}).Where("deleted_at IS NULL")
	if category != "" {
		q = q.Where("category = ?", category)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []provisiondomain.AppTemplate
	if err := q.Order("id desc").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, int(total), nil
}

func (r *Repository) FindAppTemplate(ctx context.Context, id uint64) (provisiondomain.AppTemplate, bool, error) {
	var row provisiondomain.AppTemplate
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error
	if err == nil {
		return row, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return provisiondomain.AppTemplate{}, false, nil
	}
	return provisiondomain.AppTemplate{}, false, err
}

func (r *Repository) AppTemplateNameExists(ctx context.Context, name string, excludeID uint64) (bool, error) {
	q := r.db.WithContext(ctx).Model(&provisiondomain.AppTemplate{}).Where("deleted_at IS NULL AND name = ?", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) CreateAppTemplate(ctx context.Context, template *provisiondomain.AppTemplate) error {
	return r.db.WithContext(ctx).Create(template).Error
}
func (r *Repository) UpdateAppTemplate(ctx context.Context, id uint64, updates map[string]any) (bool, error) {
	result := r.db.WithContext(ctx).Model(&provisiondomain.AppTemplate{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	return result.RowsAffected > 0, result.Error
}
func (r *Repository) SoftDeleteAppTemplate(ctx context.Context, id uint64, deletedAt time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&provisiondomain.AppTemplate{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", deletedAt)
	return result.RowsAffected > 0, result.Error
}
func (r *Repository) SeedBuiltinAppTemplates(ctx context.Context, templates []provisiondomain.AppTemplate) error {
	return r.Transaction(ctx, func(tx ports.Repository) error {
		for _, template := range templates {
			existing, found, err := tx.(*Repository).findAppTemplateByName(ctx, template.Name)
			if err != nil {
				return err
			}
			if !found {
				if err := tx.CreateAppTemplate(ctx, &template); err != nil {
					return err
				}
				continue
			}
			if template.DeployType != "helm" {
				continue
			}
			updates := map[string]any{}
			if existing.HelmRepoName == "" {
				updates["helm_repo_name"] = template.HelmRepoName
			}
			if existing.HelmRepoURL == "" {
				updates["helm_repo_url"] = template.HelmRepoURL
			}
			if existing.HelmValuesYAML == "" {
				updates["helm_values_yaml"] = template.HelmValuesYAML
			}
			if len(updates) > 0 {
				if _, err := tx.UpdateAppTemplate(ctx, existing.ID, updates); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
func (r *Repository) findAppTemplateByName(ctx context.Context, name string) (provisiondomain.AppTemplate, bool, error) {
	var row provisiondomain.AppTemplate
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND name = ?", name).First(&row).Error
	if err == nil {
		return row, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return provisiondomain.AppTemplate{}, false, nil
	}
	return provisiondomain.AppTemplate{}, false, err
}

func (r *Repository) ListCredentials(ctx context.Context, query ports.CredentialListQuery) ([]provisiondomain.SSHCredential, int, error) {
	q := r.db.WithContext(ctx).Model(&provisiondomain.SSHCredential{}).Where("deleted_at IS NULL")
	if query.Keyword != "" {
		pattern := "%" + query.Keyword + "%"
		q = q.Where("name LIKE ? OR username LIKE ?", pattern, pattern)
	}
	if query.AuthType != "" {
		q = q.Where("auth_type = ?", query.AuthType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []provisiondomain.SSHCredential
	if err := q.Order("id desc").Offset(query.Offset).Limit(query.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, int(total), nil
}
func (r *Repository) FindCredential(ctx context.Context, id uint64) (provisiondomain.SSHCredential, bool, error) {
	var row provisiondomain.SSHCredential
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error
	if err == nil {
		return row, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return provisiondomain.SSHCredential{}, false, nil
	}
	return provisiondomain.SSHCredential{}, false, err
}
func (r *Repository) CreateCredential(ctx context.Context, credential *provisiondomain.SSHCredential) error {
	return r.db.WithContext(ctx).Create(credential).Error
}
func (r *Repository) UpdateCredential(ctx context.Context, id uint64, updates map[string]any) (bool, error) {
	result := r.db.WithContext(ctx).Model(&provisiondomain.SSHCredential{}).Where("deleted_at IS NULL AND id = ?", id).Updates(updates)
	return result.RowsAffected > 0, result.Error
}
func (r *Repository) SoftDeleteCredentials(ctx context.Context, ids []uint64, deletedAt time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&provisiondomain.SSHCredential{}).Where("deleted_at IS NULL AND id IN ?", ids).Update("deleted_at", deletedAt)
	return result.RowsAffected, result.Error
}
func (r *Repository) CountServersByCredential(ctx context.Context, credentialID uint64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&provisiondomain.DeployServer{}).Where("deleted_at IS NULL AND credential_id = ?", credentialID).Count(&count).Error
	return int(count), err
}

func (r *Repository) ListServers(ctx context.Context, query ports.ServerListQuery) ([]provisiondomain.DeployServer, int, error) {
	q := r.db.WithContext(ctx).Model(&provisiondomain.DeployServer{}).Where("deleted_at IS NULL")
	if query.Keyword != "" {
		pattern := "%" + query.Keyword + "%"
		q = q.Where("name LIKE ? OR ip LIKE ?", pattern, pattern)
	}
	if query.Status != "" {
		q = q.Where("status = ?", query.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []provisiondomain.DeployServer
	if err := q.Order("id desc").Offset(query.Offset).Limit(query.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, int(total), nil
}
func (r *Repository) ServerSummary(ctx context.Context) (ports.ServerSummary, error) {
	var summary ports.ServerSummary
	err := r.db.WithContext(ctx).Model(&provisiondomain.DeployServer{}).Where("deleted_at IS NULL").Select(`COUNT(*) AS total,
		COALESCE(SUM(CASE WHEN status = 'available' THEN 1 ELSE 0 END), 0) AS available,
		COALESCE(SUM(CASE WHEN status = 'registered' THEN 1 ELSE 0 END), 0) AS registered,
		COALESCE(SUM(CASE WHEN status = 'unavailable' THEN 1 ELSE 0 END), 0) AS unavailable,
		COALESCE(SUM(cpu_cores), 0) AS cpu_cores, COALESCE(SUM(memory_mb), 0) AS memory_mb,
		COALESCE(SUM(disk_gb), 0) AS disk_gb`).Scan(&summary).Error
	return summary, err
}
func (r *Repository) FindServer(ctx context.Context, id uint64) (provisiondomain.DeployServer, bool, error) {
	var row provisiondomain.DeployServer
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error
	if err == nil {
		return row, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return provisiondomain.DeployServer{}, false, nil
	}
	return provisiondomain.DeployServer{}, false, err
}
func (r *Repository) ServerExists(ctx context.Context, ip string, port int, excludeID uint64) (bool, error) {
	q := r.db.WithContext(ctx).Model(&provisiondomain.DeployServer{}).Where("deleted_at IS NULL AND ip = ? AND ssh_port = ?", ip, port)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
func (r *Repository) CreateServer(ctx context.Context, server *provisiondomain.DeployServer) error {
	return r.db.WithContext(ctx).Create(server).Error
}
func (r *Repository) UpdateServer(ctx context.Context, id uint64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&provisiondomain.DeployServer{}).Where("id = ?", id).Updates(updates).Error
}
func (r *Repository) SoftDeleteServer(ctx context.Context, id uint64, deletedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&provisiondomain.DeployServer{}).Where("id = ?", id).Update("deleted_at", deletedAt).Error
}

func (r *Repository) ListDeployPlans(ctx context.Context, query ports.DeployPlanListQuery) ([]provisiondomain.DeployPlan, int, error) {
	q := r.db.WithContext(ctx).Model(&provisiondomain.DeployPlan{}).Where("deleted_at IS NULL")
	if query.Keyword != "" {
		pattern := "%" + query.Keyword + "%"
		q = q.Where("name LIKE ? OR cluster_name LIKE ?", pattern, pattern)
	}
	if query.Status != "" {
		q = q.Where("status = ?", query.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []provisiondomain.DeployPlan
	if err := q.Order("id desc").Offset(query.Offset).Limit(query.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, int(total), nil
}
func (r *Repository) FindDeployPlan(ctx context.Context, id uint64) (provisiondomain.DeployPlan, bool, error) {
	var row provisiondomain.DeployPlan
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error
	if err == nil {
		return row, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return provisiondomain.DeployPlan{}, false, nil
	}
	return provisiondomain.DeployPlan{}, false, err
}
func (r *Repository) DeployPlanClusterNameExists(ctx context.Context, name string, excludeID uint64) (bool, error) {
	q := r.db.WithContext(ctx).Model(&provisiondomain.DeployPlan{}).Where("deleted_at IS NULL AND cluster_name = ?", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
func (r *Repository) CountAvailableServers(ctx context.Context, ids []uint64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&provisiondomain.DeployServer{}).Where("deleted_at IS NULL AND id IN ? AND status IN ?", ids, []string{"available", "registered"}).Count(&count).Error
	return int(count), err
}
func (r *Repository) CreateDeployPlan(ctx context.Context, plan *provisiondomain.DeployPlan) error {
	return r.db.WithContext(ctx).Create(plan).Error
}
func (r *Repository) UpdateDeployPlan(ctx context.Context, id uint64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&provisiondomain.DeployPlan{}).Where("id = ?", id).Updates(updates).Error
}
func (r *Repository) ReplaceDeployPlanNodes(ctx context.Context, planID uint64, nodes []provisiondomain.DeployPlanNode) error {
	if err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Delete(&provisiondomain.DeployPlanNode{}).Error; err != nil {
		return err
	}
	if len(nodes) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&nodes).Error
}
func (r *Repository) ListDeployPlanNodes(ctx context.Context, planID uint64) ([]provisiondomain.DeployPlanNode, error) {
	var rows []provisiondomain.DeployPlanNode
	err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Order("sort_order asc, id asc").Find(&rows).Error
	return rows, err
}
func (r *Repository) SoftDeleteDeployPlan(ctx context.Context, id uint64, deletedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&provisiondomain.DeployPlan{}).Where("id = ?", id).Update("deleted_at", deletedAt).Error
}

func (r *Repository) ListDeployConfigs(ctx context.Context, osTypes []string) ([]provisiondomain.DeployConfig, error) {
	q := r.db.WithContext(ctx).Order("step_order ASC, os_type ASC")
	if len(osTypes) > 0 {
		q = q.Where("os_type IN ?", osTypes)
	}
	var rows []provisiondomain.DeployConfig
	err := q.Find(&rows).Error
	return rows, err
}
func (r *Repository) FindDeployConfig(ctx context.Context, id uint64) (provisiondomain.DeployConfig, bool, error) {
	var row provisiondomain.DeployConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err == nil {
		return row, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return provisiondomain.DeployConfig{}, false, nil
	}
	return provisiondomain.DeployConfig{}, false, err
}
func (r *Repository) FindDeployConfigByKey(ctx context.Context, stepKey string, osTypes []string) ([]provisiondomain.DeployConfig, error) {
	q := r.db.WithContext(ctx).Where("step_key = ?", stepKey)
	if len(osTypes) > 0 {
		q = q.Where("os_type IN ?", osTypes)
	}
	var rows []provisiondomain.DeployConfig
	err := q.Find(&rows).Error
	return rows, err
}
func (r *Repository) ListSupportedOSTypes(ctx context.Context) ([]string, error) {
	var values []string
	err := r.db.WithContext(ctx).Model(&provisiondomain.DeployConfig{}).Distinct().Pluck("os_type", &values).Error
	return values, err
}
func (r *Repository) CreateDeployConfigVersion(ctx context.Context, version *provisiondomain.DeployConfigVersion) error {
	return r.db.WithContext(ctx).Create(version).Error
}
func (r *Repository) UpdateDeployConfig(ctx context.Context, id uint64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&provisiondomain.DeployConfig{}).Where("id = ?", id).Updates(updates).Error
}
func (r *Repository) ListDeployConfigVersions(ctx context.Context, configID uint64, stepKey string) ([]provisiondomain.DeployConfigVersion, error) {
	q := r.db.WithContext(ctx).Order("changed_at DESC")
	if configID > 0 {
		q = q.Where("config_id = ?", configID)
	}
	if stepKey != "" {
		q = q.Where("step_key = ?", stepKey)
	}
	var rows []provisiondomain.DeployConfigVersion
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListDeployRepositories(ctx context.Context, repoType string) ([]provisiondomain.DeployRepository, error) {
	q := r.db.WithContext(ctx).Where("deleted_at IS NULL")
	if repoType != "" {
		q = q.Where("repo_type = ?", repoType)
	}
	var rows []provisiondomain.DeployRepository
	err := q.Order("priority ASC, id ASC").Find(&rows).Error
	return rows, err
}
func (r *Repository) FindDeployRepository(ctx context.Context, id uint64) (provisiondomain.DeployRepository, bool, error) {
	var row provisiondomain.DeployRepository
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error
	if err == nil {
		return row, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return provisiondomain.DeployRepository{}, false, nil
	}
	return provisiondomain.DeployRepository{}, false, err
}
func (r *Repository) CreateDeployRepository(ctx context.Context, repository *provisiondomain.DeployRepository) error {
	return r.db.WithContext(ctx).Create(repository).Error
}
func (r *Repository) UpdateDeployRepository(ctx context.Context, id uint64, updates map[string]any) (bool, error) {
	result := r.db.WithContext(ctx).Model(&provisiondomain.DeployRepository{}).Where("deleted_at IS NULL AND id = ?", id).Updates(updates)
	return result.RowsAffected > 0, result.Error
}
func (r *Repository) SoftDeleteDeployRepository(ctx context.Context, id uint64, deletedAt time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&provisiondomain.DeployRepository{}).Where("deleted_at IS NULL AND id = ?", id).Update("deleted_at", deletedAt)
	return result.RowsAffected > 0, result.Error
}

var _ ports.Repository = (*Repository)(nil)

// Keep this import-free helper local so query normalization stays in application.
func trimmed(value string) string { return strings.TrimSpace(value) }
