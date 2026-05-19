package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type PlaybookTemplateItem struct {
	ID             uint64  `json:"id"`
	Name           string  `json:"name"`
	Scenario       string  `json:"scenario"`
	Description    *string `json:"description,omitempty"`
	CurrentVersion string  `json:"current_version"`
	CreatedBy      uint64  `json:"created_by"`
	CreatedAt      string  `json:"created_at,omitempty"`
	UpdatedAt      string  `json:"updated_at,omitempty"`
}

type PlaybookTemplateDetail struct {
	PlaybookTemplateItem
}

type PlaybookTemplateVersionItem struct {
	Version     string         `json:"version"`
	CreatedAt   string         `json:"created_at,omitempty"`
	Source      model.JSONMap  `json:"source,omitempty"`
	ParamsSchema model.JSONMap `json:"params_schema,omitempty"`
	Defaults    model.JSONMap  `json:"defaults,omitempty"`
	CreatedBy   uint64         `json:"created_by"`
}

type ListPlaybookTemplatesRequest struct {
	Page     int
	PageSize int
	Keyword  string
	Scenario string
	SortBy   string
	Order    string
}

type CreatePlaybookTemplateRequest struct {
	Name        string
	Scenario    string
	Description *string
	Version     string
	Source      model.JSONMap
	ParamsSchema model.JSONMap
	Defaults    model.JSONMap
	CreatedBy   uint64
}

type PatchPlaybookTemplateRequest struct {
	Name        *string
	Scenario    *string
	Description **string
	UpdatedBy   uint64
}

type CreatePlaybookTemplateVersionRequest struct {
	Version      string
	Source       model.JSONMap
	ParamsSchema model.JSONMap
	Defaults     model.JSONMap
	CreatedBy    uint64
}

type RollbackPlaybookTemplateRequest struct {
	Version   string
	UpdatedBy uint64
}

type UploadPlaybookResult struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type PlaybookTemplateService struct {
	db        *gorm.DB
	secretKey string
}

func NewPlaybookTemplateService(db *gorm.DB, secretKey string) *PlaybookTemplateService {
	return &PlaybookTemplateService{db: db, secretKey: secretKey}
}

func (s *PlaybookTemplateService) List(ctx context.Context, req ListPlaybookTemplatesRequest) (PageResult[PlaybookTemplateItem], error) {
	if s.db == nil {
		return PageResult[PlaybookTemplateItem]{}, errors.New("db is required")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.db.WithContext(ctx).Model(&model.PlaybookTemplate{}).Where("deleted_at IS NULL")
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		q = q.Where("(name LIKE ? OR description LIKE ?)", "%"+kw+"%", "%"+kw+"%")
	}
	if sc := strings.TrimSpace(req.Scenario); sc != "" {
		q = q.Where("scenario = ?", sc)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[PlaybookTemplateItem]{}, err
	}
	orderClause := "id desc"
	if req.SortBy == "created_at" {
		if strings.ToLower(req.Order) == "asc" {
			orderClause = "created_at asc"
		} else {
			orderClause = "created_at desc"
		}
	}
	var rows []model.PlaybookTemplate
	if err := q.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[PlaybookTemplateItem]{}, err
	}
	out := make([]PlaybookTemplateItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, PlaybookTemplateItem{
			ID:             r.ID,
			Name:           r.Name,
			Scenario:       r.Scenario,
			Description:    r.Description,
			CurrentVersion: r.CurrentVersion,
			CreatedBy:      r.CreatedBy,
			CreatedAt:      r.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:      r.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return PageResult[PlaybookTemplateItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *PlaybookTemplateService) Get(ctx context.Context, id uint64) (PlaybookTemplateDetail, error) {
	if s.db == nil {
		return PlaybookTemplateDetail{}, errors.New("db is required")
	}
	if id == 0 {
		return PlaybookTemplateDetail{}, ErrInvalidParams
	}
	var row model.PlaybookTemplate
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlaybookTemplateDetail{}, ErrNotFound
		}
		return PlaybookTemplateDetail{}, err
	}
	return PlaybookTemplateDetail{
		PlaybookTemplateItem: PlaybookTemplateItem{
			ID:             row.ID,
			Name:           row.Name,
			Scenario:       row.Scenario,
			Description:    row.Description,
			CurrentVersion: row.CurrentVersion,
			CreatedBy:      row.CreatedBy,
			CreatedAt:      row.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:      row.UpdatedAt.UTC().Format(time.RFC3339),
		},
	}, nil
}

func (s *PlaybookTemplateService) Create(ctx context.Context, req CreatePlaybookTemplateRequest) (uint64, string, error) {
	if s.db == nil {
		return 0, "", errors.New("db is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, "", ErrInvalidParams
	}
	sc := strings.TrimSpace(req.Scenario)
	if sc == "" {
		sc = "service_install"
	}
	if err := validateTemplateScenario(sc); err != nil {
		return 0, "", err
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		version = "v1.0"
	}
	if err := validateTemplateVersion(version); err != nil {
		return 0, "", err
	}
	if err := validateSourceMap(req.Source); err != nil {
		return 0, "", err
	}

	created := model.PlaybookTemplate{
		Name:           name,
		Scenario:       sc,
		Description:    req.Description,
		CurrentVersion: version,
		CreatedBy:      req.CreatedBy,
		UpdatedBy:      req.CreatedBy,
	}
	ver := model.PlaybookTemplateVersion{
		Version:      version,
		Source:       req.Source,
		ParamsSchema: nonNilJSON(req.ParamsSchema),
		Defaults:     nonNilJSON(req.Defaults),
		CreatedBy:    req.CreatedBy,
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exists int64
		if err := tx.Model(&model.PlaybookTemplate{}).Where("deleted_at IS NULL AND name = ?", name).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return ErrConflict
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		ver.TemplateID = created.ID
		return tx.Create(&ver).Error
	}); err != nil {
		return 0, "", err
	}
	return created.ID, version, nil
}

func (s *PlaybookTemplateService) Patch(ctx context.Context, id uint64, req PatchPlaybookTemplateRequest) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.PlaybookTemplate
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		updates := map[string]any{}
		nextName := strings.TrimSpace(row.Name)
		if req.Name != nil {
			v := strings.TrimSpace(*req.Name)
			if v == "" {
				return ErrInvalidParams
			}
			nextName = v
			updates["name"] = v
		}
		if req.Scenario != nil {
			v := strings.TrimSpace(*req.Scenario)
			if v == "" {
				return ErrInvalidParams
			}
			if err := validateTemplateScenario(v); err != nil {
				return err
			}
			updates["scenario"] = v
		}
		if req.Description != nil {
			updates["description"] = *req.Description
		}
		if req.UpdatedBy > 0 {
			updates["updated_by"] = req.UpdatedBy
		}

		if req.Name != nil {
			var exists int64
			if err := tx.Model(&model.PlaybookTemplate{}).
				Where("deleted_at IS NULL AND id <> ? AND name = ?", id, nextName).
				Count(&exists).Error; err != nil {
				return err
			}
			if exists > 0 {
				return ErrConflict
			}
		}

		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&model.PlaybookTemplate{}).Where("deleted_at IS NULL AND id = ?", id).Updates(updates).Error
	})
}

func (s *PlaybookTemplateService) Delete(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	res := s.db.WithContext(ctx).Model(&model.PlaybookTemplate{}).Where("deleted_at IS NULL AND id = ?", id).Update("deleted_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PlaybookTemplateService) ListVersions(ctx context.Context, templateID uint64) ([]PlaybookTemplateVersionItem, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	if templateID == 0 {
		return nil, ErrInvalidParams
	}
	var rows []model.PlaybookTemplateVersion
	if err := s.db.WithContext(ctx).Where("template_id = ?", templateID).Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]PlaybookTemplateVersionItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, PlaybookTemplateVersionItem{
			Version:      r.Version,
			CreatedAt:    r.CreatedAt.UTC().Format(time.RFC3339),
			Source:       r.Source,
			ParamsSchema: r.ParamsSchema,
			Defaults:     r.Defaults,
			CreatedBy:    r.CreatedBy,
		})
	}
	return out, nil
}

func (s *PlaybookTemplateService) CreateVersion(ctx context.Context, templateID uint64, req CreatePlaybookTemplateVersionRequest) (string, error) {
	if s.db == nil {
		return "", errors.New("db is required")
	}
	if templateID == 0 {
		return "", ErrInvalidParams
	}
	if err := validateSourceMap(req.Source); err != nil {
		return "", err
	}
	version := strings.TrimSpace(req.Version)
	return version, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tpl model.PlaybookTemplate
		if err := tx.Where("deleted_at IS NULL AND id = ?", templateID).First(&tpl).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if version == "" {
			next, err := nextTemplateVersion(tx, templateID, tpl.CurrentVersion)
			if err != nil {
				return err
			}
			version = next
		}
		if err := validateTemplateVersion(version); err != nil {
			return err
		}

		var exists int64
		if err := tx.Model(&model.PlaybookTemplateVersion{}).Where("template_id = ? AND version = ?", templateID, version).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return ErrConflict
		}

		row := model.PlaybookTemplateVersion{
			TemplateID:   templateID,
			Version:      version,
			Source:       req.Source,
			ParamsSchema: nonNilJSON(req.ParamsSchema),
			Defaults:     nonNilJSON(req.Defaults),
			CreatedBy:    req.CreatedBy,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"current_version": version,
		}
		if req.CreatedBy > 0 {
			updates["updated_by"] = req.CreatedBy
		}
		return tx.Model(&model.PlaybookTemplate{}).Where("id = ? AND deleted_at IS NULL", templateID).Updates(updates).Error
	})
}

func (s *PlaybookTemplateService) Rollback(ctx context.Context, templateID uint64, req RollbackPlaybookTemplateRequest) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if templateID == 0 {
		return ErrInvalidParams
	}
	v := strings.TrimSpace(req.Version)
	if v == "" {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tpl model.PlaybookTemplate
		if err := tx.Where("deleted_at IS NULL AND id = ?", templateID).First(&tpl).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		var exists int64
		if err := tx.Model(&model.PlaybookTemplateVersion{}).Where("template_id = ? AND version = ?", templateID, v).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return ErrWithMessage(ErrNotFound, "版本不存在")
		}
		updates := map[string]any{"current_version": v}
		if req.UpdatedBy > 0 {
			updates["updated_by"] = req.UpdatedBy
		}
		return tx.Model(&model.PlaybookTemplate{}).Where("id = ? AND deleted_at IS NULL", templateID).Updates(updates).Error
	})
}

func validateTemplateScenario(s string) error {
	switch s {
	case "service_install", "machine_inspection", "config_manage":
		return nil
	default:
		return ErrWithMessage(ErrInvalidParams, "scenario 不支持")
	}
}

var versionRe = regexp.MustCompile(`^v\d+(\.\d+){0,2}$`)

func validateTemplateVersion(v string) error {
	if !versionRe.MatchString(v) {
		return ErrWithMessage(ErrInvalidParams, "version 格式不正确")
	}
	return nil
}

func nonNilJSON(m model.JSONMap) model.JSONMap {
	if m == nil {
		return model.JSONMap{}
	}
	return m
}

func validateSourceMap(src model.JSONMap) error {
	if src == nil || len(src) == 0 {
		return ErrWithMessage(ErrInvalidParams, "source 不能为空")
	}
	tp := strings.TrimSpace(anyToString(src["type"]))
	switch tp {
	case "git":
		gitObj, _ := src["git"].(map[string]any)
		if gitObj == nil {
			if mm, ok := src["git"].(model.JSONMap); ok {
				gitObj = map[string]any(mm)
			}
		}
		if gitObj == nil {
			return ErrWithMessage(ErrInvalidParams, "git 配置不能为空")
		}
		url := strings.TrimSpace(anyToString(gitObj["url"]))
		pb := strings.TrimSpace(anyToString(gitObj["playbook_path"]))
		if url == "" || pb == "" {
			return ErrWithMessage(ErrInvalidParams, "git 方式需要 url 与 playbook_path")
		}
		if !strings.Contains(url, "://") && !strings.Contains(url, "@") {
			return ErrWithMessage(ErrInvalidParams, "git url 不合法")
		}
		if strings.Contains(pb, "..") {
			return ErrWithMessage(ErrInvalidParams, "playbook_path 不合法")
		}
		return nil
	case "upload":
		upObj, _ := src["upload"].(map[string]any)
		if upObj == nil {
			if mm, ok := src["upload"].(model.JSONMap); ok {
				upObj = map[string]any(mm)
			}
		}
		if upObj == nil {
			return ErrWithMessage(ErrInvalidParams, "upload 配置不能为空")
		}
		url := strings.TrimSpace(anyToString(upObj["url"]))
		if url == "" {
			return ErrWithMessage(ErrInvalidParams, "upload 方式需要 url")
		}
		if !strings.HasPrefix(url, "/uploads/") {
			return ErrWithMessage(ErrInvalidParams, "upload url 不合法")
		}
		return nil
	case "inline":
		inObj, _ := src["inline"].(map[string]any)
		if inObj == nil {
			if mm, ok := src["inline"].(model.JSONMap); ok {
				inObj = map[string]any(mm)
			}
		}
		if inObj == nil {
			return ErrWithMessage(ErrInvalidParams, "inline 配置不能为空")
		}
		content := strings.TrimSpace(anyToString(inObj["content"]))
		if content == "" {
			return ErrWithMessage(ErrInvalidParams, "inline 方式需要 content")
		}
		return nil
	default:
		return ErrWithMessage(ErrInvalidParams, "source.type 不支持")
	}
}

func nextTemplateVersion(tx *gorm.DB, templateID uint64, current string) (string, error) {
	cur := strings.TrimSpace(current)
	if cur == "" {
		return "v1.0", nil
	}
	parts := strings.Split(strings.TrimPrefix(cur, "v"), ".")
	if len(parts) == 0 {
		return "v1.0", nil
	}
	if len(parts) == 1 {
		return "v" + parts[0] + ".1", nil
	}
	major := parts[0]
	minor := parts[1]
	minorInt := 0
	if v, err := strconvAtoiSafe(minor); err == nil {
		minorInt = v
	}
	minorInt++
	next := fmt.Sprintf("v%s.%d", major, minorInt)
	var exists int64
	if err := tx.Model(&model.PlaybookTemplateVersion{}).Where("template_id = ? AND version = ?", templateID, next).Count(&exists).Error; err != nil {
		return "", err
	}
	if exists == 0 {
		return next, nil
	}
	return fmt.Sprintf("v%s.%d.%d", major, minorInt, time.Now().UTC().Unix()), nil
}

func strconvAtoiSafe(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty")
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errors.New("not int")
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}

func (s *PlaybookTemplateService) SaveUploadedPlaybook(ctx context.Context, filename string, reader io.Reader, size int64) (UploadPlaybookResult, error) {
	_ = ctx
	if reader == nil || size <= 0 {
		return UploadPlaybookResult{}, ErrInvalidParams
	}
	const maxBytes = 2 * 1024 * 1024
	if size > maxBytes {
		return UploadPlaybookResult{}, ErrInvalidParams
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".yml" && ext != ".yaml" {
		return UploadPlaybookResult{}, ErrWithMessage(ErrInvalidParams, "仅支持 .yml/.yaml")
	}
	ym := time.Now().Format("200601")
	storedName := uuid.NewString() + ext
	relDir := filepath.Join("automation", "playbooks", ym)
	uploadsDir, err := resolveUploadsDir()
	if err != nil {
		return UploadPlaybookResult{}, err
	}
	absDir := filepath.Join(uploadsDir, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return UploadPlaybookResult{}, err
	}
	absPath := filepath.Join(absDir, storedName)
	out, err := os.Create(absPath)
	if err != nil {
		return UploadPlaybookResult{}, err
	}
	defer out.Close()
	written, err := io.Copy(out, io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return UploadPlaybookResult{}, err
	}
	if written > maxBytes {
		return UploadPlaybookResult{}, ErrInvalidParams
	}
	urlPath := path.Join("/uploads", filepath.ToSlash(filepath.Join(relDir, storedName)))
	return UploadPlaybookResult{URL: urlPath, Filename: storedName, Size: written}, nil
}

func (s *PlaybookTemplateService) ResolvePlaybookContent(ctx context.Context, source model.JSONMap) (string, error) {
	if err := validateSourceMap(source); err != nil {
		return "", err
	}
	tp := strings.TrimSpace(anyToString(source["type"]))
	switch tp {
	case "inline":
		inObj := mapFromAny(source["inline"])
		content := strings.TrimSpace(anyToString(inObj["content"]))
		return content, nil
	case "upload":
		upObj := mapFromAny(source["upload"])
		url := strings.TrimSpace(anyToString(upObj["url"]))
		return s.readUploadsFile(url)
	case "git":
		gitObj := mapFromAny(source["git"])
		url := strings.TrimSpace(anyToString(gitObj["url"]))
		ref := strings.TrimSpace(anyToString(gitObj["ref"]))
		playbookPath := strings.TrimSpace(anyToString(gitObj["playbook_path"]))
		credID := uint64(anyToInt64(gitObj["credential_id"]))
		return s.readGitFile(ctx, url, ref, playbookPath, credID)
	default:
		return "", ErrWithMessage(ErrInvalidParams, "source.type 不支持")
	}
}

func mapFromAny(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if m, ok := v.(model.JSONMap); ok {
		return map[string]any(m)
	}
	return nil
}

func anyToInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	case string:
		n, err := strconvAtoiSafe(t)
		if err != nil {
			return 0
		}
		return int64(n)
	default:
		return 0
	}
}

func (s *PlaybookTemplateService) readUploadsFile(urlPath string) (string, error) {
	if !strings.HasPrefix(urlPath, "/uploads/") {
		return "", ErrInvalidParams
	}
	rel := strings.TrimPrefix(urlPath, "/uploads/")
	rel = filepath.Clean(filepath.FromSlash(rel))
	if strings.Contains(rel, "..") || strings.HasPrefix(rel, string(filepath.Separator)) {
		return "", ErrInvalidParams
	}
	uploadsDir, err := resolveUploadsDir()
	if err != nil {
		return "", err
	}
	abs := filepath.Join(uploadsDir, rel)
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *PlaybookTemplateService) readGitFile(ctx context.Context, url, ref, playbookPath string, credID uint64) (string, error) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		return "", errors.New("git not found")
	}
	tmpDir, err := os.MkdirTemp("", "playbook-git-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	repoDir := filepath.Join(tmpDir, "repo")
	args := []string{"clone", url, repoDir}
	env := os.Environ()
	cleanup := func() {}
	if credID > 0 {
		at, err := s.loadCredentialAuth(ctx, credID)
		if err != nil {
			return "", err
		}
		if len(at.extraArgs) > 0 {
			args = append(at.extraArgs, args...)
		}
		if len(at.extraEnv) > 0 {
			env = append(env, at.extraEnv...)
		}
		cleanup = at.cleanup
	}
	defer cleanup()
	cmd := exec.CommandContext(ctx, gitBin, args...)
	cmd.Env = append(env, "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
		return "", ErrWithMessage(ErrInvalidParams, "git clone 失败")
	}
	if ref != "" {
		co := exec.CommandContext(ctx, gitBin, "-C", repoDir, "checkout", ref)
		co.Env = env
		if out, err := co.CombinedOutput(); err != nil {
			_ = out
			return "", ErrWithMessage(ErrInvalidParams, "git checkout 失败")
		}
	}
	p := filepath.Clean(filepath.Join(repoDir, filepath.FromSlash(playbookPath)))
	if !strings.HasPrefix(p, repoDir) {
		return "", ErrInvalidParams
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type gitAuth struct {
	extraArgs []string
	extraEnv  []string
	cleanup   func()
}

func (s *PlaybookTemplateService) loadCredentialAuth(ctx context.Context, credID uint64) (gitAuth, error) {
	if s.db == nil {
		return gitAuth{}, errors.New("db is required")
	}
	var row model.Credential
	if err := s.db.WithContext(ctx).Select("id", "provider", "auth_type", "data_enc").Where("deleted_at IS NULL AND id = ?", credID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gitAuth{}, ErrNotFound
		}
		return gitAuth{}, err
	}
	pt, err := decryptText(s.secretKey, row.DataEnc)
	if err != nil {
		return gitAuth{}, ErrWithMessage(ErrCrypto, "凭据解密失败")
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(pt), &data); err != nil {
		return gitAuth{}, ErrWithMessage(ErrInvalidParams, "凭据内容格式错误")
	}
	switch row.AuthType {
	case "username_password":
		u := strings.TrimSpace(anyToString(data["username"]))
		p := strings.TrimSpace(anyToString(data["password"]))
		if u == "" || p == "" {
			return gitAuth{}, ErrWithMessage(ErrInvalidParams, "凭据内容不完整")
		}
		plain := fmt.Sprintf("%s:%s", u, p)
		enc := base64.StdEncoding.EncodeToString([]byte(plain))
		h := "http.extraHeader=Authorization: Basic " + enc
		return gitAuth{extraArgs: []string{"-c", h}}, nil
	case "access_token":
		tok := strings.TrimSpace(anyToString(data["token"]))
		if tok == "" {
			return gitAuth{}, ErrWithMessage(ErrInvalidParams, "凭据内容不完整")
		}
		user := strings.TrimSpace(anyToString(data["username"]))
		if user == "" {
			user = "oauth2"
		}
		plain := fmt.Sprintf("%s:%s", user, tok)
		enc := base64.StdEncoding.EncodeToString([]byte(plain))
		h := "http.extraHeader=Authorization: Basic " + enc
		return gitAuth{extraArgs: []string{"-c", h}}, nil
	case "ssh_private_key":
		key := strings.TrimSpace(anyToString(data["private_key"]))
		if key == "" {
			return gitAuth{}, ErrWithMessage(ErrInvalidParams, "凭据内容不完整")
		}
		tmpDir, err := os.MkdirTemp("", "git-ssh-*")
		if err != nil {
			return gitAuth{}, err
		}
		keyPath := filepath.Join(tmpDir, "id")
		if err := os.WriteFile(keyPath, []byte(key), 0o600); err != nil {
			_ = os.RemoveAll(tmpDir)
			return gitAuth{}, err
		}
		knownHosts := os.DevNull
		if runtime.GOOS != "windows" {
			knownHosts = "/dev/null"
		}
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=%s", shellQuote(keyPath), shellQuote(knownHosts))
		return gitAuth{
			extraEnv: []string{"GIT_SSH_COMMAND=" + sshCmd},
			cleanup:  func() { _ = os.RemoveAll(tmpDir) },
		}, nil
	default:
		return gitAuth{}, ErrWithMessage(ErrInvalidParams, "该 auth_type 不支持用于 git")
	}
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if runtime.GOOS == "windows" {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return `'` + strings.ReplaceAll(s, `'`, `'\''`) + `'`
}
