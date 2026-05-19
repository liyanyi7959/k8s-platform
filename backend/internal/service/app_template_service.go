package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// ── 请求 / 响应 DTO ─────────────────────────────────────────────────────────

type ListAppTemplatesRequest struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Keyword    string `form:"keyword"`
	Category   string `form:"category"`
	Engine     string `form:"engine"`
	Status     string `form:"status"`
	SourceType string `form:"source_type"`
	SortBy     string `form:"sort_by"`
	Order      string `form:"order"`
}

type AppTemplateItem struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	ChartName    string    `json:"chart_name"`
	Category     string    `json:"category"`
	Version      string    `json:"version"`
	AppVersion   string    `json:"app_version"`
	Engine       string    `json:"engine"`
	Status       string    `json:"status"`
	SourceType   string    `json:"source_type"`
	SourceURL    string    `json:"source_url"`
	Summary      string    `json:"summary"`
	Tags         []string  `json:"tags"`
	Owner        string    `json:"owner"`
	ReleaseCount int       `json:"release_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AppTemplateDetail struct {
	AppTemplateItem
	Manifest           string                 `json:"manifest"`
	ValuesSchema       map[string]interface{} `json:"values_schema"`
	DefaultValues      map[string]interface{} `json:"default_values"`
	Source             string                 `json:"source"`
	SourceRef          map[string]interface{} `json:"source_ref"`
	Readme             string                 `json:"readme"`
	EnvExample         string                 `json:"env_example"`
	ProjectNameDefault string                 `json:"project_name_default"`
	InstallDirDefault  string                 `json:"install_dir_default"`
	ExtraFiles         []string               `json:"extra_files"`
}

type CreateAppTemplateRequest struct {
	Name               string                 `json:"name"`
	ChartName          string                 `json:"chart_name"`
	Category           string                 `json:"category"`
	Version            string                 `json:"version"`
	AppVersion         string                 `json:"app_version"`
	Engine             string                 `json:"engine"`
	Status             string                 `json:"status"`
	Summary            string                 `json:"summary"`
	Tags               []string               `json:"tags"`
	Manifest           string                 `json:"manifest"`
	ValuesSchema       map[string]interface{} `json:"values_schema"`
	DefaultValues      map[string]interface{} `json:"default_values"`
	Source             string                 `json:"source"`
	SourceType         string                 `json:"source_type"`
	SourceURL          string                 `json:"source_url"`
	SourceRef          map[string]interface{} `json:"source_ref"`
	Owner              string                 `json:"owner"`
	Readme             string                 `json:"readme"`
	EnvExample         string                 `json:"env_example"`
	ProjectNameDefault string                 `json:"project_name_default"`
	InstallDirDefault  string                 `json:"install_dir_default"`
	ExtraFiles         []string               `json:"extra_files"`
	CreatedBy          uint64
}

type PatchAppTemplateRequest struct {
	Name               *string                `json:"name"`
	ChartName          *string                `json:"chart_name"`
	Category           *string                `json:"category"`
	Version            *string                `json:"version"`
	AppVersion         *string                `json:"app_version"`
	Engine             *string                `json:"engine"`
	Status             *string                `json:"status"`
	Summary            *string                `json:"summary"`
	Tags               []string               `json:"tags"`
	Manifest           *string                `json:"manifest"`
	ValuesSchema       map[string]interface{} `json:"values_schema"`
	DefaultValues      map[string]interface{} `json:"default_values"`
	Source             *string                `json:"source"`
	SourceType         *string                `json:"source_type"`
	SourceURL          *string                `json:"source_url"`
	SourceRef          map[string]interface{} `json:"source_ref"`
	Owner              *string                `json:"owner"`
	Readme             *string                `json:"readme"`
	EnvExample         *string                `json:"env_example"`
	ProjectNameDefault *string                `json:"project_name_default"`
	InstallDirDefault  *string                `json:"install_dir_default"`
	ExtraFiles         []string               `json:"extra_files"`
	UpdatedBy          uint64
}

type ImportAppTemplateRequest struct {
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Engine     string   `json:"engine"`
	Tags       []string `json:"tags"`
	SourceType string   `json:"source_type"`
	SourceURL  string   `json:"source_url"`
	Owner      string   `json:"owner"`
	CreatedBy  uint64
}

// ── Service ──────────────────────────────────────────────────────────────────

type AppTemplateService struct {
	db *gorm.DB
}

func NewAppTemplateService(db *gorm.DB) *AppTemplateService {
	return &AppTemplateService{db: db}
}

// List 分页查询模板列表。
func (s *AppTemplateService) List(ctx context.Context, req ListAppTemplatesRequest) (PageResult[AppTemplateItem], error) {
	if s.db == nil {
		return PageResult[AppTemplateItem]{}, errors.New("db is required")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)

	q := s.db.WithContext(ctx).
		Table("app_templates t").
		Select("t.id, t.name, t.chart_name, t.category, t.version, t.app_version, t.engine, t.status, t.source_type, t.source_url, t.summary, t.tags, t.owner, t.created_at, t.updated_at, COUNT(r.id) AS release_count").
		Joins("LEFT JOIN app_releases r ON r.template_id = t.id AND r.deleted_at IS NULL").
		Where("t.deleted_at IS NULL").
		Group("t.id")

	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(t.name LIKE ? OR t.chart_name LIKE ? OR t.summary LIKE ? OR t.category LIKE ?)", like, like, like, like)
	}
	if cat := strings.TrimSpace(req.Category); cat != "" {
		q = q.Where("t.category LIKE ?", "%"+cat+"%")
	}
	if eng := strings.TrimSpace(req.Engine); eng != "" {
		q = q.Where("t.engine = ?", eng)
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		q = q.Where("t.status = ?", status)
	}
	if sourceType := strings.TrimSpace(req.SourceType); sourceType != "" {
		q = q.Where("t.source_type = ?", sourceType)
	}

	// 先 count（不含 GROUP BY 排序偏移）
	var total int64
	if err := s.db.WithContext(ctx).Table("app_templates t").
		Where("t.deleted_at IS NULL").
		Scopes(func(d *gorm.DB) *gorm.DB {
			if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
				like := "%" + keyword + "%"
				return d.Where("(t.name LIKE ? OR t.chart_name LIKE ? OR t.summary LIKE ? OR t.category LIKE ?)", like, like, like, like)
			}
			return d
		}).
		Scopes(func(d *gorm.DB) *gorm.DB {
			if cat := strings.TrimSpace(req.Category); cat != "" {
				return d.Where("t.category LIKE ?", "%"+cat+"%")
			}
			return d
		}).
		Scopes(func(d *gorm.DB) *gorm.DB {
			if eng := strings.TrimSpace(req.Engine); eng != "" {
				return d.Where("t.engine = ?", eng)
			}
			return d
		}).
		Scopes(func(d *gorm.DB) *gorm.DB {
			if status := strings.TrimSpace(req.Status); status != "" {
				return d.Where("t.status = ?", status)
			}
			return d
		}).
		Scopes(func(d *gorm.DB) *gorm.DB {
			if sourceType := strings.TrimSpace(req.SourceType); sourceType != "" {
				return d.Where("t.source_type = ?", sourceType)
			}
			return d
		}).
		Count(&total).Error; err != nil {
		return PageResult[AppTemplateItem]{}, err
	}

	orderExpr := "t.id DESC"
	if col := sanitizeOrderColumn(req.SortBy); col != "" {
		dir := "DESC"
		if strings.ToLower(req.Order) == "asc" {
			dir = "ASC"
		}
		orderExpr = fmt.Sprintf("t.%s %s", col, dir)
	}

	type row struct {
		model.AppTemplate
		ReleaseCount int `gorm:"column:release_count"`
	}
	var rows []row
	if err := q.Order(orderExpr).Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		return PageResult[AppTemplateItem]{}, err
	}

	out := make([]AppTemplateItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAppTemplateItem(r.AppTemplate, r.ReleaseCount))
	}
	return PageResult[AppTemplateItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

// Get 查询单个模板详情。
func (s *AppTemplateService) Get(ctx context.Context, id uint64) (*AppTemplateDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	var m model.AppTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	// 统计关联发布数
	var cnt int64
	s.db.WithContext(ctx).Model(&model.AppRelease{}).Where("template_id = ? AND deleted_at IS NULL", id).Count(&cnt)
	item := toAppTemplateItem(m, int(cnt))
	return &AppTemplateDetail{
		AppTemplateItem:    item,
		Manifest:           strPtrVal(m.Manifest),
		ValuesSchema:       jsonToMap(m.ValuesSchema),
		DefaultValues:      jsonToMap(m.DefaultValues),
		Source:             m.Source,
		SourceRef:          jsonToMap(m.SourceRef),
		Readme:             strPtrVal(m.Readme),
		EnvExample:         strPtrVal(m.EnvExample),
		ProjectNameDefault: m.ProjectName,
		InstallDirDefault:  m.InstallDir,
		ExtraFiles:         jsonToStringSlice(m.ExtraFiles),
	}, nil
}

// Create 新建模板，返回完整 detail 对象（与前端契约一致）。
func (s *AppTemplateService) Create(ctx context.Context, req CreateAppTemplateRequest) (*AppTemplateDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrWithMessage(ErrInvalidParams, "模板名称不能为空")
	}
	if req.Engine == "" {
		req.Engine = "helm"
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = "ready"
	}
	if strings.TrimSpace(req.SourceType) == "" {
		req.SourceType = "custom"
	}
	if req.Owner == "" {
		req.Owner = "当前用户"
	}
	if req.Source == "" {
		req.Source = sourceLabel(req.SourceType)
	}
	if req.ChartName == "" {
		req.ChartName = req.Name
	}

	tagsJSON := marshalJSON(req.Tags)
	schemaJSON := marshalJSONMap(req.ValuesSchema)
	defaultsJSON := marshalJSONMap(req.DefaultValues)
	sourceRefJSON := marshalJSONMap(req.SourceRef)
	extraFilesJSON := marshalJSON(req.ExtraFiles)

	m := model.AppTemplate{
		Name:          req.Name,
		ChartName:     req.ChartName,
		Category:      req.Category,
		Version:       req.Version,
		AppVersion:    req.AppVersion,
		Engine:        req.Engine,
		Status:        req.Status,
		Summary:       req.Summary,
		Tags:          strPtr(string(tagsJSON)),
		Manifest:      strPtr(req.Manifest),
		ValuesSchema:  strPtr(string(schemaJSON)),
		DefaultValues: strPtr(string(defaultsJSON)),
		Source:        req.Source,
		SourceType:    req.SourceType,
		SourceURL:     req.SourceURL,
		SourceRef:     strPtr(string(sourceRefJSON)),
		Owner:         req.Owner,
		Readme:        strPtr(req.Readme),
		EnvExample:    strPtr(req.EnvExample),
		ProjectName:   req.ProjectNameDefault,
		InstallDir:    req.InstallDirDefault,
		ExtraFiles:    strPtr(string(extraFilesJSON)),
		CreatedBy:     req.CreatedBy,
		UpdatedBy:     req.CreatedBy,
	}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	item := toAppTemplateItem(m, 0)
	return &AppTemplateDetail{
		AppTemplateItem:    item,
		Manifest:           req.Manifest,
		ValuesSchema:       req.ValuesSchema,
		DefaultValues:      req.DefaultValues,
		Source:             req.Source,
		SourceRef:          req.SourceRef,
		Readme:             req.Readme,
		EnvExample:         req.EnvExample,
		ProjectNameDefault: req.ProjectNameDefault,
		InstallDirDefault:  req.InstallDirDefault,
		ExtraFiles:         req.ExtraFiles,
	}, nil
}

// Patch 局部更新模板，返回最新详情。
func (s *AppTemplateService) Patch(ctx context.Context, id uint64, req PatchAppTemplateRequest) (*AppTemplateDetail, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	var m model.AppTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	updates := map[string]interface{}{"updated_by": req.UpdatedBy}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ChartName != nil {
		updates["chart_name"] = *req.ChartName
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.Version != nil {
		updates["version"] = *req.Version
	}
	if req.AppVersion != nil {
		updates["app_version"] = *req.AppVersion
	}
	if req.Engine != nil {
		updates["engine"] = *req.Engine
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Summary != nil {
		updates["summary"] = *req.Summary
	}
	if req.Tags != nil {
		updates["tags"] = string(marshalJSON(req.Tags))
	}
	if req.Manifest != nil {
		updates["manifest"] = *req.Manifest
	}
	if req.ValuesSchema != nil {
		updates["values_schema"] = string(marshalJSONMap(req.ValuesSchema))
	}
	if req.DefaultValues != nil {
		updates["default_values"] = string(marshalJSONMap(req.DefaultValues))
	}
	if req.Source != nil {
		updates["source"] = *req.Source
	}
	if req.SourceType != nil {
		updates["source_type"] = *req.SourceType
	}
	if req.SourceURL != nil {
		updates["source_url"] = *req.SourceURL
	}
	if req.SourceRef != nil {
		updates["source_ref"] = string(marshalJSONMap(req.SourceRef))
	}
	if req.Owner != nil {
		updates["owner"] = *req.Owner
	}
	if req.Readme != nil {
		updates["readme"] = *req.Readme
	}
	if req.EnvExample != nil {
		updates["env_example"] = *req.EnvExample
	}
	if req.ProjectNameDefault != nil {
		updates["project_name_default"] = *req.ProjectNameDefault
	}
	if req.InstallDirDefault != nil {
		updates["install_dir_default"] = *req.InstallDirDefault
	}
	if req.ExtraFiles != nil {
		updates["extra_files"] = string(marshalJSON(req.ExtraFiles))
	}
	if err := s.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
		return nil, err
	}
	// reload
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		return nil, err
	}
	var cnt int64
	s.db.WithContext(ctx).Model(&model.AppRelease{}).Where("template_id = ? AND deleted_at IS NULL", id).Count(&cnt)
	item := toAppTemplateItem(m, int(cnt))
	return &AppTemplateDetail{
		AppTemplateItem:    item,
		Manifest:           strPtrVal(m.Manifest),
		ValuesSchema:       jsonToMap(m.ValuesSchema),
		DefaultValues:      jsonToMap(m.DefaultValues),
		Source:             m.Source,
		SourceRef:          jsonToMap(m.SourceRef),
		Readme:             strPtrVal(m.Readme),
		EnvExample:         strPtrVal(m.EnvExample),
		ProjectNameDefault: m.ProjectName,
		InstallDirDefault:  m.InstallDir,
		ExtraFiles:         jsonToStringSlice(m.ExtraFiles),
	}, nil
}

// Delete 软删除模板。
func (s *AppTemplateService) Delete(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	var releaseCount int64
	if err := s.db.WithContext(ctx).
		Model(&model.AppRelease{}).
		Where("template_id = ? AND deleted_at IS NULL", id).
		Count(&releaseCount).Error; err != nil {
		return err
	}
	if releaseCount > 0 {
		return ErrWithMessage(ErrConflict, "当前模板仍有关联发布，请先处理关联发布或将模板归档")
	}
	now := time.Now()
	res := s.db.WithContext(ctx).Model(&model.AppTemplate{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ImportPackage 从上传文件中导入模板。
func (s *AppTemplateService) ImportPackage(ctx context.Context, req ImportAppTemplateRequest, filename string, content []byte) (*AppTemplateDetail, error) {
	parsed, err := parseImportedTemplate(req.Engine, filename, content)
	if err != nil {
		return nil, err
	}
	return s.Create(ctx, CreateAppTemplateRequest{
		Name:               firstNonEmpty(strings.TrimSpace(req.Name), parsed.Name),
		ChartName:          firstNonEmpty(parsed.Name, strings.TrimSpace(req.Name)),
		Category:           firstNonEmpty(strings.TrimSpace(req.Category), "未分类"),
		Version:            firstNonEmpty(parsed.Version, "1.0.0"),
		AppVersion:         parsed.AppVersion,
		Engine:             req.Engine,
		Status:             "pending_validation",
		Summary:            firstNonEmpty(parsed.Summary, "从上传文件导入"),
		Tags:               req.Tags,
		Manifest:           parsed.Manifest,
		ValuesSchema:       parsed.ValuesSchema,
		DefaultValues:      parsed.DefaultValues,
		Source:             sourceLabel(req.SourceType),
		SourceType:         firstNonEmpty(req.SourceType, "upload"),
		Owner:              firstNonEmpty(req.Owner, "当前用户"),
		Readme:             parsed.Readme,
		EnvExample:         parsed.EnvExample,
		ProjectNameDefault: parsed.ProjectNameDefault,
		InstallDirDefault:  parsed.InstallDirDefault,
		ExtraFiles:         parsed.ExtraFiles,
		SourceRef:          parsed.SourceRef,
		CreatedBy:          req.CreatedBy,
	})
}

// ImportRemote 从远程地址导入模板。
func (s *AppTemplateService) ImportRemote(ctx context.Context, req ImportAppTemplateRequest) (*AppTemplateDetail, error) {
	if strings.TrimSpace(req.SourceURL) == "" {
		return nil, ErrWithMessage(ErrInvalidParams, "远程地址不能为空")
	}
	body, filename, err := fetchRemoteTemplatePackage(ctx, req.SourceURL)
	if err != nil {
		return nil, err
	}
	parsed, err := parseImportedTemplate(req.Engine, filename, body)
	if err != nil {
		return nil, err
	}
	if parsed.SourceRef == nil {
		parsed.SourceRef = map[string]interface{}{}
	}
	parsed.SourceRef["remote_file"] = filename
	parsed.SourceRef["remote_url"] = req.SourceURL
	return s.Create(ctx, CreateAppTemplateRequest{
		Name:               firstNonEmpty(strings.TrimSpace(req.Name), parsed.Name),
		ChartName:          firstNonEmpty(parsed.Name, strings.TrimSpace(req.Name)),
		Category:           firstNonEmpty(strings.TrimSpace(req.Category), "未分类"),
		Version:            firstNonEmpty(parsed.Version, "1.0.0"),
		AppVersion:         parsed.AppVersion,
		Engine:             req.Engine,
		Status:             "pending_validation",
		Summary:            firstNonEmpty(parsed.Summary, "从远程地址导入"),
		Tags:               req.Tags,
		Manifest:           parsed.Manifest,
		ValuesSchema:       parsed.ValuesSchema,
		DefaultValues:      parsed.DefaultValues,
		Source:             sourceLabel(firstNonEmpty(req.SourceType, "remote")),
		SourceType:         firstNonEmpty(req.SourceType, "remote"),
		SourceURL:          req.SourceURL,
		Owner:              firstNonEmpty(req.Owner, "当前用户"),
		Readme:             parsed.Readme,
		EnvExample:         parsed.EnvExample,
		ProjectNameDefault: parsed.ProjectNameDefault,
		InstallDirDefault:  parsed.InstallDirDefault,
		ExtraFiles:         parsed.ExtraFiles,
		SourceRef:          parsed.SourceRef,
		CreatedBy:          req.CreatedBy,
	})
}

// Render 渲染模板（简单值替换预览，不涉及真实 Helm）。
func (s *AppTemplateService) Render(ctx context.Context, id uint64, values map[string]interface{}) (string, error) {
	detail, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	merged := mergeTemplateValues(detail.DefaultValues, values)
	if detail.Engine == "yaml" {
		lines := []string{
			fmt.Sprintf("# Compose template: %s@%s", detail.Name, detail.Version),
			detail.Manifest,
		}
		if strings.TrimSpace(detail.EnvExample) != "" {
			lines = append(lines, "", "# .env.example", detail.EnvExample)
		}
		if len(merged) > 0 {
			mergedJSON, _ := json.MarshalIndent(merged, "", "  ")
			lines = append(lines, "", "# variables", string(mergedJSON))
		}
		return strings.Join(lines, "\n"), nil
	}
	renderedManifest, err := renderAppTemplateManifest(detail, values, templateRenderOptions{})
	if err != nil {
		return "", err
	}
	mergedJSON, _ := json.MarshalIndent(merged, "", "  ")
	rendered := fmt.Sprintf("# Rendered from: %s@%s\n# engine: %s\n\n%s\n\n# --- values ---\n%s",
		detail.Name, detail.Version, detail.Engine,
		renderedManifest,
		string(mergedJSON),
	)
	return rendered, nil
}

// ── 私有辅助 ──────────────────────────────────────────────────────────────

func toAppTemplateItem(m model.AppTemplate, releaseCount int) AppTemplateItem {
	return AppTemplateItem{
		ID:           m.ID,
		Name:         m.Name,
		ChartName:    firstNonEmpty(m.ChartName, m.Name),
		Category:     m.Category,
		Version:      m.Version,
		AppVersion:   m.AppVersion,
		Engine:       m.Engine,
		Status:       m.Status,
		SourceType:   m.SourceType,
		SourceURL:    m.SourceURL,
		Summary:      m.Summary,
		Tags:         jsonToStringSlice(m.Tags),
		Owner:        m.Owner,
		ReleaseCount: releaseCount,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// sanitizeOrderColumn 仅允许安全列名排序，防止 SQL 注入。
func sanitizeOrderColumn(col string) string {
	allowed := map[string]bool{
		"id": true, "name": true, "category": true, "version": true,
		"engine": true, "status": true, "created_at": true, "updated_at": true,
	}
	if allowed[col] {
		return col
	}
	return ""
}

func strPtr(s string) *string { return &s }

func strPtrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func jsonToStringSlice(p *string) []string {
	if p == nil || *p == "" {
		return []string{}
	}
	var out []string
	_ = json.Unmarshal([]byte(*p), &out)
	return out
}

func jsonToMap(p *string) map[string]interface{} {
	if p == nil || *p == "" {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	_ = json.Unmarshal([]byte(*p), &out)
	if out == nil {
		return map[string]interface{}{}
	}
	return out
}

func marshalJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	if b == nil {
		return []byte("[]")
	}
	return b
}

func marshalJSONMap(v map[string]interface{}) []byte {
	if v == nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(v)
	if b == nil {
		return []byte("{}")
	}
	return b
}

func sourceLabel(sourceType string) string {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "upload":
		return "上传包导入"
	case "remote":
		return "远程地址导入"
	default:
		return "自定义模板"
	}
}
