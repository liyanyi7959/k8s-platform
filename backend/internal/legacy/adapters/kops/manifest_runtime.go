package kops

import (
	"context"
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

// ManifestRuntime adapts the retained Kubernetes executor and audit store to
// the Kops manifest port. This is deliberately the sole legacy dependency of
// the migrated manifest use case.
type ManifestRuntime struct {
	service *service.ManifestApplyRecordService
}

type NamespaceRuntime struct {
	k8s *service.K8sService
}

func NewNamespaceRuntime(k8s *service.K8sService) *NamespaceRuntime {
	return &NamespaceRuntime{k8s: k8s}
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
	if r == nil || r.k8s == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.k8s.CreateNamespace(ctx, clusterID, input.Name, input.Labels))
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
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	items, total, err := r.k8s.GetNamespaceResourcesSummary(ctx, clusterID, namespace)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{"key": item.Key, "label": item.Label, "count": item.Count})
	}
	return map[string]any{"namespace": namespace, "total": total, "items": out}, nil
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
func NewManifestRuntime(service *service.ManifestApplyRecordService) *ManifestRuntime {
	return &ManifestRuntime{service: service}
}

func (r *ManifestRuntime) Execute(ctx context.Context, input kopsapp.ManifestApplyInput) (*kopsapp.ManifestApplyResult, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	result, err := r.service.Execute(ctx, service.ManifestApplyExecuteRequest{
		ClusterID: input.ClusterID, YAML: input.YAML, DefaultNamespace: input.DefaultNamespace, DryRun: input.DryRun,
		CreateOnly: input.CreateOnly, SourceLabel: input.SourceLabel, SourceResource: input.SourceResource,
		WorkloadKind: input.WorkloadKind, CreatedBy: input.CreatedBy, CreatedByName: input.CreatedByName,
	})
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return &kopsapp.ManifestApplyResult{
		RecordID: result.RecordID, Status: result.Status, DryRun: result.DryRun, Summary: result.Summary,
		Items: manifestResultItems(result.Items),
	}, nil
}

func (r *ManifestRuntime) List(ctx context.Context, query kopsapp.ManifestRecordQuery) (*kopsapp.ManifestRecordPage, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	result, err := r.service.List(ctx, service.ManifestApplyRecordListParams{
		ClusterID: query.ClusterID, Page: query.Page, PageSize: query.PageSize, Keyword: query.Keyword,
		Status: query.Status, Mode: query.Mode, DefaultNamespace: query.DefaultNamespace,
	})
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	items := make([]kopsapp.ManifestRecordListItem, 0, len(result.List))
	for _, item := range result.List {
		items = append(items, manifestRecordListItem(item))
	}
	return &kopsapp.ManifestRecordPage{List: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize}, nil
}

func (r *ManifestRuntime) Get(ctx context.Context, clusterID, recordID uint64) (*kopsapp.ManifestRecordDetail, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	result, err := r.service.Get(ctx, clusterID, recordID)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return &kopsapp.ManifestRecordDetail{
		ManifestRecordListItem: manifestRecordListItem(result.ManifestApplyRecordListItem),
		ClusterID:              result.ClusterID, YAMLContent: result.YAMLContent, ResultItems: manifestResultItems(result.ResultItems),
	}, nil
}

func manifestResultItems(items []service.ManifestApplyResultItem) []kopsapp.ManifestResultItem {
	result := make([]kopsapp.ManifestResultItem, 0, len(items))
	for _, item := range items {
		result = append(result, kopsapp.ManifestResultItem{
			APIVersion: item.APIVersion, Kind: item.Kind, Namespace: item.Namespace, Name: item.Name,
			Operation: item.Operation, Resource: item.Resource, Scope: item.Scope,
		})
	}
	return result
}

func manifestRecordListItem(item service.ManifestApplyRecordListItem) kopsapp.ManifestRecordListItem {
	return kopsapp.ManifestRecordListItem{
		ID: item.ID, Status: item.Status, DryRun: item.DryRun, DefaultNamespace: item.DefaultNamespace,
		SourceLabel: item.SourceLabel, SourceResource: item.SourceResource, WorkloadKind: item.WorkloadKind,
		ResultCount: item.ResultCount, Summary: item.Summary, ErrorMessage: item.ErrorMessage,
		CreatedBy: item.CreatedBy, CreatedByName: item.CreatedByName, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func translateKopsRuntimeError(err error) error {
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
