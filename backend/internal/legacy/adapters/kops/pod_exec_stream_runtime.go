package kops

import (
	"context"
	"io"

	"k8s.io/client-go/tools/remotecommand"

	"k8s-platform-backend/internal/legacy/service"
)

type PodExecStreamRuntime struct{ service *service.K8sService }

func NewPodExecStreamRuntime(service *service.K8sService) *PodExecStreamRuntime {
	return &PodExecStreamRuntime{service: service}
}

func (r *PodExecStreamRuntime) Stream(ctx context.Context, clusterID uint64, namespace, pod string, container string, command []string, tty bool, stdin io.Reader, stdout, stderr io.Writer, resizeQueue remotecommand.TerminalSizeQueue) error {
	if r == nil || r.service == nil {
		return service.ErrInvalidParams
	}
	var target *string
	if container != "" {
		target = &container
	}
	return r.service.PodExec(ctx, clusterID, namespace, pod, target, command, tty, stdin, stdout, stderr, resizeQueue)
}
