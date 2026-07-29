package mysql

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/fleet/domain"
	"k8s-platform-backend/internal/fleet/ports"
)

type Registry struct {
	db  *gorm.DB
	key string
}

func NewRegistry(db *gorm.DB, key string) *Registry { return &Registry{db: db, key: key} }

func (r *Registry) List(ctx context.Context, filter ports.ListFilter) ([]domain.Cluster, int64, error) {
	if r.db == nil {
		return nil, 0, errors.New("db is required")
	}
	query := r.db.WithContext(ctx).Model(&clusterRow{}).Where("deleted_at IS NULL")
	if value := strings.TrimSpace(filter.Keyword); value != "" {
		query = query.Where("name LIKE ?", "%"+value+"%")
	}
	if value := strings.TrimSpace(filter.Status); value != "" {
		query = query.Where("status = ?", value)
	}
	if value := strings.TrimSpace(filter.Type); value != "" {
		query = query.Where("type = ?", value)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "id desc"
	if filter.SortBy == "created_at" && strings.EqualFold(filter.Order, "asc") {
		order = "created_at asc"
	} else if filter.SortBy == "created_at" {
		order = "created_at desc"
	}
	var rows []clusterRow
	if err := query.Order(order).Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]domain.Cluster, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomain(row))
	}
	return result, total, nil
}
func (r *Registry) CreateImported(ctx context.Context, cluster domain.Cluster, kubeconfig string) (uint64, error) {
	if r.db == nil {
		return 0, errors.New("db is required")
	}
	encrypted, err := encrypt(r.key, kubeconfig)
	if err != nil {
		return 0, err
	}
	row := fromDomain(cluster)
	row.KubeconfigEncrypted = &encrypted
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&clusterRow{}).Where("deleted_at IS NULL AND name = ?", cluster.Name).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return domain.ErrConflict
		}
		return tx.Create(&row).Error
	})
	return row.ID, err
}
func (r *Registry) Get(ctx context.Context, id uint64) (domain.Cluster, error) {
	if r.db == nil {
		return domain.Cluster{}, errors.New("db is required")
	}
	if id == 0 {
		return domain.Cluster{}, domain.ErrValidation
	}
	var row clusterRow
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Cluster{}, domain.ErrNotFound
		}
		return domain.Cluster{}, err
	}
	return toDomain(row), nil
}
func (r *Registry) Kubeconfig(ctx context.Context, id uint64) (string, error) {
	row, err := r.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if row.KubeconfigEncrypted == nil || strings.TrimSpace(*row.KubeconfigEncrypted) == "" {
		return "", domain.ErrNotFound
	}
	return decrypt(r.key, *row.KubeconfigEncrypted)
}
func (r *Registry) UpdateHealth(ctx context.Context, id uint64, apiOK bool, nodeTotal int, version string) error {
	status := "active"
	if !apiOK {
		status = "degraded"
	}
	updates := map[string]any{"status": status, "last_health_at": time.Now().UTC(), "node_count": nodeTotal}
	if version != "" {
		updates["k8s_version"] = version
	}
	return r.update(ctx, id, updates)
}
func (r *Registry) UpdateMonitorSource(ctx context.Context, id uint64, source, url, status string) error {
	return r.update(ctx, id, map[string]any{"monitor_source": source, "prometheus_url": url, "prometheus_status": status, "prometheus_detected_at": time.Now().UTC()})
}
func (r *Registry) Patch(ctx context.Context, id uint64, patch ports.Patch) error {
	if r.db == nil {
		return errors.New("db is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row clusterRow
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		updates := map[string]any{}
		if patch.Name != nil {
			var count int64
			if err := tx.Model(&clusterRow{}).Where("deleted_at IS NULL AND name = ? AND id <> ?", *patch.Name, id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return domain.ErrConflict
			}
			updates["name"] = *patch.Name
		}
		if patch.Kubeconfig != nil {
			if row.Type != "imported" {
				return domain.ErrValidation
			}
			encrypted, err := encrypt(r.key, *patch.Kubeconfig)
			if err != nil {
				return err
			}
			updates["kubeconfig_enc"] = &encrypted
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&clusterRow{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
	})
}
func (r *Registry) Delete(ctx context.Context, id uint64) error {
	if r.db == nil {
		return errors.New("db is required")
	}
	result := r.db.WithContext(ctx).Model(&clusterRow{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", time.Now().UTC())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *Registry) update(ctx context.Context, id uint64, values map[string]any) error {
	if r.db == nil {
		return errors.New("db is required")
	}
	result := r.db.WithContext(ctx).Model(&clusterRow{}).Where("id = ? AND deleted_at IS NULL", id).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type clusterRow struct {
	ID                   uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name                 string     `gorm:"column:name"`
	Type                 string     `gorm:"column:type"`
	Status               string     `gorm:"column:status"`
	KubeconfigEncrypted  *string    `gorm:"column:kubeconfig_enc"`
	K8sVersion           string     `gorm:"column:k8s_version"`
	Description          string     `gorm:"column:description"`
	NodeCount            int        `gorm:"column:node_count"`
	LastHealthAt         *time.Time `gorm:"column:last_health_at"`
	MonitorSource        string     `gorm:"column:monitor_source"`
	PrometheusURL        string     `gorm:"column:prometheus_url"`
	PrometheusStatus     string     `gorm:"column:prometheus_status"`
	PrometheusDetectedAt *time.Time `gorm:"column:prometheus_detected_at"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at"`
	DeletedAt            *time.Time `gorm:"column:deleted_at"`
}

func (clusterRow) TableName() string { return "clusters" }
func toDomain(row clusterRow) domain.Cluster {
	return domain.Cluster{ID: row.ID, Name: row.Name, Type: row.Type, Status: row.Status, KubeconfigEncrypted: row.KubeconfigEncrypted, K8sVersion: row.K8sVersion, Description: row.Description, NodeCount: row.NodeCount, LastHealthAt: row.LastHealthAt, MonitorSource: row.MonitorSource, PrometheusURL: row.PrometheusURL, PrometheusStatus: row.PrometheusStatus, PrometheusDetectedAt: row.PrometheusDetectedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: row.DeletedAt}
}
func fromDomain(row domain.Cluster) clusterRow {
	return clusterRow{Name: row.Name, Type: row.Type, Status: row.Status, Description: row.Description}
}
func encrypt(secret, plaintext string) (string, error) {
	block, err := aes.NewCipher(key(secret))
	if err != nil {
		return "", domain.ErrCrypto
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", domain.ErrCrypto
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", domain.ErrCrypto
	}
	return base64.StdEncoding.EncodeToString(append(nonce, gcm.Seal(nil, nonce, []byte(plaintext), nil)...)), nil
}
func decrypt(secret, encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", domain.ErrCrypto
	}
	block, err := aes.NewCipher(key(secret))
	if err != nil {
		return "", domain.ErrCrypto
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", domain.ErrCrypto
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", domain.ErrCrypto
	}
	return string(plain), nil
}
func key(secret string) []byte { value := sha256.Sum256([]byte(secret)); return value[:] }
