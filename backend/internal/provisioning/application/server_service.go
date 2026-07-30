package application

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net"
	"strings"
	"time"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
	"k8s-platform-backend/internal/provisioning/ports"
)

// ServerService owns registered provisioning hosts. Persistence is supplied by
// a repository so runtime use cases are independent of the database driver.
type ServerService struct {
	repository    ports.Repository
	encryptionKey string
}

func NewServerService(repository ports.Repository, encryptionKey string) *ServerService {
	return &ServerService{repository: repository, encryptionKey: encryptionKey}
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
	rows, total, err := s.repository.ListServers(ctx, ports.ServerListQuery{Keyword: strings.TrimSpace(req.Keyword), Status: strings.TrimSpace(req.Status), Offset: (page - 1) * pageSize, Limit: pageSize})
	if err != nil {
		return PageResult[DeployServerItem]{}, err
	}
	items := make([]DeployServerItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, deployServerToItem(row))
	}
	return PageResult[DeployServerItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}
func (s *ServerService) Summary(ctx context.Context) (DeployServerSummary, error) {
	summary, err := s.repository.ServerSummary(ctx)
	return DeployServerSummary(summary), err
}

func (s *ServerService) Create(ctx context.Context, req CreateDeployServerRequest) (uint64, error) {
	useReference := req.CredentialID != nil && *req.CredentialID > 0
	name, ip, user, authType, credential, err := normalizeServerInput(req.Name, req.IP, req.User, req.AuthType, req.Credential, useReference)
	if err != nil {
		return 0, err
	}
	port, err := validSSHPort(req.SSHPort)
	if err != nil {
		return 0, err
	}
	var credentialID *uint64
	if useReference {
		saved, found, err := s.repository.FindCredential(ctx, *req.CredentialID)
		if err != nil {
			return 0, err
		}
		if !found {
			return 0, ErrWithMessage(ErrNotFound, "指定的凭据不存在")
		}
		credentialID, authType, credential = req.CredentialID, saved.AuthType, saved.CredentialEnc
	} else if credential, err = encryptProvisioningSecret(s.encryptionKey, credential); err != nil {
		return 0, err
	}
	row := provisiondomain.DeployServer{Name: name, IP: ip, SSHPort: port, User: user, AuthType: authType, CredentialID: credentialID, CredentialEnc: credential, Status: "registered", Labels: provisiondomain.JSONMap(req.Labels), Remark: stringPtrOrNil(req.Remark)}
	err = s.repository.Transaction(ctx, func(tx ports.Repository) error {
		if err := ensureServerUnique(ctx, tx, ip, port, 0); err != nil {
			return err
		}
		return tx.CreateServer(ctx, &row)
	})
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}
func (s *ServerService) Get(ctx context.Context, id uint64) (DeployServerItem, error) {
	if id == 0 {
		return DeployServerItem{}, ErrInvalidParams
	}
	row, found, err := s.repository.FindServer(ctx, id)
	if err != nil {
		return DeployServerItem{}, err
	}
	if !found {
		return DeployServerItem{}, ErrNotFound
	}
	return deployServerToItem(row), nil
}

func (s *ServerService) Update(ctx context.Context, id uint64, req UpdateDeployServerRequest) error {
	if id == 0 {
		return ErrInvalidParams
	}
	return s.repository.Transaction(ctx, func(tx ports.Repository) error {
		row, found, err := tx.FindServer(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return ErrNotFound
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
			port, err = validSSHPort(*req.SSHPort)
			if err != nil {
				return err
			}
			updates["ssh_port"] = port
		}
		if ip != row.IP || port != row.SSHPort {
			if err := ensureServerUnique(ctx, tx, ip, port, id); err != nil {
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
				saved, found, err := tx.FindCredential(ctx, *credentialID)
				if err != nil {
					return err
				}
				if !found {
					return ErrWithMessage(ErrNotFound, "指定的凭据不存在")
				}
				updates["credential_id"] = *credentialID
				updates["auth_type"] = saved.AuthType
				updates["credential_enc"] = saved.CredentialEnc
				updates["status"] = "registered"
			}
		} else if req.Credential != nil {
			value := strings.TrimSpace(*req.Credential)
			if value == "" {
				return ErrWithMessage(ErrInvalidParams, "凭证不能为空")
			}
			encrypted, err := encryptProvisioningSecret(s.encryptionKey, value)
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
		return tx.UpdateServer(ctx, id, updates)
	})
}
func (s *ServerService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	return s.repository.Transaction(ctx, func(tx ports.Repository) error {
		row, found, err := tx.FindServer(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return ErrNotFound
		}
		if row.Status == "in_use" {
			return ErrWithMessage(ErrConflict, "服务器正在使用中，无法删除")
		}
		return tx.SoftDeleteServer(ctx, id, now)
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
func ensureServerUnique(ctx context.Context, repository ports.Repository, ip string, port int, excludeID uint64) error {
	exists, err := repository.ServerExists(ctx, ip, port, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return ErrWithMessage(ErrConflict, "同一 IP 和 SSH 端口已存在")
	}
	return nil
}
func deployServerToItem(row provisiondomain.DeployServer) DeployServerItem {
	return DeployServerItem{ID: row.ID, Name: row.Name, IP: row.IP, SSHPort: row.SSHPort, User: row.User, AuthType: row.AuthType, CredentialID: row.CredentialID, OS: row.OS, OSVersion: row.OSVersion, Kernel: row.Kernel, CPUCores: row.CPUCores, MemoryMB: row.MemoryMB, DiskGB: row.DiskGB, Status: row.Status, Labels: map[string]any(row.Labels), Remark: row.Remark, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
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
