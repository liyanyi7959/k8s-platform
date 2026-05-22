package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

const (
	ManifestApplyStatusRunning = "running"
	ManifestApplyStatusSuccess = "success"
	ManifestApplyStatusFailed  = "failed"
)

type ManifestApplyRecordService struct {
	db     *gorm.DB
	k8sSvc *K8sService
}

func NewManifestApplyRecordService(db *gorm.DB, k8sSvc *K8sService) *ManifestApplyRecordService {
	return &ManifestApplyRecordService{db: db, k8sSvc: k8sSvc}
}

type ManifestApplyExecuteRequest struct {
	ClusterID        uint64
	YAML             string
	DefaultNamespace string
	DryRun           bool
	SourceLabel      string
	SourceResource   string
	WorkloadKind     string
	CreatedBy        uint64
	CreatedByName    string
}

type ManifestApplyRecordListParams struct {
	ClusterID        uint64
	Page             int
	PageSize         int
	Keyword          string
	Status           string
	Mode             string
	DefaultNamespace string
}

type ManifestApplyRecordListItem struct {
	ID               uint64    `json:"id"`
	Status           string    `json:"status"`
	DryRun           bool      `json:"dry_run"`
	DefaultNamespace string    `json:"default_namespace"`
	SourceLabel      string    `json:"source_label"`
	SourceResource   string    `json:"source_resource"`
	WorkloadKind     string    `json:"workload_kind"`
	ResultCount      int       `json:"result_count"`
	Summary          string    `json:"summary"`
	ErrorMessage     string    `json:"error_message"`
	CreatedBy        uint64    `json:"created_by"`
	CreatedByName    string    `json:"created_by_name"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ManifestApplyRecordDetail struct {
	ManifestApplyRecordListItem
	ClusterID   uint64                    `json:"cluster_id"`
	YAMLContent string                    `json:"yaml_content"`
	ResultItems []ManifestApplyResultItem `json:"result_items"`
}

type ManifestApplyExecuteResult struct {
	RecordID uint64                    `json:"record_id"`
	Status   string                    `json:"status"`
	DryRun   bool                      `json:"dry_run"`
	Summary  string                    `json:"summary"`
	Items    []ManifestApplyResultItem `json:"items"`
}

func (s *ManifestApplyRecordService) Execute(ctx context.Context, req ManifestApplyExecuteRequest) (*ManifestApplyExecuteResult, error) {
	if s == nil || s.db == nil || s.k8sSvc == nil {
		return nil, ErrWithMessage(ErrConflict, "YAML 部署记录服务未初始化")
	}
	if req.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	req.YAML = strings.TrimSpace(req.YAML)
	req.DefaultNamespace = strings.TrimSpace(req.DefaultNamespace)
	req.SourceLabel = strings.TrimSpace(req.SourceLabel)
	req.SourceResource = strings.TrimSpace(req.SourceResource)
	req.WorkloadKind = strings.TrimSpace(req.WorkloadKind)
	req.CreatedByName = strings.TrimSpace(req.CreatedByName)
	if req.YAML == "" {
		return nil, ErrWithMessage(ErrInvalidParams, "yaml is required")
	}
	if req.SourceLabel == "" {
		req.SourceLabel = "通用 YAML 示例"
	}

	row := model.ManifestApplyRecord{
		ClusterID:        req.ClusterID,
		Status:           ManifestApplyStatusRunning,
		DryRun:           req.DryRun,
		DefaultNamespace: req.DefaultNamespace,
		SourceLabel:      req.SourceLabel,
		SourceResource:   req.SourceResource,
		WorkloadKind:     req.WorkloadKind,
		YAMLContent:      req.YAML,
		CreatedBy:        req.CreatedBy,
		CreatedByName:    req.CreatedByName,
		Summary:          buildManifestPendingSummary(req.DryRun),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}

	items, applyErr := s.k8sSvc.ApplyManifestYAML(ctx, req.ClusterID, req.YAML, ManifestApplyOptions{
		DefaultNamespace: req.DefaultNamespace,
		DryRun:           req.DryRun,
	})
	if applyErr != nil {
		update := map[string]any{
			"status":        ManifestApplyStatusFailed,
			"summary":       buildManifestFailureSummary(applyErr),
			"error_message": firstUserFacingError(applyErr),
		}
		_ = s.db.WithContext(ctx).Model(&model.ManifestApplyRecord{}).Where("id = ?", row.ID).Updates(update).Error
		return nil, applyErr
	}

	resultJSON, err := json.Marshal(items)
	if err != nil {
		update := map[string]any{
			"status":        ManifestApplyStatusFailed,
			"summary":       "执行成功，但结果序列化失败",
			"error_message": err.Error(),
		}
		_ = s.db.WithContext(ctx).Model(&model.ManifestApplyRecord{}).Where("id = ?", row.ID).Updates(update).Error
		return nil, ErrWithMessage(ErrConflict, "执行成功，但记录结果失败")
	}

	summary := buildManifestSuccessSummary(items, req.DryRun)
	if err := s.db.WithContext(ctx).Model(&model.ManifestApplyRecord{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":        ManifestApplyStatusSuccess,
		"result_json":   string(resultJSON),
		"result_count":  len(items),
		"summary":       summary,
		"error_message": "",
	}).Error; err != nil {
		return nil, ErrWithMessage(ErrConflict, "执行成功，但记录保存失败")
	}

	return &ManifestApplyExecuteResult{
		RecordID: row.ID,
		Status:   ManifestApplyStatusSuccess,
		DryRun:   req.DryRun,
		Summary:  summary,
		Items:    items,
	}, nil
}

func (s *ManifestApplyRecordService) List(ctx context.Context, p ManifestApplyRecordListParams) (*PageResult[ManifestApplyRecordListItem], error) {
	if s == nil || s.db == nil {
		return nil, ErrWithMessage(ErrConflict, "YAML 部署记录服务未初始化")
	}
	if p.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 || p.PageSize > 100 {
		p.PageSize = 20
	}

	q := s.db.WithContext(ctx).Model(&model.ManifestApplyRecord{}).Where("cluster_id = ?", p.ClusterID)
	if v := strings.TrimSpace(p.Keyword); v != "" {
		like := "%" + v + "%"
		q = q.Where("source_label LIKE ? OR source_resource LIKE ? OR workload_kind LIKE ? OR default_namespace LIKE ? OR summary LIKE ? OR error_message LIKE ? OR created_by_name LIKE ?", like, like, like, like, like, like, like)
	}
	if v := strings.TrimSpace(p.Status); v != "" {
		q = q.Where("status = ?", v)
	}
	if v := strings.ToLower(strings.TrimSpace(p.Mode)); v != "" {
		switch v {
		case "apply":
			q = q.Where("dry_run = ?", false)
		case "dry_run", "dryrun":
			q = q.Where("dry_run = ?", true)
		}
	}
	if v := strings.TrimSpace(p.DefaultNamespace); v != "" {
		q = q.Where("default_namespace LIKE ?", "%"+v+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var rows []model.ManifestApplyRecord
	offset := (p.Page - 1) * p.PageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(p.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ManifestApplyRecordListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toManifestRecordListItem(row))
	}
	return &PageResult[ManifestApplyRecordListItem]{List: items, Total: int(total), Page: p.Page, PageSize: p.PageSize}, nil
}

func (s *ManifestApplyRecordService) Get(ctx context.Context, clusterID, recordID uint64) (*ManifestApplyRecordDetail, error) {
	if s == nil || s.db == nil {
		return nil, ErrWithMessage(ErrConflict, "YAML 部署记录服务未初始化")
	}
	if clusterID == 0 || recordID == 0 {
		return nil, ErrInvalidParams
	}
	var row model.ManifestApplyRecord
	if err := s.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", recordID, clusterID).Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	items, err := parseManifestResultItems(row.ResultJSON)
	if err != nil {
		return nil, err
	}
	return &ManifestApplyRecordDetail{
		ManifestApplyRecordListItem: toManifestRecordListItem(row),
		ClusterID:                   row.ClusterID,
		YAMLContent:                 row.YAMLContent,
		ResultItems:                 items,
	}, nil
}

func toManifestRecordListItem(row model.ManifestApplyRecord) ManifestApplyRecordListItem {
	return ManifestApplyRecordListItem{
		ID:               row.ID,
		Status:           strings.TrimSpace(row.Status),
		DryRun:           row.DryRun,
		DefaultNamespace: row.DefaultNamespace,
		SourceLabel:      row.SourceLabel,
		SourceResource:   row.SourceResource,
		WorkloadKind:     row.WorkloadKind,
		ResultCount:      row.ResultCount,
		Summary:          strings.TrimSpace(row.Summary),
		ErrorMessage:     strings.TrimSpace(row.ErrorMessage),
		CreatedBy:        row.CreatedBy,
		CreatedByName:    row.CreatedByName,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func parseManifestResultItems(raw string) ([]ManifestApplyResultItem, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var items []ManifestApplyResultItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, ErrWithMessage(ErrConflict, "部署记录结果解析失败")
	}
	return items, nil
}

func buildManifestPendingSummary(dryRun bool) string {
	if dryRun {
		return "DryRun 校验中"
	}
	return "资源应用中"
}

func buildManifestSuccessSummary(items []ManifestApplyResultItem, dryRun bool) string {
	createCount := 0
	updateCount := 0
	for _, item := range items {
		switch strings.ToLower(strings.TrimSpace(item.Operation)) {
		case "create":
			createCount++
		case "update":
			updateCount++
		}
	}
	modeText := "Apply"
	if dryRun {
		modeText = "DryRun"
	}
	return fmt.Sprintf("%s 完成：%d 个资源，创建 %d，更新 %d", modeText, len(items), createCount, updateCount)
}

func buildManifestFailureSummary(err error) string {
	message := firstUserFacingError(err)
	if message == "" {
		message = "执行失败"
	}
	return message
}

func firstUserFacingError(err error) string {
	if msg, ok := UserMessage(err); ok {
		return strings.TrimSpace(msg)
	}
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
