package provisioning

import (
	"context"

	platformapp "k8s-platform-backend/internal/platform/application"
)

// DeploymentTaskStore is the Platform task-centre boundary used by the
// deployment workflow. Keeping the concrete TaskStore out of orchestration
// makes task persistence replaceable and keeps the workflow focused on state
// transitions rather than platform storage.
type DeploymentTaskStore interface {
	Put(*platformapp.Task) error
	Get(int64) (*platformapp.Task, bool)
	List() []*platformapp.Task
	RegisterCancel(int64, func())
	UnregisterCancel(int64)
	CancelExecution(int64)
}

// ClusterRegistrar is the Fleet boundary used after an Ansible deployment
// produces a kubeconfig. It intentionally exposes only the provisioning use
// case rather than the concrete fleet registry.
type ClusterRegistrar interface {
	Import(context.Context, string, string, string) (uint64, error)
}
