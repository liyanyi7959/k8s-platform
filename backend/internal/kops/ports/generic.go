package ports

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GenericResourceQuery is the controlled generic API boundary. It narrows the
// retained Kubernetes transport to read-only resource listing and retrieval so
// cross-context workflows (e.g. AI evidence collection) depend on the port
// rather than the concrete K8sService.
type GenericResourceQuery interface {
	List(context.Context, uint64, schema.GroupVersionResource, string, string, string, map[string]string) ([]any, error)
	GetObject(context.Context, uint64, schema.GroupVersionResource, string, string) (map[string]any, error)
}
