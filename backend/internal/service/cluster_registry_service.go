package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type ClusterItem struct {
	ID         uint64 `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	K8sVersion string `json:"k8s_version,omitempty"`
	NodeCount  int    `json:"node_count"`
	CreatedAt  string `json:"created_at,omitempty"`
}

type ClusterHealth struct {
	APIOk     bool `json:"api_ok"`
	NodeReady int  `json:"node_ready"`
	NodeTotal int  `json:"node_total"`
}

type ClusterDetail struct {
	ClusterItem
	LastHealthAt *string        `json:"last_health_at,omitempty"`
	Health       *ClusterHealth `json:"health,omitempty"`
}

type ListClustersRequest struct {
	Page     int
	PageSize int
	Keyword  string
	Status   string
	Type     string
	SortBy   string
	Order    string
}

type ClusterRegistryService struct {
	db            *gorm.DB
	kubeconfigKey string
}

func NewClusterRegistryService(db *gorm.DB, kubeconfigKey string) *ClusterRegistryService {
	return &ClusterRegistryService{db: db, kubeconfigKey: kubeconfigKey}
}

func (s *ClusterRegistryService) ListClusters(ctx context.Context, req ListClustersRequest) (PageResult[ClusterItem], error) {
	if s.db == nil {
		return PageResult[ClusterItem]{}, errors.New("db is required")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)

	q := s.db.WithContext(ctx).Model(&model.Cluster{}).Where("deleted_at IS NULL")
	kw := strings.TrimSpace(req.Keyword)
	if kw != "" {
		q = q.Where("name LIKE ?", "%"+kw+"%")
	}
	if st := strings.TrimSpace(req.Status); st != "" {
		q = q.Where("status = ?", st)
	}
	if tp := strings.TrimSpace(req.Type); tp != "" {
		q = q.Where("type = ?", tp)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[ClusterItem]{}, err
	}

	orderClause := "id desc"
	if req.SortBy == "created_at" {
		if strings.ToLower(req.Order) == "asc" {
			orderClause = "created_at asc"
		} else {
			orderClause = "created_at desc"
		}
	}

	var rows []model.Cluster
	if err := q.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[ClusterItem]{}, err
	}

	list := make([]ClusterItem, 0, len(rows))
	for _, c := range rows {
		list = append(list, ClusterItem{
			ID:         c.ID,
			Name:       c.Name,
			Type:       c.Type,
			Status:     c.Status,
			K8sVersion: c.K8sVersion,
			NodeCount:  c.NodeCount,
			CreatedAt:  c.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return PageResult[ClusterItem]{List: list, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *ClusterRegistryService) ImportCluster(ctx context.Context, name, kubeconfig string) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	n := strings.TrimSpace(name)
	kc := strings.TrimSpace(kubeconfig)
	if n == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "集群名称不能为空")
	}
	if kc == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "kubeconfig 不能为空")
	}
	enc, err := encryptText(s.kubeconfigKey, kc)
	if err != nil {
		return 0, err
	}

	var created model.Cluster
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Cluster
		if err := tx.Where("deleted_at IS NULL AND name = ?", n).First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		created = model.Cluster{
			Name:          n,
			Type:          "imported",
			Status:        "active",
			KubeconfigEnc: &enc,
		}
		return tx.Create(&created).Error
	})
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (s *ClusterRegistryService) GetCluster(ctx context.Context, id uint64) (ClusterDetail, error) {
	if s.db == nil {
		return ClusterDetail{}, errors.New("db is required")
	}
	if id == 0 {
		return ClusterDetail{}, ErrWithMessage(ErrInvalidParams, "集群ID无效")
	}
	var c model.Cluster
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ClusterDetail{}, ErrNotFound
		}
		return ClusterDetail{}, err
	}
	var last *string
	if c.LastHealthAt != nil {
		v := c.LastHealthAt.UTC().Format(time.RFC3339)
		last = &v
	}
	return ClusterDetail{
		ClusterItem: ClusterItem{
			ID:        c.ID,
			Name:      c.Name,
			Type:      c.Type,
			Status:    c.Status,
			CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
		},
		LastHealthAt: last,
		Health:       nil,
	}, nil
}

func (s *ClusterRegistryService) GetKubeconfig(ctx context.Context, id uint64) (string, error) {
	if s.db == nil {
		return "", errors.New("db is required")
	}
	if id == 0 {
		return "", ErrWithMessage(ErrInvalidParams, "集群ID无效")
	}
	var c model.Cluster
	if err := s.db.WithContext(ctx).Select("id", "kubeconfig_enc").Where("deleted_at IS NULL AND id = ?", id).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	if c.KubeconfigEnc == nil || strings.TrimSpace(*c.KubeconfigEnc) == "" {
		return "", ErrNotFound
	}
	return decryptText(s.kubeconfigKey, *c.KubeconfigEnc)
}

func (s *ClusterRegistryService) UpdateClusterHealth(ctx context.Context, id uint64, apiOK bool, nodeReady, nodeTotal int, k8sVersion string) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "集群ID无效")
	}
	status := "active"
	if !apiOK {
		status = "degraded"
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"status":         status,
		"last_health_at": &now,
		"node_count":     nodeTotal,
	}
	if k8sVersion != "" {
		updates["k8s_version"] = k8sVersion
	}
	return s.db.WithContext(ctx).Model(&model.Cluster{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
}

type PatchClusterRequest struct {
	Name       *string
	Kubeconfig *string
}

func (s *ClusterRegistryService) PatchCluster(ctx context.Context, id uint64, req PatchClusterRequest) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "集群ID无效")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.Cluster
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		updates := map[string]any{}
		if req.Name != nil {
			v := strings.TrimSpace(*req.Name)
			if v == "" {
				return ErrWithMessage(ErrInvalidParams, "集群名称不能为空")
			}
			if v != strings.TrimSpace(row.Name) {
				var existing model.Cluster
				if err := tx.Select("id").Where("deleted_at IS NULL AND name = ? AND id <> ?", v, id).First(&existing).Error; err == nil {
					return ErrConflict
				} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			updates["name"] = v
		}

		if req.Kubeconfig != nil {
			if strings.TrimSpace(row.Type) != "imported" {
				return ErrWithMessage(ErrInvalidParams, "仅导入集群支持更新凭据")
			}
			kc := strings.TrimSpace(*req.Kubeconfig)
			if kc == "" {
				return ErrWithMessage(ErrInvalidParams, "kubeconfig 不能为空")
			}
			enc, err := encryptText(s.kubeconfigKey, kc)
			if err != nil {
				return err
			}
			updates["kubeconfig_enc"] = &enc
		}

		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&model.Cluster{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
	})
}

func (s *ClusterRegistryService) DeleteCluster(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "集群ID无效")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.Cluster
		if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&model.Cluster{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", &now).Error
	})
}
