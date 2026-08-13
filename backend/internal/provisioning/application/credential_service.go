package application

import (
	"context"
	"strings"
	"time"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
	"k8s-platform-backend/internal/provisioning/ports"
)

// CredentialService owns reusable credentials. It stores only encrypted
// secret material and exposes metadata-only DTOs to HTTP adapters.
type CredentialService struct {
	repository    ports.Repository
	encryptionKey string
}

func NewCredentialService(repository ports.Repository, encryptionKey string) *CredentialService {
	return &CredentialService{repository: repository, encryptionKey: encryptionKey}
}

type CredentialItem struct {
	ID          uint64  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
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
	Type     string
}

type CreateCredentialRequest struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	AuthType   string `json:"auth_type"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
	Remark     string `json:"remark"`
}

type UpdateCredentialRequest struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	AuthType   string  `json:"auth_type"`
	Username   string  `json:"username"`
	Credential string  `json:"credential"`
	Remark     *string `json:"remark"`
}

// normalizeCredentialType 归一化凭据类型，空值默认为 ssh。
func normalizeCredentialType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "ssh":
		return "ssh"
	case "kubeconfig":
		return "kubeconfig"
	case "git":
		return "git"
	case "docker-registry", "docker_registry", "registry":
		return "docker-registry"
	case "token":
		return "token"
	default:
		return ""
	}
}

// validateCredentialFields 按凭据类型校验必填字段，返回 (username, credential, error)。
func validateCredentialFields(credType, authType, username, credential string) (string, string, error) {
	credential = strings.TrimSpace(credential)
	if credential == "" {
		return "", "", ErrWithMessage(ErrInvalidParams, "凭证不能为空")
	}
	switch credType {
	case "ssh":
		at := normalizeAuthType(authType)
		if at == "" {
			return "", "", ErrWithMessage(ErrInvalidParams, "SSH 认证类型必须为 password 或 key")
		}
		un := strings.TrimSpace(username)
		if un == "" {
			un = "root"
		}
		return un, credential, nil
	case "kubeconfig":
		// kubeconfig 类型不需要 username 和 auth_type
		return strings.TrimSpace(username), credential, nil
	case "git":
		un := strings.TrimSpace(username)
		if un == "" {
			return "", "", ErrWithMessage(ErrInvalidParams, "Git 凭据需要填写用户名")
		}
		return un, credential, nil
	case "docker-registry":
		un := strings.TrimSpace(username)
		if un == "" {
			return "", "", ErrWithMessage(ErrInvalidParams, "Registry 凭据需要填写用户名")
		}
		return un, credential, nil
	case "token":
		return strings.TrimSpace(username), credential, nil
	default:
		return "", "", ErrWithMessage(ErrInvalidParams, "不支持的凭据类型")
	}
}

func (s *CredentialService) List(ctx context.Context, req ListCredentialsRequest) (PageResult[CredentialItem], error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	rows, total, err := s.repository.ListCredentials(ctx, ports.CredentialListQuery{
		Keyword: strings.TrimSpace(req.Keyword), AuthType: strings.TrimSpace(req.AuthType),
		Type:   strings.TrimSpace(req.Type),
		Offset: (page - 1) * pageSize, Limit: pageSize,
	})
	if err != nil {
		return PageResult[CredentialItem]{}, err
	}
	items := make([]CredentialItem, 0, len(rows))
	for _, row := range rows {
		count, err := s.countServers(ctx, row.ID)
		if err != nil {
			return PageResult[CredentialItem]{}, err
		}
		items = append(items, credentialToItem(row, count))
	}
	return PageResult[CredentialItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *CredentialService) Create(ctx context.Context, req CreateCredentialRequest) (uint64, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "凭据名称不能为空")
	}
	credType := normalizeCredentialType(req.Type)
	if credType == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "凭据类型不支持")
	}
	username, credential, err := validateCredentialFields(credType, req.AuthType, req.Username, req.Credential)
	if err != nil {
		return 0, err
	}
	encrypted, err := encryptProvisioningSecret(s.encryptionKey, credential)
	if err != nil {
		return 0, err
	}
	authType := ""
	if credType == "ssh" {
		authType = normalizeAuthType(req.AuthType)
	}
	row := provisiondomain.Credential{
		Name: name, Type: credType, AuthType: authType, Username: username,
		CredentialEnc: encrypted, Remark: stringPtrOrNil(req.Remark),
	}
	if err := s.repository.CreateCredential(ctx, &row); err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *CredentialService) Get(ctx context.Context, id uint64) (CredentialItem, error) {
	if id == 0 {
		return CredentialItem{}, ErrInvalidParams
	}
	row, found, err := s.repository.FindCredential(ctx, id)
	if err != nil {
		return CredentialItem{}, err
	}
	if !found {
		return CredentialItem{}, ErrNotFound
	}
	count, err := s.countServers(ctx, row.ID)
	if err != nil {
		return CredentialItem{}, err
	}
	return credentialToItem(row, count), nil
}

func (s *CredentialService) Update(ctx context.Context, id uint64, req UpdateCredentialRequest) error {
	if id == 0 {
		return ErrInvalidParams
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ErrWithMessage(ErrInvalidParams, "凭据名称不能为空")
	}
	credType := normalizeCredentialType(req.Type)
	if credType == "" {
		return ErrWithMessage(ErrInvalidParams, "凭据类型不支持")
	}
	username, credential, err := validateCredentialFields(credType, req.AuthType, req.Username, req.Credential)
	if err != nil {
		return err
	}
	authType := ""
	if credType == "ssh" {
		authType = normalizeAuthType(req.AuthType)
	}
	updates := map[string]any{
		"name": name, "type": credType, "auth_type": authType,
		"username": username, "remark": req.Remark,
	}
	if credential != "" {
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

func credentialToItem(row provisiondomain.Credential, serverCount int) CredentialItem {
	t := row.Type
	if t == "" {
		t = "ssh"
	}
	return CredentialItem{
		ID: row.ID, Name: row.Name, Type: t, AuthType: row.AuthType, Username: row.Username,
		Remark: row.Remark, ServerCount: serverCount,
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
