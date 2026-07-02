package service

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type DeployService struct {
	db              *gorm.DB
	encryptionKey   string
	taskStore       *TaskStore
	clusterRegistry *ClusterRegistryService
	deployConfig    *DeployConfigService
}

func NewDeployService(db *gorm.DB, encryptionKey string, taskStore *TaskStore, clusterRegistry *ClusterRegistryService) *DeployService {
	return &DeployService{db: db, encryptionKey: encryptionKey, taskStore: taskStore, clusterRegistry: clusterRegistry, deployConfig: NewDeployConfigService(db)}
}

func (s *DeployService) GetTaskStore() *TaskStore {
	return s.taskStore
}

type DeployServerItem struct {
	ID           uint64         `json:"id"`
	Name         string         `json:"name"`
	IP           string         `json:"ip"`
	SSHPort      int            `json:"ssh_port"`
	User         string         `json:"user"`
	AuthType     string         `json:"auth_type"`
	CredentialID *uint64        `json:"credential_id,omitempty"`
	OS           *string        `json:"os,omitempty"`
	OSVersion    *string        `json:"os_version,omitempty"`
	Kernel       *string        `json:"kernel,omitempty"`
	CPUCores     *uint          `json:"cpu_cores,omitempty"`
	MemoryMB     *uint64        `json:"memory_mb,omitempty"`
	DiskGB       *uint64        `json:"disk_gb,omitempty"`
	Status       string         `json:"status"`
	Labels       map[string]any `json:"labels,omitempty"`
	Remark       *string        `json:"remark,omitempty"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
}

type ListDeployServersRequest struct {
	Page     int
	PageSize int
	Keyword  string
	Status   string
}

type CreateDeployServerRequest struct {
	Name         string         `json:"name"`
	IP           string         `json:"ip"`
	SSHPort      int            `json:"ssh_port"`
	User         string         `json:"user"`
	AuthType     string         `json:"auth_type"`
	CredentialID *uint64        `json:"credential_id"`
	Credential   string         `json:"credential"`
	Labels       map[string]any `json:"labels"`
	Remark       string         `json:"remark"`
}

type UpdateDeployServerRequest struct {
	Name         *string        `json:"name"`
	IP           *string        `json:"ip"`
	SSHPort      *int           `json:"ssh_port"`
	User         *string        `json:"user"`
	AuthType     *string        `json:"auth_type"`
	CredentialID **uint64       `json:"credential_id"`
	Credential   *string        `json:"credential"`
	Labels       map[string]any `json:"labels"`
	Remark       *string        `json:"remark"`
}

func (s *DeployService) ListServers(ctx context.Context, req ListDeployServersRequest) (PageResult[DeployServerItem], error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.db.WithContext(ctx).Model(&model.DeployServer{}).Where("deleted_at IS NULL")
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		q = q.Where("name LIKE ? OR ip LIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	if st := strings.TrimSpace(req.Status); st != "" {
		q = q.Where("status = ?", st)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[DeployServerItem]{}, err
	}
	var rows []model.DeployServer
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[DeployServerItem]{}, err
	}
	items := make([]DeployServerItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, deployServerToItem(row))
	}
	return PageResult[DeployServerItem]{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *DeployService) CreateServer(ctx context.Context, req CreateDeployServerRequest) (uint64, error) {
	useCredentialRef := req.CredentialID != nil && *req.CredentialID > 0
	name, ip, user, authType, credential, err := normalizeServerInput(req.Name, req.IP, req.User, req.AuthType, req.Credential, useCredentialRef)
	if err != nil {
		return 0, err
	}
	port := req.SSHPort
	if port == 0 {
		port = 22
	}
	if port < 1 || port > 65535 {
		return 0, ErrWithMessage(ErrInvalidParams, "SSH 端口范围必须为 1-65535")
	}
	// 如果指定了 credential_id，从凭据表获取凭证
	var credentialID *uint64
	if req.CredentialID != nil && *req.CredentialID > 0 {
		var cred model.SSHCredential
		if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", *req.CredentialID).First(&cred).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, ErrWithMessage(ErrNotFound, "指定的凭据不存在")
			}
			return 0, err
		}
		credentialID = req.CredentialID
		authType = cred.AuthType
		credential = cred.CredentialEnc // 直接使用已加密的值
	} else {
		// 手动输入的凭证需要加密
		enc, err := encryptText(s.encryptionKey, credential)
		if err != nil {
			return 0, err
		}
		credential = enc
	}
	remark := stringPtrOrNil(req.Remark)
	row := model.DeployServer{Name: name, IP: ip, SSHPort: port, User: user, AuthType: authType, CredentialID: credentialID, CredentialEnc: credential, Status: "registered", Labels: model.JSONMap(req.Labels), Remark: remark}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureServerUnique(tx, ip, port, 0); err != nil {
			return err
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *DeployService) GetServer(ctx context.Context, id uint64) (DeployServerItem, error) {
	var row model.DeployServer
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeployServerItem{}, ErrNotFound
		}
		return DeployServerItem{}, err
	}
	return deployServerToItem(row), nil
}

func (s *DeployService) UpdateServer(ctx context.Context, id uint64, req UpdateDeployServerRequest) error {
	if id == 0 {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.DeployServer
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		updates := map[string]any{}
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return ErrWithMessage(ErrInvalidParams, "服务器名称不能为空")
			}
			updates["name"] = name
		}
		ip := row.IP
		port := row.SSHPort
		if req.IP != nil {
			ip = strings.TrimSpace(*req.IP)
			if net.ParseIP(ip) == nil {
				return ErrWithMessage(ErrInvalidParams, "IP 地址格式不正确")
			}
			updates["ip"] = ip
		}
		if req.SSHPort != nil {
			port = *req.SSHPort
			if port < 1 || port > 65535 {
				return ErrWithMessage(ErrInvalidParams, "SSH 端口范围必须为 1-65535")
			}
			updates["ssh_port"] = port
		}
		if ip != row.IP || port != row.SSHPort {
			if err := ensureServerUnique(tx, ip, port, id); err != nil {
				return err
			}
		}
		if req.User != nil {
			user := strings.TrimSpace(*req.User)
			if user == "" {
				user = "root"
			}
			updates["user"] = user
		}
		if req.AuthType != nil {
			authType := normalizeAuthType(*req.AuthType)
			if authType == "" {
				return ErrWithMessage(ErrInvalidParams, "认证类型必须为 password 或 key")
			}
			updates["auth_type"] = authType
		}
		if req.CredentialID != nil {
			// 使用已有凭据
			credID := *req.CredentialID
			if credID != nil && *credID > 0 {
				var cred model.SSHCredential
				if err := tx.Where("deleted_at IS NULL AND id = ?", *credID).First(&cred).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return ErrWithMessage(ErrNotFound, "指定的凭据不存在")
					}
					return err
				}
				updates["credential_id"] = *credID
				updates["auth_type"] = cred.AuthType
				updates["credential_enc"] = cred.CredentialEnc
				updates["status"] = "registered"
			} else {
				updates["credential_id"] = nil
			}
		} else if req.Credential != nil {
			credential := strings.TrimSpace(*req.Credential)
			if credential == "" {
				return ErrWithMessage(ErrInvalidParams, "凭证不能为空")
			}
			enc, err := encryptText(s.encryptionKey, credential)
			if err != nil {
				return err
			}
			updates["credential_id"] = nil
			updates["credential_enc"] = enc
			updates["status"] = "registered"
		}
		if req.Labels != nil {
			updates["labels"] = model.JSONMap(req.Labels)
		}
		if req.Remark != nil {
			updates["remark"] = stringPtrOrNil(*req.Remark)
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&model.DeployServer{}).Where("id = ?", id).Updates(updates).Error
	})
}

func (s *DeployService) DeleteServer(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.DeployServer
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if row.Status == "in_use" {
			return ErrWithMessage(ErrConflict, "服务器正在使用中，无法删除")
		}
		return tx.Model(&model.DeployServer{}).Where("id = ?", id).Update("deleted_at", &now).Error
	})
}

func (s *DeployService) MarkServerAvailable(ctx context.Context, id uint64) error {
	return s.updateServerStatus(ctx, id, "available")
}

func (s *DeployService) updateServerStatus(ctx context.Context, id uint64, status string) error {
	res := s.db.WithContext(ctx).Model(&model.DeployServer{}).Where("deleted_at IS NULL AND id = ?", id).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func deployServerToItem(row model.DeployServer) DeployServerItem {
	return DeployServerItem{
		ID: row.ID, Name: row.Name, IP: row.IP, SSHPort: row.SSHPort, User: row.User, AuthType: row.AuthType,
		CredentialID: row.CredentialID,
		OS:           row.OS, OSVersion: row.OSVersion, Kernel: row.Kernel, CPUCores: row.CPUCores, MemoryMB: row.MemoryMB, DiskGB: row.DiskGB,
		Status: row.Status, Labels: map[string]any(row.Labels), Remark: row.Remark,
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func normalizeServerInput(name, ip, user, authType, credential string, skipCredentialCheck bool) (string, string, string, string, string, error) {
	name = strings.TrimSpace(name)
	ip = strings.TrimSpace(ip)
	user = strings.TrimSpace(user)
	credential = strings.TrimSpace(credential)
	if name == "" {
		return "", "", "", "", "", ErrWithMessage(ErrInvalidParams, "服务器名称不能为空")
	}
	if net.ParseIP(ip) == nil {
		return "", "", "", "", "", ErrWithMessage(ErrInvalidParams, "IP 地址格式不正确")
	}
	if user == "" {
		user = "root"
	}
	authType = normalizeAuthType(authType)
	if authType == "" {
		return "", "", "", "", "", ErrWithMessage(ErrInvalidParams, "认证类型必须为 password 或 key")
	}
	if !skipCredentialCheck && credential == "" {
		return "", "", "", "", "", ErrWithMessage(ErrInvalidParams, "凭证不能为空")
	}
	return name, ip, user, authType, credential, nil
}

func normalizeAuthType(v string) string {
	switch strings.TrimSpace(v) {
	case "", "password":
		return "password"
	case "key":
		return "key"
	default:
		return ""
	}
}

func ensureServerUnique(tx *gorm.DB, ip string, port int, excludeID uint64) error {
	q := tx.Where("deleted_at IS NULL AND ip = ? AND ssh_port = ?", ip, port)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var existing model.DeployServer
	if err := q.Select("id").First(&existing).Error; err == nil {
		return ErrWithMessage(ErrConflict, "同一 IP 和 SSH 端口已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func stringPtrOrNil(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
