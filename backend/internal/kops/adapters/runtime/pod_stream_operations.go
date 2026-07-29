package runtime

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

// PodStreamTransport supplies only the REST configuration required for the
// Kubernetes log and exec streaming protocol. The composition root owns the
// retained credential service; this runtime never imports it directly.
type PodStreamTransport interface {
	RESTConfig(context.Context, uint64) (*rest.Config, error)
}

// PodStreamOperations owns Kubernetes pod-log and SPDY exec transport. Pod
// cache/listing policy remains in the retained service while the runtime-only
// protocol is isolated in the Kops bounded context.
type PodStreamOperations struct{ transport PodStreamTransport }

func NewPodStreamOperations(transport PodStreamTransport) *PodStreamOperations {
	return &PodStreamOperations{transport: transport}
}

func (o *PodStreamOperations) PodLogStream(ctx context.Context, clusterID uint64, namespace, pod, container string, follow bool, tailLines int64, previous bool) (io.ReadCloser, error) {
	namespace = strings.TrimSpace(namespace)
	pod = strings.TrimSpace(pod)
	if namespace == "" || pod == "" {
		return nil, kopsapp.ErrInvalidParams
	}

	config, err := o.restConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	streamConfig := rest.CopyConfig(config)
	streamConfig.Timeout = 0
	client, err := kubernetes.NewForConfig(streamConfig)
	if err != nil {
		return nil, nodeOperationError(err)
	}
	stream, err := client.CoreV1().Pods(namespace).GetLogs(pod, buildPodLogOptions(container, follow, tailLines, previous)).Stream(ctx)
	if err != nil {
		return nil, nodeOperationError(err)
	}
	return stream, nil
}

func (o *PodStreamOperations) PodLogs(ctx context.Context, clusterID uint64, namespace, pod, container string, tailLines int64, previous bool) (string, error) {
	stream, err := o.PodLogStream(ctx, clusterID, namespace, pod, container, false, tailLines, previous)
	if err != nil {
		return "", err
	}
	defer func() { _ = stream.Close() }()
	raw, err := io.ReadAll(stream)
	if err != nil {
		return "", nodeOperationError(err)
	}
	return string(raw), nil
}

func (o *PodStreamOperations) PodExec(
	ctx context.Context,
	clusterID uint64,
	namespace, pod string,
	container *string,
	command []string,
	tty bool,
	stdin io.Reader,
	stdout, stderr io.Writer,
	resizeQueue remotecommand.TerminalSizeQueue,
) error {
	namespace = strings.TrimSpace(namespace)
	pod = strings.TrimSpace(pod)
	if namespace == "" || pod == "" {
		return kopsapp.ErrInvalidParams
	}
	if len(command) == 0 {
		command = []string{"/bin/sh"}
	}

	config, err := o.restConfig(ctx, clusterID)
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nodeOperationError(err)
	}
	podObject, err := client.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return nodeOperationError(err)
	}

	streamConfig := rest.CopyConfig(config)
	streamConfig.Timeout = 0
	streamStderr := stderr
	if tty {
		streamStderr = nil
	}

	candidates := podExecCommandCandidates(command)
	var lastErr error
	for index, candidate := range candidates {
		options := buildPodExecOptions(container, candidate, tty, stdin, stdout, stderr)
		request := client.CoreV1().RESTClient().
			Post().
			Resource("pods").
			Name(pod).
			Namespace(namespace).
			SubResource("exec")
		request.VersionedParams(options, scheme.ParameterCodec)

		executor, err := remotecommand.NewSPDYExecutor(streamConfig, "POST", request.URL())
		if err != nil {
			return normalizePodExecError(err, podObject, candidate)
		}
		err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:             stdin,
			Stdout:            stdout,
			Stderr:            streamStderr,
			Tty:               tty,
			TerminalSizeQueue: resizeQueue,
		})
		if err == nil {
			return nil
		}

		lastErr = normalizePodExecError(err, podObject, candidate)
		if index < len(candidates)-1 && isExecCommandNotFoundError(err) {
			continue
		}
		return lastErr
	}
	return lastErr
}

func (o *PodStreamOperations) restConfig(ctx context.Context, clusterID uint64) (*rest.Config, error) {
	if o == nil || o.transport == nil {
		return nil, kopsapp.ErrConflict
	}
	config, err := o.transport.RESTConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, kopsapp.ErrRuntime
	}
	return config, nil
}

func buildPodLogOptions(container string, follow bool, tailLines int64, previous bool) *corev1.PodLogOptions {
	options := &corev1.PodLogOptions{Container: strings.TrimSpace(container), Follow: follow, Previous: previous}
	if tailLines < 0 {
		tailLines = 200
	}
	if tailLines > 0 {
		options.TailLines = &tailLines
	}
	return options
}

func buildPodExecOptions(container *string, command []string, tty bool, stdin io.Reader, stdout, stderr io.Writer) *corev1.PodExecOptions {
	if len(command) == 0 {
		command = []string{"/bin/sh"}
	}
	options := &corev1.PodExecOptions{
		Command: command,
		Stdin:   stdin != nil,
		Stdout:  stdout != nil,
		Stderr:  !tty && stderr != nil,
		TTY:     tty,
	}
	if container != nil {
		options.Container = strings.TrimSpace(*container)
	}
	return options
}

func isExecCommandNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(lower, "executable file not found") || strings.Contains(lower, "no such file or directory")
}

func podExecCommandCandidates(command []string) [][]string {
	trimmed := make([]string, 0, len(command))
	for _, item := range command {
		if value := strings.TrimSpace(item); value != "" {
			trimmed = append(trimmed, value)
		}
	}
	if len(trimmed) == 0 {
		trimmed = []string{"/bin/sh"}
	}
	if len(trimmed) != 1 {
		return [][]string{trimmed}
	}

	commonShells := []string{trimmed[0], "/bin/sh", "sh", "/bin/bash", "bash", "/bin/ash", "ash"}
	seen := make(map[string]struct{}, len(commonShells))
	candidates := make([][]string, 0, len(commonShells))
	for _, shell := range commonShells {
		shell = strings.TrimSpace(shell)
		if shell == "" {
			continue
		}
		if _, ok := seen[shell]; ok {
			continue
		}
		seen[shell] = struct{}{}
		candidates = append(candidates, []string{shell})
	}
	return candidates
}

func normalizePodExecError(err error, pod *corev1.Pod, command []string) error {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	if lower == "" {
		return nodeOperationError(err)
	}
	if strings.Contains(lower, "pod does not exist") {
		return kopsapp.ErrWithMessage(kopsapp.ErrNotFound, "目标 Pod 不存在或已被重新调度，请刷新列表后重试")
	}
	if strings.Contains(lower, "connection refused") && strings.Contains(lower, ":10250") {
		nodeName := ""
		if pod != nil {
			nodeName = strings.TrimSpace(pod.Spec.NodeName)
		}
		address := extractExecBackendAddress(err.Error())
		if host, port, splitErr := net.SplitHostPort(address); splitErr == nil {
			address = net.JoinHostPort(host, port)
		}
		if nodeName != "" && address != "" {
			return kopsapp.ErrWithMessage(kopsapp.ErrRuntimeNetwork, fmt.Sprintf("目标节点 %s 的 kubelet(%s) 不可达，无法建立 PodShell，请检查节点网络、kubelet 进程和 10250 端口", nodeName, address))
		}
		if nodeName != "" {
			return kopsapp.ErrWithMessage(kopsapp.ErrRuntimeNetwork, fmt.Sprintf("目标节点 %s 的 kubelet 不可达，无法建立 PodShell，请检查节点网络、kubelet 进程和 10250 端口", nodeName))
		}
		return kopsapp.ErrWithMessage(kopsapp.ErrRuntimeNetwork, "目标节点的 kubelet 不可达，无法建立 PodShell，请检查节点网络、kubelet 进程和 10250 端口")
	}
	if isExecCommandNotFoundError(err) {
		commandName := "sh"
		if len(command) > 0 && strings.TrimSpace(command[0]) != "" {
			commandName = strings.TrimSpace(command[0])
		}
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, fmt.Sprintf("容器内不存在命令 %s，请改用 /bin/sh、bash 或镜像实际提供的 shell", commandName))
	}
	return nodeOperationError(err)
}

func extractExecBackendAddress(message string) string {
	text := strings.TrimSpace(message)
	if text == "" {
		return ""
	}
	marker := "dial tcp "
	index := strings.Index(strings.ToLower(text), marker)
	if index < 0 {
		return ""
	}
	rest := strings.TrimSpace(text[index+len(marker):])
	if end := strings.Index(rest, ": connect"); end >= 0 {
		return strings.TrimSpace(rest[:end])
	}
	return rest
}
