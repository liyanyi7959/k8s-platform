package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type AutomationCatalogItem struct {
	ID                 uint64   `json:"id"`
	Name               string   `json:"name"`
	Category           string   `json:"category"`
	Description        *string  `json:"description,omitempty"`
	RecommendedVersion string   `json:"recommended_version"`
	Tags               []string `json:"tags"`
	Published          bool     `json:"published"`
	Visibility         string   `json:"visibility"`
	IconURL            *string  `json:"icon_url,omitempty"`
	TemplateID         uint64   `json:"template_id"`
	TemplateVersion    string   `json:"template_version"`
	TemplateName       *string  `json:"template_name,omitempty"`
	SortOrder          int      `json:"sort_order"`
	CreatedBy          uint64   `json:"created_by"`
	UpdatedBy          uint64   `json:"updated_by"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

type ListAutomationCatalogItemsRequest struct {
	Page       int
	PageSize   int
	Keyword    string
	Category   string
	Published  *bool
	Visibility string
	SortBy     string
	Order      string
}

type UpsertAutomationCatalogItemRequest struct {
	Name               string
	Category           string
	Description        *string
	RecommendedVersion string
	Tags               []string
	Published          bool
	Visibility         string
	IconURL            *string
	TemplateID         uint64
	TemplateVersion    string
	SortOrder          int
	OperatorID         uint64
}

type AutomationCatalogService struct {
	db *gorm.DB
}

type automationCatalogRow struct {
	model.AutomationCatalogItem
	TemplateName *string `gorm:"column:template_name"`
}

func NewAutomationCatalogService(db *gorm.DB) *AutomationCatalogService {
	return &AutomationCatalogService{db: db}
}

func (s *AutomationCatalogService) buildListQuery(ctx context.Context, req ListAutomationCatalogItemsRequest) *gorm.DB {
	q := s.db.WithContext(ctx).
		Table("automation_catalog_items aci").
		Select("aci.*, pt.name AS template_name").
		Joins("LEFT JOIN playbook_templates pt ON pt.id = aci.template_id AND pt.deleted_at IS NULL").
		Where("aci.deleted_at IS NULL")
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(aci.name LIKE ? OR aci.category LIKE ? OR aci.description LIKE ?)", like, like, like)
	}
	if category := strings.TrimSpace(req.Category); category != "" {
		q = q.Where("aci.category = ?", category)
	}
	if req.Published != nil {
		q = q.Where("aci.published = ?", *req.Published)
	}
	if visibility := strings.TrimSpace(req.Visibility); visibility != "" {
		q = q.Where("aci.visibility = ?", visibility)
	}
	return q
}

func (s *AutomationCatalogService) List(ctx context.Context, req ListAutomationCatalogItemsRequest) (PageResult[AutomationCatalogItem], error) {
	if s.db == nil {
		return PageResult[AutomationCatalogItem]{}, errors.New("db is required")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.buildListQuery(ctx, req)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[AutomationCatalogItem]{}, err
	}
	orderClause := "aci.sort_order ASC, aci.updated_at DESC, aci.id DESC"
	if strings.TrimSpace(req.SortBy) == "name" {
		if strings.EqualFold(strings.TrimSpace(req.Order), "asc") {
			orderClause = "aci.name ASC, aci.id DESC"
		} else {
			orderClause = "aci.name DESC, aci.id DESC"
		}
	}
	var rows []automationCatalogRow
	if err := q.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[AutomationCatalogItem]{}, err
	}
	out := make([]AutomationCatalogItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAutomationCatalogItem(row))
	}
	return PageResult[AutomationCatalogItem]{List: out, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *AutomationCatalogService) Get(ctx context.Context, id uint64) (AutomationCatalogItem, error) {
	if s.db == nil {
		return AutomationCatalogItem{}, errors.New("db is required")
	}
	if id == 0 {
		return AutomationCatalogItem{}, ErrWithMessage(ErrInvalidParams, "应用 ID 无效")
	}
	var row automationCatalogRow
	err := s.db.WithContext(ctx).
		Table("automation_catalog_items aci").
		Select("aci.*, pt.name AS template_name").
		Joins("LEFT JOIN playbook_templates pt ON pt.id = aci.template_id AND pt.deleted_at IS NULL").
		Where("aci.deleted_at IS NULL AND aci.id = ?", id).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AutomationCatalogItem{}, ErrNotFound
		}
		return AutomationCatalogItem{}, err
	}
	return toAutomationCatalogItem(row), nil
}

func (s *AutomationCatalogService) Create(ctx context.Context, req UpsertAutomationCatalogItemRequest) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	row, err := s.buildCatalogModel(ctx, 0, req)
	if err != nil {
		return 0, err
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *AutomationCatalogService) Patch(ctx context.Context, id uint64, req UpsertAutomationCatalogItemRequest) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "应用 ID 无效")
	}
	row, err := s.buildCatalogModel(ctx, id, req)
	if err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Model(&model.AutomationCatalogItem{}).Where("deleted_at IS NULL AND id = ?", id).Updates(map[string]any{
		"name":                row.Name,
		"category":            row.Category,
		"description":         row.Description,
		"recommended_version": row.RecommendedVersion,
		"tags":                row.Tags,
		"published":           row.Published,
		"visibility":          row.Visibility,
		"icon_url":            row.IconURL,
		"template_id":         row.TemplateID,
		"template_version":    row.TemplateVersion,
		"sort_order":          row.SortOrder,
		"updated_by":          row.UpdatedBy,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *AutomationCatalogService) Delete(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "应用 ID 无效")
	}
	res := s.db.WithContext(ctx).Model(&model.AutomationCatalogItem{}).Where("deleted_at IS NULL AND id = ?", id).Update("deleted_at", time.Now().UTC())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *AutomationCatalogService) buildCatalogModel(ctx context.Context, id uint64, req UpsertAutomationCatalogItemRequest) (model.AutomationCatalogItem, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return model.AutomationCatalogItem{}, ErrWithMessage(ErrInvalidParams, "应用名称不能为空")
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		return model.AutomationCatalogItem{}, ErrWithMessage(ErrInvalidParams, "分类不能为空")
	}
	visibility := strings.TrimSpace(req.Visibility)
	if visibility == "" {
		visibility = "public"
	}
	if visibility != "public" && visibility != "tenant" && visibility != "org" && visibility != "project" {
		return model.AutomationCatalogItem{}, ErrWithMessage(ErrInvalidParams, "可见性不合法")
	}
	if req.TemplateID == 0 {
		return model.AutomationCatalogItem{}, ErrWithMessage(ErrInvalidParams, "请选择关联模板")
	}
	tpl, err := getPlaybookTemplateRow(ctx, s.db, req.TemplateID)
	if err != nil {
		return model.AutomationCatalogItem{}, err
	}
	templateVersion := strings.TrimSpace(req.TemplateVersion)
	if templateVersion == "" {
		templateVersion = strings.TrimSpace(tpl.CurrentVersion)
	}
	if templateVersion == "" {
		return model.AutomationCatalogItem{}, ErrWithMessage(ErrInvalidParams, "模板版本不能为空")
	}
	if err := ensurePlaybookTemplateVersionExists(ctx, s.db, req.TemplateID, templateVersion); err != nil {
		return model.AutomationCatalogItem{}, err
	}
	recommendedVersion := strings.TrimSpace(req.RecommendedVersion)
	if recommendedVersion == "" {
		recommendedVersion = templateVersion
	}
	if err := ensureCatalogNameUnique(ctx, s.db, id, name); err != nil {
		return model.AutomationCatalogItem{}, err
	}
	row := model.AutomationCatalogItem{
		Name:               name,
		Category:           category,
		Description:        trimStringPtr(req.Description),
		RecommendedVersion: recommendedVersion,
		Tags:               model.JSONStringArray(normalizeCatalogTags(req.Tags)),
		Published:          req.Published,
		Visibility:         visibility,
		IconURL:            trimStringPtr(req.IconURL),
		TemplateID:         req.TemplateID,
		TemplateVersion:    templateVersion,
		SortOrder:          req.SortOrder,
		CreatedBy:          req.OperatorID,
		UpdatedBy:          req.OperatorID,
	}
	if id > 0 {
		row.ID = id
	}
	return row, nil
}

func getPlaybookTemplateRow(ctx context.Context, db *gorm.DB, id uint64) (model.PlaybookTemplate, error) {
	var row model.PlaybookTemplate
	if err := db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.PlaybookTemplate{}, ErrWithMessage(ErrNotFound, "关联模板不存在")
		}
		return model.PlaybookTemplate{}, err
	}
	return row, nil
}

func ensurePlaybookTemplateVersionExists(ctx context.Context, db *gorm.DB, templateID uint64, version string) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.PlaybookTemplateVersion{}).Where("template_id = ? AND version = ?", templateID, version).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrWithMessage(ErrNotFound, "模板版本不存在")
	}
	return nil
}

func ensureCatalogNameUnique(ctx context.Context, db *gorm.DB, id uint64, name string) error {
	var count int64
	q := db.WithContext(ctx).Model(&model.AutomationCatalogItem{}).Where("deleted_at IS NULL AND name = ?", name)
	if id > 0 {
		q = q.Where("id <> ?", id)
	}
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrWithMessage(ErrConflict, "应用名称已存在")
	}
	return nil
}

func normalizeCatalogTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(tag)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func toAutomationCatalogItem(row automationCatalogRow) AutomationCatalogItem {
	return AutomationCatalogItem{
		ID:                 row.ID,
		Name:               row.Name,
		Category:           row.Category,
		Description:        row.Description,
		RecommendedVersion: row.RecommendedVersion,
		Tags:               []string(row.Tags),
		Published:          row.Published,
		Visibility:         row.Visibility,
		IconURL:            row.IconURL,
		TemplateID:         row.TemplateID,
		TemplateVersion:    row.TemplateVersion,
		TemplateName:       row.TemplateName,
		SortOrder:          row.SortOrder,
		CreatedBy:          row.CreatedBy,
		UpdatedBy:          row.UpdatedBy,
		CreatedAt:          row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
