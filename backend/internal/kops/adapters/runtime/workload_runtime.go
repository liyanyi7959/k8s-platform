package runtime

import (
	"context"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

type WorkloadRuntime struct {
	service    *service.K8sService
	operations *WorkloadOperations
}

func NewWorkloadRuntime(service *service.K8sService, operations *WorkloadOperations) *WorkloadRuntime {
	return &WorkloadRuntime{service: service, operations: operations}
}

func (r *WorkloadRuntime) List(ctx context.Context, query kopsapp.WorkloadQuery) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	list := func(kind kopsapp.WorkloadKind) ([]any, error) {
		gvr, err := workloadResourceGVR(kind)
		if err != nil {
			return nil, err
		}
		value, err := r.service.List(ctx, query.ClusterID, gvr, query.Namespace, query.SortBy, query.Order, map[string]string{"label_selector": query.LabelSelector})
		return value, translateKopsRuntimeError(err)
	}
	if query.Kind != "" {
		value, err := list(query.Kind)
		if err != nil {
			return nil, err
		}
		return map[string]any{"list": value}, nil
	}
	merged := make([]any, 0, 128)
	for _, kind := range []kopsapp.WorkloadKind{kopsapp.WorkloadDeployment, kopsapp.WorkloadStatefulSet, kopsapp.WorkloadDaemonSet} {
		value, err := list(kind)
		if err != nil {
			return nil, err
		}
		merged = append(merged, value...)
	}
	return map[string]any{"list": merged}, nil
}

func (r *WorkloadRuntime) History(ctx context.Context, ref kopsapp.WorkloadRef) (any, error) {
	if r == nil || r.operations == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.operations.RolloutHistory(ctx, ref.ClusterID, ref.Namespace, ref.Name, string(ref.Kind))
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"history": value}, nil
}

func (r *WorkloadRuntime) Undo(ctx context.Context, ref kopsapp.WorkloadRef, revision int) error {
	if r == nil || r.operations == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.operations.RolloutUndo(ctx, ref.ClusterID, ref.Namespace, ref.Name, string(ref.Kind), revision))
}

func (r *WorkloadRuntime) Patch(ctx context.Context, ref kopsapp.WorkloadRef, patch map[string]any) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, err := workloadResourceGVR(ref.Kind)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name, patch))
}

func (r *WorkloadRuntime) Image(ctx context.Context, input kopsapp.WorkloadImage) error {
	if r == nil || r.operations == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.operations.UpdateWorkloadImage(ctx, input.ClusterID, input.Namespace, input.Name, string(input.Kind), input.Container, input.Image))
}

func (r *WorkloadRuntime) Pause(ctx context.Context, ref kopsapp.WorkloadRef, paused bool) error {
	if r == nil || r.operations == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.operations.UpdateWorkloadPaused(ctx, ref.ClusterID, ref.Namespace, ref.Name, string(ref.Kind), paused))
}

func (r *WorkloadRuntime) Object(ctx context.Context, ref kopsapp.WorkloadRef) (map[string]any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	gvr, err := workloadResourceGVR(ref.Kind)
	if err != nil {
		return nil, err
	}
	value, err := r.service.GetObject(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	return value, translateKopsRuntimeError(err)
}

func (r *WorkloadRuntime) ApplyYAML(ctx context.Context, input kopsapp.WorkloadYAMLEdit) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, err := workloadResourceGVR(input.Kind)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.service.ApplyYAML(ctx, input.ClusterID, gvr, input.Namespace, input.YAML))
}

func (r *WorkloadRuntime) Delete(ctx context.Context, ref kopsapp.WorkloadRef) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, err := workloadResourceGVR(ref.Kind)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name))
}

func (r *WorkloadRuntime) YAML(ctx context.Context, ref kopsapp.WorkloadRef) (string, error) {
	if r == nil || r.service == nil {
		return "", kopsapp.ErrConflict
	}
	gvr, err := workloadResourceGVR(ref.Kind)
	if err != nil {
		return "", err
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	return value, translateKopsRuntimeError(err)
}

func workloadResourceGVR(kind kopsapp.WorkloadKind) (schema.GroupVersionResource, error) {
	switch kind {
	case kopsapp.WorkloadDeployment:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil
	case kopsapp.WorkloadStatefulSet:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, nil
	case kopsapp.WorkloadDaemonSet:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, nil
	default:
		return schema.GroupVersionResource{}, kopsapp.ErrInvalidParams
	}
}

// ActionProposalRuntime is the retained Kubernetes/manifest adapter for the
// Kops action-proposal application service. It intentionally contains only
// API translation; risk policy and operation orchestration live in Kops.
type ActionProposalRuntime struct {
	k8s       *service.K8sService
	nodes     *NodeOperations
	pods      *PodOperations
	workloads *WorkloadOperations
	batch     *BatchRuntime
	manifests kopsapp.ManifestRuntime
}

func NewActionProposalRuntime(k8s *service.K8sService, nodes *NodeOperations, pods *PodOperations, workloads *WorkloadOperations, manifests kopsapp.ManifestRuntime) *ActionProposalRuntime {
	return &ActionProposalRuntime{k8s: k8s, nodes: nodes, pods: pods, workloads: workloads, batch: NewBatchRuntime(k8s), manifests: manifests}
}

func (r *ActionProposalRuntime) Inspect(ctx context.Context, clusterID uint64, actionType string, target kopsapp.ActionProposalTarget) (map[string]any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	gvr, namespace, err := actionProposalGVR(actionType, target)
	if err != nil {
		return nil, err
	}
	value, err := r.k8s.GetObject(ctx, clusterID, gvr, namespace, target.Name)
	return value, translateKopsRuntimeError(err)
}

func (r *ActionProposalRuntime) Restart(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget) error {
	if r == nil || r.k8s == nil {
		return kopsapp.ErrConflict
	}
	gvr, err := actionProposalWorkloadGVR(target.Kind)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.k8s.PatchJSON(ctx, clusterID, gvr, target.Namespace, target.Name, map[string]any{
		"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]any{
			"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339),
		}}}},
	}))
}

func (r *ActionProposalRuntime) Scale(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, replicas int) error {
	if r == nil || r.k8s == nil {
		return kopsapp.ErrConflict
	}
	gvr, err := actionProposalWorkloadGVR(target.Kind)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.k8s.PatchJSON(ctx, clusterID, gvr, target.Namespace, target.Name, map[string]any{"spec": map[string]any{"replicas": replicas}}))
}

func (r *ActionProposalRuntime) UpdateImage(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, container, image string) error {
	if r == nil || r.workloads == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.workloads.UpdateWorkloadImage(ctx, clusterID, target.Namespace, target.Name, target.Kind, container, image))
}

func (r *ActionProposalRuntime) Pause(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, paused bool) error {
	if r == nil || r.workloads == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.workloads.UpdateWorkloadPaused(ctx, clusterID, target.Namespace, target.Name, target.Kind, paused))
}

func (r *ActionProposalRuntime) Undo(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, revision int) error {
	if r == nil || r.workloads == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.workloads.RolloutUndo(ctx, clusterID, target.Namespace, target.Name, target.Kind, revision))
}

func (r *ActionProposalRuntime) DeleteWorkload(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget) error {
	if r == nil || r.k8s == nil {
		return kopsapp.ErrConflict
	}
	gvr, err := actionProposalWorkloadGVR(target.Kind)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.k8s.Delete(ctx, clusterID, gvr, target.Namespace, target.Name))
}

func (r *ActionProposalRuntime) DeleteResource(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget) error {
	if r == nil || r.k8s == nil {
		return kopsapp.ErrConflict
	}
	gvr, namespaced, err := actionProposalResourceGVR(target.Kind)
	if err != nil {
		return err
	}
	namespace := target.Namespace
	if !namespaced {
		namespace = ""
	}
	return translateKopsRuntimeError(r.k8s.Delete(ctx, clusterID, gvr, namespace, target.Name))
}

func (r *ActionProposalRuntime) DeletePod(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, force bool) error {
	if r == nil || r.pods == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.pods.Delete(ctx, clusterID, target.Namespace, target.Name, force))
}

func (r *ActionProposalRuntime) SetNodeSchedulable(ctx context.Context, clusterID uint64, name string, unschedulable bool) error {
	if r == nil || r.nodes == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.nodes.SetSchedulable(ctx, clusterID, name, unschedulable))
}

func (r *ActionProposalRuntime) DrainNode(ctx context.Context, clusterID uint64, name string, options kopsapp.ActionProposalDrainOptions) error {
	if r == nil || r.nodes == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.nodes.Drain(ctx, clusterID, name, NodeDrainOptions{TimeoutSeconds: options.TimeoutSeconds, Force: options.Force, IgnoreDaemonSets: options.IgnoreDaemonSets}))
}

func (r *ActionProposalRuntime) TriggerCronJob(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget) (string, error) {
	if r == nil || r.batch == nil {
		return "", kopsapp.ErrConflict
	}
	result, err := r.batch.TriggerCronJob(ctx, kopsapp.BatchReference{ClusterID: clusterID, Namespace: target.Namespace, Name: target.Name})
	if err != nil {
		return "", err
	}
	return result.JobName, nil
}

func (r *ActionProposalRuntime) SuspendCronJob(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, suspend bool) error {
	if r == nil || r.batch == nil {
		return kopsapp.ErrConflict
	}
	return r.batch.SuspendCronJob(ctx, kopsapp.BatchReference{ClusterID: clusterID, Namespace: target.Namespace, Name: target.Name}, suspend)
}

func (r *ActionProposalRuntime) DeleteCompletedJobs(ctx context.Context, clusterID uint64, namespace string, olderThanHours int) (int, error) {
	if r == nil || r.batch == nil {
		return 0, kopsapp.ErrConflict
	}
	return r.batch.DeleteCompletedJobs(ctx, clusterID, namespace, olderThanHours)
}

func (r *ActionProposalRuntime) ApplyManifest(ctx context.Context, input kopsapp.ActionProposalManifestRequest) (kopsapp.ActionProposalManifestResult, error) {
	if r == nil || r.manifests == nil {
		return kopsapp.ActionProposalManifestResult{}, kopsapp.ErrConflict
	}
	result, err := r.manifests.Execute(ctx, kopsapp.ManifestApplyInput{
		ClusterID: input.ClusterID, YAML: input.YAML, DefaultNamespace: input.DefaultNamespace, DryRun: false,
		SourceLabel: input.SourceLabel, SourceResource: input.SourceResource, CreatedBy: input.CreatedBy, CreatedByName: input.CreatedByName,
	})
	if err != nil {
		return kopsapp.ActionProposalManifestResult{}, err
	}
	return kopsapp.ActionProposalManifestResult{RecordID: result.RecordID, Status: result.Status, Summary: result.Summary}, nil
}

func actionProposalGVR(actionType string, target kopsapp.ActionProposalTarget) (schema.GroupVersionResource, string, error) {
	switch strings.TrimSpace(actionType) {
	case "restart_workload", "scale_workload", "update_workload_image", "pause_workload_rollout", "rollout_undo", "delete_workload":
		gvr, err := actionProposalWorkloadGVR(target.Kind)
		return gvr, target.Namespace, err
	case "delete_resource":
		gvr, namespaced, err := actionProposalResourceGVR(target.Kind)
		if !namespaced {
			return gvr, "", err
		}
		return gvr, target.Namespace, err
	case "delete_pod":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, target.Namespace, nil
	case "cordon_node", "uncordon_node", "drain_node":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, "", nil
	case "trigger_cronjob", "suspend_cronjob":
		return schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}, target.Namespace, nil
	default:
		return schema.GroupVersionResource{}, "", kopsapp.ErrInvalidParams
	}
}

func actionProposalWorkloadGVR(kind string) (schema.GroupVersionResource, error) {
	switch strings.TrimSpace(kind) {
	case "Deployment":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil
	case "StatefulSet":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, nil
	case "DaemonSet":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, nil
	default:
		return schema.GroupVersionResource{}, kopsapp.ErrInvalidParams
	}
}

func actionProposalResourceGVR(kind string) (schema.GroupVersionResource, bool, error) {
	type resourceRef struct {
		group, version, resource string
		namespaced               bool
	}
	resources := map[string]resourceRef{
		"Namespace": {"", "v1", "namespaces", false}, "Node": {"", "v1", "nodes", false}, "Pod": {"", "v1", "pods", true},
		"Deployment": {"apps", "v1", "deployments", true}, "StatefulSet": {"apps", "v1", "statefulsets", true}, "DaemonSet": {"apps", "v1", "daemonsets", true}, "ReplicaSet": {"apps", "v1", "replicasets", true},
		"Service": {"", "v1", "services", true}, "Ingress": {"networking.k8s.io", "v1", "ingresses", true}, "IngressClass": {"networking.k8s.io", "v1", "ingressclasses", false}, "NetworkPolicy": {"networking.k8s.io", "v1", "networkpolicies", true},
		"ConfigMap": {"", "v1", "configmaps", true}, "Secret": {"", "v1", "secrets", true}, "ServiceAccount": {"", "v1", "serviceaccounts", true}, "Endpoint": {"", "v1", "endpoints", true}, "EndpointSlice": {"discovery.k8s.io", "v1", "endpointslices", true}, "Lease": {"coordination.k8s.io", "v1", "leases", true},
		"PodDisruptionBudget": {"policy", "v1", "poddisruptionbudgets", true}, "Role": {"rbac.authorization.k8s.io", "v1", "roles", true}, "ClusterRole": {"rbac.authorization.k8s.io", "v1", "clusterroles", false}, "RoleBinding": {"rbac.authorization.k8s.io", "v1", "rolebindings", true}, "ClusterRoleBinding": {"rbac.authorization.k8s.io", "v1", "clusterrolebindings", false},
		"HorizontalPodAutoscaler": {"autoscaling", "v2", "horizontalpodautoscalers", true}, "Job": {"batch", "v1", "jobs", true}, "CronJob": {"batch", "v1", "cronjobs", true}, "PersistentVolumeClaim": {"", "v1", "persistentvolumeclaims", true}, "PersistentVolume": {"", "v1", "persistentvolumes", false}, "StorageClass": {"storage.k8s.io", "v1", "storageclasses", false},
		"CSIDriver": {"storage.k8s.io", "v1", "csidrivers", false}, "CSINode": {"storage.k8s.io", "v1", "csinodes", false}, "CSIStorageCapacity": {"storage.k8s.io", "v1", "csistoragecapacities", true}, "VolumeAttachment": {"storage.k8s.io", "v1", "volumeattachments", false}, "VolumeSnapshot": {"snapshot.storage.k8s.io", "v1", "volumesnapshots", true}, "VolumeSnapshotClass": {"snapshot.storage.k8s.io", "v1", "volumesnapshotclasses", false}, "VolumeSnapshotContent": {"snapshot.storage.k8s.io", "v1", "volumesnapshotcontents", false},
		"ResourceQuota": {"", "v1", "resourcequotas", true}, "LimitRange": {"", "v1", "limitranges", true}, "CustomResourceDefinition": {"apiextensions.k8s.io", "v1", "customresourcedefinitions", false}, "APIService": {"apiregistration.k8s.io", "v1", "apiservices", false}, "PriorityClass": {"scheduling.k8s.io", "v1", "priorityclasses", false}, "RuntimeClass": {"node.k8s.io", "v1", "runtimeclasses", false},
		"ValidatingWebhookConfiguration": {"admissionregistration.k8s.io", "v1", "validatingwebhookconfigurations", false}, "MutatingWebhookConfiguration": {"admissionregistration.k8s.io", "v1", "mutatingwebhookconfigurations", false}, "ValidatingAdmissionPolicy": {"admissionregistration.k8s.io", "v1", "validatingadmissionpolicies", false}, "ValidatingAdmissionPolicyBinding": {"admissionregistration.k8s.io", "v1", "validatingadmissionpolicybindings", false},
	}
	ref, ok := resources[strings.TrimSpace(kind)]
	if !ok {
		return schema.GroupVersionResource{}, false, kopsapp.ErrInvalidParams
	}
	return schema.GroupVersionResource{Group: ref.group, Version: ref.version, Resource: ref.resource}, ref.namespaced, nil
}

var _ kopsapp.ActionProposalRuntime = (*ActionProposalRuntime)(nil)
