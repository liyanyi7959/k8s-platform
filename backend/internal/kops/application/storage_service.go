package application

import (
	"context"
	"strings"
	"sync"
	"time"
)

type StorageResource string

const (
	StoragePersistentVolumeClaim StorageResource = "persistentvolumeclaim"
	StoragePersistentVolume      StorageResource = "persistentvolume"
	StorageClass                 StorageResource = "storageclass"
	StorageVolumeSnapshot        StorageResource = "volumesnapshot"
	StorageVolumeSnapshotClass   StorageResource = "volumesnapshotclass"
	StorageVolumeSnapshotContent StorageResource = "volumesnapshotcontent"
)

type StorageListQuery struct {
	ClusterID uint64
	Namespace string
	SortBy    string
	Order     string
}

type StorageReference struct {
	ClusterID uint64
	Namespace string
	Name      string
}

type StorageEdit struct {
	ClusterID uint64
	Namespace string
	YAML      string
}

type CreatePVCInput struct {
	ClusterID    uint64   `json:"-"`
	Namespace    string   `json:"namespace"`
	Name         string   `json:"name"`
	StorageClass string   `json:"storage_class"`
	AccessModes  []string `json:"access_modes"`
	Capacity     string   `json:"capacity"`
}

type ResourceSupport struct {
	ReplicaSets                     *bool `json:"replicasets,omitempty"`
	PodMetrics                      *bool `json:"podmetrics,omitempty"`
	ClusterRoles                    *bool `json:"clusterroles,omitempty"`
	Endpoints                       *bool `json:"endpoints,omitempty"`
	EndpointSlices                  *bool `json:"endpointslices,omitempty"`
	NetworkPolicies                 *bool `json:"networkpolicies,omitempty"`
	VolumeAttachments               *bool `json:"volumeattachments,omitempty"`
	ResourceQuotas                  *bool `json:"resourcequotas,omitempty"`
	LimitRanges                     *bool `json:"limitranges,omitempty"`
	CustomResourceDefinitions       *bool `json:"customresourcedefinitions,omitempty"`
	APIServices                     *bool `json:"apiservices,omitempty"`
	PriorityClasses                 *bool `json:"priorityclasses,omitempty"`
	ValidatingWebhookConfigurations *bool `json:"validatingwebhookconfigurations,omitempty"`
	MutatingWebhookConfigurations   *bool `json:"mutatingwebhookconfigurations,omitempty"`
	Leases                          *bool `json:"leases,omitempty"`
	VolumeSnapshots                 *bool `json:"volumesnapshots,omitempty"`
	VolumeSnapshotClasses           *bool `json:"volumesnapshotclasses,omitempty"`
	VolumeSnapshotContents          *bool `json:"volumesnapshotcontents,omitempty"`
}

type StorageCapability string

const (
	CapabilityReplicaSets                     StorageCapability = "replicasets"
	CapabilityPodMetrics                      StorageCapability = "podmetrics"
	CapabilityClusterRoles                    StorageCapability = "clusterroles"
	CapabilityEndpoints                       StorageCapability = "endpoints"
	CapabilityEndpointSlices                  StorageCapability = "endpointslices"
	CapabilityNetworkPolicies                 StorageCapability = "networkpolicies"
	CapabilityVolumeAttachments               StorageCapability = "volumeattachments"
	CapabilityResourceQuotas                  StorageCapability = "resourcequotas"
	CapabilityLimitRanges                     StorageCapability = "limitranges"
	CapabilityCustomResourceDefinitions       StorageCapability = "customresourcedefinitions"
	CapabilityAPIServices                     StorageCapability = "apiservices"
	CapabilityPriorityClasses                 StorageCapability = "priorityclasses"
	CapabilityValidatingWebhookConfigurations StorageCapability = "validatingwebhookconfigurations"
	CapabilityMutatingWebhookConfigurations   StorageCapability = "mutatingwebhookconfigurations"
	CapabilityLeases                          StorageCapability = "leases"
	CapabilityVolumeSnapshots                 StorageCapability = "volumesnapshots"
	CapabilityVolumeSnapshotClasses           StorageCapability = "volumesnapshotclasses"
	CapabilityVolumeSnapshotContents          StorageCapability = "volumesnapshotcontents"
)

type StorageRuntime interface {
	List(context.Context, StorageResource, StorageListQuery) (any, error)
	YAML(context.Context, StorageResource, StorageReference) (any, error)
	Delete(context.Context, StorageResource, StorageReference) error
	Apply(context.Context, StorageResource, StorageEdit) error
	CreatePVC(context.Context, CreatePVCInput) error
	Supports(context.Context, uint64, StorageCapability) (bool, error)
}

type StorageService struct{ runtime StorageRuntime }

func NewStorageService(runtime StorageRuntime) *StorageService {
	return &StorageService{runtime: runtime}
}

func (s *StorageService) List(ctx context.Context, resource StorageResource, query StorageListQuery) (any, error) {
	if !validStorageResource(resource) || query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	query.Namespace, query.SortBy, query.Order = strings.TrimSpace(query.Namespace), strings.TrimSpace(query.SortBy), strings.TrimSpace(query.Order)
	if !storageResourceIsNamespaced(resource) {
		query.Namespace = ""
	}
	return s.runtime.List(ctx, resource, query)
}

func (s *StorageService) YAML(ctx context.Context, resource StorageResource, ref StorageReference) (any, error) {
	if err := validateStorageReference(resource, ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.YAML(ctx, resource, normalizeStorageReference(resource, ref))
}

func (s *StorageService) Delete(ctx context.Context, resource StorageResource, ref StorageReference) error {
	if err := validateStorageReference(resource, ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, resource, normalizeStorageReference(resource, ref))
}

func (s *StorageService) Apply(ctx context.Context, resource StorageResource, edit StorageEdit) error {
	if !validStorageResource(resource) || resource == StoragePersistentVolumeClaim || resource == StoragePersistentVolume || edit.ClusterID == 0 || strings.TrimSpace(edit.YAML) == "" {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	edit.Namespace, edit.YAML = strings.TrimSpace(edit.Namespace), strings.TrimSpace(edit.YAML)
	if storageResourceIsNamespaced(resource) && edit.Namespace == "" {
		return ErrInvalidParams
	}
	if !storageResourceIsNamespaced(resource) {
		edit.Namespace = ""
	}
	return s.runtime.Apply(ctx, resource, edit)
}

func (s *StorageService) CreatePVC(ctx context.Context, input CreatePVCInput) error {
	input.Namespace, input.Name, input.StorageClass, input.Capacity = strings.TrimSpace(input.Namespace), strings.TrimSpace(input.Name), strings.TrimSpace(input.StorageClass), strings.TrimSpace(input.Capacity)
	if input.ClusterID == 0 || input.Namespace == "" || input.Name == "" || input.Capacity == "" {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.CreatePVC(ctx, input)
}

func (s *StorageService) ResourceSupport(ctx context.Context, clusterID uint64) (ResourceSupport, error) {
	if clusterID == 0 {
		return ResourceSupport{}, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ResourceSupport{}, ErrConflict
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	type check struct {
		capability StorageCapability
		set        func(*ResourceSupport, *bool)
	}
	checks := []check{
		{CapabilityReplicaSets, func(out *ResourceSupport, value *bool) { out.ReplicaSets = value }}, {CapabilityPodMetrics, func(out *ResourceSupport, value *bool) { out.PodMetrics = value }},
		{CapabilityClusterRoles, func(out *ResourceSupport, value *bool) { out.ClusterRoles = value }}, {CapabilityEndpoints, func(out *ResourceSupport, value *bool) { out.Endpoints = value }},
		{CapabilityEndpointSlices, func(out *ResourceSupport, value *bool) { out.EndpointSlices = value }}, {CapabilityNetworkPolicies, func(out *ResourceSupport, value *bool) { out.NetworkPolicies = value }},
		{CapabilityVolumeAttachments, func(out *ResourceSupport, value *bool) { out.VolumeAttachments = value }}, {CapabilityResourceQuotas, func(out *ResourceSupport, value *bool) { out.ResourceQuotas = value }},
		{CapabilityLimitRanges, func(out *ResourceSupport, value *bool) { out.LimitRanges = value }}, {CapabilityCustomResourceDefinitions, func(out *ResourceSupport, value *bool) { out.CustomResourceDefinitions = value }},
		{CapabilityAPIServices, func(out *ResourceSupport, value *bool) { out.APIServices = value }}, {CapabilityPriorityClasses, func(out *ResourceSupport, value *bool) { out.PriorityClasses = value }},
		{CapabilityValidatingWebhookConfigurations, func(out *ResourceSupport, value *bool) { out.ValidatingWebhookConfigurations = value }}, {CapabilityMutatingWebhookConfigurations, func(out *ResourceSupport, value *bool) { out.MutatingWebhookConfigurations = value }},
		{CapabilityLeases, func(out *ResourceSupport, value *bool) { out.Leases = value }}, {CapabilityVolumeSnapshots, func(out *ResourceSupport, value *bool) { out.VolumeSnapshots = value }},
		{CapabilityVolumeSnapshotClasses, func(out *ResourceSupport, value *bool) { out.VolumeSnapshotClasses = value }}, {CapabilityVolumeSnapshotContents, func(out *ResourceSupport, value *bool) { out.VolumeSnapshotContents = value }},
	}
	values := make([]*bool, len(checks))
	semaphore := make(chan struct{}, 4)
	var wait sync.WaitGroup
	for index, item := range checks {
		wait.Add(1)
		semaphore <- struct{}{}
		go func(index int, item check) {
			defer wait.Done()
			defer func() { <-semaphore }()
			supported, err := s.runtime.Supports(ctx, clusterID, item.capability)
			if err == nil {
				values[index] = &supported
			}
		}(index, item)
	}
	wait.Wait()
	var result ResourceSupport
	for index, item := range checks {
		item.set(&result, values[index])
	}
	return result, nil
}

func validStorageResource(resource StorageResource) bool {
	switch resource {
	case StoragePersistentVolumeClaim, StoragePersistentVolume, StorageClass, StorageVolumeSnapshot, StorageVolumeSnapshotClass, StorageVolumeSnapshotContent:
		return true
	}
	return false
}
func storageResourceIsNamespaced(resource StorageResource) bool {
	return resource == StoragePersistentVolumeClaim || resource == StorageVolumeSnapshot
}
func validateStorageReference(resource StorageResource, ref StorageReference) error {
	if !validStorageResource(resource) || ref.ClusterID == 0 || strings.TrimSpace(ref.Name) == "" || (storageResourceIsNamespaced(resource) && strings.TrimSpace(ref.Namespace) == "") {
		return ErrInvalidParams
	}
	return nil
}
func normalizeStorageReference(resource StorageResource, ref StorageReference) StorageReference {
	ref.Name, ref.Namespace = strings.TrimSpace(ref.Name), strings.TrimSpace(ref.Namespace)
	if !storageResourceIsNamespaced(resource) {
		ref.Namespace = ""
	}
	return ref
}
