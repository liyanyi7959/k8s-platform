package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type CredentialItem struct {
	ID        uint64        `json:"id"`
	Name      string        `json:"name"`
	Provider  string        `json:"provider"`
	AuthType  string        `json:"auth_type"`
	Desc      *string       `json:"desc,omitempty"`
	Meta      model.JSONMap `json:"meta,omitempty"`
	CreatedBy uint64        `json:"created_by"`
	CreatedAt string        `json:"created_at,omitempty"`
	UpdatedAt string        `json:"updated_at,omitempty"`
}

type CredentialDetail struct {
	CredentialItem
}

type ListCredentialsRequest struct {
	Page     int
	PageSize int
	Keyword  string
	Provider string
	AuthType string
	SortBy   string
	Order    string
}

type CreateCredentialRequest struct {
	Name      string
	Provider  string
	AuthType  string
	Desc      *string
	Data      map[string]any
	CreatedBy uint64
}

type PatchCredentialRequest struct {
	Name      *string
	Provider  *string
	AuthType  *string
	Desc      **string
	Data      *map[string]any
	UpdatedBy uint64
}

type CredentialService struct {
	db        *gorm.DB
	secretKey string
}

func NewCredentialService(db *gorm.DB, secretKey string) *CredentialService {
	return &CredentialService{db: db, secretKey: secretKey}
}

func (s *CredentialService) ListCredentials(ctx context.Context, req ListCredentialsRequest) (PageResult[CredentialItem], error) {
	if s.db == nil {
		return PageResult[CredentialItem]{}, errors.New("db is required")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.db.WithContext(ctx).Model(&model.Credential{}).Where("deleted_at IS NULL")

	if v := strings.TrimSpace(req.Provider); v != "" {
		q = q.Where("provider = ?", v)
	}
	if v := strings.TrimSpace(req.AuthType); v != "" {
		q = q.Where("auth_type = ?", v)
	}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		q = q.Where("(name LIKE ? OR `desc` LIKE ?)", "%"+kw+"%", "%"+kw+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[CredentialItem]{}, err
	}

	orderClause := "id desc"
	if req.SortBy == "created_at" {
		if strings.ToLower(req.Order) == "asc" {
			orderClause = "created_at asc"
		} else {
			orderClause = "created_at desc"
		}
	}

	var rows []model.Credential
	if err := q.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[CredentialItem]{}, err
	}

	list := make([]CredentialItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, CredentialItem{
			ID:        r.ID,
			Name:      r.Name,
			Provider:  r.Provider,
			AuthType:  r.AuthType,
			Desc:      r.Desc,
			Meta:      r.Meta,
			CreatedBy: r.CreatedBy,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return PageResult[CredentialItem]{List: list, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *CredentialService) GetCredential(ctx context.Context, id uint64) (CredentialDetail, error) {
	if s.db == nil {
		return CredentialDetail{}, errors.New("db is required")
	}
	if id == 0 {
		return CredentialDetail{}, ErrInvalidParams
	}
	var row model.Credential
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CredentialDetail{}, ErrNotFound
		}
		return CredentialDetail{}, err
	}
	return CredentialDetail{
		CredentialItem: CredentialItem{
			ID:        row.ID,
			Name:      row.Name,
			Provider:  row.Provider,
			AuthType:  row.AuthType,
			Desc:      row.Desc,
			Meta:      row.Meta,
			CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339),
		},
	}, nil
}

func (s *CredentialService) CreateCredential(ctx context.Context, req CreateCredentialRequest) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	name := strings.TrimSpace(req.Name)
	provider := strings.TrimSpace(req.Provider)
	authType := strings.TrimSpace(req.AuthType)
	if name == "" || provider == "" || authType == "" {
		return 0, ErrInvalidParams
	}
	if err := validateCredentialProvider(provider); err != nil {
		return 0, err
	}
	if err := validateCredentialAuthType(authType); err != nil {
		return 0, err
	}

	data := req.Data
	if data == nil {
		data = map[string]any{}
	}
	if err := validateCredentialData(authType, data); err != nil {
		return 0, err
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return 0, ErrInvalidParams
	}
	enc, err := encryptText(s.secretKey, string(dataJSON))
	if err != nil {
		return 0, err
	}
	meta := sanitizeCredentialMeta(data)

	created := model.Credential{
		Name:      name,
		Provider:  provider,
		AuthType:  authType,
		Desc:      req.Desc,
		Meta:      meta,
		DataEnc:   enc,
		CreatedBy: req.CreatedBy,
		UpdatedBy: req.CreatedBy,
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exists int64
		if err := tx.Model(&model.Credential{}).Where("deleted_at IS NULL AND name = ? AND provider = ?", name, provider).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return ErrConflict
		}
		return tx.Create(&created).Error
	}); err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (s *CredentialService) PatchCredential(ctx context.Context, id uint64, req PatchCredentialRequest) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.Credential
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		updates := map[string]any{}
		nextName := strings.TrimSpace(row.Name)
		nextProvider := strings.TrimSpace(row.Provider)
		nextAuthType := strings.TrimSpace(row.AuthType)

		if req.Name != nil {
			v := strings.TrimSpace(*req.Name)
			if v == "" {
				return ErrInvalidParams
			}
			nextName = v
			updates["name"] = v
		}
		if req.Provider != nil {
			v := strings.TrimSpace(*req.Provider)
			if v == "" {
				return ErrInvalidParams
			}
			if err := validateCredentialProvider(v); err != nil {
				return err
			}
			nextProvider = v
			updates["provider"] = v
		}
		if req.AuthType != nil {
			v := strings.TrimSpace(*req.AuthType)
			if v == "" {
				return ErrInvalidParams
			}
			if err := validateCredentialAuthType(v); err != nil {
				return err
			}
			nextAuthType = v
			updates["auth_type"] = v
		}
		if req.Desc != nil {
			updates["desc"] = *req.Desc
		}

		if (req.AuthType != nil) && req.Data == nil {
			return ErrWithMessage(ErrInvalidParams, "修改认证方式时必须同时更新内容")
		}

		if len(updates) > 0 && (req.Name != nil || req.Provider != nil) {
			var exists int64
			if err := tx.Model(&model.Credential{}).
				Where("deleted_at IS NULL AND id <> ? AND name = ? AND provider = ?", id, nextName, nextProvider).
				Count(&exists).Error; err != nil {
				return err
			}
			if exists > 0 {
				return ErrConflict
			}
		}

		if req.Data != nil {
			if err := validateCredentialData(nextAuthType, *req.Data); err != nil {
				return err
			}
			dataJSON, err := json.Marshal(*req.Data)
			if err != nil {
				return ErrInvalidParams
			}
			enc, err := encryptText(s.secretKey, string(dataJSON))
			if err != nil {
				return err
			}
			updates["data_enc"] = enc
			updates["meta"] = sanitizeCredentialMeta(*req.Data)
		}

		if req.UpdatedBy > 0 {
			updates["updated_by"] = req.UpdatedBy
		}

		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&model.Credential{}).Where("deleted_at IS NULL AND id = ?", id).Updates(updates).Error
	})
}

func (s *CredentialService) DeleteCredential(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	res := s.db.WithContext(ctx).Model(&model.Credential{}).Where("deleted_at IS NULL AND id = ?", id).Update("deleted_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *CredentialService) GetCredentialData(ctx context.Context, id uint64) (map[string]any, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	if id == 0 {
		return nil, ErrInvalidParams
	}
	var row model.Credential
	if err := s.db.WithContext(ctx).Select("data_enc").Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	pt, err := decryptText(s.secretKey, row.DataEnc)
	if err != nil {
		return nil, ErrWithMessage(ErrCrypto, "凭据解密失败")
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(pt), &out); err != nil {
		return nil, ErrWithMessage(ErrInvalidParams, "凭据内容格式错误")
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func sanitizeCredentialMeta(data map[string]any) model.JSONMap {
	if len(data) == 0 {
		return nil
	}
	secretHints := []string{"password", "passwd", "token", "secret", "private_key", "access_key", "secret_key", "apikey", "api_key"}
	sort.Strings(secretHints)
	out := model.JSONMap{}
	for k, v := range data {
		key := strings.ToLower(strings.TrimSpace(k))
		if key == "" {
			continue
		}
		drop := false
		for _, h := range secretHints {
			if strings.Contains(key, h) {
				drop = true
				break
			}
		}
		if drop {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func validateCredentialProvider(provider string) error {
	switch provider {
	case "git", "gitlab", "github", "gitea", "harbor", "docker_registry", "generic":
		return nil
	default:
		return ErrWithMessage(ErrInvalidParams, "provider 不支持")
	}
}

func validateCredentialAuthType(authType string) error {
	switch authType {
	case "access_token", "username_password", "ssh_private_key", "docker_config_json", "generic_json":
		return nil
	default:
		return ErrWithMessage(ErrInvalidParams, "auth_type 不支持")
	}
}

func validateCredentialData(authType string, data map[string]any) error {
	if data == nil {
		data = map[string]any{}
	}
	switch authType {
	case "access_token":
		v, ok := data["token"]
		if !ok || strings.TrimSpace(anyToString(v)) == "" {
			return ErrWithMessage(ErrInvalidParams, "access_token 方式需要 token")
		}
	case "username_password":
		u, okU := data["username"]
		p, okP := data["password"]
		if !okU || strings.TrimSpace(anyToString(u)) == "" {
			return ErrWithMessage(ErrInvalidParams, "用户名密码方式需要 username")
		}
		if !okP || strings.TrimSpace(anyToString(p)) == "" {
			return ErrWithMessage(ErrInvalidParams, "用户名密码方式需要 password")
		}
	case "ssh_private_key":
		k, ok := data["private_key"]
		if !ok || strings.TrimSpace(anyToString(k)) == "" {
			return ErrWithMessage(ErrInvalidParams, "ssh_private_key 方式需要 private_key")
		}
	case "docker_config_json":
		c, ok := data["config_json"]
		if !ok || strings.TrimSpace(anyToString(c)) == "" {
			return ErrWithMessage(ErrInvalidParams, "docker_config_json 方式需要 config_json")
		}
	case "generic_json":
		return nil
	default:
		return ErrWithMessage(ErrInvalidParams, "auth_type 不支持")
	}
	return nil
}

func anyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}
