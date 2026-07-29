package application

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

// InspectionResourceReference identifies a Kubernetes object without coupling
// the application layer to a client-go resource type.
type InspectionResourceReference struct {
	ClusterID uint64
	Kind      string
	Namespace string
	Name      string
}

// InspectionResourceDescriptor is the canonical mapping from a supported
// inspection kind to the Kubernetes API resource used by runtime adapters.
type InspectionResourceDescriptor struct {
	Group      string
	Version    string
	Resource   string
	Namespaced bool
}

// InspectionResult preserves the existing inspection payload shape for both
// the Kops HTTP routes and AI tools, while keeping it transport-neutral.
type InspectionResult struct {
	Summary  string         `json:"summary"`
	Evidence map[string]any `json:"evidence"`
	RawRef   map[string]any `json:"raw_ref"`
}

// InspectionResourceRead is intentionally raw. The AI boundary applies its
// own sensitive-data export policy before exposing its Object or YAML fields.
type InspectionResourceRead struct {
	Reference InspectionResourceReference
	Object    map[string]any
	YAML      string
	YAMLError string
}

// ResourceInspectionRuntime supplies Kubernetes read operations used by the
// inspection application service. Implementations belong in adapters.
type ResourceInspectionRuntime interface {
	Object(context.Context, InspectionResourceReference) (map[string]any, error)
	YAML(context.Context, InspectionResourceReference) (string, error)
	NodeEvents(context.Context, uint64, string) ([]any, error)
	RolloutHistory(context.Context, InspectionResourceReference) ([]any, error)
	Supports(context.Context, uint64, string) (bool, error)
	List(context.Context, uint64, string, string) ([]any, error)
	PodLogs(context.Context, uint64, string, string, int64) (string, error)
}

func (s *InspectionService) Pod(ctx context.Context, clusterID uint64, namespace, name string) (*InspectionResult, error) {
	ref, err := normalizeInspectionReference(clusterID, "Pod", namespace, name, "inspection")
	if err != nil {
		return nil, err
	}
	runtime, err := s.resourceRuntime()
	if err != nil {
		return nil, err
	}
	obj, err := runtime.Object(ctx, ref)
	if err != nil {
		return nil, err
	}

	pod := BuildInspectionPodOverview(obj)
	relationships := inspectPodRelationships(ctx, runtime, ref, obj)
	containers, resources := buildInspectionPodContainerResources(obj)
	conditions := InspectionResourceConditions(obj)
	metrics, metricsSummary := inspectSinglePodMetrics(ctx, runtime, ref)
	logs, logsSummary := inspectPodLogs(ctx, runtime, ref)

	parts := []string{fmt.Sprintf("Pod %s/%s", ref.Namespace, ref.Name)}
	if phase := strings.TrimSpace(fmt.Sprint(pod["phase"])); phase != "" {
		parts = append(parts, "phase "+phase)
	}
	if controller := inspectionRelationshipControllerSummary(relationships); controller != "" {
		parts = append(parts, "owner "+controller)
	}
	if ready := strings.TrimSpace(fmt.Sprint(pod["ready"])); ready != "" {
		parts = append(parts, "ready "+ready)
	}
	if strings.TrimSpace(metricsSummary) != "" {
		parts = append(parts, metricsSummary)
	}
	if strings.TrimSpace(logsSummary) != "" {
		parts = append(parts, logsSummary)
	}

	evidence := map[string]any{
		"pod":           pod,
		"conditions":    conditions,
		"containers":    containers,
		"resources":     resources,
		"relationships": relationships,
		"metrics":       metrics,
	}
	for key, value := range logs {
		evidence[key] = value
	}
	return &InspectionResult{
		Summary:  strings.Join(parts, ", "),
		Evidence: evidence,
		RawRef: map[string]any{
			"cluster_id": ref.ClusterID,
			"namespace":  ref.Namespace,
			"name":       ref.Name,
			"source":     "pod.inspect",
		},
	}, nil
}

func (s *InspectionService) Node(ctx context.Context, clusterID uint64, name string) (*InspectionResult, error) {
	ref, err := normalizeInspectionReference(clusterID, "Node", "", name, "inspection")
	if err != nil {
		return nil, err
	}
	runtime, err := s.resourceRuntime()
	if err != nil {
		return nil, err
	}
	obj, err := runtime.Object(ctx, ref)
	if err != nil {
		return nil, err
	}
	evidence := map[string]any{"object": obj}
	summary := fmt.Sprintf("Read node %s object information", ref.Name)
	events, eventsErr := runtime.NodeEvents(ctx, ref.ClusterID, ref.Name)
	if eventsErr == nil {
		evidence["events"] = events
		summary = fmt.Sprintf("Read node %s object information and related events", ref.Name)
	} else {
		evidence["events_error"] = inspectionErrorMessage(eventsErr)
	}
	return &InspectionResult{
		Summary:  summary,
		Evidence: evidence,
		RawRef:   map[string]any{"cluster_id": ref.ClusterID, "name": ref.Name, "source": "node.inspect"},
	}, nil
}

func (s *InspectionService) Deployment(ctx context.Context, clusterID uint64, namespace, name string) (*InspectionResult, error) {
	ref, err := normalizeInspectionReference(clusterID, "Deployment", namespace, name, "inspection")
	if err != nil {
		return nil, err
	}
	runtime, err := s.resourceRuntime()
	if err != nil {
		return nil, err
	}
	obj, err := runtime.Object(ctx, ref)
	if err != nil {
		return nil, err
	}
	workload := BuildInspectionWorkloadSummary(ref.Kind, obj)
	conditions := InspectionResourceConditions(obj)
	evidence := map[string]any{"deployment": workload, "conditions": conditions, "object": obj}
	summary := BuildInspectionDeploymentSummary(ref.Namespace, ref.Name, workload, conditions)
	if history, historyErr := runtime.RolloutHistory(ctx, ref); historyErr == nil {
		evidence["rollout_history"] = history
		if len(history) > 0 {
			evidence["rollout_revision_count"] = len(history)
			summary += fmt.Sprintf(", rollout revisions %d", len(history))
		} else {
			summary += ", rollout history included"
		}
	} else {
		evidence["rollout_history_error"] = inspectionErrorMessage(historyErr)
	}
	return &InspectionResult{
		Summary:  summary,
		Evidence: evidence,
		RawRef:   map[string]any{"cluster_id": ref.ClusterID, "namespace": ref.Namespace, "name": ref.Name, "source": "deployment.inspect"},
	}, nil
}

// Resource reads a supported object and opportunistically retrieves its YAML.
// A YAML retrieval failure is evidence, not a failure of an otherwise valid
// object inspection, matching the historical AI tool behavior.
func (s *InspectionService) Resource(ctx context.Context, clusterID uint64, kind, namespace, name string) (*InspectionResourceRead, error) {
	ref, err := normalizeInspectionReference(clusterID, kind, namespace, name, "inspection")
	if err != nil {
		return nil, err
	}
	runtime, err := s.resourceRuntime()
	if err != nil {
		return nil, err
	}
	obj, err := runtime.Object(ctx, ref)
	if err != nil {
		return nil, err
	}
	value := &InspectionResourceRead{Reference: ref, Object: obj}
	if yamlText, yamlErr := runtime.YAML(ctx, ref); yamlErr == nil {
		value.YAML = yamlText
	} else {
		value.YAMLError = inspectionErrorMessage(yamlErr)
	}
	return value, nil
}

func (s *InspectionService) ResourceYAML(ctx context.Context, clusterID uint64, kind, namespace, name string) (*InspectionResourceRead, error) {
	ref, err := normalizeInspectionReference(clusterID, kind, namespace, name, "export")
	if err != nil {
		return nil, err
	}
	runtime, err := s.resourceRuntime()
	if err != nil {
		return nil, err
	}
	yamlText, err := runtime.YAML(ctx, ref)
	if err != nil {
		return nil, err
	}
	return &InspectionResourceRead{Reference: ref, YAML: yamlText}, nil
}

func (s *InspectionService) resourceRuntime() (ResourceInspectionRuntime, error) {
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	runtime, ok := s.runtime.(ResourceInspectionRuntime)
	if !ok || runtime == nil {
		return nil, ErrConflict
	}
	return runtime, nil
}

func normalizeInspectionReference(clusterID uint64, kind, namespace, name, purpose string) (InspectionResourceReference, error) {
	ref := InspectionResourceReference{ClusterID: clusterID, Kind: strings.TrimSpace(kind), Namespace: strings.TrimSpace(namespace), Name: strings.TrimSpace(name)}
	if ref.ClusterID == 0 || ref.Kind == "" || ref.Name == "" {
		return InspectionResourceReference{}, ErrInvalidParams
	}
	descriptor, ok := InspectionResourceDescriptorFor(ref.Kind)
	if !ok {
		return InspectionResourceReference{}, ErrWithMessage(ErrInvalidParams, "unsupported resource kind for AI "+strings.TrimSpace(purpose))
	}
	if descriptor.Namespaced && ref.Namespace == "" {
		return InspectionResourceReference{}, ErrWithMessage(ErrInvalidParams, "namespace is required for the selected resource kind")
	}
	if !descriptor.Namespaced {
		ref.Namespace = ""
	}
	return ref, nil
}

// InspectionResourceDescriptorFor returns the canonical supported resource
// table. Keeping this table in Kops prevents API adapters and callers from
// drifting in their kind-to-resource mapping.
func InspectionResourceDescriptorFor(kind string) (InspectionResourceDescriptor, bool) {
	refs := map[string]InspectionResourceDescriptor{
		"namespace": {Version: "v1", Resource: "namespaces"}, "node": {Version: "v1", Resource: "nodes"},
		"pod": {Version: "v1", Resource: "pods", Namespaced: true}, "deployment": {Group: "apps", Version: "v1", Resource: "deployments", Namespaced: true},
		"statefulset": {Group: "apps", Version: "v1", Resource: "statefulsets", Namespaced: true}, "daemonset": {Group: "apps", Version: "v1", Resource: "daemonsets", Namespaced: true},
		"replicaset": {Group: "apps", Version: "v1", Resource: "replicasets", Namespaced: true}, "service": {Version: "v1", Resource: "services", Namespaced: true},
		"ingress": {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses", Namespaced: true}, "ingressclass": {Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"},
		"networkpolicy": {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies", Namespaced: true}, "configmap": {Version: "v1", Resource: "configmaps", Namespaced: true},
		"secret": {Version: "v1", Resource: "secrets", Namespaced: true}, "serviceaccount": {Version: "v1", Resource: "serviceaccounts", Namespaced: true},
		"endpoint": {Version: "v1", Resource: "endpoints", Namespaced: true}, "endpointslice": {Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices", Namespaced: true},
		"lease": {Group: "coordination.k8s.io", Version: "v1", Resource: "leases", Namespaced: true}, "pdb": {Group: "policy", Version: "v1", Resource: "poddisruptionbudgets", Namespaced: true},
		"poddisruptionbudget": {Group: "policy", Version: "v1", Resource: "poddisruptionbudgets", Namespaced: true}, "role": {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles", Namespaced: true},
		"clusterrole": {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}, "rolebinding": {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings", Namespaced: true},
		"clusterrolebinding": {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}, "hpa": {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers", Namespaced: true},
		"horizontalpodautoscaler": {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers", Namespaced: true}, "event": {Version: "v1", Resource: "events", Namespaced: true},
		"pvc": {Version: "v1", Resource: "persistentvolumeclaims", Namespaced: true}, "persistentvolumeclaim": {Version: "v1", Resource: "persistentvolumeclaims", Namespaced: true},
		"pv": {Version: "v1", Resource: "persistentvolumes"}, "persistentvolume": {Version: "v1", Resource: "persistentvolumes"},
		"storageclass": {Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}, "csidriver": {Group: "storage.k8s.io", Version: "v1", Resource: "csidrivers"},
		"csinode": {Group: "storage.k8s.io", Version: "v1", Resource: "csinodes"}, "csistoragecapacity": {Group: "storage.k8s.io", Version: "v1", Resource: "csistoragecapacities", Namespaced: true},
		"volumeattachment": {Group: "storage.k8s.io", Version: "v1", Resource: "volumeattachments"}, "volumesnapshot": {Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshots", Namespaced: true},
		"volumesnapshotclass": {Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotclasses"}, "volumesnapshotcontent": {Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotcontents"},
		"resourcequota": {Version: "v1", Resource: "resourcequotas", Namespaced: true}, "limitrange": {Version: "v1", Resource: "limitranges", Namespaced: true},
		"customresourcedefinition": {Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}, "crd": {Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"},
		"apiservice": {Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}, "priorityclass": {Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"},
		"runtimeclass": {Group: "node.k8s.io", Version: "v1", Resource: "runtimeclasses"}, "validatingwebhookconfiguration": {Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"},
		"mutatingwebhookconfiguration": {Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}, "validatingadmissionpolicy": {Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingadmissionpolicies"},
		"validatingadmissionpolicybinding": {Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingadmissionpolicybindings"}, "job": {Group: "batch", Version: "v1", Resource: "jobs", Namespaced: true}, "cronjob": {Group: "batch", Version: "v1", Resource: "cronjobs", Namespaced: true},
	}
	ref, ok := refs[strings.ToLower(strings.TrimSpace(kind))]
	return ref, ok
}

func BuildInspectionResourceOverview(kind, namespace, name string, obj map[string]any) map[string]any {
	overview := map[string]any{"kind": strings.TrimSpace(kind), "namespace": strings.TrimSpace(namespace), "name": strings.TrimSpace(name)}
	if obj == nil {
		return overview
	}
	overview["api_version"] = strings.TrimSpace(fmt.Sprint(obj["apiVersion"]))
	if metadata := inspectionMap(obj["metadata"]); metadata != nil {
		overview["generation"] = metadata["generation"]
		overview["creation_timestamp"] = metadata["creationTimestamp"]
		if labels := inspectionMap(metadata["labels"]); labels != nil {
			overview["label_count"] = len(labels)
		}
		if annotations := inspectionMap(metadata["annotations"]); annotations != nil {
			overview["annotation_count"] = len(annotations)
		}
	}
	if status := inspectionMap(obj["status"]); status != nil {
		if phase := strings.TrimSpace(fmt.Sprint(status["phase"])); phase != "" {
			overview["phase"] = phase
		}
		if replicas := strings.TrimSpace(fmt.Sprint(status["replicas"])); replicas != "" && replicas != "<nil>" {
			overview["replicas"] = replicas
		}
		if ready := strings.TrimSpace(fmt.Sprint(status["readyReplicas"])); ready != "" && ready != "<nil>" {
			overview["ready_replicas"] = ready
		}
	}
	return overview
}

func InspectionResourceConditions(obj map[string]any) []map[string]any {
	if status := inspectionMap(obj["status"]); status != nil {
		raw, _ := status["conditions"].([]any)
		conditions := make([]map[string]any, 0, minInspectionInt(len(raw), 10))
		for _, value := range raw {
			condition := inspectionMap(value)
			if condition == nil {
				continue
			}
			conditions = append(conditions, map[string]any{
				"type": strings.TrimSpace(fmt.Sprint(condition["type"])), "status": strings.TrimSpace(fmt.Sprint(condition["status"])),
				"reason": strings.TrimSpace(fmt.Sprint(condition["reason"])), "message": strings.TrimSpace(fmt.Sprint(condition["message"])),
				"last_transition_at": strings.TrimSpace(fmt.Sprint(condition["lastTransitionTime"])),
			})
			if len(conditions) >= 10 {
				break
			}
		}
		return conditions
	}
	return nil
}

func BuildInspectionResourceSummary(kind, namespace, name string, obj map[string]any, masked bool) string {
	scope := strings.TrimSpace(name)
	if ns := strings.TrimSpace(namespace); ns != "" {
		scope = ns + "/" + scope
	}
	parts := []string{fmt.Sprintf("Read %s %s", strings.TrimSpace(kind), scope)}
	status := inspectionMap(obj["status"])
	if phase := strings.TrimSpace(fmt.Sprint(status["phase"])); phase != "" {
		parts = append(parts, "phase "+phase)
	}
	if ready := strings.TrimSpace(fmt.Sprint(status["readyReplicas"])); ready != "" && ready != "<nil>" {
		replicas := strings.TrimSpace(fmt.Sprint(status["replicas"]))
		if replicas == "" || replicas == "<nil>" {
			parts = append(parts, "ready replicas "+ready)
		} else {
			parts = append(parts, fmt.Sprintf("ready replicas %s/%s", ready, replicas))
		}
	}
	if conditions := InspectionResourceConditions(obj); len(conditions) > 0 {
		first := conditions[0]
		if kind := strings.TrimSpace(fmt.Sprint(first["type"])); kind != "" {
			parts = append(parts, fmt.Sprintf("condition %s=%s", kind, strings.TrimSpace(fmt.Sprint(first["status"]))))
		}
	}
	if masked {
		parts = append(parts, "sensitive fields masked")
	}
	return strings.Join(parts, ", ")
}

// BuildInspectionPodOverview creates the transport-neutral pod read model used
// by Kops resource inspection and AI list projections.
func BuildInspectionPodOverview(obj map[string]any) map[string]any {
	overview := map[string]any{"name": inspectionObjectMetaString(obj, "name"), "namespace": inspectionObjectMetaString(obj, "namespace")}
	if obj == nil {
		return overview
	}
	spec, status := inspectionMap(obj["spec"]), inspectionMap(obj["status"])
	overview["phase"] = strings.TrimSpace(fmt.Sprint(status["phase"]))
	overview["pod_ip"] = strings.TrimSpace(fmt.Sprint(status["podIP"]))
	overview["host_ip"] = strings.TrimSpace(fmt.Sprint(status["hostIP"]))
	overview["node_name"] = strings.TrimSpace(fmt.Sprint(spec["nodeName"]))
	overview["service_account"] = strings.TrimSpace(fmt.Sprint(spec["serviceAccountName"]))
	overview["qos_class"] = strings.TrimSpace(fmt.Sprint(status["qosClass"]))
	overview["creation_timestamp"] = strings.TrimSpace(fmt.Sprint(inspectionMap(obj["metadata"])["creationTimestamp"]))
	overview["start_time"] = strings.TrimSpace(fmt.Sprint(status["startTime"]))
	statuses, _ := status["containerStatuses"].([]any)
	ready, restarts := 0, 0
	containerStates := make([]map[string]any, 0, len(statuses))
	for _, value := range statuses {
		container := inspectionMap(value)
		if container == nil {
			continue
		}
		if inspectionBool(container["ready"]) {
			ready++
		}
		restarts += inspectionInt(container["restartCount"])
		containerStates = append(containerStates, map[string]any{"name": strings.TrimSpace(fmt.Sprint(container["name"])), "ready": inspectionBool(container["ready"]), "restart_count": inspectionInt(container["restartCount"]), "image": strings.TrimSpace(fmt.Sprint(container["image"])), "image_id": strings.TrimSpace(fmt.Sprint(container["imageID"]))})
	}
	overview["ready"] = fmt.Sprintf("%d/%d", ready, maxInspectionInt(len(statuses), len(inspectionContainerNames(spec))))
	overview["restart_count"] = restarts
	if len(containerStates) > 0 {
		overview["container_statuses"] = containerStates
	}
	return overview
}

func buildInspectionPodContainerResources(obj map[string]any) ([]map[string]any, map[string]any) {
	spec := inspectionMap(obj["spec"])
	if spec == nil {
		return nil, nil
	}
	containers := make([]map[string]any, 0, 8)
	var requestCPU, limitCPU, requestMemory, limitMemory, requestEphemeral, limitEphemeral int64
	appendContainers := func(values []any, kind string) {
		for _, value := range values {
			container := inspectionMap(value)
			if container == nil {
				continue
			}
			requests := inspectionResourceValueMap(inspectionMap(inspectionMap(container["resources"])["requests"]))
			limits := inspectionResourceValueMap(inspectionMap(inspectionMap(container["resources"])["limits"]))
			requestCPU += inspectionQuantity(requests["cpu"], true)
			limitCPU += inspectionQuantity(limits["cpu"], true)
			requestMemory += inspectionQuantity(requests["memory"], false)
			limitMemory += inspectionQuantity(limits["memory"], false)
			requestEphemeral += inspectionGenericQuantity(requests["ephemeral-storage"])
			limitEphemeral += inspectionGenericQuantity(limits["ephemeral-storage"])
			containers = append(containers, map[string]any{"name": strings.TrimSpace(fmt.Sprint(container["name"])), "type": kind, "image": strings.TrimSpace(fmt.Sprint(container["image"])), "requests": requests, "limits": limits, "cpu_request": strings.TrimSpace(fmt.Sprint(requests["cpu"])), "cpu_limit": strings.TrimSpace(fmt.Sprint(limits["cpu"])), "memory_request": strings.TrimSpace(fmt.Sprint(requests["memory"])), "memory_limit": strings.TrimSpace(fmt.Sprint(limits["memory"])), "ephemeral_request": strings.TrimSpace(fmt.Sprint(requests["ephemeral-storage"])), "ephemeral_limit": strings.TrimSpace(fmt.Sprint(limits["ephemeral-storage"]))})
		}
	}
	if values, _ := spec["containers"].([]any); len(values) > 0 {
		appendContainers(values, "container")
	}
	if values, _ := spec["initContainers"].([]any); len(values) > 0 {
		appendContainers(values, "initContainer")
	}
	return containers, map[string]any{
		"requests": map[string]any{"cpu_millicores": requestCPU, "cpu": formatInspectionMillicores(requestCPU), "memory_bytes": requestMemory, "memory": formatInspectionMemory(requestMemory), "ephemeral_storage_bytes": requestEphemeral, "ephemeral_storage": formatInspectionMemory(requestEphemeral)},
		"limits":   map[string]any{"cpu_millicores": limitCPU, "cpu": formatInspectionMillicores(limitCPU), "memory_bytes": limitMemory, "memory": formatInspectionMemory(limitMemory), "ephemeral_storage_bytes": limitEphemeral, "ephemeral_storage": formatInspectionMemory(limitEphemeral)},
	}
}

func inspectPodRelationships(ctx context.Context, runtime ResourceInspectionRuntime, ref InspectionResourceReference, pod map[string]any) map[string]any {
	owners := inspectionOwnerReferences(pod)
	result := map[string]any{"owner_references": owners}
	current := inspectionControllerOwner(owners)
	if len(current) == 0 {
		result["message"] = "pod has no controller owner reference"
		return result
	}
	chain, errorsList := make([]map[string]any, 0, 4), make([]string, 0, 2)
	visited := map[string]struct{}{}
	for depth := 0; depth < 4 && len(current) > 0; depth++ {
		kind, name := strings.TrimSpace(fmt.Sprint(current["kind"])), strings.TrimSpace(fmt.Sprint(current["name"]))
		if kind == "" || name == "" {
			break
		}
		cacheKey := kind + ":" + name
		if _, seen := visited[cacheKey]; seen {
			break
		}
		visited[cacheKey] = struct{}{}
		entry := cloneInspectionMap(current)
		descriptor, ok := InspectionResourceDescriptorFor(kind)
		if !ok {
			chain = append(chain, entry)
			break
		}
		namespace := ref.Namespace
		if !descriptor.Namespaced {
			namespace = ""
		}
		obj, err := runtime.Object(ctx, InspectionResourceReference{ClusterID: ref.ClusterID, Kind: kind, Namespace: namespace, Name: name})
		if err != nil {
			message := inspectionErrorMessage(err)
			entry["resolution_error"] = message
			chain = append(chain, entry)
			errorsList = append(errorsList, fmt.Sprintf("%s/%s: %s", kind, name, message))
			break
		}
		summary := BuildInspectionWorkloadSummary(kind, obj)
		summary["owner_references"] = inspectionOwnerReferences(obj)
		entry["summary"] = summary
		chain = append(chain, entry)
		next := inspectionControllerOwner(inspectionOwnerReferences(obj))
		if len(next) == 0 {
			break
		}
		current = next
	}
	result["controller_chain"] = chain
	result["immediate_owner"] = chain[0]
	result["top_workload"] = chain[len(chain)-1]
	if len(errorsList) > 0 {
		result["resolution_errors"] = errorsList
	}
	return result
}

func inspectSinglePodMetrics(ctx context.Context, runtime ResourceInspectionRuntime, ref InspectionResourceReference) (map[string]any, string) {
	supported, err := runtime.Supports(ctx, ref.ClusterID, "pod_metrics")
	if err != nil {
		return map[string]any{"supported": false, "error": inspectionErrorMessage(err)}, "metrics unavailable"
	}
	if !supported {
		return map[string]any{"supported": false, "message": "metrics API is not available in the current cluster"}, "metrics unavailable"
	}
	items, err := runtime.List(ctx, ref.ClusterID, "pod_metrics", ref.Namespace)
	if err != nil {
		return map[string]any{"supported": true, "available": false, "error": inspectionErrorMessage(err)}, "metrics query failed"
	}
	var target map[string]any
	for _, item := range items {
		obj := inspectionMap(item)
		if obj != nil && inspectionObjectMetaString(obj, "name") == ref.Name {
			target = obj
			break
		}
	}
	if target == nil {
		return map[string]any{"supported": true, "available": false, "message": "no PodMetrics sample found for the pod"}, "metrics missing"
	}
	cpu, memory := inspectionMetricTotals(target)
	containers := make([]map[string]any, 0, 4)
	values, _ := target["containers"].([]any)
	for _, value := range values {
		container := inspectionMap(value)
		if container == nil {
			continue
		}
		usage := inspectionMap(container["usage"])
		containers = append(containers, map[string]any{"name": strings.TrimSpace(fmt.Sprint(container["name"])), "cpu": strings.TrimSpace(fmt.Sprint(usage["cpu"])), "memory": strings.TrimSpace(fmt.Sprint(usage["memory"])), "cpu_millicores": inspectionQuantity(usage["cpu"], true), "memory_bytes": inspectionQuantity(usage["memory"], false)})
	}
	return map[string]any{"supported": true, "available": true, "timestamp": strings.TrimSpace(fmt.Sprint(target["timestamp"])), "window": strings.TrimSpace(fmt.Sprint(target["window"])), "cpu": formatInspectionMillicores(cpu), "cpu_millicores": cpu, "memory": formatInspectionMemory(memory), "memory_bytes": memory, "containers": containers}, fmt.Sprintf("CPU %s, memory %s", formatInspectionMillicores(cpu), formatInspectionMemory(memory))
}

func inspectPodLogs(ctx context.Context, runtime ResourceInspectionRuntime, ref InspectionResourceReference) (map[string]any, string) {
	logs, err := runtime.PodLogs(ctx, ref.ClusterID, ref.Namespace, ref.Name, 80)
	if err != nil {
		return map[string]any{"logs_error": inspectionErrorMessage(err)}, "logs unavailable"
	}
	return map[string]any{"logs_excerpt": truncateInspectionText(logs, 1500)}, "recent logs included"
}

// BuildInspectionWorkloadSummary creates a common workload projection for
// Deployment, StatefulSet and DaemonSet resources.
func BuildInspectionWorkloadSummary(kind string, obj map[string]any) map[string]any {
	summary := map[string]any{"kind": strings.TrimSpace(kind), "name": inspectionObjectMetaString(obj, "name"), "namespace": inspectionObjectMetaString(obj, "namespace")}
	if obj == nil {
		return summary
	}
	status, spec := inspectionMap(obj["status"]), inspectionMap(obj["spec"])
	summary["desired_replicas"] = inspectionWorkloadDesired(kind, spec, status)
	summary["ready_replicas"] = inspectionWorkloadReady(kind, status)
	if value := inspectionInt(status["availableReplicas"]); value > 0 || strings.EqualFold(kind, "Deployment") {
		summary["available_replicas"] = value
	}
	if value := inspectionInt(status["updatedReplicas"]); value > 0 || strings.EqualFold(kind, "Deployment") || strings.EqualFold(kind, "StatefulSet") {
		summary["updated_replicas"] = value
	}
	if value := inspectionInt(status["currentReplicas"]); value > 0 || strings.EqualFold(kind, "StatefulSet") {
		summary["current_replicas"] = value
	}
	if value := inspectionInt(status["currentNumberScheduled"]); value > 0 || strings.EqualFold(kind, "DaemonSet") {
		summary["current_replicas"] = value
	}
	if value := inspectionInt(status["numberAvailable"]); value > 0 || strings.EqualFold(kind, "DaemonSet") {
		summary["available_replicas"] = value
	}
	if value := inspectionInt(status["updatedNumberScheduled"]); value > 0 || strings.EqualFold(kind, "DaemonSet") {
		summary["updated_replicas"] = value
	}
	if selector := inspectionMap(spec["selector"]); selector != nil {
		if labels := inspectionMap(selector["matchLabels"]); len(labels) > 0 {
			summary["selector"] = labels
		}
	}
	if containers := inspectionTemplateContainers(obj); len(containers) > 0 {
		summary["containers"] = containers
	}
	return summary
}

// BuildInspectionDeploymentSummary renders the concise deployment inspection
// summary while keeping presentation rules in the Kops application layer.
func BuildInspectionDeploymentSummary(namespace, name string, workload map[string]any, conditions []map[string]any) string {
	parts := []string{fmt.Sprintf("Deployment %s/%s", strings.TrimSpace(namespace), strings.TrimSpace(name)), fmt.Sprintf("replicas ready %d/%d", inspectionInt(workload["ready_replicas"]), inspectionInt(workload["desired_replicas"])), fmt.Sprintf("available %d", inspectionInt(workload["available_replicas"])), fmt.Sprintf("updated %d", inspectionInt(workload["updated_replicas"]))}
	if containers, _ := workload["containers"].([]map[string]any); len(containers) > 0 {
		images := make([]string, 0, len(containers))
		for _, container := range containers {
			name, image := strings.TrimSpace(fmt.Sprint(container["name"])), strings.TrimSpace(fmt.Sprint(container["image"]))
			if name == "" && image == "" {
				continue
			}
			if name == "" {
				images = append(images, image)
			} else {
				images = append(images, name+"="+image)
			}
		}
		if len(images) > 0 {
			parts = append(parts, "images "+strings.Join(images, ", "))
		}
	}
	if len(conditions) > 0 {
		if kind := strings.TrimSpace(fmt.Sprint(conditions[0]["type"])); kind != "" {
			parts = append(parts, fmt.Sprintf("condition %s=%s", kind, strings.TrimSpace(fmt.Sprint(conditions[0]["status"]))))
		}
	}
	return strings.Join(parts, ", ")
}

func inspectionOwnerReferences(obj map[string]any) []map[string]any {
	metadata := inspectionMap(obj["metadata"])
	if metadata == nil {
		return nil
	}
	raw, _ := metadata["ownerReferences"].([]any)
	if len(raw) == 0 {
		return nil
	}
	owners := make([]map[string]any, 0, len(raw))
	for _, value := range raw {
		if owner := inspectionMap(value); owner != nil {
			owners = append(owners, map[string]any{"api_version": strings.TrimSpace(fmt.Sprint(owner["apiVersion"])), "kind": strings.TrimSpace(fmt.Sprint(owner["kind"])), "name": strings.TrimSpace(fmt.Sprint(owner["name"])), "uid": strings.TrimSpace(fmt.Sprint(owner["uid"])), "controller": inspectionBool(owner["controller"])})
		}
	}
	return owners
}

func inspectionControllerOwner(owners []map[string]any) map[string]any {
	for _, owner := range owners {
		if inspectionBool(owner["controller"]) {
			return cloneInspectionMap(owner)
		}
	}
	if len(owners) == 0 {
		return nil
	}
	return cloneInspectionMap(owners[0])
}

func inspectionRelationshipControllerSummary(relationships map[string]any) string {
	top := inspectionMap(relationships["top_workload"])
	if top == nil {
		return ""
	}
	if summary := inspectionMap(top["summary"]); summary != nil {
		kind, name := strings.TrimSpace(fmt.Sprint(summary["kind"])), strings.TrimSpace(fmt.Sprint(summary["name"]))
		if kind != "" && name != "" {
			return kind + "/" + name
		}
	}
	kind, name := strings.TrimSpace(fmt.Sprint(top["kind"])), strings.TrimSpace(fmt.Sprint(top["name"]))
	if kind != "" && name != "" {
		return kind + "/" + name
	}
	return ""
}

func inspectionTemplateContainers(obj map[string]any) []map[string]any {
	spec := inspectionMap(obj["spec"])
	if spec == nil {
		return nil
	}
	if template := inspectionMap(spec["template"]); template != nil {
		if templateSpec := inspectionMap(template["spec"]); templateSpec != nil {
			return inspectionContainerNames(templateSpec)
		}
	}
	return inspectionContainerNames(spec)
}

func inspectionContainerNames(spec map[string]any) []map[string]any {
	if spec == nil {
		return nil
	}
	containers := make([]map[string]any, 0, 8)
	appendList := func(values []any, kind string) {
		for _, value := range values {
			if container := inspectionMap(value); container != nil {
				containers = append(containers, map[string]any{"name": strings.TrimSpace(fmt.Sprint(container["name"])), "image": strings.TrimSpace(fmt.Sprint(container["image"])), "type": kind})
			}
		}
	}
	if values, _ := spec["containers"].([]any); len(values) > 0 {
		appendList(values, "container")
	}
	if values, _ := spec["initContainers"].([]any); len(values) > 0 {
		appendList(values, "initContainer")
	}
	return containers
}

func inspectionWorkloadDesired(kind string, spec, status map[string]any) int {
	if strings.EqualFold(strings.TrimSpace(kind), "daemonset") {
		if status != nil {
			if value := inspectionInt(status["desiredNumberScheduled"]); value > 0 {
				return value
			}
			return inspectionInt(status["currentNumberScheduled"])
		}
		return 0
	}
	if spec != nil {
		if value := inspectionInt(spec["replicas"]); value > 0 {
			return value
		}
	}
	if status != nil {
		return inspectionInt(status["replicas"])
	}
	return 0
}

func inspectionWorkloadReady(kind string, status map[string]any) int {
	if status == nil {
		return 0
	}
	if strings.EqualFold(strings.TrimSpace(kind), "daemonset") {
		return inspectionInt(status["numberReady"])
	}
	return inspectionInt(status["readyReplicas"])
}

func inspectionResourceValueMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	for key, value := range values {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" && text != "<nil>" {
			out[key] = text
		}
	}
	return out
}
func inspectionMetricTotals(obj map[string]any) (int64, int64) {
	values, _ := obj["containers"].([]any)
	var cpu, memory int64
	for _, value := range values {
		container := inspectionMap(value)
		usage := inspectionMap(container["usage"])
		if usage != nil {
			cpu += inspectionQuantity(usage["cpu"], true)
			memory += inspectionQuantity(usage["memory"], false)
		}
	}
	return cpu, memory
}
func inspectionQuantity(raw any, cpu bool) int64 {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" {
		return 0
	}
	quantity, err := resource.ParseQuantity(text)
	if err != nil {
		return 0
	}
	if cpu {
		return quantity.MilliValue()
	}
	return quantity.Value()
}
func inspectionGenericQuantity(raw any) int64 {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" {
		return 0
	}
	quantity, err := resource.ParseQuantity(text)
	if err != nil {
		return 0
	}
	return quantity.Value()
}
func formatInspectionMillicores(value int64) string {
	if value >= 1000 || value <= -1000 {
		return fmt.Sprintf("%.2f cores", float64(value)/1000)
	}
	return fmt.Sprintf("%dm", value)
}
func formatInspectionMemory(value int64) string {
	const (
		ki int64 = 1024
		mi       = 1024 * ki
		gi       = 1024 * mi
		ti       = 1024 * gi
	)
	switch {
	case value >= ti || value <= -ti:
		return fmt.Sprintf("%.2f Ti", float64(value)/float64(ti))
	case value >= gi || value <= -gi:
		return fmt.Sprintf("%.2f Gi", float64(value)/float64(gi))
	case value >= mi || value <= -mi:
		return fmt.Sprintf("%.1f Mi", float64(value)/float64(mi))
	case value >= ki || value <= -ki:
		return fmt.Sprintf("%.1f Ki", float64(value)/float64(ki))
	default:
		return fmt.Sprintf("%d B", value)
	}
}
func truncateInspectionText(input string, limit int) string {
	raw := strings.TrimSpace(input)
	if limit <= 0 || len([]rune(raw)) <= limit {
		return raw
	}
	return string([]rune(raw)[:limit]) + "..."
}
func inspectionMap(value any) map[string]any { result, _ := value.(map[string]any); return result }
func inspectionObjectMetaString(obj map[string]any, key string) string {
	return strings.TrimSpace(fmt.Sprint(inspectionMap(obj["metadata"])[key]))
}
func inspectionBool(value any) bool {
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
func inspectionInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return 0
}
func inspectionErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var carrier interface{ UserMessage() string }
	if as, ok := err.(interface{ UserMessage() string }); ok {
		carrier = as
	}
	if carrier != nil {
		if message := strings.TrimSpace(carrier.UserMessage()); message != "" {
			return message
		}
	}
	return strings.TrimSpace(err.Error())
}
func cloneInspectionMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			output[key] = cloneInspectionMap(typed)
		case []any:
			items := make([]any, len(typed))
			copy(items, typed)
			output[key] = items
		default:
			output[key] = value
		}
	}
	return output
}
func minInspectionInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
func maxInspectionInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

// Keep sort imported as a compile-time guard for deterministic collection
// presentation helpers added alongside this service.
var _ = sort.SliceStable
