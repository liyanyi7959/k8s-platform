package application

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	ActionTypeRestartWorkload      = "restart_workload"
	ActionTypeScaleWorkload        = "scale_workload"
	ActionTypeUpdateWorkloadImage  = "update_workload_image"
	ActionTypePauseWorkloadRollout = "pause_workload_rollout"
	ActionTypeRolloutUndo          = "rollout_undo"
	ActionTypeDeleteWorkload       = "delete_workload"
	ActionTypeDeleteResource       = "delete_resource"
	ActionTypeDeletePod            = "delete_pod"
	ActionTypeCordonNode           = "cordon_node"
	ActionTypeUncordonNode         = "uncordon_node"
	ActionTypeDrainNode            = "drain_node"
	ActionTypeTriggerCronJob       = "trigger_cronjob"
	ActionTypeSuspendCronJob       = "suspend_cronjob"
	ActionTypeDeleteCompletedJobs  = "delete_completed_jobs"
	ActionTypeApplyManifest        = "apply_manifest"
)

// NormalizeActionType validates the small, explicit action vocabulary that AI
// is allowed to propose. Execution remains behind the change approval flow.
func NormalizeActionType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ActionTypeRestartWorkload:
		return ActionTypeRestartWorkload
	case ActionTypeScaleWorkload:
		return ActionTypeScaleWorkload
	case ActionTypeUpdateWorkloadImage:
		return ActionTypeUpdateWorkloadImage
	case ActionTypePauseWorkloadRollout:
		return ActionTypePauseWorkloadRollout
	case ActionTypeRolloutUndo:
		return ActionTypeRolloutUndo
	case ActionTypeDeleteWorkload:
		return ActionTypeDeleteWorkload
	case ActionTypeDeleteResource:
		return ActionTypeDeleteResource
	case ActionTypeDeletePod:
		return ActionTypeDeletePod
	case ActionTypeCordonNode:
		return ActionTypeCordonNode
	case ActionTypeUncordonNode:
		return ActionTypeUncordonNode
	case ActionTypeDrainNode:
		return ActionTypeDrainNode
	case ActionTypeTriggerCronJob:
		return ActionTypeTriggerCronJob
	case ActionTypeSuspendCronJob:
		return ActionTypeSuspendCronJob
	case ActionTypeDeleteCompletedJobs:
		return ActionTypeDeleteCompletedJobs
	case ActionTypeApplyManifest:
		return ActionTypeApplyManifest
	default:
		return ""
	}
}

// NormalizeActionTarget gives Kubernetes resource kinds their canonical form
// before a proposal is persisted or passed to a runtime adapter.
func NormalizeActionTarget(target ActionTargetResource) ActionTargetResource {
	kind := strings.TrimSpace(target.Kind)
	switch strings.ToLower(kind) {
	case "namespace":
		kind = "Namespace"
	case "node":
		kind = "Node"
	case "pod":
		kind = "Pod"
	case "deployment":
		kind = "Deployment"
	case "statefulset":
		kind = "StatefulSet"
	case "daemonset":
		kind = "DaemonSet"
	case "replicaset":
		kind = "ReplicaSet"
	case "service":
		kind = "Service"
	case "ingress":
		kind = "Ingress"
	case "configmap":
		kind = "ConfigMap"
	case "secret":
		kind = "Secret"
	case "persistentvolumeclaim", "pvc":
		kind = "PersistentVolumeClaim"
	case "persistentvolume", "pv":
		kind = "PersistentVolume"
	case "storageclass":
		kind = "StorageClass"
	case "job":
		kind = "Job"
	case "cronjob":
		kind = "CronJob"
	}
	return ActionTargetResource{
		Kind:      kind,
		Namespace: strings.TrimSpace(target.Namespace),
		Name:      strings.TrimSpace(target.Name),
	}
}

// ActionWorkloadGVR maps the workload subset that supports rollout actions.
func ActionWorkloadGVR(kind string) (schema.GroupVersionResource, bool) {
	switch strings.TrimSpace(kind) {
	case "Deployment":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, true
	case "StatefulSet":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, true
	case "DaemonSet":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, true
	default:
		return schema.GroupVersionResource{}, false
	}
}
