package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// ---------------------------------------------------------------------------
// Node operations
// ---------------------------------------------------------------------------

// UpdateNodeSchedulable 更新节点调度状态（Cordon/Uncordon）。
func (s *K8sService) UpdateNodeSchedulable(ctx context.Context, clusterID uint64, nodeName string, unschedulable bool) error {
	client, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}
	patchData := []byte(fmt.Sprintf(`{"spec":{"unschedulable":%v}}`, unschedulable))
	_, err = client.CoreV1().Nodes().Patch(ctx, nodeName, types.MergePatchType, patchData, metav1.PatchOptions{})
	return normalizeK8sErr(err)
}

func (s *K8sService) ListPodsOnNode(ctx context.Context, clusterID uint64, nodeName string, sortBy, order string) ([]any, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(nodeName)
	if name == "" {
		return nil, ErrInvalidParams
	}

	opts := metav1.ListOptions{
		Limit:         k8sListPageLimit,
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", name).String(),
	}
	pods := make([]*corev1.Pod, 0, 256)
	for {
		pl, err := cs.CoreV1().Pods(metav1.NamespaceAll).List(ctx, opts)
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
	sortPods(pods, sortBy, order)
	return podsToAnyList(pods), nil
}

func (s *K8sService) ListNodeEvents(ctx context.Context, clusterID uint64, nodeName string) ([]any, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(nodeName)
	if name == "" {
		return nil, ErrInvalidParams
	}

	n, err := cs.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, normalizeK8sErr(err)
	}
	uid := strings.TrimSpace(string(n.UID))

	sels := []fields.Selector{
		fields.OneTermEqualSelector("involvedObject.kind", "Node"),
		fields.OneTermEqualSelector("involvedObject.name", name),
	}
	if uid != "" {
		sels = append(sels, fields.OneTermEqualSelector("involvedObject.uid", uid))
	}
	fieldSelector := fields.AndSelectors(sels...).String()

	opts := metav1.ListOptions{Limit: k8sListPageLimit, FieldSelector: fieldSelector}
	events := make([]*corev1.Event, 0, 128)
	for {
		el, err := cs.CoreV1().Events(metav1.NamespaceAll).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range el.Items {
			events = append(events, el.Items[i].DeepCopy())
		}
		token := strings.TrimSpace(el.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	return eventsToAnyList(events), nil
}

func (s *K8sService) evictPodCompatible(ctx context.Context, clusterID uint64, cs *kubernetes.Clientset, namespace, name string) error {
	if cs == nil {
		return ErrK8s
	}
	v1Eviction := &policyv1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := cs.CoreV1().Pods(namespace).EvictV1(ctx, v1Eviction)
	if err == nil || !isMissingAPIResourceErr(err) {
		return err
	}

	v1beta1Eviction := &policyv1beta1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return cs.CoreV1().Pods(namespace).EvictV1beta1(ctx, v1beta1Eviction)
}

// ---------------------------------------------------------------------------
// DrainNode
// ---------------------------------------------------------------------------

type DrainNodeOptions struct {
	TimeoutSeconds     int
	Force              bool
	IgnoreDaemonSets   bool
	PollInterval       time.Duration
	EvictionBackoff    time.Duration
	EvictionMaxBackoff time.Duration
}

func (s *K8sService) DrainNode(ctx context.Context, clusterID uint64, nodeName string, opts DrainNodeOptions) error {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(nodeName)
	if name == "" {
		return ErrInvalidParams
	}

	timeoutSeconds := opts.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 600
	}
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	evictBackoff := opts.EvictionBackoff
	if evictBackoff <= 0 {
		evictBackoff = 2 * time.Second
	}
	evictMaxBackoff := opts.EvictionMaxBackoff
	if evictMaxBackoff <= 0 {
		evictMaxBackoff = 15 * time.Second
	}

	ignoreDaemonSets := opts.IgnoreDaemonSets
	if !opts.Force {
		ignoreDaemonSets = true
	}

	dctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	if err := s.UpdateNodeSchedulable(dctx, clusterID, name, true); err != nil {
		return err
	}

	pods, err := cs.CoreV1().Pods("").List(dctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", name).String(),
	})
	if err != nil {
		return normalizeK8sErr(err)
	}

	isMirrorPod := func(p *corev1.Pod) bool {
		if p == nil || p.Annotations == nil {
			return false
		}
		_, ok := p.Annotations[corev1.MirrorPodAnnotationKey]
		return ok
	}
	isDaemonSetPod := func(p *corev1.Pod) bool {
		if p == nil {
			return false
		}
		ref := metav1.GetControllerOf(p)
		return ref != nil && ref.Kind == "DaemonSet"
	}
	isTerminal := func(p *corev1.Pod) bool {
		if p == nil {
			return true
		}
		return p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed
	}

	for i := range pods.Items {
		p := &pods.Items[i]
		if p.DeletionTimestamp != nil || isTerminal(p) || isMirrorPod(p) {
			continue
		}
		if isDaemonSetPod(p) && ignoreDaemonSets {
			continue
		}

		backoff := evictBackoff
		for {
			err := s.evictPodCompatible(dctx, clusterID, cs, p.Namespace, p.Name)
			if err == nil || apierrors.IsNotFound(err) {
				break
			}
			if apierrors.IsTooManyRequests(err) {
				select {
				case <-dctx.Done():
					return normalizeK8sErr(dctx.Err())
				case <-time.After(backoff):
				}
				backoff *= 2
				if backoff > evictMaxBackoff {
					backoff = evictMaxBackoff
				}
				continue
			}
			if opts.Force {
				grace := int64(0)
				derr := cs.CoreV1().Pods(p.Namespace).Delete(dctx, p.Name, metav1.DeleteOptions{GracePeriodSeconds: &grace})
				if derr == nil || apierrors.IsNotFound(derr) {
					break
				}
				return normalizeK8sErr(derr)
			}
			return normalizeK8sErr(err)
		}

		for {
			_, err := cs.CoreV1().Pods(p.Namespace).Get(dctx, p.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				break
			}
			if err != nil {
				return normalizeK8sErr(err)
			}
			select {
			case <-dctx.Done():
				return normalizeK8sErr(dctx.Err())
			case <-time.After(pollInterval):
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// CreateNamespace / CheckHealth
// ---------------------------------------------------------------------------

func (s *K8sService) CreateNamespace(ctx context.Context, clusterID uint64, name string, labels map[string]string) error {
	dc, err := s.dynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	n := strings.TrimSpace(name)
	if n == "" {
		return ErrInvalidParams
	}
	obj := map[string]any{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]any{
			"name":   n,
			"labels": labels,
		},
	}
	u := &unstructured.Unstructured{Object: obj}
	_, err = dc.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(ctx, u, metav1.CreateOptions{})
	return normalizeK8sErr(err)
}

// CheckHealth 检查集群基本可用性，并统计 Node Ready 数量。
func (s *K8sService) CheckHealth(ctx context.Context, clusterID uint64) (apiOK bool, nodeReady int, nodeTotal int, k8sVersion string, err error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return false, 0, 0, "", err
	}
	verInfo, err := cs.Discovery().ServerVersion()
	if err != nil {
		return false, 0, 0, "", normalizeK8sErr(err)
	}
	k8sVersion = verInfo.GitVersion

	nodes, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return true, 0, 0, k8sVersion, normalizeK8sErr(err)
	}
	total := len(nodes.Items)
	ready := 0
	for i := range nodes.Items {
		if isNodeReady(&nodes.Items[i]) {
			ready++
		}
	}
	return true, ready, total, k8sVersion, nil
}

func isNodeReady(n *corev1.Node) bool {
	if n == nil {
		return false
	}
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
