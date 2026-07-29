package application

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"k8s-platform-backend/internal/ai/domain"
)

// ResourceReference describes a Kubernetes resource without exposing a
// client-go type to the AI use-case layer.
type ResourceReference struct {
	Group    string
	Version  string
	Resource string
}

// ResourceQueryRuntime is the Kubernetes read boundary used by AI resource
// queries. The retained legacy implementation owns client selection, API
// compatibility and transport-error translation.
type ResourceQueryRuntime interface {
	List(context.Context, uint64, ResourceReference, string, string, string, map[string]string) ([]map[string]any, error)
	Get(context.Context, uint64, ResourceReference, string, string) (map[string]any, error)
	ListNodeEvents(context.Context, uint64, string) ([]map[string]any, error)
	PodLogs(context.Context, uint64, string, string, string, int64, bool) (string, error)
}

// ResourceQueryPresenter preserves the established AI evidence shape while
// the legacy read-model formatters finish their migration separately.
type ResourceQueryPresenter interface {
	ListItemSummary(string, map[string]any) domain.JSONMap
}

// ResourceQueryPort lets the remaining tool registry depend on the AI use
// case instead of a legacy service implementation.
type ResourceQueryPort interface {
	ListResources(context.Context, uint64, string, string, string, string, string, int) (ToolResult, error)
	SearchResources(context.Context, uint64, string, string, []string, int) (ToolResult, error)
	GetResourceEvents(context.Context, uint64, string, string, string) (ToolResult, error)
	GetResourceLogs(context.Context, uint64, string, string, string, string, int64, bool) (ToolResult, error)
	GetRelatedResources(context.Context, uint64, string, string, string) (ToolResult, error)
}

type ResourceQueryService struct {
	runtime   ResourceQueryRuntime
	presenter ResourceQueryPresenter
}

func NewResourceQueryService(runtime ResourceQueryRuntime, presenter ResourceQueryPresenter) *ResourceQueryService {
	return &ResourceQueryService{runtime: runtime, presenter: presenter}
}

func (s *ResourceQueryService) ListResources(ctx context.Context, clusterID uint64, kind, namespace, labelSelector, sortBy, order string, limit int) (ToolResult, error) {
	if !s.ready() {
		return ToolResult{}, ErrConflict
	}
	resourceKind := strings.TrimSpace(kind)
	namespace = strings.TrimSpace(namespace)
	resource, namespaced, ok := supportedResource(resourceKind)
	if !ok {
		return ToolResult{}, ErrWithMessage(ErrInvalidParams, "unsupported resource kind for AI list")
	}
	if !namespaced {
		namespace = ""
	}
	labelSelector = strings.TrimSpace(labelSelector)
	items, err := s.runtime.List(ctx, clusterID, resource, namespace, firstNonEmpty(sortBy, "metadata.name"), firstNonEmpty(order, "asc"), map[string]string{"label_selector": labelSelector})
	if err != nil {
		return ToolResult{}, err
	}
	limited := clampResultLimit(limit, 50)
	summaries := make([]domain.JSONMap, 0, min(len(items), limited))
	for _, item := range items {
		summaries = append(summaries, s.listItemSummary(resourceKind, item))
		if len(summaries) >= limited {
			break
		}
	}
	scope := resourceKind
	if namespace != "" {
		scope = fmt.Sprintf("%s in namespace %s", resourceKind, namespace)
	}
	summary := fmt.Sprintf("Listed %d of %d %s resources", len(summaries), len(items), scope)
	if labelSelector != "" {
		summary += " with label selector " + labelSelector
	}
	return ToolResult{Summary: summary, Evidence: domain.JSONMap{
		"kind": resourceKind, "namespace": namespace, "label_selector": labelSelector,
		"total": len(items), "returned": len(summaries), "items": summaries,
	}, RawRef: domain.JSONMap{"cluster_id": clusterID, "kind": resourceKind, "namespace": namespace, "source": "resource.list"}}, nil
}

func (s *ResourceQueryService) SearchResources(ctx context.Context, clusterID uint64, keyword, namespace string, kinds []string, limit int) (ToolResult, error) {
	if !s.ready() {
		return ToolResult{}, ErrConflict
	}
	query := strings.ToLower(strings.TrimSpace(keyword))
	if query == "" {
		return ToolResult{}, ErrWithMessage(ErrInvalidParams, "search keyword is required")
	}
	namespace = strings.TrimSpace(namespace)
	candidates := normalizeSearchKinds(kinds, namespace != "")
	matches := make([]domain.JSONMap, 0, 64)
	for _, kind := range candidates {
		resource, namespaced, ok := supportedResource(kind)
		if !ok {
			continue
		}
		targetNamespace := namespace
		if !namespaced {
			targetNamespace = ""
		}
		items, err := s.runtime.List(ctx, clusterID, resource, targetNamespace, "metadata.name", "asc", nil)
		if err != nil {
			return ToolResult{}, err
		}
		for _, item := range items {
			if !strings.Contains(strings.ToLower(objectMetaString(item, "name")), query) {
				continue
			}
			matches = append(matches, s.listItemSummary(kind, item))
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return resultSortKey(matches[i]) < resultSortKey(matches[j]) })
	limited := clampResultLimit(limit, 50)
	if len(matches) > limited {
		matches = matches[:limited]
	}
	scope := "all supported resources"
	if namespace != "" {
		scope = "namespace " + namespace
	}
	return ToolResult{Summary: fmt.Sprintf("Search for %q returned %d matches in %s", keyword, len(matches), scope), Evidence: domain.JSONMap{
		"keyword": strings.TrimSpace(keyword), "namespace": namespace, "kinds": candidates, "returned": len(matches), "items": matches,
	}, RawRef: domain.JSONMap{"cluster_id": clusterID, "namespace": namespace, "source": "resource.search"}}, nil
}

func (s *ResourceQueryService) GetResourceEvents(ctx context.Context, clusterID uint64, kind, namespace, name string) (ToolResult, error) {
	if !s.ready() {
		return ToolResult{}, ErrConflict
	}
	resourceKind, namespace, name := strings.TrimSpace(kind), strings.TrimSpace(namespace), strings.TrimSpace(name)
	if resourceKind == "" || name == "" {
		return ToolResult{}, ErrInvalidParams
	}
	if strings.EqualFold(resourceKind, "Node") {
		events, err := s.runtime.ListNodeEvents(ctx, clusterID, name)
		if err != nil {
			return ToolResult{}, err
		}
		return eventResult(clusterID, resourceKind, "", name, events), nil
	}
	resource, namespaced, ok := supportedResource(resourceKind)
	if !ok {
		return ToolResult{}, ErrWithMessage(ErrInvalidParams, "unsupported resource kind for AI events")
	}
	if namespaced && namespace == "" {
		return ToolResult{}, ErrWithMessage(ErrInvalidParams, "namespace is required for the selected resource kind")
	}
	if !namespaced {
		namespace = ""
	}
	object, err := s.runtime.Get(ctx, clusterID, resource, namespace, name)
	if err != nil {
		return ToolResult{}, err
	}
	events, err := s.runtime.List(ctx, clusterID, ResourceReference{Version: "v1", Resource: "events"}, namespace, "", "", map[string]string{"field_selector": eventFieldSelector(resourceKind, namespace, name, object)})
	if err != nil {
		return ToolResult{}, err
	}
	return eventResult(clusterID, resourceKind, namespace, name, events), nil
}

func (s *ResourceQueryService) GetResourceLogs(ctx context.Context, clusterID uint64, kind, namespace, name, container string, tailLines int64, previous bool) (ToolResult, error) {
	if !s.ready() {
		return ToolResult{}, ErrConflict
	}
	resourceKind, namespace, name := strings.TrimSpace(kind), strings.TrimSpace(namespace), strings.TrimSpace(name)
	if resourceKind == "" || name == "" {
		return ToolResult{}, ErrInvalidParams
	}
	if tailLines <= 0 {
		tailLines = 80
	}
	podName := name
	relatedPods := []domain.JSONMap(nil)
	if !strings.EqualFold(resourceKind, "Pod") {
		var err error
		relatedPods, err = s.relatedPods(ctx, clusterID, resourceKind, namespace, name)
		if err != nil {
			return ToolResult{}, err
		}
		if len(relatedPods) == 0 {
			return ToolResult{}, ErrWithMessage(ErrNotFound, "no related pods found for the selected resource")
		}
		podName = strings.TrimSpace(fmt.Sprint(relatedPods[0]["name"]))
	}
	logs, err := s.runtime.PodLogs(ctx, clusterID, namespace, podName, strings.TrimSpace(container), tailLines, previous)
	if err != nil {
		return ToolResult{}, err
	}
	evidence := domain.JSONMap{"kind": resourceKind, "namespace": namespace, "name": name, "pod_name": podName, "container": strings.TrimSpace(container), "tail_lines": tailLines, "previous": previous, "logs": truncate(logs, 3000)}
	if len(relatedPods) > 0 {
		evidence["related_pods"] = relatedPods
	}
	return ToolResult{Summary: fmt.Sprintf("Collected logs for %s via pod %s/%s", resourceKind, namespace, podName), Evidence: evidence, RawRef: domain.JSONMap{"cluster_id": clusterID, "namespace": namespace, "name": name, "pod_name": podName, "kind": resourceKind, "source": "resource.logs"}}, nil
}

func (s *ResourceQueryService) GetRelatedResources(ctx context.Context, clusterID uint64, kind, namespace, name string) (ToolResult, error) {
	if !s.ready() {
		return ToolResult{}, ErrConflict
	}
	resourceKind, namespace, name := strings.TrimSpace(kind), strings.TrimSpace(namespace), strings.TrimSpace(name)
	if resourceKind == "" || name == "" {
		return ToolResult{}, ErrInvalidParams
	}
	var pods, controllers []domain.JSONMap
	var err error
	switch strings.ToLower(resourceKind) {
	case "configmap":
		pods, controllers, err = s.namedResourceConsumers(ctx, clusterID, namespace, name, podUsesConfigMap)
	case "secret":
		pods, controllers, err = s.namedResourceConsumers(ctx, clusterID, namespace, name, podUsesSecret)
	default:
		pods, err = s.relatedPods(ctx, clusterID, resourceKind, namespace, name)
		controllers = controllersFromPods(pods)
	}
	if err != nil {
		return ToolResult{}, err
	}
	return relatedResult(clusterID, namespace, resourceKind, name, pods, controllers), nil
}

func (s *ResourceQueryService) ready() bool { return s != nil && s.runtime != nil }

func (s *ResourceQueryService) listItemSummary(kind string, object map[string]any) domain.JSONMap {
	if s.presenter != nil {
		return s.presenter.ListItemSummary(kind, object)
	}
	return domain.JSONMap{"kind": strings.TrimSpace(kind), "namespace": objectMetaString(object, "namespace"), "name": objectMetaString(object, "name")}
}

func (s *ResourceQueryService) relatedPods(ctx context.Context, clusterID uint64, kind, namespace, name string) ([]domain.JSONMap, error) {
	resourceKind := strings.TrimSpace(kind)
	switch strings.ToLower(resourceKind) {
	case "pod":
		object, err := s.runtime.Get(ctx, clusterID, ResourceReference{Version: "v1", Resource: "pods"}, namespace, name)
		if err != nil {
			return nil, err
		}
		return []domain.JSONMap{podReference(object)}, nil
	case "deployment", "statefulset", "daemonset", "replicaset", "job":
		resource, _, ok := supportedResource(resourceKind)
		if !ok {
			return nil, ErrInvalidParams
		}
		object, err := s.runtime.Get(ctx, clusterID, resource, namespace, name)
		if err != nil {
			return nil, err
		}
		selector := workloadLabelSelector(object)
		if selector == "" {
			return nil, nil
		}
		return s.podsByLabelSelector(ctx, clusterID, namespace, selector)
	case "cronjob":
		object, err := s.runtime.Get(ctx, clusterID, ResourceReference{Group: "batch", Version: "v1", Resource: "cronjobs"}, namespace, name)
		if err != nil {
			return nil, err
		}
		selector := cronJobTemplateSelector(object)
		if selector == "" {
			return nil, nil
		}
		return s.podsByLabelSelector(ctx, clusterID, namespace, selector)
	case "service":
		object, err := s.runtime.Get(ctx, clusterID, ResourceReference{Version: "v1", Resource: "services"}, namespace, name)
		if err != nil {
			return nil, err
		}
		selector := mapToLabelSelector(mapValue(mapValue(object, "spec"), "selector"))
		if selector == "" {
			return nil, nil
		}
		return s.podsByLabelSelector(ctx, clusterID, namespace, selector)
	case "persistentvolumeclaim":
		items, err := s.runtime.List(ctx, clusterID, ResourceReference{Version: "v1", Resource: "pods"}, namespace, "metadata.name", "asc", nil)
		if err != nil {
			return nil, err
		}
		pods := make([]domain.JSONMap, 0, 16)
		for _, item := range items {
			if podUsesPVC(item, name) {
				pods = append(pods, podReference(item))
			}
		}
		return pods, nil
	default:
		return nil, nil
	}
}

func (s *ResourceQueryService) podsByLabelSelector(ctx context.Context, clusterID uint64, namespace, selector string) ([]domain.JSONMap, error) {
	items, err := s.runtime.List(ctx, clusterID, ResourceReference{Version: "v1", Resource: "pods"}, namespace, "metadata.name", "asc", map[string]string{"label_selector": strings.TrimSpace(selector)})
	if err != nil {
		return nil, err
	}
	pods := make([]domain.JSONMap, 0, min(len(items), 20))
	for _, item := range items {
		pods = append(pods, podReference(item))
		if len(pods) >= 20 {
			break
		}
	}
	return pods, nil
}

func (s *ResourceQueryService) namedResourceConsumers(ctx context.Context, clusterID uint64, namespace, name string, matches func(map[string]any, string) bool) ([]domain.JSONMap, []domain.JSONMap, error) {
	items, err := s.runtime.List(ctx, clusterID, ResourceReference{Version: "v1", Resource: "pods"}, namespace, "metadata.name", "asc", nil)
	if err != nil {
		return nil, nil, err
	}
	pods := make([]domain.JSONMap, 0, 16)
	controllersByKey := map[string]domain.JSONMap{}
	for _, item := range items {
		if !matches(item, name) {
			continue
		}
		pods = append(pods, podReference(item))
		for _, owner := range ownerReferences(item) {
			kind, ownerName := strings.TrimSpace(fmt.Sprint(owner["kind"])), strings.TrimSpace(fmt.Sprint(owner["name"]))
			if kind != "" || ownerName != "" {
				controllersByKey[kind+"/"+ownerName] = domain.JSONMap{"kind": kind, "name": ownerName}
			}
		}
	}
	controllers := make([]domain.JSONMap, 0, len(controllersByKey))
	for _, controller := range controllersByKey {
		controllers = append(controllers, controller)
	}
	sortControllers(controllers)
	return pods, controllers, nil
}

func eventResult(clusterID uint64, kind, namespace, name string, events []map[string]any) ToolResult {
	items := eventEvidenceItems(events)
	summary := fmt.Sprintf("Collected %d events for %s %s/%s", len(items), kind, namespace, name)
	if strings.EqualFold(kind, "Node") {
		summary = fmt.Sprintf("Collected %d node events for %s", len(items), name)
	}
	evidence := domain.JSONMap{"kind": kind, "name": name, "events": items}
	return ToolResult{Summary: summary, Evidence: evidence, RawRef: domain.JSONMap{"cluster_id": clusterID, "namespace": namespace, "name": name, "kind": kind, "source": "resource.events"}}
}

func relatedResult(clusterID uint64, namespace, kind, name string, pods, controllers []domain.JSONMap) ToolResult {
	return ToolResult{Summary: fmt.Sprintf("Found %d related pods and %d related controllers for %s %s/%s", len(pods), len(controllers), kind, namespace, name), Evidence: domain.JSONMap{
		"kind": kind, "namespace": namespace, "name": name, "pod_count": len(pods), "pods": pods, "controllers": controllers,
	}, RawRef: domain.JSONMap{"cluster_id": clusterID, "namespace": namespace, "name": name, "kind": kind, "source": "resource.related"}}
}

func supportedResource(kind string) (ResourceReference, bool, bool) {
	type definition struct{ group, version, resource string; namespaced bool }
	resources := map[string]definition{
		"namespace": {"", "v1", "namespaces", false}, "node": {"", "v1", "nodes", false}, "pod": {"", "v1", "pods", true},
		"deployment": {"apps", "v1", "deployments", true}, "statefulset": {"apps", "v1", "statefulsets", true}, "daemonset": {"apps", "v1", "daemonsets", true}, "replicaset": {"apps", "v1", "replicasets", true},
		"service": {"", "v1", "services", true}, "ingress": {"networking.k8s.io", "v1", "ingresses", true}, "networkpolicy": {"networking.k8s.io", "v1", "networkpolicies", true},
		"configmap": {"", "v1", "configmaps", true}, "secret": {"", "v1", "secrets", true}, "serviceaccount": {"", "v1", "serviceaccounts", true},
		"persistentvolumeclaim": {"", "v1", "persistentvolumeclaims", true}, "pvc": {"", "v1", "persistentvolumeclaims", true}, "persistentvolume": {"", "v1", "persistentvolumes", false}, "pv": {"", "v1", "persistentvolumes", false},
		"storageclass": {"storage.k8s.io", "v1", "storageclasses", false}, "job": {"batch", "v1", "jobs", true}, "cronjob": {"batch", "v1", "cronjobs", true},
	}
	resource, ok := resources[strings.ToLower(strings.TrimSpace(kind))]
	if !ok {
		return ResourceReference{}, false, false
	}
	return ResourceReference{Group: resource.group, Version: resource.version, Resource: resource.resource}, resource.namespaced, true
}

func normalizeSearchKinds(kinds []string, namespacedOnly bool) []string {
	if len(kinds) == 0 {
		result := []string{"Pod", "Deployment", "StatefulSet", "DaemonSet", "Service", "Ingress", "ConfigMap", "Secret", "PersistentVolumeClaim", "Job", "CronJob"}
		if !namespacedOnly {
			result = append(result, "Namespace", "Node", "PersistentVolume", "StorageClass")
		}
		return result
	}
	result, seen := make([]string, 0, len(kinds)), map[string]struct{}{}
	for _, kind := range kinds {
		normalized := normalizeResourceKind(kind)
		if normalized == "" {
			continue
		}
		_, namespaced, ok := supportedResource(normalized)
		if !ok || (namespacedOnly && !namespaced) {
			continue
		}
		if _, exists := seen[normalized]; !exists {
			seen[normalized] = struct{}{}
			result = append(result, normalized)
		}
	}
	return result
}

func normalizeResourceKind(kind string) string {
	known := map[string]string{"namespace": "Namespace", "node": "Node", "pod": "Pod", "deployment": "Deployment", "statefulset": "StatefulSet", "daemonset": "DaemonSet", "replicaset": "ReplicaSet", "service": "Service", "ingress": "Ingress", "networkpolicy": "NetworkPolicy", "configmap": "ConfigMap", "secret": "Secret", "serviceaccount": "ServiceAccount", "persistentvolumeclaim": "PersistentVolumeClaim", "pvc": "PersistentVolumeClaim", "persistentvolume": "PersistentVolume", "pv": "PersistentVolume", "storageclass": "StorageClass", "job": "Job", "cronjob": "CronJob"}
	if normalized, ok := known[strings.ToLower(strings.TrimSpace(kind))]; ok {
		return normalized
	}
	return strings.TrimSpace(kind)
}

func eventFieldSelector(kind, namespace, name string, object map[string]any) string {
	parts := []string{"involvedObject.kind=" + strings.TrimSpace(kind), "involvedObject.name=" + strings.TrimSpace(name)}
	if namespace = strings.TrimSpace(namespace); namespace != "" {
		parts = append(parts, "involvedObject.namespace="+namespace)
	}
	if uid := strings.TrimSpace(fmt.Sprint(mapValue(object, "metadata")["uid"])); uid != "" {
		parts = append(parts, "involvedObject.uid="+uid)
	}
	return strings.Join(parts, ",")
}

func eventEvidenceItems(items []map[string]any) []domain.JSONMap {
	evidence := make([]domain.JSONMap, 0, min(len(items), 20))
	for _, item := range items {
		involved := mapValue(item, "involvedObject")
		evidence = append(evidence, domain.JSONMap{
			"type": strings.TrimSpace(fmt.Sprint(item["type"])), "reason": strings.TrimSpace(fmt.Sprint(item["reason"])), "message": strings.TrimSpace(fmt.Sprint(item["message"])),
			"namespace": objectMetaString(item, "namespace"), "name": objectMetaString(item, "name"),
			"event_time": firstNonEmpty(strings.TrimSpace(fmt.Sprint(item["eventTime"])), strings.TrimSpace(fmt.Sprint(item["lastTimestamp"])), strings.TrimSpace(fmt.Sprint(mapValue(item, "metadata")["creationTimestamp"]))),
			"involved_kind": strings.TrimSpace(fmt.Sprint(involved["kind"])), "involved_name": strings.TrimSpace(fmt.Sprint(involved["name"])), "involved_uid": strings.TrimSpace(fmt.Sprint(involved["uid"])), "count": intValue(item["count"]),
		})
		if len(evidence) >= 20 {
			break
		}
	}
	return evidence
}

func podReference(object map[string]any) domain.JSONMap {
	spec, status := mapValue(object, "spec"), mapValue(object, "status")
	ready, restarts := podReadyAndRestarts(object)
	return domain.JSONMap{"kind": "Pod", "namespace": objectMetaString(object, "namespace"), "name": objectMetaString(object, "name"), "phase": strings.TrimSpace(fmt.Sprint(status["phase"])), "node": strings.TrimSpace(fmt.Sprint(spec["nodeName"])), "ready": ready, "restarts": restarts, "owners": ownerReferences(object)}
}

func podReadyAndRestarts(object map[string]any) (string, int) {
	statuses, _ := mapValue(object, "status")["containerStatuses"].([]any)
	if len(statuses) == 0 {
		return "-", 0
	}
	ready, restarts := 0, 0
	for _, raw := range statuses {
		status, _ := raw.(map[string]any)
		if boolValue(status["ready"]) {
			ready++
		}
		restarts += intValue(status["restartCount"])
	}
	return fmt.Sprintf("%d/%d", ready, len(statuses)), restarts
}

func ownerReferences(object map[string]any) []domain.JSONMap {
	rawOwners, _ := mapValue(object, "metadata")["ownerReferences"].([]any)
	owners := make([]domain.JSONMap, 0, len(rawOwners))
	for _, raw := range rawOwners {
		owner, _ := raw.(map[string]any)
		if owner == nil {
			continue
		}
		owners = append(owners, domain.JSONMap{"api_version": strings.TrimSpace(fmt.Sprint(owner["apiVersion"])), "kind": strings.TrimSpace(fmt.Sprint(owner["kind"])), "name": strings.TrimSpace(fmt.Sprint(owner["name"])), "uid": strings.TrimSpace(fmt.Sprint(owner["uid"])), "controller": boolValue(owner["controller"])})
	}
	return owners
}

func workloadLabelSelector(object map[string]any) string {
	selector := mapValue(mapValue(object, "spec"), "selector")
	return mapToLabelSelector(mapValue(selector, "matchLabels"))
}

func cronJobTemplateSelector(object map[string]any) string {
	template := mapValue(mapValue(mapValue(mapValue(object, "spec"), "jobTemplate"), "spec"), "template")
	return mapToLabelSelector(mapValue(mapValue(template, "metadata"), "labels"))
}

func podUsesPVC(pod map[string]any, name string) bool {
	volumes, _ := mapValue(pod, "spec")["volumes"].([]any)
	for _, raw := range volumes {
		volume, _ := raw.(map[string]any)
		if strings.TrimSpace(fmt.Sprint(mapValue(volume, "persistentVolumeClaim")["claimName"])) == name {
			return true
		}
	}
	return false
}

func podUsesConfigMap(pod map[string]any, name string) bool {
	spec := mapValue(pod, "spec")
	volumes, _ := spec["volumes"].([]any)
	for _, raw := range volumes {
		volume, _ := raw.(map[string]any)
		if strings.TrimSpace(fmt.Sprint(mapValue(volume, "configMap")["name"])) == name {
			return true
		}
		sources, _ := mapValue(volume, "projected")["sources"].([]any)
		for _, source := range sources {
			if strings.TrimSpace(fmt.Sprint(mapValue(mapFromAny(source), "configMap")["name"])) == name {
				return true
			}
		}
	}
	return containersUseNamedRef(spec, name, "configMapRef", "configMapKeyRef")
}

func podUsesSecret(pod map[string]any, name string) bool {
	spec := mapValue(pod, "spec")
	volumes, _ := spec["volumes"].([]any)
	for _, raw := range volumes {
		volume, _ := raw.(map[string]any)
		if strings.TrimSpace(fmt.Sprint(mapValue(volume, "secret")["secretName"])) == name || strings.TrimSpace(fmt.Sprint(mapValue(mapValue(volume, "csi"), "nodePublishSecretRef")["name"])) == name {
			return true
		}
		sources, _ := mapValue(volume, "projected")["sources"].([]any)
		for _, source := range sources {
			if strings.TrimSpace(fmt.Sprint(mapValue(mapFromAny(source), "secret")["name"])) == name {
				return true
			}
		}
	}
	imagePullSecrets, _ := spec["imagePullSecrets"].([]any)
	for _, raw := range imagePullSecrets {
		if strings.TrimSpace(fmt.Sprint(mapFromAny(raw)["name"])) == name {
			return true
		}
	}
	return containersUseNamedRef(spec, name, "secretRef", "secretKeyRef")
}

func containersUseNamedRef(spec map[string]any, name, fromKey, valueKey string) bool {
	for _, key := range []string{"containers", "initContainers"} {
		containers, _ := spec[key].([]any)
		for _, raw := range containers {
			container := mapFromAny(raw)
			envFrom, _ := container["envFrom"].([]any)
			for _, rawRef := range envFrom {
				if strings.TrimSpace(fmt.Sprint(mapValue(mapFromAny(rawRef), fromKey)["name"])) == name {
					return true
				}
			}
			env, _ := container["env"].([]any)
			for _, rawEnv := range env {
				valueFrom := mapValue(mapFromAny(rawEnv), "valueFrom")
				if strings.TrimSpace(fmt.Sprint(mapValue(valueFrom, valueKey)["name"])) == name {
					return true
				}
			}
		}
	}
	return false
}

func controllersFromPods(pods []domain.JSONMap) []domain.JSONMap {
	byKey := map[string]domain.JSONMap{}
	for _, pod := range pods {
		owners, _ := pod["owners"].([]domain.JSONMap)
		for _, owner := range owners {
			kind, name := strings.TrimSpace(fmt.Sprint(owner["kind"])), strings.TrimSpace(fmt.Sprint(owner["name"]))
			if kind != "" && name != "" {
				byKey[kind+"/"+name] = domain.JSONMap{"kind": kind, "name": name}
			}
		}
	}
	controllers := make([]domain.JSONMap, 0, len(byKey))
	for _, controller := range byKey {
		controllers = append(controllers, controller)
	}
	sortControllers(controllers)
	return controllers
}

func sortControllers(controllers []domain.JSONMap) {
	sort.SliceStable(controllers, func(i, j int) bool {
		return strings.TrimSpace(fmt.Sprint(controllers[i]["kind"]))+"/"+strings.TrimSpace(fmt.Sprint(controllers[i]["name"])) < strings.TrimSpace(fmt.Sprint(controllers[j]["kind"]))+"/"+strings.TrimSpace(fmt.Sprint(controllers[j]["name"]))
	})
}

func mapToLabelSelector(labels map[string]any) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strings.TrimSpace(fmt.Sprint(labels[key])))
	}
	return strings.Join(parts, ",")
}

func resultSortKey(result domain.JSONMap) string {
	return strings.TrimSpace(fmt.Sprint(result["kind"])) + "\x00" + strings.TrimSpace(fmt.Sprint(result["namespace"])) + "\x00" + strings.TrimSpace(fmt.Sprint(result["name"]))
}

func objectMetaString(object map[string]any, key string) string {
	return strings.TrimSpace(fmt.Sprint(mapValue(object, "metadata")[key]))
}

func mapValue(source map[string]any, key string) map[string]any {
	if source == nil {
		return nil
	}
	value, _ := source[key].(map[string]any)
	return value
}

func mapFromAny(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return err == nil && parsed
	default:
		return false
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	default:
		return 0
	}
}

func truncate(input string, limit int) string {
	raw := strings.TrimSpace(input)
	if limit <= 0 || len([]rune(raw)) <= limit {
		return raw
	}
	return string([]rune(raw)[:limit]) + "..."
}

func clampResultLimit(limit, fallback int) int {
	if fallback <= 0 {
		fallback = 50
	}
	if limit <= 0 {
		limit = fallback
	}
	if limit < 1 {
		return 1
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
