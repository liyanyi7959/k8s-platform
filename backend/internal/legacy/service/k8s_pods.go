package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/remotecommand"
)

type NamespacePodHealthItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Phase        string `json:"phase"`
	Ready        string `json:"ready"`
	RestartCount int    `json:"restart_count"`
	Reason       string `json:"reason,omitempty"`
}

type NamespaceEventItem struct {
	Type          string `json:"type"`
	Reason        string `json:"reason,omitempty"`
	Message       string `json:"message,omitempty"`
	InvolvedKind  string `json:"involved_kind,omitempty"`
	InvolvedName  string `json:"involved_name,omitempty"`
	Count         int32  `json:"count,omitempty"`
	LastTimestamp string `json:"last_timestamp,omitempty"`
}

// ---------------------------------------------------------------------------
// Pod listing — cached / direct / Redis
// ---------------------------------------------------------------------------

func (s *K8sService) listPodsCached(ctx context.Context, clusterID uint64, namespace string, sortBy, order string, labelSelector string) ([]any, error) {
	entry, err := s.getOrStartPodCache(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	var selector labels.Selector
	if strings.TrimSpace(labelSelector) != "" {
		sel, err := labels.Parse(labelSelector)
		if err != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "label_selector 无效")
		}
		selector = sel
	}

	cacheEnabled := s != nil && s.cache != nil && s.cache.Enabled() && s.podTTL > 0
	if cacheEnabled {
		if out, ok := s.listPodsFromRedisExact(ctx, clusterID, namespace, labelSelector, sortBy, order); ok {
			return out, nil
		}
		if out, ok := s.listPodsFromRedisAll(ctx, clusterID, namespace, selector, sortBy, order); ok {
			return out, nil
		}
		if strings.TrimSpace(namespace) == "" && selector == nil {
			s.setPodsAllToRedis(clusterID, entry)
		}
	}

	if !entry.informer.HasSynced() {
		pods, err := s.listPodsDirectPods(ctx, clusterID, namespace, labelSelector)
		if err != nil {
			return nil, err
		}
		if cacheEnabled {
			s.setPodsListPodsToRedis(clusterID, namespace, labelSelector, pods)
		}
		sortPods(pods, sortBy, order)
		return podsToAnyList(pods), nil
	}

	pods := filterPodsFromInformer(entry.informer, namespace, selector)
	if cacheEnabled {
		s.setPodsListPodsToRedis(clusterID, namespace, labelSelector, pods)
	}
	sortPods(pods, sortBy, order)
	return podsToAnyList(pods), nil
}

func (s *K8sService) listPodsDirect(ctx context.Context, clusterID uint64, namespace string, sortBy, order string, labelSelector string) ([]any, error) {
	pods, err := s.listPodsDirectPods(ctx, clusterID, namespace, labelSelector)
	if err != nil {
		return nil, err
	}
	sortPods(pods, sortBy, order)
	return podsToAnyList(pods), nil
}

func (s *K8sService) listPodsDirectPods(ctx context.Context, clusterID uint64, namespace string, labelSelector string) ([]*corev1.Pod, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	if ls := strings.TrimSpace(labelSelector); ls != "" {
		opts.LabelSelector = ls
	}

	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	pods := make([]*corev1.Pod, 0, 256)
	for {
		pl, err := cs.CoreV1().Pods(ns).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range pl.Items {
			pods = append(pods, pl.Items[i].DeepCopy())
		}
		token := strings.TrimSpace(pl.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	return pods, nil
}

func (s *K8sService) DeletePod(ctx context.Context, clusterID uint64, namespace, name string, force bool) error {
	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(name) == "" {
		return ErrInvalidParams
	}
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}
	opts := metav1.DeleteOptions{}
	if force {
		var zero int64 = 0
		opts.GracePeriodSeconds = &zero
	}
	return normalizeK8sErr(cs.CoreV1().Pods(namespace).Delete(ctx, name, opts))
}

// ---------------------------------------------------------------------------
// Pod Redis cache helpers
// ---------------------------------------------------------------------------

func (s *K8sService) listPodsFromRedisExact(ctx context.Context, clusterID uint64, namespace string, labelSelector string, sortBy, order string) ([]any, bool) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || clusterID == 0 || s.podTTL <= 0 {
		return nil, false
	}
	key := s.podsListCacheKey(clusterID, namespace, labelSelector)
	pods, ok := s.podsFromRedisKey(ctx, key)
	if !ok {
		return nil, false
	}
	sortPods(pods, sortBy, order)
	return podsToAnyList(pods), true
}

func (s *K8sService) listPodsFromRedisAll(ctx context.Context, clusterID uint64, namespace string, selector labels.Selector, sortBy, order string) ([]any, bool) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || clusterID == 0 || s.podTTL <= 0 {
		return nil, false
	}
	pods, ok := s.podsFromRedisKey(ctx, s.podsAllCacheKey(clusterID))
	if !ok {
		return nil, false
	}
	filtered := make([]*corev1.Pod, 0, len(pods))
	for _, p := range pods {
		if p == nil {
			continue
		}
		if namespace != "" && p.Namespace != namespace {
			continue
		}
		if selector != nil && !selector.Matches(labels.Set(p.Labels)) {
			continue
		}
		filtered = append(filtered, p.DeepCopy())
	}
	sortPods(filtered, sortBy, order)
	return podsToAnyList(filtered), true
}

func (s *K8sService) podsFromRedisKey(ctx context.Context, key string) ([]*corev1.Pod, bool) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 {
		return nil, false
	}
	k := strings.TrimSpace(key)
	if k == "" {
		return nil, false
	}
	rctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	b, ok, err := s.cache.Get(rctx, k)
	cancel()
	if err != nil || !ok || b == nil {
		return nil, false
	}
	var arr []corev1.Pod
	if json.Unmarshal(b, &arr) != nil {
		return nil, false
	}
	out := make([]*corev1.Pod, 0, len(arr))
	for i := range arr {
		out = append(out, arr[i].DeepCopy())
	}
	return out, true
}

func (s *K8sService) setPodsAllToRedis(clusterID uint64, entry *podCacheEntry) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 || clusterID == 0 || entry == nil || entry.informer == nil {
		return
	}
	pods := podsFromInformerStore(entry.informer)
	s.setPodsListPodsToRedis(clusterID, "", "", pods)
}

func (s *K8sService) setPodsAllPodsToRedis(clusterID uint64, pods []*corev1.Pod) {
	s.setPodsListPodsToRedis(clusterID, "", "", pods)
}

func (s *K8sService) setPodsListPodsToRedis(clusterID uint64, namespace string, labelSelector string, pods []*corev1.Pod) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 || clusterID == 0 {
		return
	}
	key := s.podsListCacheKey(clusterID, namespace, labelSelector)
	arr := make([]corev1.Pod, 0, len(pods))
	for _, p := range pods {
		if p != nil {
			arr = append(arr, *p)
		}
	}
	b, err := json.Marshal(arr)
	if err != nil || b == nil {
		return
	}
	ttl := s.podTTL
	wctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_ = s.cache.Set(wctx, key, b, ttl)
	cancel()
}

// ---------------------------------------------------------------------------
// Pod informer / filter / sort / convert helpers
// ---------------------------------------------------------------------------

func podsFromInformerStore(informer cache.SharedIndexInformer) []*corev1.Pod {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*corev1.Pod, 0, len(objs))
	for _, obj := range objs {
		p, ok := obj.(*corev1.Pod)
		if !ok || p == nil {
			continue
		}
		out = append(out, p.DeepCopy())
	}
	return out
}

func filterPodsFromInformer(informer cache.SharedIndexInformer, namespace string, selector labels.Selector) []*corev1.Pod {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*corev1.Pod, 0, len(objs))
	for _, obj := range objs {
		p, ok := obj.(*corev1.Pod)
		if !ok || p == nil {
			continue
		}
		if namespace != "" && p.Namespace != namespace {
			continue
		}
		if selector != nil && !selector.Matches(labels.Set(p.Labels)) {
			continue
		}
		out = append(out, p.DeepCopy())
	}
	return out
}

func podsToAnyList(pods []*corev1.Pod) []any {
	out := make([]any, 0, len(pods))
	for _, p := range pods {
		if p == nil {
			continue
		}
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(p)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func eventsToAnyList(events []*corev1.Event) []any {
	out := make([]any, 0, len(events))
	for _, ev := range events {
		if ev == nil {
			continue
		}
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ev)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func namespacePodReadyCount(pod *corev1.Pod) (int, int) {
	if pod == nil {
		return 0, 0
	}
	total := len(pod.Status.ContainerStatuses)
	ready := 0
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			ready++
		}
	}
	if total == 0 {
		total = len(pod.Spec.Containers)
	}
	return ready, total
}

func namespacePodRestartCount(pod *corev1.Pod) int {
	if pod == nil {
		return 0
	}
	total := 0
	for _, status := range pod.Status.ContainerStatuses {
		total += int(status.RestartCount)
	}
	return total
}

func namespacePodReason(pod *corev1.Pod) string {
	if pod == nil {
		return ""
	}
	if strings.TrimSpace(pod.Status.Reason) != "" {
		return strings.TrimSpace(pod.Status.Reason)
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil && strings.TrimSpace(status.State.Waiting.Reason) != "" {
			return strings.TrimSpace(status.State.Waiting.Reason)
		}
		if status.State.Terminated != nil && strings.TrimSpace(status.State.Terminated.Reason) != "" {
			return strings.TrimSpace(status.State.Terminated.Reason)
		}
	}
	return strings.TrimSpace(string(pod.Status.Phase))
}

func namespacePodIsAbnormal(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}
	if pod.DeletionTimestamp != nil {
		return false
	}
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodPending {
		return true
	}
	ready, total := namespacePodReadyCount(pod)
	if total > 0 && ready < total {
		return true
	}
	return namespacePodRestartCount(pod) > 0
}

func namespacePodHealthItem(pod *corev1.Pod) NamespacePodHealthItem {
	ready, total := namespacePodReadyCount(pod)
	return NamespacePodHealthItem{
		Name:         pod.Name,
		Namespace:    pod.Namespace,
		Phase:        string(pod.Status.Phase),
		Ready:        fmt.Sprintf("%d/%d", ready, total),
		RestartCount: namespacePodRestartCount(pod),
		Reason:       namespacePodReason(pod),
	}
}

func namespaceEventLastTimestamp(ev *corev1.Event) time.Time {
	if ev == nil {
		return time.Time{}
	}
	if !ev.LastTimestamp.IsZero() {
		return ev.LastTimestamp.Time
	}
	if !ev.EventTime.IsZero() {
		return ev.EventTime.Time
	}
	if !ev.CreationTimestamp.IsZero() {
		return ev.CreationTimestamp.Time
	}
	return time.Time{}
}

func namespaceEventItem(ev *corev1.Event) NamespaceEventItem {
	lastAt := namespaceEventLastTimestamp(ev)
	return NamespaceEventItem{
		Type:          strings.TrimSpace(ev.Type),
		Reason:        strings.TrimSpace(ev.Reason),
		Message:       strings.TrimSpace(ev.Message),
		InvolvedKind:  strings.TrimSpace(ev.InvolvedObject.Kind),
		InvolvedName:  strings.TrimSpace(ev.InvolvedObject.Name),
		Count:         ev.Count,
		LastTimestamp: lastAt.UTC().Format(time.RFC3339),
	}
}

func sortNamespaceEvents(events []*corev1.Event) {
	sort.SliceStable(events, func(i, j int) bool {
		left := namespaceEventLastTimestamp(events[i])
		right := namespaceEventLastTimestamp(events[j])
		if left.Equal(right) {
			if events[i] == nil || events[j] == nil {
				return i < j
			}
			return events[i].Name > events[j].Name
		}
		return left.After(right)
	})
}

func (s *K8sService) GetNamespaceHealth(ctx context.Context, clusterID uint64, namespace string) (map[string]any, error) {
	ns := strings.TrimSpace(namespace)
	if clusterID == 0 || ns == "" {
		return nil, ErrInvalidParams
	}
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	pods, err := s.listPodsDirectPods(ctx, clusterID, ns, "")
	if err != nil {
		return nil, err
	}

	podItems := make([]NamespacePodHealthItem, 0, minInt(len(pods), 20))
	counts := map[string]int{
		"total":     len(pods),
		"running":   0,
		"pending":   0,
		"failed":    0,
		"succeeded": 0,
		"abnormal":  0,
	}
	totalRestarts := 0
	for _, pod := range pods {
		if pod == nil {
			continue
		}
		switch pod.Status.Phase {
		case corev1.PodRunning:
			counts["running"]++
		case corev1.PodPending:
			counts["pending"]++
		case corev1.PodFailed:
			counts["failed"]++
		case corev1.PodSucceeded:
			counts["succeeded"]++
		}
		restarts := namespacePodRestartCount(pod)
		totalRestarts += restarts
		if namespacePodIsAbnormal(pod) {
			counts["abnormal"]++
			if len(podItems) < 20 {
				podItems = append(podItems, namespacePodHealthItem(pod))
			}
		}
	}

	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	events := make([]*corev1.Event, 0, 128)
	for {
		el, err := cs.CoreV1().Events(ns).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range el.Items {
			item := el.Items[i].DeepCopy()
			if strings.EqualFold(strings.TrimSpace(item.Type), corev1.EventTypeWarning) {
				events = append(events, item)
			}
		}
		token := strings.TrimSpace(el.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	sortNamespaceEvents(events)
	eventItems := make([]NamespaceEventItem, 0, minInt(len(events), 20))
	for _, ev := range events {
		if ev == nil {
			continue
		}
		if len(eventItems) >= 20 {
			break
		}
		eventItems = append(eventItems, namespaceEventItem(ev))
	}

	return map[string]any{
		"namespace":           ns,
		"pod_counts":          counts,
		"total_restarts":      totalRestarts,
		"abnormal_pods":       podItems,
		"warning_event_count": len(events),
		"warning_events":      eventItems,
	}, nil
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func sortPods(pods []*corev1.Pod, sortBy, order string) {
	sb := strings.TrimSpace(sortBy)
	if sb == "" {
		return
	}
	desc := strings.ToLower(strings.TrimSpace(order)) == "desc"
	if sb == "metadata.name" {
		sort.SliceStable(pods, func(i, j int) bool {
			if desc {
				return pods[i].Name > pods[j].Name
			}
			return pods[i].Name < pods[j].Name
		})
		return
	}
	if sb == "metadata.namespace" {
		sort.SliceStable(pods, func(i, j int) bool {
			if desc {
				return pods[i].Namespace > pods[j].Namespace
			}
			return pods[i].Namespace < pods[j].Namespace
		})
	}
}

// ---------------------------------------------------------------------------
// PodLogs / PodExec
// ---------------------------------------------------------------------------

// PodLogStream 获取 Pod 日志流。
// follow=true 时返回持续输出的流；tailLines=0 表示全量日志；tailLines<0 时回退到最近 200 行。
// previous=true 时返回上一个容器实例的日志。
func (s *K8sService) PodLogStream(ctx context.Context, clusterID uint64, namespace, pod string, container string, follow bool, tailLines int64, previous bool) (io.ReadCloser, error) {
	cfg, err := s.restConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	streamCfg := rest.CopyConfig(cfg)
	streamCfg.Timeout = 0
	cs, err := kubernetes.NewForConfig(streamCfg)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(namespace)
	name := strings.TrimSpace(pod)
	if ns == "" || name == "" {
		return nil, ErrInvalidParams
	}
	stream, err := cs.CoreV1().Pods(ns).GetLogs(name, buildPodLogOptions(container, follow, tailLines, previous)).Stream(ctx)
	if err != nil {
		return nil, normalizeK8sErr(err)
	}
	return stream, nil
}

// PodLogs 获取 Pod 日志。
func (s *K8sService) PodLogs(ctx context.Context, clusterID uint64, namespace, pod string, container string, tailLines int64, previous bool) (string, error) {
	stream, err := s.PodLogStream(ctx, clusterID, namespace, pod, container, false, tailLines, previous)
	if err != nil {
		return "", err
	}
	defer func() { _ = stream.Close() }()
	raw, err := io.ReadAll(stream)
	if err != nil {
		return "", normalizeK8sErr(err)
	}
	return string(raw), nil
}

func buildPodLogOptions(container string, follow bool, tailLines int64, previous bool) *corev1.PodLogOptions {
	options := &corev1.PodLogOptions{
		Container: strings.TrimSpace(container),
		Follow:    follow,
		Previous:  previous,
	}
	if tailLines < 0 {
		tailLines = 200
	}
	if tailLines > 0 {
		options.TailLines = &tailLines
	}
	return options
}

func buildPodExecOptions(container *string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) *corev1.PodExecOptions {
	if len(command) == 0 {
		command = []string{"/bin/sh"}
	}
	options := &corev1.PodExecOptions{
		Container: "",
		Command:   command,
		Stdin:     stdin != nil,
		Stdout:    stdout != nil,
		Stderr:    !tty && stderr != nil,
		TTY:       tty,
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
		value := strings.TrimSpace(item)
		if value != "" {
			trimmed = append(trimmed, value)
		}
	}
	if len(trimmed) == 0 {
		trimmed = []string{"/bin/sh"}
	}
	if len(trimmed) != 1 {
		return [][]string{trimmed}
	}

	first := trimmed[0]
	commonShells := []string{first, "/bin/sh", "sh", "/bin/bash", "bash", "/bin/ash", "ash"}
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
		return normalizeK8sErr(err)
	}
	if strings.Contains(lower, "pod does not exist") {
		return ErrWithMessage(ErrNotFound, "目标 Pod 不存在或已被重新调度，请刷新列表后重试")
	}
	if strings.Contains(lower, "connection refused") && strings.Contains(lower, ":10250") {
		nodeName := ""
		if pod != nil {
			nodeName = strings.TrimSpace(pod.Spec.NodeName)
		}
		addr := ""
		if host, port, splitErr := net.SplitHostPort(extractExecBackendAddress(err.Error())); splitErr == nil {
			addr = net.JoinHostPort(host, port)
		} else {
			addr = extractExecBackendAddress(err.Error())
		}
		if nodeName != "" && addr != "" {
			return ErrWithMessage(ErrK8sNetwork, fmt.Sprintf("目标节点 %s 的 kubelet(%s) 不可达，无法建立 PodShell，请检查节点网络、kubelet 进程和 10250 端口", nodeName, addr))
		}
		if nodeName != "" {
			return ErrWithMessage(ErrK8sNetwork, fmt.Sprintf("目标节点 %s 的 kubelet 不可达，无法建立 PodShell，请检查节点网络、kubelet 进程和 10250 端口", nodeName))
		}
		return ErrWithMessage(ErrK8sNetwork, "目标节点的 kubelet 不可达，无法建立 PodShell，请检查节点网络、kubelet 进程和 10250 端口")
	}
	if strings.Contains(lower, "executable file not found") || strings.Contains(lower, "no such file or directory") {
		cmd := "sh"
		if len(command) > 0 && strings.TrimSpace(command[0]) != "" {
			cmd = strings.TrimSpace(command[0])
		}
		return ErrWithMessage(ErrInvalidParams, fmt.Sprintf("容器内不存在命令 %s，请改用 /bin/sh、bash 或镜像实际提供的 shell", cmd))
	}
	return normalizeK8sErr(err)
}

func extractExecBackendAddress(message string) string {
	text := strings.TrimSpace(message)
	if text == "" {
		return ""
	}
	marker := "dial tcp "
	idx := strings.Index(strings.ToLower(text), marker)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(text[idx+len(marker):])
	if rest == "" {
		return ""
	}
	if end := strings.Index(rest, ": connect"); end >= 0 {
		return strings.TrimSpace(rest[:end])
	}
	return rest
}

// PodExec 在指定 Pod/Container 中执行命令，并建立 SPDY stream 进行交互。
func (s *K8sService) PodExec(
	ctx context.Context,
	clusterID uint64,
	namespace string,
	pod string,
	container *string,
	command []string,
	tty bool,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	resizeQueue remotecommand.TerminalSizeQueue,
) error {
	ns := strings.TrimSpace(namespace)
	name := strings.TrimSpace(pod)
	if ns == "" || name == "" {
		return ErrInvalidParams
	}
	if len(command) == 0 {
		command = []string{"/bin/sh"}
	}

	cfg, err := s.restConfig(ctx, clusterID)
	if err != nil {
		return err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	podObj, getErr := cs.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if getErr != nil {
		return normalizeK8sErr(getErr)
	}

	streamCfg := rest.CopyConfig(cfg)
	streamCfg.Timeout = 0
	streamStderr := stderr
	if tty {
		streamStderr = nil
	}

	candidates := podExecCommandCandidates(command)
	var lastErr error
	for idx, candidate := range candidates {
		opts := buildPodExecOptions(container, candidate, tty, stdin, stdout, stderr)

		req := cs.CoreV1().RESTClient().
			Post().
			Resource("pods").
			Name(name).
			Namespace(ns).
			SubResource("exec")

		req.VersionedParams(opts, scheme.ParameterCodec)
		exec, execErr := remotecommand.NewSPDYExecutor(streamCfg, "POST", req.URL())
		if execErr != nil {
			return normalizePodExecError(execErr, podObj, candidate)
		}

		execErr = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:             stdin,
			Stdout:            stdout,
			Stderr:            streamStderr,
			Tty:               tty,
			TerminalSizeQueue: resizeQueue,
		})
		if execErr == nil {
			return nil
		}

		lastErr = normalizePodExecError(execErr, podObj, candidate)
		if idx < len(candidates)-1 && isExecCommandNotFoundError(execErr) {
			continue
		}
		return lastErr
	}
	if lastErr != nil {
		return lastErr
	}
	return nil
}
