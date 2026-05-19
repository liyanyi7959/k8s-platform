package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// SSH 哨兵错误已统一迁移至 errors.go（ErrSSHNetwork / ErrSSHTimeout / ErrSSHAuth）。

type ServerItem struct {
	// ServerItem 为对外返回的服务器列表项。
	// 注意：不会返回 password/private_key 等敏感字段（这些字段仅以加密形式存储在 DB 中）。
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	IP          string   `json:"ip"`
	Port        int      `json:"port"`
	AuthType    string   `json:"auth_type"`
	Username    string   `json:"username"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
	LastCheckAt *string  `json:"last_check_at,omitempty"`
	CreatedBy   uint64   `json:"created_by"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

type ServerDetail struct {
	// ServerDetail 为服务器详情结构（基于列表项扩展更新时间）。
	// 仍然不包含任何敏感认证信息字段。
	ServerItem
	UpdatedAt string `json:"updated_at,omitempty"`
}

type ServerSSHAuth struct {
	Addr       string
	Username   string
	AuthType   string
	Password   string
	PrivateKey string
	Status     string
}

type CheckSSHResult struct {
	OK          bool    `json:"ok"`
	CheckedAt   string  `json:"checked_at"`
	Status      string  `json:"status,omitempty"`
	LastCheckAt *string `json:"last_check_at,omitempty"`
	Message     string  `json:"message,omitempty"`
}

type ListServersRequest struct {
	// ListServersRequest 为列表查询参数。
	// controller 层通常从 query string 解析后传入。
	Page     int
	PageSize int
	Keyword  string
	Tag      string
	Status   string
	SortBy   string
	Order    string
}

type CreateServerRequest struct {
	// CreateServerRequest 为创建服务器入参。
	// password/private_key 为明文，仅用于本次请求；service 内部会加密后入库。
	Name       string
	IP         string
	Port       int
	AuthType   string
	Username   string
	Password   string
	PrivateKey string
	Tags       []string
	Status     string
	CreatedBy  uint64
}

type PatchServerRequest struct {
	// PatchServerRequest 为部分更新入参。
	// 其中 password/private_key 为 **string 用于表达“三态语义”：
	// - nil：不修改
	// - 指向 nil：清空
	// - 指向非空字符串：更新为新值（入库前加密）
	Name       *string
	IP         *string
	Port       *int
	AuthType   *string
	Username   *string
	Password   **string
	PrivateKey **string
	Tags       *[]string
	Status     *string
}

type ServerService struct {
	// db 为 GORM 数据库连接。
	// secretKey 用于对 password/private_key 进行对称加密存储（AES）。
	db        *gorm.DB
	secretKey string
}

func NewServerService(db *gorm.DB, secretKey string) *ServerService {
	// NewServerService 创建 ServerService。
	// secretKey 与 kubeconfigKey 复用同一配置项：保证部署侧只需维护一份密钥。
	return &ServerService{db: db, secretKey: secretKey}
}

func (s *ServerService) ListServers(ctx context.Context, req ListServersRequest) (PageResult[ServerItem], error) {
	// ListServers 分页列出服务器。
	// - keyword 支持 name/ip 模糊查询
	// - tag 为 tags(JSON 数组) 的包含查询（使用 LIKE 进行粗匹配）
	// - 默认按 id desc；支持 created_at 排序
	if s.db == nil {
		return PageResult[ServerItem]{}, errors.New("db is required")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)

	q := s.db.WithContext(ctx).Model(&model.Server{}).Where("deleted_at IS NULL")
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		q = q.Where("(name LIKE ? OR ip LIKE ?)", "%"+kw+"%", "%"+kw+"%")
	}
	if st := strings.TrimSpace(req.Status); st != "" {
		q = q.Where("status = ?", st)
	}
	if tag := strings.TrimSpace(req.Tag); tag != "" {
		q = q.Where("tags LIKE ?", "%\""+tag+"\"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[ServerItem]{}, err
	}

	orderClause := "id desc"
	if req.SortBy == "created_at" {
		if strings.ToLower(strings.TrimSpace(req.Order)) == "asc" {
			orderClause = "created_at asc"
		} else {
			orderClause = "created_at desc"
		}
	}

	var rows []model.Server
	if err := q.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[ServerItem]{}, err
	}

	out := make([]ServerItem, 0, len(rows))
	for i := range rows {
		out = append(out, toServerItem(&rows[i]))
	}
	return PageResult[ServerItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *ServerService) GetServer(ctx context.Context, id uint64) (ServerDetail, error) {
	// GetServer 获取单个服务器（不含敏感字段）。
	// 若记录被软删除（deleted_at 非空）则视为不存在。
	if s.db == nil {
		return ServerDetail{}, errors.New("db is required")
	}
	if id == 0 {
		return ServerDetail{}, ErrInvalidParams
	}
	var row model.Server
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ServerDetail{}, ErrNotFound
		}
		return ServerDetail{}, err
	}
	item := toServerItem(&row)
	return ServerDetail{
		ServerItem: item,
		UpdatedAt:  row.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// GetServerCredentials 获取服务器解密后的凭据（密码/私钥）。
// 注意：此方法返回敏感信息，仅限内部业务（如 Ansible/SSH 终端）调用，禁止直接暴露给 Controller。
func (s *ServerService) GetServerCredentials(ctx context.Context, id uint64) (string, string, error) {
	if s.db == nil {
		return "", "", errors.New("db is required")
	}
	var row model.Server
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		return "", "", err
	}

	pass := ""
	if row.PasswordEnc != nil && *row.PasswordEnc != "" {
		p, err := decryptText(s.secretKey, *row.PasswordEnc)
		if err != nil {
			return "", "", fmt.Errorf("decrypt password failed: %w", err)
		}
		pass = p
	}

	key := ""
	if row.PrivateKeyEnc != nil && *row.PrivateKeyEnc != "" {
		k, err := decryptText(s.secretKey, *row.PrivateKeyEnc)
		if err != nil {
			return "", "", fmt.Errorf("decrypt private_key failed: %w", err)
		}
		key = k
	}

	return pass, key, nil
}

func (s *ServerService) GetServerSSHAuth(ctx context.Context, id uint64) (ServerSSHAuth, error) {
	if s.db == nil {
		return ServerSSHAuth{}, errors.New("db is required")
	}
	if id == 0 {
		return ServerSSHAuth{}, ErrInvalidParams
	}
	var row model.Server
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ServerSSHAuth{}, ErrNotFound
		}
		return ServerSSHAuth{}, err
	}
	port := row.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(strings.TrimSpace(row.IP), strconv.Itoa(port))

	out := ServerSSHAuth{
		Addr:     addr,
		Username: strings.TrimSpace(row.Username),
		AuthType: strings.TrimSpace(row.AuthType),
		Status:   strings.TrimSpace(row.Status),
	}
	switch out.AuthType {
	case "password":
		if row.PasswordEnc == nil || strings.TrimSpace(*row.PasswordEnc) == "" {
			return ServerSSHAuth{}, ErrInvalidParams
		}
		plain, err := decryptText(s.secretKey, *row.PasswordEnc)
		if err != nil {
			return ServerSSHAuth{}, err
		}
		out.Password = plain
	case "key":
		if row.PrivateKeyEnc == nil || strings.TrimSpace(*row.PrivateKeyEnc) == "" {
			return ServerSSHAuth{}, ErrInvalidParams
		}
		plain, err := decryptText(s.secretKey, *row.PrivateKeyEnc)
		if err != nil {
			return ServerSSHAuth{}, err
		}
		out.PrivateKey = plain
	default:
		return ServerSSHAuth{}, ErrInvalidParams
	}
	return out, nil
}

func normalizeSSHErr(err error) error {
	if err == nil {
		return nil
	}
	var nerr net.Error
	if errors.As(err, &nerr) && nerr != nil && nerr.Timeout() {
		return ErrWithMessage(ErrSSHTimeout, "SSH 连接超时")
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "unable to authenticate") ||
		strings.Contains(lower, "no supported methods remain") ||
		strings.Contains(lower, "permission denied") {
		return ErrWithMessage(ErrSSHAuth, "SSH 凭据不正确")
	}
	if strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "no route to host") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "timeout") {
		return ErrWithMessage(ErrSSHNetwork, "SSH 网络连接失败")
	}
	return ErrWithMessage(ErrSSHNetwork, "SSH 连接失败")
}

func (s *ServerService) CheckSSH(ctx context.Context, id uint64) (CheckSSHResult, error) {
	if s.db == nil {
		return CheckSSHResult{}, errors.New("db is required")
	}
	info, err := s.GetServerSSHAuth(ctx, id)
	if err != nil {
		return CheckSSHResult{}, err
	}

	var authMethod ssh.AuthMethod
	switch info.AuthType {
	case "password":
		if strings.TrimSpace(info.Password) == "" {
			return CheckSSHResult{}, ErrInvalidParams
		}
		authMethod = ssh.Password(info.Password)
	case "key":
		if strings.TrimSpace(info.PrivateKey) == "" {
			return CheckSSHResult{}, ErrInvalidParams
		}
		signer, err := ssh.ParsePrivateKey([]byte(info.PrivateKey))
		if err != nil {
			return CheckSSHResult{}, ErrInvalidParams
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		return CheckSSHResult{}, ErrInvalidParams
	}

	checkedAt := time.Now().UTC()
	ok := true
	sshCfg := &ssh.ClientConfig{
		User:            info.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         12 * time.Second,
	}
	client, err := ssh.Dial("tcp", info.Addr, sshCfg)
	if err != nil {
		ok = false
	} else {
		_ = client.Close()
	}

	nextStatus := "active"
	if !ok {
		nextStatus = "disabled"
	}

	res := s.db.WithContext(ctx).Model(&model.Server{}).Where("deleted_at IS NULL AND id = ?", id).Updates(map[string]any{
		"last_check_at": checkedAt,
		"status":        nextStatus,
	})
	if res.Error != nil {
		return CheckSSHResult{}, res.Error
	}
	if res.RowsAffected == 0 {
		return CheckSSHResult{}, ErrNotFound
	}

	checkedAtStr := checkedAt.Format(time.RFC3339)
	lastCheckAt := checkedAtStr
	out := CheckSSHResult{
		OK:          ok,
		CheckedAt:   checkedAtStr,
		Status:      nextStatus,
		LastCheckAt: &lastCheckAt,
	}
	if !ok {
		nerr := normalizeSSHErr(err)
		if msg, ok := UserMessage(nerr); ok {
			out.Message = msg
		} else if nerr != nil {
			out.Message = nerr.Error()
		}
	}
	return out, nil
}

func (s *ServerService) CreateServer(ctx context.Context, req CreateServerRequest) (uint64, error) {
	// CreateServer 创建服务器。
	// 校验策略：
	// - name/ip/username 不能为空；ip 必须可 ParseIP；port 在 1..65535
	// - auth_type: password|key
	// - password 模式必须提供 password；key 模式必须提供 private_key
	// - status: active|disabled
	// 约束策略：
	// - name 唯一（与 migration 的 uk_servers_name 保持一致）
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	n := strings.TrimSpace(req.Name)
	ip := strings.TrimSpace(req.IP)
	user := strings.TrimSpace(req.Username)
	authType := strings.TrimSpace(req.AuthType)
	status := strings.TrimSpace(req.Status)
	if n == "" || ip == "" || user == "" {
		return 0, ErrInvalidParams
	}
	if net.ParseIP(ip) == nil {
		return 0, ErrInvalidParams
	}
	port := req.Port
	if port == 0 {
		port = 22
	}
	if port <= 0 || port > 65535 {
		return 0, ErrInvalidParams
	}
	if authType == "" {
		authType = "password"
	}
	if authType != "password" && authType != "key" {
		return 0, ErrInvalidParams
	}
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "disabled" {
		return 0, ErrInvalidParams
	}

	var passEnc *string
	var keyEnc *string
	switch authType {
	case "password":
		if strings.TrimSpace(req.Password) == "" {
			return 0, ErrInvalidParams
		}
		enc, err := encryptText(s.secretKey, req.Password)
		if err != nil {
			return 0, err
		}
		passEnc = &enc
	case "key":
		if strings.TrimSpace(req.PrivateKey) == "" {
			return 0, ErrInvalidParams
		}
		enc, err := encryptText(s.secretKey, req.PrivateKey)
		if err != nil {
			return 0, err
		}
		keyEnc = &enc
	}

	tagsJSON, err := marshalTags(req.Tags)
	if err != nil {
		return 0, ErrInvalidParams
	}

	var created model.Server
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Server
		if err := tx.Where("deleted_at IS NULL AND name = ?", n).First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		created = model.Server{
			Name:          n,
			IP:            ip,
			Port:          port,
			AuthType:      authType,
			Username:      user,
			PasswordEnc:   passEnc,
			PrivateKeyEnc: keyEnc,
			Tags:          tagsJSON,
			Status:        status,
			CreatedBy:     req.CreatedBy,
		}
		return tx.Create(&created).Error
	})
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (s *ServerService) PatchServer(ctx context.Context, id uint64, req PatchServerRequest) error {
	// PatchServer 部分更新服务器。
	// 注意点：
	// - auth_type 改变后会强制校验对应凭据仍然存在，避免出现“password 模式但无 password_enc”的脏数据
	// - password/private_key 支持清空（指向 nil），但清空后若与 auth_type 不匹配会返回参数错误
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.Server
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		updates := map[string]any{}
		name := strings.TrimSpace(row.Name)
		ip := strings.TrimSpace(row.IP)
		port := row.Port
		authType := strings.TrimSpace(row.AuthType)
		username := strings.TrimSpace(row.Username)
		status := strings.TrimSpace(row.Status)
		passEnc := row.PasswordEnc
		keyEnc := row.PrivateKeyEnc

		if req.Name != nil {
			v := strings.TrimSpace(*req.Name)
			if v == "" {
				return ErrInvalidParams
			}
			if v != name {
				var existing model.Server
				if err := tx.Select("id").Where("deleted_at IS NULL AND name = ? AND id <> ?", v, id).First(&existing).Error; err == nil {
					return ErrConflict
				} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			name = v
			updates["name"] = v
		}

		if req.IP != nil {
			v := strings.TrimSpace(*req.IP)
			if v == "" || net.ParseIP(v) == nil {
				return ErrInvalidParams
			}
			ip = v
			updates["ip"] = v
		}

		if req.Port != nil {
			v := *req.Port
			if v <= 0 || v > 65535 {
				return ErrInvalidParams
			}
			port = v
			updates["port"] = v
		}

		if req.AuthType != nil {
			v := strings.TrimSpace(*req.AuthType)
			if v != "password" && v != "key" {
				return ErrInvalidParams
			}
			authType = v
			updates["auth_type"] = v
		}

		if req.Username != nil {
			v := strings.TrimSpace(*req.Username)
			if v == "" {
				return ErrInvalidParams
			}
			username = v
			updates["username"] = v
		}

		if req.Status != nil {
			v := strings.TrimSpace(*req.Status)
			if v != "active" && v != "disabled" {
				return ErrInvalidParams
			}
			status = v
			updates["status"] = v
		}

		if req.Password != nil {
			if *req.Password == nil {
				passEnc = nil
				updates["password_enc"] = nil
			} else {
				plain := strings.TrimSpace(**req.Password)
				if plain == "" {
					return ErrInvalidParams
				}
				enc, err := encryptText(s.secretKey, plain)
				if err != nil {
					return err
				}
				passEnc = &enc
				updates["password_enc"] = &enc
			}
		}

		if req.PrivateKey != nil {
			if *req.PrivateKey == nil {
				keyEnc = nil
				updates["private_key_enc"] = nil
			} else {
				plain := strings.TrimSpace(**req.PrivateKey)
				if plain == "" {
					return ErrInvalidParams
				}
				enc, err := encryptText(s.secretKey, plain)
				if err != nil {
					return err
				}
				keyEnc = &enc
				updates["private_key_enc"] = &enc
			}
		}

		if req.Tags != nil {
			tagsJSON, err := marshalTags(*req.Tags)
			if err != nil {
				return ErrInvalidParams
			}
			updates["tags"] = tagsJSON
		}

		_ = name
		_ = ip
		_ = port
		_ = username
		_ = status

		switch authType {
		case "password":
			if passEnc == nil || strings.TrimSpace(*passEnc) == "" {
				return ErrInvalidParams
			}
		case "key":
			if keyEnc == nil || strings.TrimSpace(*keyEnc) == "" {
				return ErrInvalidParams
			}
		default:
			return ErrInvalidParams
		}

		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&model.Server{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
	})
}

func (s *ServerService) DeleteServer(ctx context.Context, id uint64) error {
	// DeleteServer 软删除服务器：写入 deleted_at。
	// 软删除优点：
	// - 保留历史数据，便于审计与恢复
	// - 避免外键/关联清理复杂化（未来可能与集群、任务等关联）
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.Server
		if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&model.Server{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", &now).Error
	})
}

func marshalTags(tags []string) (*string, error) {
	// marshalTags 将 tags 规范化并序列化为 JSON 字符串。
	// - 去空白、去重
	// - nil 表示“不设置”（数据库字段可为 NULL）
	if tags == nil {
		return nil, nil
	}
	clean := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, t := range tags {
		v := strings.TrimSpace(t)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		clean = append(clean, v)
	}
	b, err := json.Marshal(clean)
	if err != nil {
		return nil, err
	}
	s := string(b)
	return &s, nil
}

func parseTags(tagsJSON *string) []string {
	// parseTags 将数据库中存储的 JSON 字符串解析为 tags 数组。
	// 解析失败时返回空数组，避免因脏数据影响接口可用性。
	if tagsJSON == nil || strings.TrimSpace(*tagsJSON) == "" {
		return []string{}
	}
	var out []string
	if json.Unmarshal([]byte(*tagsJSON), &out) != nil {
		return []string{}
	}
	clean := make([]string, 0, len(out))
	seen := map[string]bool{}
	for _, t := range out {
		v := strings.TrimSpace(t)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		clean = append(clean, v)
	}
	return clean
}

func toServerItem(sv *model.Server) ServerItem {
	// toServerItem 将 DB 模型转换为对外返回结构。
	// 约束：不透出加密字段（password_enc/private_key_enc）。
	if sv == nil {
		return ServerItem{}
	}
	var last *string
	if sv.LastCheckAt != nil {
		v := sv.LastCheckAt.UTC().Format(time.RFC3339)
		last = &v
	}
	return ServerItem{
		ID:          sv.ID,
		Name:        sv.Name,
		IP:          sv.IP,
		Port:        sv.Port,
		AuthType:    sv.AuthType,
		Username:    sv.Username,
		Tags:        parseTags(sv.Tags),
		Status:      sv.Status,
		LastCheckAt: last,
		CreatedBy:   sv.CreatedBy,
		CreatedAt:   sv.CreatedAt.UTC().Format(time.RFC3339),
	}
}
