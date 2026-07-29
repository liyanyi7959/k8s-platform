package application

import (
	"context"
	"strings"
)

type RelationshipResource string

const (
	RelationshipReplicaSet       RelationshipResource = "replicaset"
	RelationshipVolumeAttachment RelationshipResource = "volumeattachment"
)

type RelationshipResourceListQuery struct {
	ClusterID uint64
	Namespace string
	SortBy    string
	Order     string
}

type RelationshipResourceReference struct {
	ClusterID uint64
	Namespace string
	Name      string
}

type RelationshipResourceEdit struct {
	ClusterID uint64
	Namespace string
	YAML      string
}

type RelationshipResourceRuntime interface {
	List(context.Context, RelationshipResource, RelationshipResourceListQuery) (any, error)
	YAML(context.Context, RelationshipResource, RelationshipResourceReference) (any, error)
	Apply(context.Context, RelationshipResource, RelationshipResourceEdit) error
	Delete(context.Context, RelationshipResource, RelationshipResourceReference) error
}

type RelationshipResourceService struct{ runtime RelationshipResourceRuntime }

func NewRelationshipResourceService(runtime RelationshipResourceRuntime) *RelationshipResourceService {
	return &RelationshipResourceService{runtime: runtime}
}

func (s *RelationshipResourceService) List(ctx context.Context, resource RelationshipResource, query RelationshipResourceListQuery) (any, error) {
	if !validRelationshipResource(resource) || query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	query.Namespace = strings.TrimSpace(query.Namespace)
	query.SortBy = strings.TrimSpace(query.SortBy)
	query.Order = strings.TrimSpace(query.Order)
	if !relationshipResourceIsNamespaced(resource) {
		query.Namespace = ""
	}
	return s.runtime.List(ctx, resource, query)
}

func (s *RelationshipResourceService) YAML(ctx context.Context, resource RelationshipResource, ref RelationshipResourceReference) (any, error) {
	if err := validateRelationshipReference(resource, ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.YAML(ctx, resource, normalizedRelationshipReference(resource, ref))
}

func (s *RelationshipResourceService) Apply(ctx context.Context, resource RelationshipResource, edit RelationshipResourceEdit) error {
	if !validRelationshipResource(resource) || edit.ClusterID == 0 || strings.TrimSpace(edit.YAML) == "" {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	edit.Namespace = strings.TrimSpace(edit.Namespace)
	edit.YAML = strings.TrimSpace(edit.YAML)
	if relationshipResourceIsNamespaced(resource) && edit.Namespace == "" {
		return ErrInvalidParams
	}
	if !relationshipResourceIsNamespaced(resource) {
		edit.Namespace = ""
	}
	return s.runtime.Apply(ctx, resource, edit)
}

func (s *RelationshipResourceService) Delete(ctx context.Context, resource RelationshipResource, ref RelationshipResourceReference) error {
	if err := validateRelationshipReference(resource, ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, resource, normalizedRelationshipReference(resource, ref))
}

func validRelationshipResource(resource RelationshipResource) bool {
	return resource == RelationshipReplicaSet || resource == RelationshipVolumeAttachment
}

func relationshipResourceIsNamespaced(resource RelationshipResource) bool {
	return resource == RelationshipReplicaSet
}

func validateRelationshipReference(resource RelationshipResource, ref RelationshipResourceReference) error {
	if !validRelationshipResource(resource) || ref.ClusterID == 0 || strings.TrimSpace(ref.Name) == "" {
		return ErrInvalidParams
	}
	if relationshipResourceIsNamespaced(resource) && strings.TrimSpace(ref.Namespace) == "" {
		return ErrInvalidParams
	}
	return nil
}

func normalizedRelationshipReference(resource RelationshipResource, ref RelationshipResourceReference) RelationshipResourceReference {
	ref.Name = strings.TrimSpace(ref.Name)
	ref.Namespace = strings.TrimSpace(ref.Namespace)
	if !relationshipResourceIsNamespaced(resource) {
		ref.Namespace = ""
	}
	return ref
}
