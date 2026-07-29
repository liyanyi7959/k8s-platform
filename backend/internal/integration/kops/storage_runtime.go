package kops

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

type StorageRuntime struct{ service *service.K8sService }

func NewStorageRuntime(service *service.K8sService) *StorageRuntime {
	return &StorageRuntime{service: service}
}
func (r *StorageRuntime) List(ctx context.Context, resource kopsapp.StorageResource, query kopsapp.StorageListQuery) (any, error) {
	gvr, err := storageGVR(resource)
	if err != nil {
		return nil, err
	}
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.List(ctx, query.ClusterID, gvr, query.Namespace, query.SortBy, query.Order, nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}
func (r *StorageRuntime) YAML(ctx context.Context, resource kopsapp.StorageResource, ref kopsapp.StorageReference) (any, error) {
	gvr, err := storageGVR(resource)
	if err != nil {
		return nil, err
	}
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}
func (r *StorageRuntime) Delete(ctx context.Context, resource kopsapp.StorageResource, ref kopsapp.StorageReference) error {
	gvr, err := storageGVR(resource)
	if err != nil {
		return err
	}
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name))
}
func (r *StorageRuntime) Apply(ctx context.Context, resource kopsapp.StorageResource, edit kopsapp.StorageEdit) error {
	gvr, err := storageGVR(resource)
	if err != nil {
		return err
	}
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.ApplyYAML(ctx, edit.ClusterID, gvr, edit.Namespace, edit.YAML))
}
func (r *StorageRuntime) CreatePVC(ctx context.Context, input kopsapp.CreatePVCInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	client, err := r.service.TypedClient(ctx, input.ClusterID)
	if err != nil {
		return translateKopsRuntimeError(err)
	}
	quantity, err := apiresource.ParseQuantity(input.Capacity)
	if err != nil || quantity.Sign() <= 0 {
		return kopsapp.ErrInvalidParams
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: strings.TrimSpace(input.Name), Namespace: strings.TrimSpace(input.Namespace)},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: pvcAccessModes(input.AccessModes),
			Resources:   corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: quantity}},
		},
	}
	if storageClass := strings.TrimSpace(input.StorageClass); storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}
	_, err = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	return translateKopsRuntimeError(service.NormalizeKubernetesError(err))
}

func pvcAccessModes(values []string) []corev1.PersistentVolumeAccessMode {
	if len(values) == 0 {
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	modes := make([]corev1.PersistentVolumeAccessMode, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			modes = append(modes, corev1.PersistentVolumeAccessMode(value))
		}
	}
	if len(modes) == 0 {
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	return modes
}
func (r *StorageRuntime) Supports(ctx context.Context, clusterID uint64, capability kopsapp.StorageCapability) (bool, error) {
	if r == nil || r.service == nil {
		return false, kopsapp.ErrConflict
	}
	gvr, err := storageCapabilityGVR(capability)
	if err != nil {
		return false, err
	}
	value, err := r.service.SupportsCompatibleGVR(ctx, clusterID, gvr)
	if err != nil {
		return false, translateKopsRuntimeError(err)
	}
	return value, nil
}
func storageGVR(resource kopsapp.StorageResource) (schema.GroupVersionResource, error) {
	switch resource {
	case kopsapp.StoragePersistentVolumeClaim:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}, nil
	case kopsapp.StoragePersistentVolume:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}, nil
	case kopsapp.StorageClass:
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}, nil
	case kopsapp.StorageVolumeSnapshot:
		return schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshots"}, nil
	case kopsapp.StorageVolumeSnapshotClass:
		return schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotclasses"}, nil
	case kopsapp.StorageVolumeSnapshotContent:
		return schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotcontents"}, nil
	}
	return schema.GroupVersionResource{}, fmt.Errorf("unknown storage resource")
}
func storageCapabilityGVR(capability kopsapp.StorageCapability) (schema.GroupVersionResource, error) {
	values := map[kopsapp.StorageCapability]schema.GroupVersionResource{
		kopsapp.CapabilityReplicaSets: {Group: "apps", Version: "v1", Resource: "replicasets"}, kopsapp.CapabilityPodMetrics: {Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}, kopsapp.CapabilityClusterRoles: {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}, kopsapp.CapabilityEndpoints: {Group: "", Version: "v1", Resource: "endpoints"}, kopsapp.CapabilityEndpointSlices: {Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"}, kopsapp.CapabilityNetworkPolicies: {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}, kopsapp.CapabilityVolumeAttachments: {Group: "storage.k8s.io", Version: "v1", Resource: "volumeattachments"}, kopsapp.CapabilityResourceQuotas: {Group: "", Version: "v1", Resource: "resourcequotas"}, kopsapp.CapabilityLimitRanges: {Group: "", Version: "v1", Resource: "limitranges"}, kopsapp.CapabilityCustomResourceDefinitions: {Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}, kopsapp.CapabilityAPIServices: {Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}, kopsapp.CapabilityPriorityClasses: {Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"}, kopsapp.CapabilityValidatingWebhookConfigurations: {Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}, kopsapp.CapabilityMutatingWebhookConfigurations: {Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}, kopsapp.CapabilityLeases: {Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}, kopsapp.CapabilityVolumeSnapshots: {Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshots"}, kopsapp.CapabilityVolumeSnapshotClasses: {Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotclasses"}, kopsapp.CapabilityVolumeSnapshotContents: {Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotcontents"}}
	value, ok := values[capability]
	if !ok {
		return schema.GroupVersionResource{}, errors.New("unknown storage capability")
	}
	return value, nil
}
