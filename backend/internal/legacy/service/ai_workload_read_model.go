package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"

	model "k8s-platform-backend/internal/ai/domain"
)

func (s *NamespaceDiagnosisService) GetNamespaceWorkloadInventory(ctx context.Context, clusterID uint64, namespace string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	evidence, summary, err := s.namespaceWorkloadInventoryEvidence(ctx, clusterID, ns)
	if err != nil {
		return AIToolResult{}, err
	}
	return AIToolResult{
		Summary:  summary,
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"source":     "namespace.workloads",
		},
	}, nil
}

func (s *NamespaceDiagnosisService) namespaceWorkloadInventoryEvidence(ctx context.Context, clusterID uint64, namespace string) (model.JSONMap, string, error) {
	workloadSpecs := []struct {
		kind string
		gvr  schema.GroupVersionResource
		key  string
	}{
		{kind: "Deployment", gvr: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, key: "deployments"},
		{kind: "StatefulSet", gvr: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, key: "statefulsets"},
		{kind: "DaemonSet", gvr: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, key: "daemonsets"},
	}

	itemsByKind := model.JSONMap{}
	flattened := make([]model.JSONMap, 0, 32)
	counts := model.JSONMap{}
	totalDesired := 0
	totalReady := 0

	for _, spec := range workloadSpecs {
		list, err := s.k8sSvc.List(ctx, clusterID, spec.gvr, namespace, "metadata.name", "asc", nil)
		if err != nil {
			return nil, "", err
		}

		summaries := make([]model.JSONMap, 0, len(list))
		for _, item := range list {
			obj, _ := item.(map[string]any)
			if obj == nil {
				continue
			}
			summary := buildAIWorkloadSummaryFromObject(spec.kind, obj)
			summaries = append(summaries, summary)
			flattened = append(flattened, summary)
			totalDesired += aiIntValue(summary["desired_replicas"])
			totalReady += aiIntValue(summary["ready_replicas"])
		}

		itemsByKind[spec.key] = summaries
		counts[spec.key] = len(summaries)
	}

	sort.SliceStable(flattened, func(i, j int) bool {
		leftKind := strings.TrimSpace(fmt.Sprint(flattened[i]["kind"]))
		rightKind := strings.TrimSpace(fmt.Sprint(flattened[j]["kind"]))
		if leftKind == rightKind {
			return strings.TrimSpace(fmt.Sprint(flattened[i]["name"])) < strings.TrimSpace(fmt.Sprint(flattened[j]["name"]))
		}
		return leftKind < rightKind
	})

	summary := fmt.Sprintf(
		"Namespace %s workload inventory: %d workloads (%d Deployments, %d StatefulSets, %d DaemonSets), ready replicas %d/%d",
		namespace,
		len(flattened),
		aiIntValue(counts["deployments"]),
		aiIntValue(counts["statefulsets"]),
		aiIntValue(counts["daemonsets"]),
		totalReady,
		totalDesired,
	)

	return model.JSONMap{
		"namespace": namespace,
		"counts":    counts,
		"items":     flattened,
		"by_kind":   itemsByKind,
		"replicas": model.JSONMap{
			"desired": totalDesired,
			"ready":   totalReady,
		},
	}, summary, nil
}

func buildAIWorkloadSummaryFromObject(kind string, obj map[string]any) model.JSONMap {
	summary := model.JSONMap{
		"kind":      strings.TrimSpace(kind),
		"name":      aiObjectMetaString(obj, "name"),
		"namespace": aiObjectMetaString(obj, "namespace"),
	}
	if obj == nil {
		return summary
	}

	status, _ := obj["status"].(map[string]any)
	spec, _ := obj["spec"].(map[string]any)
	summary["desired_replicas"] = aiWorkloadDesiredReplicas(kind, spec, status)
	summary["ready_replicas"] = aiWorkloadReadyReplicas(kind, status)

	if value := aiIntValue(status["availableReplicas"]); value > 0 || strings.EqualFold(kind, "Deployment") {
		summary["available_replicas"] = value
	}
	if value := aiIntValue(status["updatedReplicas"]); value > 0 || strings.EqualFold(kind, "Deployment") || strings.EqualFold(kind, "StatefulSet") {
		summary["updated_replicas"] = value
	}
	if value := aiIntValue(status["currentReplicas"]); value > 0 || strings.EqualFold(kind, "StatefulSet") {
		summary["current_replicas"] = value
	}
	if value := aiIntValue(status["currentNumberScheduled"]); value > 0 || strings.EqualFold(kind, "DaemonSet") {
		summary["current_replicas"] = value
	}
	if value := aiIntValue(status["numberAvailable"]); value > 0 || strings.EqualFold(kind, "DaemonSet") {
		summary["available_replicas"] = value
	}
	if value := aiIntValue(status["updatedNumberScheduled"]); value > 0 || strings.EqualFold(kind, "DaemonSet") {
		summary["updated_replicas"] = value
	}

	if selector, ok := spec["selector"].(map[string]any); ok && selector != nil {
		if matchLabels, ok := selector["matchLabels"].(map[string]any); ok && len(matchLabels) > 0 {
			summary["selector"] = matchLabels
		}
	}

	containers := aiTemplateContainersFromObject(obj)
	if len(containers) > 0 {
		summary["containers"] = containers
	}

	return summary
}

func aiWorkloadDesiredReplicas(kind string, spec, status map[string]any) int {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "daemonset":
		if status != nil {
			if value := aiIntValue(status["desiredNumberScheduled"]); value > 0 {
				return value
			}
			return aiIntValue(status["currentNumberScheduled"])
		}
		return 0
	default:
		if spec != nil {
			if value := aiIntValue(spec["replicas"]); value > 0 {
				return value
			}
		}
		if status != nil {
			return aiIntValue(status["replicas"])
		}
		return 0
	}
}

func aiWorkloadReadyReplicas(kind string, status map[string]any) int {
	if status == nil {
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "daemonset":
		return aiIntValue(status["numberReady"])
	default:
		return aiIntValue(status["readyReplicas"])
	}
}

func aiTemplateContainersFromObject(obj map[string]any) []model.JSONMap {
	spec, _ := obj["spec"].(map[string]any)
	if spec == nil {
		return nil
	}
	if template, ok := spec["template"].(map[string]any); ok && template != nil {
		if templateSpec, ok := template["spec"].(map[string]any); ok && templateSpec != nil {
			return aiContainerNameImageList(templateSpec)
		}
	}
	return aiContainerNameImageList(spec)
}

func aiContainerNameImageList(spec map[string]any) []model.JSONMap {
	if spec == nil {
		return nil
	}
	containers := make([]model.JSONMap, 0, 8)
	appendList := func(items []any, containerType string) {
		for _, item := range items {
			container, _ := item.(map[string]any)
			if container == nil {
				continue
			}
			containers = append(containers, model.JSONMap{
				"name":  strings.TrimSpace(fmt.Sprint(container["name"])),
				"image": strings.TrimSpace(fmt.Sprint(container["image"])),
				"type":  containerType,
			})
		}
	}
	if list, _ := spec["containers"].([]any); len(list) > 0 {
		appendList(list, "container")
	}
	if list, _ := spec["initContainers"].([]any); len(list) > 0 {
		appendList(list, "initContainer")
	}
	return containers
}

func buildAIPodOverview(obj map[string]any) model.JSONMap {
	overview := model.JSONMap{
		"name":      aiObjectMetaString(obj, "name"),
		"namespace": aiObjectMetaString(obj, "namespace"),
	}
	if obj == nil {
		return overview
	}

	spec, _ := obj["spec"].(map[string]any)
	status, _ := obj["status"].(map[string]any)
	overview["phase"] = strings.TrimSpace(fmt.Sprint(status["phase"]))
	overview["pod_ip"] = strings.TrimSpace(fmt.Sprint(status["podIP"]))
	overview["host_ip"] = strings.TrimSpace(fmt.Sprint(status["hostIP"]))
	overview["node_name"] = strings.TrimSpace(fmt.Sprint(spec["nodeName"]))
	overview["service_account"] = strings.TrimSpace(fmt.Sprint(spec["serviceAccountName"]))
	overview["qos_class"] = strings.TrimSpace(fmt.Sprint(status["qosClass"]))
	overview["creation_timestamp"] = strings.TrimSpace(fmt.Sprint(aiMapValue(obj, "metadata")["creationTimestamp"]))
	overview["start_time"] = strings.TrimSpace(fmt.Sprint(status["startTime"]))

	containerStatuses, _ := status["containerStatuses"].([]any)
	totalContainers := len(containerStatuses)
	readyContainers := 0
	restartCount := 0
	containerStates := make([]model.JSONMap, 0, len(containerStatuses))
	for _, item := range containerStatuses {
		container, _ := item.(map[string]any)
		if container == nil {
			continue
		}
		if aiBoolValue(container["ready"]) {
			readyContainers++
		}
		restartCount += aiIntValue(container["restartCount"])
		containerStates = append(containerStates, model.JSONMap{
			"name":          strings.TrimSpace(fmt.Sprint(container["name"])),
			"ready":         aiBoolValue(container["ready"]),
			"restart_count": aiIntValue(container["restartCount"]),
			"image":         strings.TrimSpace(fmt.Sprint(container["image"])),
			"image_id":      strings.TrimSpace(fmt.Sprint(container["imageID"])),
		})
	}

	overview["ready"] = fmt.Sprintf("%d/%d", readyContainers, maxInt(totalContainers, len(aiContainerNameImageList(spec))))
	overview["restart_count"] = restartCount
	if len(containerStates) > 0 {
		overview["container_statuses"] = containerStates
	}
	return overview
}

func buildAIPodContainerResources(obj map[string]any) ([]model.JSONMap, model.JSONMap) {
	spec, _ := obj["spec"].(map[string]any)
	if spec == nil {
		return nil, nil
	}

	containers := make([]model.JSONMap, 0, 8)
	var totalRequestCPU int64
	var totalLimitCPU int64
	var totalRequestMemory int64
	var totalLimitMemory int64
	var totalRequestEphemeral int64
	var totalLimitEphemeral int64

	appendContainers := func(items []any, containerType string) {
		for _, item := range items {
			container, _ := item.(map[string]any)
			if container == nil {
				continue
			}
			requests := aiResourceValueMap(aiMapValue(aiMapValue(container, "resources"), "requests"))
			limits := aiResourceValueMap(aiMapValue(aiMapValue(container, "resources"), "limits"))
			totalRequestCPU += parseAIMetricQuantity(requests["cpu"], true)
			totalLimitCPU += parseAIMetricQuantity(limits["cpu"], true)
			totalRequestMemory += parseAIMetricQuantity(requests["memory"], false)
			totalLimitMemory += parseAIMetricQuantity(limits["memory"], false)
			totalRequestEphemeral += aiParseGenericQuantity(requests["ephemeral-storage"])
			totalLimitEphemeral += aiParseGenericQuantity(limits["ephemeral-storage"])

			containers = append(containers, model.JSONMap{
				"name":              strings.TrimSpace(fmt.Sprint(container["name"])),
				"type":              containerType,
				"image":             strings.TrimSpace(fmt.Sprint(container["image"])),
				"requests":          requests,
				"limits":            limits,
				"cpu_request":       strings.TrimSpace(fmt.Sprint(requests["cpu"])),
				"cpu_limit":         strings.TrimSpace(fmt.Sprint(limits["cpu"])),
				"memory_request":    strings.TrimSpace(fmt.Sprint(requests["memory"])),
				"memory_limit":      strings.TrimSpace(fmt.Sprint(limits["memory"])),
				"ephemeral_request": strings.TrimSpace(fmt.Sprint(requests["ephemeral-storage"])),
				"ephemeral_limit":   strings.TrimSpace(fmt.Sprint(limits["ephemeral-storage"])),
			})
		}
	}

	if items, _ := spec["containers"].([]any); len(items) > 0 {
		appendContainers(items, "container")
	}
	if items, _ := spec["initContainers"].([]any); len(items) > 0 {
		appendContainers(items, "initContainer")
	}

	totals := model.JSONMap{
		"requests": model.JSONMap{
			"cpu_millicores":          totalRequestCPU,
			"cpu":                     formatAIMillicores(totalRequestCPU),
			"memory_bytes":            totalRequestMemory,
			"memory":                  formatAIMemoryBytes(totalRequestMemory),
			"ephemeral_storage_bytes": totalRequestEphemeral,
			"ephemeral_storage":       formatAIMemoryBytes(totalRequestEphemeral),
		},
		"limits": model.JSONMap{
			"cpu_millicores":          totalLimitCPU,
			"cpu":                     formatAIMillicores(totalLimitCPU),
			"memory_bytes":            totalLimitMemory,
			"memory":                  formatAIMemoryBytes(totalLimitMemory),
			"ephemeral_storage_bytes": totalLimitEphemeral,
			"ephemeral_storage":       formatAIMemoryBytes(totalLimitEphemeral),
		},
	}
	return containers, totals
}

func aiResourceValueMap(values map[string]any) model.JSONMap {
	if len(values) == 0 {
		return model.JSONMap{}
	}
	out := model.JSONMap{}
	for key, value := range values {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" || text == "<nil>" {
			continue
		}
		out[key] = text
	}
	return out
}

func aiParseGenericQuantity(raw any) int64 {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" {
		return 0
	}
	q, err := resource.ParseQuantity(text)
	if err != nil {
		return 0
	}
	return q.Value()
}

func (s *ResourceInspectionService) inspectPodRelationships(ctx context.Context, clusterID uint64, namespace string, podObj map[string]any) model.JSONMap {
	owners := extractAIOwnerReferences(podObj)
	result := model.JSONMap{
		"owner_references": owners,
	}
	controller := pickAIControllerOwner(owners)
	if len(controller) == 0 {
		result["message"] = "pod has no controller owner reference"
		return result
	}

	chain := make([]model.JSONMap, 0, 4)
	resolutionErrors := make([]string, 0, 2)
	visited := make(map[string]struct{}, 4)
	current := controller

	for depth := 0; depth < 4 && len(current) > 0; depth++ {
		kind := strings.TrimSpace(fmt.Sprint(current["kind"]))
		name := strings.TrimSpace(fmt.Sprint(current["name"]))
		if kind == "" || name == "" {
			break
		}
		cacheKey := kind + ":" + name
		if _, exists := visited[cacheKey]; exists {
			break
		}
		visited[cacheKey] = struct{}{}

		entry := cloneJSONMap(current)
		gvr, namespaced, ok := aiSupportedResourceGVR(kind)
		if !ok {
			chain = append(chain, entry)
			break
		}

		targetNamespace := namespace
		if !namespaced {
			targetNamespace = ""
		}
		obj, err := s.k8sSvc.GetObject(ctx, clusterID, gvr, targetNamespace, name)
		if err != nil {
			entry["resolution_error"] = firstUserFacingError(err)
			chain = append(chain, entry)
			resolutionErrors = append(resolutionErrors, fmt.Sprintf("%s/%s: %s", kind, name, firstUserFacingError(err)))
			break
		}

		resolvedSummary := buildAIWorkloadSummaryFromObject(kind, obj)
		resolvedSummary["owner_references"] = extractAIOwnerReferences(obj)
		entry["summary"] = resolvedSummary
		chain = append(chain, entry)

		next := pickAIControllerOwner(extractAIOwnerReferences(obj))
		if len(next) == 0 {
			break
		}
		current = next
	}

	result["controller_chain"] = chain
	result["immediate_owner"] = chain[0]
	result["top_workload"] = chain[len(chain)-1]
	if len(resolutionErrors) > 0 {
		result["resolution_errors"] = resolutionErrors
	}
	return result
}

func extractAIOwnerReferences(obj map[string]any) []model.JSONMap {
	metadata, _ := obj["metadata"].(map[string]any)
	if metadata == nil {
		return nil
	}
	rawOwners, _ := metadata["ownerReferences"].([]any)
	if len(rawOwners) == 0 {
		return nil
	}
	owners := make([]model.JSONMap, 0, len(rawOwners))
	for _, item := range rawOwners {
		owner, _ := item.(map[string]any)
		if owner == nil {
			continue
		}
		owners = append(owners, model.JSONMap{
			"api_version": strings.TrimSpace(fmt.Sprint(owner["apiVersion"])),
			"kind":        strings.TrimSpace(fmt.Sprint(owner["kind"])),
			"name":        strings.TrimSpace(fmt.Sprint(owner["name"])),
			"uid":         strings.TrimSpace(fmt.Sprint(owner["uid"])),
			"controller":  aiBoolValue(owner["controller"]),
		})
	}
	return owners
}

func pickAIControllerOwner(owners []model.JSONMap) model.JSONMap {
	for _, owner := range owners {
		if aiBoolValue(owner["controller"]) {
			return cloneJSONMap(owner)
		}
	}
	if len(owners) == 0 {
		return nil
	}
	return cloneJSONMap(owners[0])
}

func aiRelationshipControllerSummary(relationships model.JSONMap) string {
	if relationships == nil {
		return ""
	}
	top, _ := relationships["top_workload"].(model.JSONMap)
	if top == nil {
		if rawTop, ok := relationships["top_workload"].(map[string]any); ok {
			top = model.JSONMap(rawTop)
		}
	}
	if top == nil {
		return ""
	}
	summary, _ := top["summary"].(model.JSONMap)
	if summary == nil {
		if rawSummary, ok := top["summary"].(map[string]any); ok {
			summary = model.JSONMap(rawSummary)
		}
	}
	if summary != nil {
		kind := strings.TrimSpace(fmt.Sprint(summary["kind"]))
		name := strings.TrimSpace(fmt.Sprint(summary["name"]))
		if kind != "" && name != "" {
			return kind + "/" + name
		}
	}
	kind := strings.TrimSpace(fmt.Sprint(top["kind"]))
	name := strings.TrimSpace(fmt.Sprint(top["name"]))
	if kind != "" && name != "" {
		return kind + "/" + name
	}
	return ""
}

func (s *ResourceInspectionService) inspectSinglePodMetrics(ctx context.Context, clusterID uint64, namespace, podName string) (model.JSONMap, string) {
	metricsGVR := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
	supported, err := s.k8sSvc.SupportsCompatibleGVR(ctx, clusterID, metricsGVR)
	if err != nil {
		return model.JSONMap{
			"supported": false,
			"error":     firstUserFacingError(err),
		}, "metrics unavailable"
	}
	if !supported {
		return model.JSONMap{
			"supported": false,
			"message":   "metrics API is not available in the current cluster",
		}, "metrics unavailable"
	}

	items, err := s.k8sSvc.List(ctx, clusterID, metricsGVR, namespace, "metadata.name", "asc", nil)
	if err != nil {
		return model.JSONMap{
			"supported": true,
			"available": false,
			"error":     firstUserFacingError(err),
		}, "metrics query failed"
	}

	var target map[string]any
	for _, item := range items {
		obj, _ := item.(map[string]any)
		if obj == nil {
			continue
		}
		if aiObjectMetaString(obj, "name") == podName {
			target = obj
			break
		}
	}
	if target == nil {
		return model.JSONMap{
			"supported": true,
			"available": false,
			"message":   "no PodMetrics sample found for the pod",
		}, "metrics missing"
	}

	cpuMilli, memoryBytes := aiMetricUsageTotals(target)
	containers := make([]model.JSONMap, 0, 4)
	rawContainers, _ := target["containers"].([]any)
	for _, item := range rawContainers {
		container, _ := item.(map[string]any)
		if container == nil {
			continue
		}
		usage, _ := container["usage"].(map[string]any)
		containers = append(containers, model.JSONMap{
			"name":           strings.TrimSpace(fmt.Sprint(container["name"])),
			"cpu":            strings.TrimSpace(fmt.Sprint(usage["cpu"])),
			"memory":         strings.TrimSpace(fmt.Sprint(usage["memory"])),
			"cpu_millicores": parseAIMetricQuantity(usage["cpu"], true),
			"memory_bytes":   parseAIMetricQuantity(usage["memory"], false),
		})
	}

	return model.JSONMap{
		"supported":      true,
		"available":      true,
		"timestamp":      strings.TrimSpace(fmt.Sprint(target["timestamp"])),
		"window":         strings.TrimSpace(fmt.Sprint(target["window"])),
		"cpu":            formatAIMillicores(cpuMilli),
		"cpu_millicores": cpuMilli,
		"memory":         formatAIMemoryBytes(memoryBytes),
		"memory_bytes":   memoryBytes,
		"containers":     containers,
	}, fmt.Sprintf("CPU %s, memory %s", formatAIMillicores(cpuMilli), formatAIMemoryBytes(memoryBytes))
}

func (s *ResourceInspectionService) inspectPodLogEvidence(ctx context.Context, clusterID uint64, namespace, podName string) (model.JSONMap, string) {
	logText, err := s.k8sSvc.PodLogs(ctx, clusterID, namespace, podName, "", 80, false)
	if err != nil {
		return model.JSONMap{
			"logs_error": firstUserFacingError(err),
		}, "logs unavailable"
	}
	return model.JSONMap{
		"logs_excerpt": truncateForModel(logText, 1500),
	}, "recent logs included"
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
