package kops

import (
	"context"
	"io"

	"k8s-platform-backend/internal/legacy/service"
)

type PodLogStreamRuntime struct{ service *service.K8sService }

func NewPodLogStreamRuntime(service *service.K8sService) *PodLogStreamRuntime {
	return &PodLogStreamRuntime{service: service}
}

func (r *PodLogStreamRuntime) Stream(ctx context.Context, clusterID uint64, namespace, pod, container string, follow bool, tailLines int64, previous bool) (io.ReadCloser, error) {
	if r == nil || r.service == nil {
		return nil, service.ErrInvalidParams
	}
	return r.service.PodLogStream(ctx, clusterID, namespace, pod, container, follow, tailLines, previous)
}
