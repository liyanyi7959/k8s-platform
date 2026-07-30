package application

import (
	"context"
	"strings"
	"time"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
	"k8s-platform-backend/internal/provisioning/ports"
)

// CredentialService owns reusable SSH credentials. It stores only encrypted
// secret material and exposes metadata-only DTOs to HTTP adapters.
type CredentialService struct {
	repository    ports.Repository
	encryptionKey string
}

func NewCredentialService(repository ports.Repository, encryptionKey string) *CredentialService {
	return &CredentialService{repository: repository, encryptionKey: encryptionKey}
}

type SSHCredentialItem struct {
	ID          uint64  `json:"id"`
	Name        string  `json:"name"`
	AuthType    string  `json:"auth_type"`
	Username    string  `json:"username"`
	Remark      *string `json:"remark,omitempty"`
	ServerCount int     `json:"server_count"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type ListCredentialsRequest struct {
	Page     int
	PageSize int
	Keyword  string
	AuthType string
}

type CreateSSHCredentialRequest struct {
	Name       string `json:"name"`
	AuthType   string `json:"auth_type"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
	Remark     string `json:"remark"`
}

type UpdateSSHCredentialRequest struct {
	Name       string  `json:"name"`
	AuthType   string  `json:"auth_type"`
	Username   string  `json:"username"`
	Credential string  `json:"credential"`
	Remark     *string `json:"remark"`
}

func (s *CredentialService) List(ctx context.Context, req ListCredentialsRequest) (PageResult[SSHCredentialItem], error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	rows, total, err := s.repository.ListCredentials(ctx, ports.CredentialListQuery{
		Keyword: strings.TrimSpace(req.Keyword), AuthType: strings.TrimSpace(req.AuthType), Offset: (page - 1) * pageSize, Limit: pageSize,
	})
	if err != nil {
		return PageResult[SSHCredentialItem]{}, err
	}
	items := make([]SSHCredentialItem, 0, len(rows))
	for _, row := range rows {
		count, err := s.countServers(ctx, row.ID)
		if err != nil {
			return PageResult[SSHCredentialItem]{}, err
		}
		items = append(items, sshCredentialToItem(row, count))
	}
	return PageResult[SSHCredentialItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *CredentialService) Create(ctx context.Context, req CreateSSHCredentialRequest) (uint64, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "凭据名称不能为空")
	}
	authType := normalizeAuthType(req.AuthType)
	if authType == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "认证类型必须为 password 或 key")
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = "root"
	}
	credential := strings.TrimSpace(req.Credential)
	if credential == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "凭证不能为空")
	}
	encrypted, err := encryptProvisioningSecret(s.encryptionKey, credential)
	if err != nil {
		return 0, err
	}
	row := provisiondomain.SSHCredential{Name: name, AuthType: authType, Username: username, CredentialEnc: encrypted, Remark: stringPtrOrNil(req.Remark)}
	if err := s.repository.CreateCredential(ctx, &row); err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *CredentialService) Get(ctx context.Context, id uint64) (SSHCredentialItem, error) {
	if id == 0 {
		return SSHCredentialItem{}, ErrInvalidParams
	}
	row, found, err := s.repository.FindCredential(ctx, id)
	if err != nil {
		return SSHCredentialItem{}, err
	}
	if !found {
		return SSHCredentialItem{}, ErrNotFound
	}
	count, err := s.countServers(ctx, row.ID)
	if err != nil {
		return SSHCredentialItem{}, err
	}
	return sshCredentialToItem(row, count), nil
}

func (s *CredentialService) Update(ctx context.Context, id uint64, req UpdateSSHCredentialRequest) error {
	if id == 0 {
		return ErrInvalidParams
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ErrWithMessage(ErrInvalidParams, "凭据名称不能为空")
	}
	authType := normalizeAuthType(req.AuthType)
	if authType == "" {
		return ErrWithMessage(ErrInvalidParams, "认证类型必须为 password 或 key")
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = "root"
	}
	updates := map[string]any{"name": name, "auth_type": authType, "username": username, "remark": req.Remark}
	if credential := strings.TrimSpace(req.Credential); credential != "" {
		encrypted, err := encryptProvisioningSecret(s.encryptionKey, credential)
		if err != nil {
			return err
		}
		updates["credential_enc"] = encrypted
	}
	updated, err := s.repository.UpdateCredential(ctx, id, updates)
	if err != nil {
		return err
	}
	if !updated {
		return ErrNotFound
	}
	return nil
}

func (s *CredentialService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	updated, err := s.repository.SoftDeleteCredentials(ctx, []uint64{id}, now)
	if err != nil {
		return err
	}
	if updated == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *CredentialService) BatchDelete(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	updated, err := s.repository.SoftDeleteCredentials(ctx, ids, now)
	if err != nil {
		return err
	}
	if updated == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *CredentialService) countServers(ctx context.Context, credentialID uint64) (int, error) {
	return s.repository.CountServersByCredential(ctx, credentialID)
}

func sshCredentialToItem(row provisiondomain.SSHCredential, serverCount int) SSHCredentialItem {
	return SSHCredentialItem{
		ID: row.ID, Name: row.Name, AuthType: row.AuthType, Username: row.Username, Remark: row.Remark, ServerCount: serverCount,
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
