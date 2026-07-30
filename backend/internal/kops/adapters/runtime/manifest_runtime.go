package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
	kopsdomain "k8s-platform-backend/internal/kops/domain"
)

// ManifestRuntime persists manifest operation records and delegates the
// Kubernetes apply itself to the retained runtime. It is an anti-corruption
// adapter; Kops application code has no dependency on legacy services.
type ManifestRuntime struct {
	db  *gorm.DB
	k8s *service.K8sService
}

type NamespaceRuntime struct {
	k8s     *service.K8sService
	nodes   *NodeOperations
	summary kopsapp.NamespaceResourceSummaryReader
}

func NewNamespaceRuntime(k8s *service.K8sService, nodes *NodeOperations, summary kopsapp.NamespaceResourceSummaryReader) *NamespaceRuntime {
	return &NamespaceRuntime{k8s: k8s, nodes: nodes, summary: summary}
}
func (r *NamespaceRuntime) List(ctx context.Context, clusterID uint64, sortBy, order string) (any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.k8s.List(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}, "", sortBy, order, nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}
func (r *NamespaceRuntime) Create(ctx context.Context, clusterID uint64, input kopsapp.NamespaceCreateInput) error {
	if r == nil || r.nodes == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.nodes.CreateNamespace(ctx, clusterID, input.Name, input.Labels))
}
func (r *NamespaceRuntime) Delete(ctx context.Context, clusterID uint64, namespace string) error {
	if r == nil || r.k8s == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.k8s.Delete(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}, "", namespace))
}
func (r *NamespaceRuntime) YAML(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.k8s.GetYAML(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}, "", namespace)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}
func (r *NamespaceRuntime) Summary(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if r == nil || r.summary == nil {
		return nil, kopsapp.ErrConflict
	}
	summary, err := r.summary.Summary(ctx, clusterID, namespace)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	out := make([]map[string]any, 0, len(summary.Items))
	for _, item := range summary.Items {
		out = append(out, map[string]any{"key": item.Key, "label": item.Label, "count": item.Count})
	}
	return map[string]any{"namespace": namespace, "total": summary.Total, "items": out}, nil
}
func (r *NamespaceRuntime) Events(ctx context.Context, query kopsapp.EventListQuery) (any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	selectors := make([]fields.Selector, 0, 3)
	if query.InvolvedObjectKind != "" {
		selectors = append(selectors, fields.OneTermEqualSelector("involvedObject.kind", query.InvolvedObjectKind))
	}
	if query.InvolvedObjectName != "" {
		selectors = append(selectors, fields.OneTermEqualSelector("involvedObject.name", query.InvolvedObjectName))
	}
	if query.InvolvedObjectUID != "" {
		selectors = append(selectors, fields.OneTermEqualSelector("involvedObject.uid", query.InvolvedObjectUID))
	}
	var extra map[string]string
	if len(selectors) > 0 {
		extra = map[string]string{"field_selector": strings.TrimSpace(fields.AndSelectors(selectors...).String())}
	}
	value, err := r.k8s.List(ctx, query.ClusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}, query.Namespace, query.SortBy, query.Order, extra)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}
func NewManifestRuntime(db *gorm.DB, k8s *service.K8sService) *ManifestRuntime {
	return &ManifestRuntime{db: db, k8s: k8s}
}

func (r *ManifestRuntime) Execute(ctx context.Context, input kopsapp.ManifestApplyInput) (*kopsapp.ManifestApplyResult, error) {
	if r == nil || r.db == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	if input.ClusterID == 0 || strings.TrimSpace(input.YAML) == "" {
		return nil, kopsapp.ErrInvalidParams
	}
	input.YAML, input.DefaultNamespace = strings.TrimSpace(input.YAML), strings.TrimSpace(input.DefaultNamespace)
	input.SourceLabel, input.SourceResource = strings.TrimSpace(input.SourceLabel), strings.TrimSpace(input.SourceResource)
	input.WorkloadKind, input.CreatedByName = strings.TrimSpace(input.WorkloadKind), strings.TrimSpace(input.CreatedByName)
	if input.SourceLabel == "" {
		input.SourceLabel = "通用 YAML 变更"
	}
	row := kopsdomain.ManifestApplyRecord{
		ClusterID: input.ClusterID, Status: "running", DryRun: input.DryRun, DefaultNamespace: input.DefaultNamespace,
		SourceLabel: input.SourceLabel, SourceResource: input.SourceResource, WorkloadKind: input.WorkloadKind,
		YAMLContent: input.YAML, CreatedBy: input.CreatedBy, CreatedByName: input.CreatedByName, Summary: manifestPendingSummary(input.DryRun),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	items, applyErr := r.applyManifestYAML(ctx, input.ClusterID, input.YAML, manifestApplyOptions{DefaultNamespace: input.DefaultNamespace, DryRun: input.DryRun, CreateOnly: input.CreateOnly})
	if applyErr != nil {
		_ = r.db.WithContext(ctx).Model(&kopsdomain.ManifestApplyRecord{}).Where("id = ?", row.ID).Updates(map[string]any{"status": "failed", "summary": manifestFailureSummary(applyErr), "error_message": manifestErrorMessage(applyErr)}).Error
		return nil, translateKopsRuntimeError(applyErr)
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		_ = r.db.WithContext(ctx).Model(&kopsdomain.ManifestApplyRecord{}).Where("id = ?", row.ID).Updates(map[string]any{"status": "failed", "summary": "执行成功，但结果序列化失败", "error_message": err.Error()}).Error
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrConflict, "执行成功，但记录结果失败")
	}
	summary := manifestSuccessSummary(items, input.DryRun)
	if err := r.db.WithContext(ctx).Model(&kopsdomain.ManifestApplyRecord{}).Where("id = ?", row.ID).Updates(map[string]any{"status": "success", "result_json": string(encoded), "result_count": len(items), "summary": summary, "error_message": ""}).Error; err != nil {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrConflict, "执行成功，但记录保存失败")
	}
	return &kopsapp.ManifestApplyResult{
		RecordID: row.ID, Status: "success", DryRun: input.DryRun, Summary: summary, Items: manifestResultItems(items),
	}, nil
}

func (r *ManifestRuntime) List(ctx context.Context, query kopsapp.ManifestRecordQuery) (*kopsapp.ManifestRecordPage, error) {
	if r == nil || r.db == nil {
		return nil, kopsapp.ErrConflict
	}
	if query.ClusterID == 0 {
		return nil, kopsapp.ErrInvalidParams
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 20
	}
	q := r.db.WithContext(ctx).Model(&kopsdomain.ManifestApplyRecord{}).Where("cluster_id = ?", query.ClusterID)
	if value := strings.TrimSpace(query.Keyword); value != "" {
		like := "%" + value + "%"
		q = q.Where("source_label LIKE ? OR source_resource LIKE ? OR workload_kind LIKE ? OR default_namespace LIKE ? OR summary LIKE ? OR error_message LIKE ? OR created_by_name LIKE ?", like, like, like, like, like, like, like)
	}
	if value := strings.TrimSpace(query.Status); value != "" {
		q = q.Where("status = ?", value)
	}
	switch strings.ToLower(strings.TrimSpace(query.Mode)) {
	case "apply":
		q = q.Where("dry_run = ?", false)
	case "dry_run", "dryrun":
		q = q.Where("dry_run = ?", true)
	}
	if value := strings.TrimSpace(query.DefaultNamespace); value != "" {
		q = q.Where("default_namespace LIKE ?", "%"+value+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []kopsdomain.ManifestApplyRecord
	if err := q.Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]kopsapp.ManifestRecordListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, manifestRecordListItem(row))
	}
	return &kopsapp.ManifestRecordPage{List: items, Total: int(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *ManifestRuntime) Get(ctx context.Context, clusterID, recordID uint64) (*kopsapp.ManifestRecordDetail, error) {
	if r == nil || r.db == nil {
		return nil, kopsapp.ErrConflict
	}
	if clusterID == 0 || recordID == 0 {
		return nil, kopsapp.ErrInvalidParams
	}
	var row kopsdomain.ManifestApplyRecord
	if err := r.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", recordID, clusterID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, kopsapp.ErrNotFound
		}
		return nil, err
	}
	items, err := parseManifestResultItems(row.ResultJSON)
	if err != nil {
		return nil, err
	}
	return &kopsapp.ManifestRecordDetail{ManifestRecordListItem: manifestRecordListItem(row), ClusterID: row.ClusterID, YAMLContent: row.YAMLContent, ResultItems: manifestResultItems(items)}, nil
}

func manifestResultItems(items []manifestApplyResultItem) []kopsapp.ManifestResultItem {
	result := make([]kopsapp.ManifestResultItem, 0, len(items))
	for _, item := range items {
		result = append(result, kopsapp.ManifestResultItem{APIVersion: item.APIVersion, Kind: item.Kind, Namespace: item.Namespace, Name: item.Name, Operation: item.Operation, Resource: item.Resource, Scope: item.Scope})
	}
	return result
}

func manifestRecordListItem(item kopsdomain.ManifestApplyRecord) kopsapp.ManifestRecordListItem {
	return kopsapp.ManifestRecordListItem{
		ID: item.ID, Status: item.Status, DryRun: item.DryRun, DefaultNamespace: item.DefaultNamespace,
		SourceLabel: item.SourceLabel, SourceResource: item.SourceResource, WorkloadKind: item.WorkloadKind,
		ResultCount: item.ResultCount, Summary: item.Summary, ErrorMessage: item.ErrorMessage,
		CreatedBy: item.CreatedBy, CreatedByName: item.CreatedByName, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func parseManifestResultItems(raw string) ([]manifestApplyResultItem, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var items []manifestApplyResultItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrConflict, "部署记录结果解析失败")
	}
	return items, nil
}

func manifestPendingSummary(dryRun bool) string {
	if dryRun {
		return "DryRun 校验中"
	}
	return "资源应用中"
}

func manifestSuccessSummary(items []manifestApplyResultItem, dryRun bool) string {
	created, updated := 0, 0
	for _, item := range items {
		switch strings.ToLower(strings.TrimSpace(item.Operation)) {
		case "create":
			created++
		case "update":
			updated++
		}
	}
	mode := "Apply"
	if dryRun {
		mode = "DryRun"
	}
	return fmt.Sprintf("%s 完成，%d 个资源，创建 %d，更新 %d", mode, len(items), created, updated)
}

func manifestErrorMessage(err error) string {
	if message, ok := service.UserMessage(err); ok {
		return strings.TrimSpace(message)
	}
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func manifestFailureSummary(err error) string {
	return firstManifestText(manifestErrorMessage(err), "执行失败")
}
func firstManifestText(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

// TranslateKopsRuntimeError maps shared Kubernetes transport errors to the
// Kops application error vocabulary. Cross-context integration adapters may
// use it at their boundary without importing a legacy implementation.
func TranslateKopsRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	for _, candidate := range []struct{ legacy, target error }{
		{service.ErrInvalidParams, kopsapp.ErrInvalidParams}, {service.ErrNotFound, kopsapp.ErrNotFound}, {service.ErrConflict, kopsapp.ErrConflict},
		{service.ErrK8sNetwork, kopsapp.ErrRuntimeNetwork}, {service.ErrK8sTimeout, kopsapp.ErrRuntimeTimeout},
		{service.ErrK8sUnauthorized, kopsapp.ErrRuntimeUnauthorized}, {service.ErrK8sForbidden, kopsapp.ErrRuntimeForbidden},
		{service.ErrK8sTLS, kopsapp.ErrRuntimeTLS}, {service.ErrK8s, kopsapp.ErrRuntime},
	} {
		if errors.Is(err, candidate.legacy) {
			if message, ok := service.UserMessage(err); ok {
				return kopsapp.ErrWithMessage(candidate.target, message)
			}
			return candidate.target
		}
	}
	return err
}

func translateKopsRuntimeError(err error) error {
	return TranslateKopsRuntimeError(err)
}
