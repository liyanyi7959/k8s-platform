package application

import (
	"context"
	"strings"
)

type PlatformResource string

const (
	PlatformRuntimeClass                     PlatformResource = "runtimeclass"
	PlatformCSIDriver                        PlatformResource = "csidriver"
	PlatformCSINode                          PlatformResource = "csinode"
	PlatformCSIStorageCapacity               PlatformResource = "csistoragecapacity"
	PlatformValidatingAdmissionPolicy        PlatformResource = "validatingadmissionpolicy"
	PlatformValidatingAdmissionPolicyBinding PlatformResource = "validatingadmissionpolicybinding"
	PlatformPodDisruptionBudget              PlatformResource = "poddisruptionbudget"
	PlatformRole                             PlatformResource = "role"
	PlatformClusterRole                      PlatformResource = "clusterrole"
	PlatformRoleBinding                      PlatformResource = "rolebinding"
	PlatformClusterRoleBinding               PlatformResource = "clusterrolebinding"
	PlatformCustomResourceDefinition         PlatformResource = "customresourcedefinition"
	PlatformAPIService                       PlatformResource = "apiservice"
	PlatformPriorityClass                    PlatformResource = "priorityclass"
	PlatformValidatingWebhookConfiguration   PlatformResource = "validatingwebhookconfiguration"
	PlatformMutatingWebhookConfiguration     PlatformResource = "mutatingwebhookconfiguration"
	PlatformNetworkPolicy                    PlatformResource = "networkpolicy"
	PlatformResourceQuota                    PlatformResource = "resourcequota"
	PlatformLimitRange                       PlatformResource = "limitrange"
	PlatformServiceAccount                   PlatformResource = "serviceaccount"
	PlatformHorizontalPodAutoscaler          PlatformResource = "horizontalpodautoscaler"
)

type PlatformResourceListQuery struct {
	ClusterID uint64
	Namespace string
	SortBy    string
	Order     string
}

type PlatformResourceReference struct {
	ClusterID uint64
	Namespace string
	Name      string
}

type PlatformResourceEdit struct {
	ClusterID uint64
	Namespace string
	YAML      string
}

type PlatformResourceRuntime interface {
	List(context.Context, PlatformResource, PlatformResourceListQuery) (any, error)
	YAML(context.Context, PlatformResource, PlatformResourceReference) (any, error)
	Apply(context.Context, PlatformResource, PlatformResourceEdit) error
	Delete(context.Context, PlatformResource, PlatformResourceReference) error
}

type PlatformResourceService struct{ runtime PlatformResourceRuntime }

func NewPlatformResourceService(runtime PlatformResourceRuntime) *PlatformResourceService {
	return &PlatformResourceService{runtime: runtime}
}

func (s *PlatformResourceService) List(ctx context.Context, resource PlatformResource, query PlatformResourceListQuery) (any, error) {
	if !validPlatformResource(resource) || query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	query.Namespace = strings.TrimSpace(query.Namespace)
	query.SortBy = strings.TrimSpace(query.SortBy)
	query.Order = strings.TrimSpace(query.Order)
	if !platformResourceIsNamespaced(resource) {
		query.Namespace = ""
	}
	return s.runtime.List(ctx, resource, query)
}

func (s *PlatformResourceService) YAML(ctx context.Context, resource PlatformResource, ref PlatformResourceReference) (any, error) {
	if err := validatePlatformReference(resource, ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.YAML(ctx, resource, normalizedPlatformReference(resource, ref))
}

func (s *PlatformResourceService) Apply(ctx context.Context, resource PlatformResource, edit PlatformResourceEdit) error {
	if !validPlatformResource(resource) || edit.ClusterID == 0 || strings.TrimSpace(edit.YAML) == "" {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	edit.Namespace = strings.TrimSpace(edit.Namespace)
	edit.YAML = strings.TrimSpace(edit.YAML)
	if platformResourceApplyRequiresNamespace(resource) && edit.Namespace == "" {
		return ErrInvalidParams
	}
	if !platformResourceIsNamespaced(resource) {
		edit.Namespace = ""
	}
	return s.runtime.Apply(ctx, resource, edit)
}

func (s *PlatformResourceService) Delete(ctx context.Context, resource PlatformResource, ref PlatformResourceReference) error {
	if err := validatePlatformReference(resource, ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, resource, normalizedPlatformReference(resource, ref))
}

func validPlatformResource(resource PlatformResource) bool {
	switch resource {
	case PlatformRuntimeClass, PlatformCSIDriver, PlatformCSINode, PlatformCSIStorageCapacity, PlatformValidatingAdmissionPolicy, PlatformValidatingAdmissionPolicyBinding,
		PlatformPodDisruptionBudget, PlatformRole, PlatformClusterRole, PlatformRoleBinding, PlatformClusterRoleBinding,
		PlatformCustomResourceDefinition, PlatformAPIService, PlatformPriorityClass, PlatformValidatingWebhookConfiguration, PlatformMutatingWebhookConfiguration,
		PlatformNetworkPolicy, PlatformResourceQuota, PlatformLimitRange, PlatformServiceAccount, PlatformHorizontalPodAutoscaler:
		return true
	default:
		return false
	}
}

func platformResourceIsNamespaced(resource PlatformResource) bool {
	return resource == PlatformCSIStorageCapacity || resource == PlatformPodDisruptionBudget || resource == PlatformRole || resource == PlatformRoleBinding ||
		resource == PlatformNetworkPolicy || resource == PlatformResourceQuota || resource == PlatformLimitRange || resource == PlatformServiceAccount || resource == PlatformHorizontalPodAutoscaler
}

func platformResourceApplyRequiresNamespace(resource PlatformResource) bool {
	return resource == PlatformCSIStorageCapacity || resource == PlatformPodDisruptionBudget || resource == PlatformRole || resource == PlatformRoleBinding || resource == PlatformServiceAccount || resource == PlatformHorizontalPodAutoscaler
}

func validatePlatformReference(resource PlatformResource, ref PlatformResourceReference) error {
	if !validPlatformResource(resource) || ref.ClusterID == 0 || strings.TrimSpace(ref.Name) == "" {
		return ErrInvalidParams
	}
	if platformResourceIsNamespaced(resource) && strings.TrimSpace(ref.Namespace) == "" {
		return ErrInvalidParams
	}
	return nil
}

func normalizedPlatformReference(resource PlatformResource, ref PlatformResourceReference) PlatformResourceReference {
	ref.Name = strings.TrimSpace(ref.Name)
	ref.Namespace = strings.TrimSpace(ref.Namespace)
	if !platformResourceIsNamespaced(resource) {
		ref.Namespace = ""
	}
	return ref
}
