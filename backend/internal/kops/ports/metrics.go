package ports

import (
	"context"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// MetricsCluster is the monitoring configuration that Kops needs from the
// Fleet context. It is purpose-built rather than exposing Fleet's cluster
// aggregate to the metrics workflow.
type MetricsCluster struct {
	MonitorSource        string
	PrometheusURL        string
	PrometheusStatus     string
	PrometheusDetectedAt any
}

// MetricsClusterStore is the Fleet-facing port used by metrics provider
// selection and persistence.
type MetricsClusterStore interface {
	MonitorSource(context.Context, uint64) (MetricsCluster, error)
	UpdateMonitorSource(context.Context, uint64, string, string, string) error
}

// MetricsTransport is the Kubernetes-facing port. The concrete client cache
// and credential handling remain in the Kubernetes adapter.
type MetricsTransport interface {
	TypedClient(context.Context, uint64) (*kubernetes.Clientset, error)
	DynamicClient(context.Context, uint64) (*dynamic.DynamicClient, error)
}
