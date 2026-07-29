package kops

import (
	"context"
	"io"

	"k8s.io/client-go/tools/remotecommand"

	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

type PodExecStreamRuntime struct {
	streams *kopsruntime.PodStreamOperations
}

func NewPodExecStreamRuntime(streams *kopsruntime.PodStreamOperations) *PodExecStreamRuntime {
	return &PodExecStreamRuntime{streams: streams}
}

func (r *PodExecStreamRuntime) Stream(ctx context.Context, clusterID uint64, namespace, pod string, container string, command []string, tty bool, stdin io.Reader, stdout, stderr io.Writer, resizeQueue remotecommand.TerminalSizeQueue) error {
	if r == nil || r.streams == nil {
		return kopsapp.ErrConflict
	}
	var target *string
	if container != "" {
		target = &container
	}
	return r.streams.PodExec(ctx, clusterID, namespace, pod, target, command, tty, stdin, stdout, stderr, resizeQueue)
}
