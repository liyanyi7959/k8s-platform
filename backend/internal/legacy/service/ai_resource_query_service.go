package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s-platform-backend/internal/legacy/model"
)

type ResourceQueryService struct {
	k8sSvc *K8sService
}

func NewResourceQueryService(k8sSvc *K8sService) *ResourceQueryService {
	return &ResourceQueryService{k8sSvc: k8sSvc}
}

func (s *ResourceQueryService) ListResources(
	ctx context.Context,
	clusterID uint64,
	kind,
	namespace,
	labelSelector,
	sortBy,
	order string,
	limit int,
) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	resKind := strings.TrimSpace(kind)
	ns := strings.TrimSpace(namespace)
	gvr, namespaced, ok := aiSupportedResourceGVR(resKind)
	if !ok {
		return AIToolResult{}, ErrWithMessage(ErrInvalidParams, "unsupported resource kind for AI list")
	}
	if !namespaced {
		ns = ""
	}

	items, err := s.k8sSvc.List(ctx, clusterID, gvr, ns, firstNonEmpty(sortBy, "metadata.name"), firstNonEmpty(order, "asc"), map[string]string{
		"label_selector": strings.TrimSpace(labelSelector),
	})
	if err != nil {
		return AIToolResult{}, err
	}

	limited := clampAIResultLimit(limit, 50)
	summaries := make([]model.JSONMap, 0, minInt(len(items), limited))
	for _, item := range items {
		obj, _ := item.(map[string]any)
		if obj == nil {
			continue
		}
		summaries = append(summaries, buildAIListItemSummary(resKind, obj))
		if len(summaries) >= limited {
			break
		}
	}

	scopeText := resKind
	if ns != "" {
		scopeText = fmt.Sprintf("%s in namespace %s", resKind, ns)
	}
	summary := fmt.Sprintf("Listed %d of %d %s resources", len(summaries), len(items), scopeText)
	if labelSelector = strings.TrimSpace(labelSelector); labelSelector != "" {
		summary += " with label selector " + labelSelector
	}

	return AIToolResult{
		Summary: summary,
		Evidence: model.JSONMap{
			"kind":          resKind,
			"namespace":     ns,
			"label_selector": strings.TrimSpace(labelSelector),
			"total":         len(items),
			"returned":      len(summaries),
			"items":         summaries,
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"kind":       resKind,
			"namespace":  ns,
			"source":     "resource.list",
		},
	}, nil
}

func (s *ResourceQueryService) SearchResources(
	ctx context.Context,
	clusterID uint64,
	keyword,
	namespace string,
	kinds []string,
	limit int,
) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	query := strings.ToLower(strings.TrimSpace(keyword))
	if query == "" {
		return AIToolResult{}, ErrWithMessage(ErrInvalidParams, "search keyword is required")
	}
	ns := strings.TrimSpace(namespace)

	candidateKinds := normalizeAIResourceSearchKinds(kinds, ns != "")
	matches := make([]model.JSONMap, 0, 64)
	for _, kind := range candidateKinds {
		gvr, namespaced, ok := aiSupportedResourceGVR(kind)
		if !ok {
			continue
		}
		targetNamespace := ns
		if !namespaced {
			targetNamespace = ""
		}
		items, err := s.k8sSvc.List(ctx, clusterID, gvr, targetNamespace, "metadata.name", "asc", nil)
		if err != nil {
			return AIToolResult{}, err
		}
		for _, item := range items {
			obj, _ := item.(map[string]any)
			if obj == nil {
				continue
			}
			name := strings.ToLower(aiObjectMetaString(obj, "name"))
			if !strings.Contains(name, query) {
				continue
			}
			matches = append(matches, buildAIListItemSummary(kind, obj))
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		leftKind := strings.TrimSpace(fmt.Sprint(matches[i]["kind"]))
		rightKind := strings.TrimSpace(fmt.Sprint(matches[j]["kind"]))
		if leftKind == rightKind {
			leftNamespace := strings.TrimSpace(fmt.Sprint(matches[i]["namespace"]))
			rightNamespace := strings.TrimSpace(fmt.Sprint(matches[j]["namespace"]))
			if leftNamespace == rightNamespace {
				return strings.TrimSpace(fmt.Sprint(matches[i]["name"])) < strings.TrimSpace(fmt.Sprint(matches[j]["name"]))
			}
			return leftNamespace < rightNamespace
		}
		return leftKind < rightKind
	})

	limited := clampAIResultLimit(limit, 50)
	if len(matches) > limited {
		matches = matches[:limited]
	}

	scopeText := "all supported resources"
	if ns != "" {
		scopeText = "namespace " + ns
	}
	return AIToolResult{
		Summary: fmt.Sprintf("Search for %q returned %d matches in %s", keyword, len(matches), scopeText),
		Evidence: model.JSONMap{
			"keyword":   strings.TrimSpace(keyword),
			"namespace": ns,
			"kinds":     candidateKinds,
			"returned":  len(matches),
			"items":     matches,
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"source":     "resource.search",
		},
	}, nil
}

func (s *ResourceQueryService) GetResourceEvents(ctx context.Context, clusterID uint64, kind, namespace, name string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	resKind := strings.TrimSpace(kind)
	ns := strings.TrimSpace(namespace)
	resName := strings.TrimSpace(name)
	if resKind == "" || resName == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	if strings.EqualFold(resKind, "Node") {
		events, err := s.k8sSvc.ListNodeEvents(ctx, clusterID, resName)
		if err != nil {
			return AIToolResult{}, err
		}
		evidenceItems := buildAIEventEvidenceItems(events)
		return AIToolResult{
			Summary: fmt.Sprintf("Collected %d node events for %s", len(evidenceItems), resName),
			Evidence: model.JSONMap{
				"kind":   resKind,
				"name":   resName,
				"events": evidenceItems,
			},
			RawRef: model.JSONMap{
				"cluster_id": clusterID,
				"name":       resName,
				"kind":       resKind,
				"source":     "resource.events",
			},
		}, nil
	}

	gvr, namespaced, ok := aiSupportedResourceGVR(resKind)
	if !ok {
		return AIToolResult{}, ErrWithMessage(ErrInvalidParams, "unsupported resource kind for AI events")
	}
	if namespaced && ns == "" {
		return AIToolResult{}, ErrWithMessage(ErrInvalidParams, "namespace is required for the selected resource kind")
	}
	if !namespaced {
		ns = ""
	}
	obj, err := s.k8sSvc.GetObject(ctx, clusterID, gvr, ns, resName)
	if err != nil {
		return AIToolResult{}, err
	}
	fieldSelector := buildAIResourceEventFieldSelector(resKind, ns, resName, obj)
	items, err := s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}, ns, "", "", map[string]string{
		"field_selector": fieldSelector,
	})
	if err != nil {
		return AIToolResult{}, err
	}
	evidenceItems := buildAIEventEvidenceItems(items)
	return AIToolResult{
		Summary: fmt.Sprintf("Collected %d events for %s %s/%s", len(evidenceItems), resKind, ns, resName),
		Evidence: model.JSONMap{
			"kind":   resKind,
			"name":   resName,
			"events": evidenceItems,
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"name":       resName,
			"kind":       resKind,
			"source":     "resource.events",
		},
	}, nil
}

func (s *ResourceQueryService) GetResourceLogs(
	ctx context.Context,
	clusterID uint64,
	kind,
	namespace,
	name,
	container string,
	tailLines int64,
	previous bool,
) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	resKind := strings.TrimSpace(kind)
	ns := strings.TrimSpace(namespace)
	resName := strings.TrimSpace(name)
	if resKind == "" || resName == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	if tailLines <= 0 {
		tailLines = 80
	}

	podName := resName
	relatedPods := make([]model.JSONMap, 0, 4)
	if !strings.EqualFold(resKind, "Pod") {
		pods, err := s.listRelatedPodsForResource(ctx, clusterID, resKind, ns, resName)
		if err != nil {
			return AIToolResult{}, err
		}
		if len(pods) == 0 {
			return AIToolResult{}, ErrWithMessage(ErrNotFound, "no related pods found for the selected resource")
		}
		podName = strings.TrimSpace(fmt.Sprint(pods[0]["name"]))
		relatedPods = pods
	}

	logText, err := s.k8sSvc.PodLogs(ctx, clusterID, ns, podName, strings.TrimSpace(container), tailLines, previous)
	if err != nil {
		return AIToolResult{}, err
	}
	evidence := model.JSONMap{
		"kind":       resKind,
		"namespace":  ns,
		"name":       resName,
		"pod_name":   podName,
		"container":  strings.TrimSpace(container),
		"tail_lines": tailLines,
		"previous":   previous,
		"logs":       truncateForModel(logText, 3000),
	}
	if len(relatedPods) > 0 {
		evidence["related_pods"] = relatedPods
	}
	return AIToolResult{
		Summary: fmt.Sprintf("Collected logs for %s via pod %s/%s", resKind, ns, podName),
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"name":       resName,
			"pod_name":   podName,
			"kind":       resKind,
			"source":     "resource.logs",
		},
	}, nil
}

func (s *ResourceQueryService) GetRelatedResources(ctx context.Context, clusterID uint64, kind, namespace, name string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	resKind := strings.TrimSpace(kind)
	ns := strings.TrimSpace(namespace)
	resName := strings.TrimSpace(name)
	if resKind == "" || resName == "" {
		return AIToolResult{}, ErrInvalidParams
	}

	switch strings.ToLower(resKind) {
	case "configmap":
		pods, controllers, err := s.findConfigMapConsumers(ctx, clusterID, ns, resName)
		if err != nil {
			return AIToolResult{}, err
		}
		return buildAIRelatedResourceResult(clusterID, ns, resKind, resName, pods, controllers), nil
	case "secret":
		pods, controllers, err := s.findSecretConsumers(ctx, clusterID, ns, resName)
		if err != nil {
			return AIToolResult{}, err
		}
		return buildAIRelatedResourceResult(clusterID, ns, resKind, resName, pods, controllers), nil
	default:
		pods, err := s.listRelatedPodsForResource(ctx, clusterID, resKind, ns, resName)
		if err != nil {
			return AIToolResult{}, err
		}
		controllers := buildAIControllersFromPods(pods)
		return buildAIRelatedResourceResult(clusterID, ns, resKind, resName, pods, controllers), nil
	}
}

func buildAIListItemSummary(kind string, obj map[string]any) model.JSONMap {
	resKind := strings.TrimSpace(kind)
	switch strings.ToLower(resKind) {
	case "deployment", "statefulset", "daemonset":
		return buildAIWorkloadSummaryFromObject(resKind, obj)
	case "pod":
		return buildAIPodOverview(obj)
	default:
		return buildAIResourceOverview(resKind, aiObjectMetaString(obj, "namespace"), aiObjectMetaString(obj, "name"), obj)
	}
}

func normalizeAIResourceSearchKinds(kinds []string, namespacedOnly bool) []string {
	if len(kinds) == 0 {
		defaultKinds := []string{
			"Pod", "Deployment", "StatefulSet", "DaemonSet", "Service", "Ingress",
			"ConfigMap", "Secret", "PersistentVolumeClaim", "Job", "CronJob",
		}
		if !namespacedOnly {
			defaultKinds = append(defaultKinds, "Namespace", "Node", "PersistentVolume", "StorageClass")
		}
		return defaultKinds
	}

	out := make([]string, 0, len(kinds))
	seen := make(map[string]struct{}, len(kinds))
	for _, item := range kinds {
		kind := strings.TrimSpace(item)
		if kind == "" {
			continue
		}
		normalized := normalizeAIResourceKind(kind)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		if _, namespaced, ok := aiSupportedResourceGVR(normalized); ok {
			if namespacedOnly && !namespaced {
				continue
			}
			seen[normalized] = struct{}{}
			out = append(out, normalized)
		}
	}
	return out
}

func normalizeAIResourceKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "namespace":
		return "Namespace"
	case "node":
		return "Node"
	case "pod":
		return "Pod"
	case "deployment":
		return "Deployment"
	case "statefulset":
		return "StatefulSet"
	case "daemonset":
		return "DaemonSet"
	case "replicaset":
		return "ReplicaSet"
	case "service":
		return "Service"
	case "ingress":
		return "Ingress"
	case "networkpolicy":
		return "NetworkPolicy"
	case "configmap":
		return "ConfigMap"
	case "secret":
		return "Secret"
	case "serviceaccount":
		return "ServiceAccount"
	case "persistentvolumeclaim", "pvc":
		return "PersistentVolumeClaim"
	case "persistentvolume", "pv":
		return "PersistentVolume"
	case "storageclass":
		return "StorageClass"
	case "job":
		return "Job"
	case "cronjob":
		return "CronJob"
	default:
		return strings.TrimSpace(kind)
	}
}

func clampAIResultLimit(limit, fallback int) int {
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

func buildAIResourceEventFieldSelector(kind, namespace, name string, obj map[string]any) string {
	parts := []string{
		"involvedObject.kind=" + strings.TrimSpace(kind),
		"involvedObject.name=" + strings.TrimSpace(name),
	}
	if strings.TrimSpace(namespace) != "" {
		parts = append(parts, "involvedObject.namespace="+strings.TrimSpace(namespace))
	}
	metadata, _ := obj["metadata"].(map[string]any)
	if metadata != nil {
		if uid := strings.TrimSpace(fmt.Sprint(metadata["uid"])); uid != "" {
			parts = append(parts, "involvedObject.uid="+uid)
		}
	}
	return strings.Join(parts, ",")
}

func buildAIEventEvidenceItems(items []any) []model.JSONMap {
	evidence := make([]model.JSONMap, 0, minInt(len(items), 20))
	for _, item := range items {
		obj, _ := item.(map[string]any)
		if obj == nil {
			continue
		}
		involved, _ := obj["involvedObject"].(map[string]any)
		evidence = append(evidence, model.JSONMap{
			"type":           strings.TrimSpace(fmt.Sprint(obj["type"])),
			"reason":         strings.TrimSpace(fmt.Sprint(obj["reason"])),
			"message":        strings.TrimSpace(fmt.Sprint(obj["message"])),
			"namespace":      aiObjectMetaString(obj, "namespace"),
			"name":           aiObjectMetaString(obj, "name"),
			"event_time":     firstNonEmpty(strings.TrimSpace(fmt.Sprint(obj["eventTime"])), strings.TrimSpace(fmt.Sprint(obj["lastTimestamp"])), strings.TrimSpace(fmt.Sprint(aiMapValue(obj, "metadata")["creationTimestamp"]))),
			"involved_kind":  strings.TrimSpace(fmt.Sprint(involved["kind"])),
			"involved_name":  strings.TrimSpace(fmt.Sprint(involved["name"])),
			"involved_uid":   strings.TrimSpace(fmt.Sprint(involved["uid"])),
			"count":          aiIntValue(obj["count"]),
		})
		if len(evidence) >= 20 {
			break
		}
	}
	return evidence
}

func (s *ResourceQueryService) listRelatedPodsForResource(ctx context.Context, clusterID uint64, kind, namespace, name string) ([]model.JSONMap, error) {
	resKind := strings.TrimSpace(kind)
	ns := strings.TrimSpace(namespace)
	switch strings.ToLower(resKind) {
	case "pod":
		obj, err := s.k8sSvc.GetObject(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, ns, name)
		if err != nil {
			return nil, err
		}
		return []model.JSONMap{buildAIPodReference(obj)}, nil
	case "deployment", "statefulset", "daemonset", "replicaset", "job":
		gvr, _, ok := aiSupportedResourceGVR(resKind)
		if !ok {
			return nil, ErrInvalidParams
		}
		obj, err := s.k8sSvc.GetObject(ctx, clusterID, gvr, ns, name)
		if err != nil {
			return nil, err
		}
		selector := aiLabelSelectorFromObject(obj)
		if selector == "" {
			return nil, nil
		}
		return s.listPodsByLabelSelector(ctx, clusterID, ns, selector)
	case "cronjob":
		obj, err := s.k8sSvc.GetObject(ctx, clusterID, schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}, ns, name)
		if err != nil {
			return nil, err
		}
		selector := aiJobTemplateSelector(obj)
		if selector == "" {
			return nil, nil
		}
		return s.listPodsByLabelSelector(ctx, clusterID, ns, selector)
	case "service":
		obj, err := s.k8sSvc.GetObject(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}, ns, name)
		if err != nil {
			return nil, err
		}
		selector := aiMapToLabelSelector(aiMapValue(aiMapValue(obj, "spec"), "selector"))
		if selector == "" {
			return nil, nil
		}
		return s.listPodsByLabelSelector(ctx, clusterID, ns, selector)
	case "persistentvolumeclaim":
		pods, err := s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, ns, "metadata.name", "asc", nil)
		if err != nil {
			return nil, err
		}
		related := make([]model.JSONMap, 0, 16)
		for _, item := range pods {
			obj, _ := item.(map[string]any)
			if obj == nil || !aiPodUsesPVC(obj, name) {
				continue
			}
			related = append(related, buildAIPodReference(obj))
		}
		return related, nil
	default:
		return nil, nil
	}
}

func (s *ResourceQueryService) listPodsByLabelSelector(ctx context.Context, clusterID uint64, namespace, labelSelector string) ([]model.JSONMap, error) {
	items, err := s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, namespace, "metadata.name", "asc", map[string]string{
		"label_selector": strings.TrimSpace(labelSelector),
	})
	if err != nil {
		return nil, err
	}
	pods := make([]model.JSONMap, 0, minInt(len(items), 20))
	for _, item := range items {
		obj, _ := item.(map[string]any)
		if obj == nil {
			continue
		}
		pods = append(pods, buildAIPodReference(obj))
		if len(pods) >= 20 {
			break
		}
	}
	return pods, nil
}

func buildAIPodReference(obj map[string]any) model.JSONMap {
	spec, _ := obj["spec"].(map[string]any)
	status, _ := obj["status"].(map[string]any)
	ready, restarts := aiPodReadyAndRestarts(obj)
	return model.JSONMap{
		"kind":       "Pod",
		"namespace":  aiObjectMetaString(obj, "namespace"),
		"name":       aiObjectMetaString(obj, "name"),
		"phase":      strings.TrimSpace(fmt.Sprint(status["phase"])),
		"node":       strings.TrimSpace(fmt.Sprint(spec["nodeName"])),
		"ready":      ready,
		"restarts":   restarts,
		"owners":     extractAIOwnerReferences(obj),
	}
}

func aiPodReadyAndRestarts(obj map[string]any) (string, int) {
	status, _ := obj["status"].(map[string]any)
	containerStatuses, _ := status["containerStatuses"].([]any)
	if len(containerStatuses) == 0 {
		return "-", 0
	}
	ready := 0
	restarts := 0
	for _, item := range containerStatuses {
		container, _ := item.(map[string]any)
		if container == nil {
			continue
		}
		if aiBoolValue(container["ready"]) {
			ready++
		}
		restarts += aiIntValue(container["restartCount"])
	}
	return fmt.Sprintf("%d/%d", ready, len(containerStatuses)), restarts
}

func aiLabelSelectorFromObject(obj map[string]any) string {
	spec, _ := obj["spec"].(map[string]any)
	if spec == nil {
		return ""
	}
	if selector, ok := spec["selector"].(map[string]any); ok && selector != nil {
		if matchLabels, ok := selector["matchLabels"].(map[string]any); ok && len(matchLabels) > 0 {
			return aiMapToLabelSelector(matchLabels)
		}
	}
	return ""
}

func aiJobTemplateSelector(obj map[string]any) string {
	spec, _ := obj["spec"].(map[string]any)
	if spec == nil {
		return ""
	}
	jobTemplate, _ := spec["jobTemplate"].(map[string]any)
	if jobTemplate == nil {
		return ""
	}
	jobSpec, _ := jobTemplate["spec"].(map[string]any)
	if jobSpec == nil {
		return ""
	}
	template, _ := jobSpec["template"].(map[string]any)
	if template == nil {
		return ""
	}
	metadata, _ := template["metadata"].(map[string]any)
	if metadata == nil {
		return ""
	}
	labels, _ := metadata["labels"].(map[string]any)
	return aiMapToLabelSelector(labels)
}

func aiMapToLabelSelector(labels map[string]any) string {
	if len(labels) == 0 {
		return ""
	}
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

func aiPodUsesPVC(pod map[string]any, pvcName string) bool {
	spec, _ := pod["spec"].(map[string]any)
	if spec == nil {
		return false
	}
	volumes, _ := spec["volumes"].([]any)
	for _, item := range volumes {
		volume, _ := item.(map[string]any)
		if volume == nil {
			continue
		}
		pvc, _ := volume["persistentVolumeClaim"].(map[string]any)
		if pvc == nil {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(pvc["claimName"])) == pvcName {
			return true
		}
	}
	return false
}

func (s *ResourceQueryService) findConfigMapConsumers(ctx context.Context, clusterID uint64, namespace, name string) ([]model.JSONMap, []model.JSONMap, error) {
	return s.findNamedResourceConsumers(ctx, clusterID, namespace, name, aiPodUsesConfigMap)
}

func (s *ResourceQueryService) findSecretConsumers(ctx context.Context, clusterID uint64, namespace, name string) ([]model.JSONMap, []model.JSONMap, error) {
	return s.findNamedResourceConsumers(ctx, clusterID, namespace, name, aiPodUsesSecret)
}

func (s *ResourceQueryService) findNamedResourceConsumers(
	ctx context.Context,
	clusterID uint64,
	namespace,
	name string,
	matcher func(map[string]any, string) bool,
) ([]model.JSONMap, []model.JSONMap, error) {
	items, err := s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, namespace, "metadata.name", "asc", nil)
	if err != nil {
		return nil, nil, err
	}
	pods := make([]model.JSONMap, 0, 16)
	controllerMap := make(map[string]model.JSONMap)
	for _, item := range items {
		obj, _ := item.(map[string]any)
		if obj == nil || !matcher(obj, name) {
			continue
		}
		podRef := buildAIPodReference(obj)
		pods = append(pods, podRef)
		for _, owner := range extractAIOwnerReferences(obj) {
			key := strings.TrimSpace(fmt.Sprint(owner["kind"])) + "/" + strings.TrimSpace(fmt.Sprint(owner["name"]))
			if key == "/" {
				continue
			}
			controllerMap[key] = model.JSONMap{
				"kind": strings.TrimSpace(fmt.Sprint(owner["kind"])),
				"name": strings.TrimSpace(fmt.Sprint(owner["name"])),
			}
		}
	}
	controllers := make([]model.JSONMap, 0, len(controllerMap))
	for _, controller := range controllerMap {
		controllers = append(controllers, controller)
	}
	sort.SliceStable(controllers, func(i, j int) bool {
		left := strings.TrimSpace(fmt.Sprint(controllers[i]["kind"])) + "/" + strings.TrimSpace(fmt.Sprint(controllers[i]["name"]))
		right := strings.TrimSpace(fmt.Sprint(controllers[j]["kind"])) + "/" + strings.TrimSpace(fmt.Sprint(controllers[j]["name"]))
		return left < right
	})
	return pods, controllers, nil
}

func aiPodUsesConfigMap(pod map[string]any, configMapName string) bool {
	spec, _ := pod["spec"].(map[string]any)
	if spec == nil {
		return false
	}
	volumes, _ := spec["volumes"].([]any)
	for _, item := range volumes {
		volume, _ := item.(map[string]any)
		if volume == nil {
			continue
		}
		configMap, _ := volume["configMap"].(map[string]any)
		if strings.TrimSpace(fmt.Sprint(configMap["name"])) == configMapName {
			return true
		}
		projected, _ := volume["projected"].(map[string]any)
		sources, _ := projected["sources"].([]any)
		for _, source := range sources {
			sourceMap, _ := source.(map[string]any)
			if sourceMap == nil {
				continue
			}
			configMap, _ := sourceMap["configMap"].(map[string]any)
			if strings.TrimSpace(fmt.Sprint(configMap["name"])) == configMapName {
				return true
			}
		}
	}
	return aiContainersUseNamedRef(spec, configMapName, "configMapRef", "configMapKeyRef")
}

func aiPodUsesSecret(pod map[string]any, secretName string) bool {
	spec, _ := pod["spec"].(map[string]any)
	if spec == nil {
		return false
	}
	volumes, _ := spec["volumes"].([]any)
	for _, item := range volumes {
		volume, _ := item.(map[string]any)
		if volume == nil {
			continue
		}
		secret, _ := volume["secret"].(map[string]any)
		if strings.TrimSpace(fmt.Sprint(secret["secretName"])) == secretName {
			return true
		}
		projected, _ := volume["projected"].(map[string]any)
		sources, _ := projected["sources"].([]any)
		for _, source := range sources {
			sourceMap, _ := source.(map[string]any)
			if sourceMap == nil {
				continue
			}
			secret, _ := sourceMap["secret"].(map[string]any)
			if strings.TrimSpace(fmt.Sprint(secret["name"])) == secretName {
				return true
			}
		}
		csi, _ := volume["csi"].(map[string]any)
		nodePublish, _ := csi["nodePublishSecretRef"].(map[string]any)
		if strings.TrimSpace(fmt.Sprint(nodePublish["name"])) == secretName {
			return true
		}
	}
	imagePullSecrets, _ := spec["imagePullSecrets"].([]any)
	for _, item := range imagePullSecrets {
		secret, _ := item.(map[string]any)
		if secret == nil {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(secret["name"])) == secretName {
			return true
		}
	}
	return aiContainersUseNamedRef(spec, secretName, "secretRef", "secretKeyRef")
}

func aiContainersUseNamedRef(spec map[string]any, targetName, envFromRefKey, envValueRefKey string) bool {
	for _, key := range []string{"containers", "initContainers"} {
		containers, _ := spec[key].([]any)
		for _, item := range containers {
			container, _ := item.(map[string]any)
			if container == nil {
				continue
			}
			envFrom, _ := container["envFrom"].([]any)
			for _, envFromItem := range envFrom {
				envFromMap, _ := envFromItem.(map[string]any)
				if envFromMap == nil {
					continue
				}
				ref, _ := envFromMap[envFromRefKey].(map[string]any)
				if strings.TrimSpace(fmt.Sprint(ref["name"])) == targetName {
					return true
				}
			}
			env, _ := container["env"].([]any)
			for _, envItem := range env {
				envMap, _ := envItem.(map[string]any)
				if envMap == nil {
					continue
				}
				valueFrom, _ := envMap["valueFrom"].(map[string]any)
				if valueFrom == nil {
					continue
				}
				ref, _ := valueFrom[envValueRefKey].(map[string]any)
				if strings.TrimSpace(fmt.Sprint(ref["name"])) == targetName {
					return true
				}
			}
		}
	}
	return false
}

func buildAIControllersFromPods(pods []model.JSONMap) []model.JSONMap {
	controllerMap := make(map[string]model.JSONMap)
	for _, pod := range pods {
		owners, _ := pod["owners"].([]model.JSONMap)
		for _, owner := range owners {
			kind := strings.TrimSpace(fmt.Sprint(owner["kind"]))
			name := strings.TrimSpace(fmt.Sprint(owner["name"]))
			if kind == "" || name == "" {
				continue
			}
			controllerMap[kind+"/"+name] = model.JSONMap{
				"kind": kind,
				"name": name,
			}
		}
	}
	controllers := make([]model.JSONMap, 0, len(controllerMap))
	for _, item := range controllerMap {
		controllers = append(controllers, item)
	}
	sort.SliceStable(controllers, func(i, j int) bool {
		left := strings.TrimSpace(fmt.Sprint(controllers[i]["kind"])) + "/" + strings.TrimSpace(fmt.Sprint(controllers[i]["name"]))
		right := strings.TrimSpace(fmt.Sprint(controllers[j]["kind"])) + "/" + strings.TrimSpace(fmt.Sprint(controllers[j]["name"]))
		return left < right
	})
	return controllers
}

func buildAIRelatedResourceResult(
	clusterID uint64,
	namespace,
	kind,
	name string,
	pods,
	controllers []model.JSONMap,
) AIToolResult {
	return AIToolResult{
		Summary: fmt.Sprintf("Found %d related pods and %d related controllers for %s %s/%s", len(pods), len(controllers), kind, namespace, name),
		Evidence: model.JSONMap{
			"kind":        kind,
			"namespace":   namespace,
			"name":        name,
			"pod_count":   len(pods),
			"pods":        pods,
			"controllers": controllers,
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  namespace,
			"name":       name,
			"kind":       kind,
			"source":     "resource.related",
		},
	}
}
