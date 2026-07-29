package service

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// These Kops resource lookup helpers are retained beside legacy K8s runtime
// code until the runtime itself moves behind the Kops application ports.
func truncateForModel(input string, limit int) string {
	raw := strings.TrimSpace(input)
	if limit <= 0 || len([]rune(raw)) <= limit {
		return raw
	}
	return string([]rune(raw)[:limit]) + "..."
}

func AIObjectMetaString(item any, key string) string {
	obj, ok := item.(map[string]any)
	if !ok {
		return ""
	}
	meta, ok := obj["metadata"].(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(meta[key]))
}

func aiMapValue(source map[string]any, key string) map[string]any {
	if source == nil {
		return nil
	}
	value, _ := source[key].(map[string]any)
	return value
}

func aiBoolValue(value any) bool {
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

func aiSupportedResourceGVR(kind string) (schema.GroupVersionResource, bool, bool) {
	type resourceRef struct {
		group, version, resource string
		namespaced               bool
	}
	refs := map[string]resourceRef{
		"namespace": {"", "v1", "namespaces", false}, "node": {"", "v1", "nodes", false},
		"pod": {"", "v1", "pods", true}, "deployment": {"apps", "v1", "deployments", true},
		"statefulset": {"apps", "v1", "statefulsets", true}, "daemonset": {"apps", "v1", "daemonsets", true},
		"replicaset": {"apps", "v1", "replicasets", true}, "service": {"", "v1", "services", true},
		"ingress": {"networking.k8s.io", "v1", "ingresses", true}, "ingressclass": {"networking.k8s.io", "v1", "ingressclasses", false},
		"networkpolicy": {"networking.k8s.io", "v1", "networkpolicies", true}, "configmap": {"", "v1", "configmaps", true},
		"secret": {"", "v1", "secrets", true}, "serviceaccount": {"", "v1", "serviceaccounts", true},
		"endpoint": {"", "v1", "endpoints", true}, "endpointslice": {"discovery.k8s.io", "v1", "endpointslices", true},
		"lease": {"coordination.k8s.io", "v1", "leases", true}, "pdb": {"policy", "v1", "poddisruptionbudgets", true},
		"poddisruptionbudget": {"policy", "v1", "poddisruptionbudgets", true}, "role": {"rbac.authorization.k8s.io", "v1", "roles", true},
		"clusterrole": {"rbac.authorization.k8s.io", "v1", "clusterroles", false}, "rolebinding": {"rbac.authorization.k8s.io", "v1", "rolebindings", true},
		"clusterrolebinding": {"rbac.authorization.k8s.io", "v1", "clusterrolebindings", false}, "hpa": {"autoscaling", "v2", "horizontalpodautoscalers", true},
		"horizontalpodautoscaler": {"autoscaling", "v2", "horizontalpodautoscalers", true}, "event": {"", "v1", "events", true},
		"pvc": {"", "v1", "persistentvolumeclaims", true}, "persistentvolumeclaim": {"", "v1", "persistentvolumeclaims", true},
		"pv": {"", "v1", "persistentvolumes", false}, "persistentvolume": {"", "v1", "persistentvolumes", false},
		"storageclass": {"storage.k8s.io", "v1", "storageclasses", false}, "csidriver": {"storage.k8s.io", "v1", "csidrivers", false},
		"csinode": {"storage.k8s.io", "v1", "csinodes", false}, "csistoragecapacity": {"storage.k8s.io", "v1", "csistoragecapacities", true},
		"volumeattachment": {"storage.k8s.io", "v1", "volumeattachments", false}, "volumesnapshot": {"snapshot.storage.k8s.io", "v1", "volumesnapshots", true},
		"volumesnapshotclass": {"snapshot.storage.k8s.io", "v1", "volumesnapshotclasses", false}, "volumesnapshotcontent": {"snapshot.storage.k8s.io", "v1", "volumesnapshotcontents", false},
		"resourcequota": {"", "v1", "resourcequotas", true}, "limitrange": {"", "v1", "limitranges", true},
		"customresourcedefinition": {"apiextensions.k8s.io", "v1", "customresourcedefinitions", false}, "crd": {"apiextensions.k8s.io", "v1", "customresourcedefinitions", false},
		"apiservice": {"apiregistration.k8s.io", "v1", "apiservices", false}, "priorityclass": {"scheduling.k8s.io", "v1", "priorityclasses", false},
		"runtimeclass": {"node.k8s.io", "v1", "runtimeclasses", false}, "validatingwebhookconfiguration": {"admissionregistration.k8s.io", "v1", "validatingwebhookconfigurations", false},
		"mutatingwebhookconfiguration": {"admissionregistration.k8s.io", "v1", "mutatingwebhookconfigurations", false}, "validatingadmissionpolicy": {"admissionregistration.k8s.io", "v1", "validatingadmissionpolicies", false},
		"validatingadmissionpolicybinding": {"admissionregistration.k8s.io", "v1", "validatingadmissionpolicybindings", false}, "job": {"batch", "v1", "jobs", true}, "cronjob": {"batch", "v1", "cronjobs", true},
	}
	ref, ok := refs[strings.ToLower(strings.TrimSpace(kind))]
	if !ok {
		return schema.GroupVersionResource{}, false, false
	}
	return schema.GroupVersionResource{Group: ref.group, Version: ref.version, Resource: ref.resource}, ref.namespaced, true
}

func aiGenericNamespacedGVR(kind string) (schema.GroupVersionResource, bool) {
	gvr, namespaced, ok := aiSupportedResourceGVR(kind)
	if !ok || !namespaced {
		return schema.GroupVersionResource{}, false
	}
	return gvr, true
}
