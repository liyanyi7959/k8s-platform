package application

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	"gorm.io/gorm"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
)

// ServerService owns registered provisioning hosts. SSH probing and terminal
// transport remain runtime adapters and consume the persisted host afterward.
type ServerService struct {
	db            *gorm.DB
	encryptionKey string
}

func NewServerService(db *gorm.DB, encryptionKey string) *ServerService {
	return &ServerService{db: db, encryptionKey: encryptionKey}
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

type DeployServerSummary struct {
	Total       int64 `json:"total"`
	Available   int64 `json:"available"`
	Registered  int64 `json:"registered"`
	Unavailable int64 `json:"unavailable"`
	CPUCores    int64 `json:"cpu_cores"`
	MemoryMB    int64 `json:"memory_mb"`
	DiskGB      int64 `json:"disk_gb"`
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

func (s *ServerService) List(ctx context.Context, req ListDeployServersRequest) (PageResult[DeployServerItem], error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.db.WithContext(ctx).Model(&provisiondomain.DeployServer{}).Where("deleted_at IS NULL")
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		q = q.Where("name LIKE ? OR ip LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[DeployServerItem]{}, err
	}
	var rows []provisiondomain.DeployServer
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[DeployServerItem]{}, err
	}
	items := make([]DeployServerItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, deployServerToItem(row))
	}
	return PageResult[DeployServerItem]{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *ServerService) Summary(ctx context.Context) (DeployServerSummary, error) {
	var summary DeployServerSummary
	err := s.db.WithContext(ctx).Model(&provisiondomain.DeployServer{}).
		Where("deleted_at IS NULL").
		Select(`COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN status = 'available' THEN 1 ELSE 0 END), 0) AS available,
			COALESCE(SUM(CASE WHEN status = 'registered' THEN 1 ELSE 0 END), 0) AS registered,
			COALESCE(SUM(CASE WHEN status = 'unavailable' THEN 1 ELSE 0 END), 0) AS unavailable,
			COALESCE(SUM(cpu_cores), 0) AS cpu_cores,
			COALESCE(SUM(memory_mb), 0) AS memory_mb,
			COALESCE(SUM(disk_gb), 0) AS disk_gb`).
		Scan(&summary).Error
	return summary, err
}

func (s *ServerService) Create(ctx context.Context, req CreateDeployServerRequest) (uint64, error) {
	useCredentialReference := req.CredentialID != nil && *req.CredentialID > 0
	name, ip, user, authType, credential, err := normalizeServerInput(req.Name, req.IP, req.User, req.AuthType, req.Credential, useCredentialReference)
	if err != nil {
		return 0, err
	}
	port, err := validSSHPort(req.SSHPort)
	if err != nil {
		return 0, err
	}
	var credentialID *uint64
	if useCredentialReference {
		var savedCredential provisiondomain.SSHCredential
		if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", *req.CredentialID).First(&savedCredential).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, ErrWithMessage(ErrNotFound, "指定的凭据不存在")
			}
			return 0, err
		}
		credentialID, authType, credential = req.CredentialID, savedCredential.AuthType, savedCredential.CredentialEnc
	} else if credential, err = encryptProvisioningSecret(s.encryptionKey, credential); err != nil {
		return 0, err
	}
	row := provisiondomain.DeployServer{
		Name: name, IP: ip, SSHPort: port, User: user, AuthType: authType, CredentialID: credentialID, CredentialEnc: credential,
		Status: "registered", Labels: provisiondomain.JSONMap(req.Labels), Remark: stringPtrOrNil(req.Remark),
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureServerUnique(tx, ip, port, 0); err != nil {
			return err
		}
		return tx.Create(&row).Error
	}); err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *ServerService) Get(ctx context.Context, id uint64) (DeployServerItem, error) {
	if id == 0 {
		return DeployServerItem{}, ErrInvalidParams
	}
	var row provisiondomain.DeployServer
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeployServerItem{}, ErrNotFound
		}
		return DeployServerItem{}, err
	}
	return deployServerToItem(row), nil
}

func (s *ServerService) Update(ctx context.Context, id uint64, req UpdateDeployServerRequest) error {
	if id == 0 {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row provisiondomain.DeployServer
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		updates, ip, port := map[string]any{}, row.IP, row.SSHPort
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return ErrWithMessage(ErrInvalidParams, "服务器名称不能为空")
			}
			updates["name"] = name
		}
		if req.IP != nil {
			ip = strings.TrimSpace(*req.IP)
			if net.ParseIP(ip) == nil {
				return ErrWithMessage(ErrInvalidParams, "IP 地址格式不正确")
			}
			updates["ip"] = ip
		}
		if req.SSHPort != nil {
			var err error
			if port, err = validSSHPort(*req.SSHPort); err != nil {
				return err
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
			credentialID := *req.CredentialID
			if credentialID == nil || *credentialID == 0 {
				updates["credential_id"] = nil
			} else {
				var savedCredential provisiondomain.SSHCredential
				if err := tx.Where("deleted_at IS NULL AND id = ?", *credentialID).First(&savedCredential).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return ErrWithMessage(ErrNotFound, "指定的凭据不存在")
					}
					return err
				}
				updates["credential_id"] = *credentialID
				updates["auth_type"] = savedCredential.AuthType
				updates["credential_enc"] = savedCredential.CredentialEnc
				updates["status"] = "registered"
			}
		} else if req.Credential != nil {
			credential := strings.TrimSpace(*req.Credential)
			if credential == "" {
				return ErrWithMessage(ErrInvalidParams, "凭证不能为空")
			}
			encrypted, err := encryptProvisioningSecret(s.encryptionKey, credential)
			if err != nil {
				return err
			}
			updates["credential_id"] = nil
			updates["credential_enc"] = encrypted
			updates["status"] = "registered"
		}
		if req.Labels != nil {
			updates["labels"] = provisiondomain.JSONMap(req.Labels)
		}
		if req.Remark != nil {
			updates["remark"] = stringPtrOrNil(*req.Remark)
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&provisiondomain.DeployServer{}).Where("id = ?", id).Updates(updates).Error
	})
}

func (s *ServerService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row provisiondomain.DeployServer
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if row.Status == "in_use" {
			return ErrWithMessage(ErrConflict, "服务器正在使用中，无法删除")
		}
		return tx.Model(&provisiondomain.DeployServer{}).Where("id = ?", id).Update("deleted_at", &now).Error
	})
}

func validSSHPort(value int) (int, error) {
	if value == 0 {
		return 22, nil
	}
	if value < 1 || value > 65535 {
		return 0, ErrWithMessage(ErrInvalidParams, "SSH 端口范围必须为 1-65535")
	}
	return value, nil
}

func normalizeServerInput(name, ip, user, authType, credential string, skipCredentialCheck bool) (string, string, string, string, string, error) {
	name, ip, user, credential = strings.TrimSpace(name), strings.TrimSpace(ip), strings.TrimSpace(user), strings.TrimSpace(credential)
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

func normalizeAuthType(value string) string {
	switch strings.TrimSpace(value) {
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
	var existing provisiondomain.DeployServer
	if err := q.Select("id").First(&existing).Error; err == nil {
		return ErrWithMessage(ErrConflict, "同一 IP 和 SSH 端口已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func deployServerToItem(row provisiondomain.DeployServer) DeployServerItem {
	return DeployServerItem{
		ID: row.ID, Name: row.Name, IP: row.IP, SSHPort: row.SSHPort, User: row.User, AuthType: row.AuthType, CredentialID: row.CredentialID,
		OS: row.OS, OSVersion: row.OSVersion, Kernel: row.Kernel, CPUCores: row.CPUCores, MemoryMB: row.MemoryMB, DiskGB: row.DiskGB,
		Status: row.Status, Labels: map[string]any(row.Labels), Remark: row.Remark,
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func stringPtrOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func encryptProvisioningSecret(secret, plaintext string) (string, error) {
	sum := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(append(nonce, gcm.Seal(nil, nonce, []byte(plaintext), nil)...)), nil
}
