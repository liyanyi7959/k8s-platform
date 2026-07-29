package kops

import (
	"context"
	"io"

	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

type PodLogStreamRuntime struct {
	streams *kopsruntime.PodStreamOperations
}

func NewPodLogStreamRuntime(streams *kopsruntime.PodStreamOperations) *PodLogStreamRuntime {
	return &PodLogStreamRuntime{streams: streams}
}

func (r *PodLogStreamRuntime) Stream(ctx context.Context, clusterID uint64, namespace, pod, container string, follow bool, tailLines int64, previous bool) (io.ReadCloser, error) {
	if r == nil || r.streams == nil {
		return nil, kopsapp.ErrConflict
	}
	return r.streams.PodLogStream(ctx, clusterID, namespace, pod, container, follow, tailLines, previous)
}
