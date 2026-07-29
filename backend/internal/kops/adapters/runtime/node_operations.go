package runtime

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

const nodeOperationListPageLimit int64 = 500

// NodeOperations owns the Kubernetes API behaviour specific to node
// lifecycle, node-scoped inspection and namespace creation. Its client
// transport is injected at composition so this bounded adapter has no legacy
// service dependency.
type NodeOperations struct{ transport NodeOperationsTransport }

type NodeOperationsTransport interface {
	TypedClient(context.Context, uint64) (*kubernetes.Clientset, error)
	DynamicClient(context.Context, uint64) (*dynamic.DynamicClient, error)
}

func NewNodeOperations(transport NodeOperationsTransport) *NodeOperations {
	return &NodeOperations{transport: transport}
}

type NodeDrainOptions struct {
	TimeoutSeconds     int
	Force              bool
	IgnoreDaemonSets   bool
	PollInterval       time.Duration
	EvictionBackoff    time.Duration
	EvictionMaxBackoff time.Duration
}

func (o *NodeOperations) SetSchedulable(ctx context.Context, clusterID uint64, nodeName string, unschedulable bool) error {
	client, err := o.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(nodeName)
	if name == "" {
		return kopsapp.ErrInvalidParams
	}
	patch := []byte(fmt.Sprintf(`{"spec":{"unschedulable":%v}}`, unschedulable))
	_, err = client.CoreV1().Nodes().Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	return nodeOperationError(err)
}

func (o *NodeOperations) ListPods(ctx context.Context, clusterID uint64, nodeName, sortBy, order string) ([]any, error) {
	client, err := o.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(nodeName)
	if name == "" {
		return nil, kopsapp.ErrInvalidParams
	}

	options := metav1.ListOptions{
		Limit:         nodeOperationListPageLimit,
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", name).String(),
	}
	pods := make([]*corev1.Pod, 0, 256)
	for {
		list, listErr := client.CoreV1().Pods(metav1.NamespaceAll).List(ctx, options)
		if listErr != nil {
			return nil, nodeOperationError(listErr)
		}
		for index := range list.Items {
			pods = append(pods, list.Items[index].DeepCopy())
		}
		if strings.TrimSpace(list.Continue) == "" {
			break
		}
		options.Continue = list.Continue
	}
	kopsapp.SortObjectsByMetadata(pods, sortBy, order)
	return nodePodsToAnyList(pods), nil
}

func (o *NodeOperations) ListEvents(ctx context.Context, clusterID uint64, nodeName string) ([]any, error) {
	client, err := o.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(nodeName)
	if name == "" {
		return nil, kopsapp.ErrInvalidParams
	}

	node, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, nodeOperationError(err)
	}
	selectors := []fields.Selector{
		fields.OneTermEqualSelector("involvedObject.kind", "Node"),
		fields.OneTermEqualSelector("involvedObject.name", name),
	}
	if uid := strings.TrimSpace(string(node.UID)); uid != "" {
		selectors = append(selectors, fields.OneTermEqualSelector("involvedObject.uid", uid))
	}
	options := metav1.ListOptions{
		Limit:         nodeOperationListPageLimit,
		FieldSelector: fields.AndSelectors(selectors...).String(),
	}
	events := make([]*corev1.Event, 0, 128)
	for {
		list, listErr := client.CoreV1().Events(metav1.NamespaceAll).List(ctx, options)
		if listErr != nil {
			return nil, nodeOperationError(listErr)
		}
		for index := range list.Items {
			events = append(events, list.Items[index].DeepCopy())
		}
		if strings.TrimSpace(list.Continue) == "" {
			break
		}
		options.Continue = list.Continue
	}
	return nodeEventsToAnyList(events), nil
}

func (o *NodeOperations) Drain(ctx context.Context, clusterID uint64, nodeName string, opts NodeDrainOptions) error {
	client, err := o.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(nodeName)
	if name == "" {
		return kopsapp.ErrInvalidParams
	}

	timeout := opts.TimeoutSeconds
	if timeout <= 0 {
		timeout = 600
	}
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	evictionBackoff := opts.EvictionBackoff
	if evictionBackoff <= 0 {
		evictionBackoff = 2 * time.Second
	}
	evictionMaxBackoff := opts.EvictionMaxBackoff
	if evictionMaxBackoff <= 0 {
		evictionMaxBackoff = 15 * time.Second
	}
	ignoreDaemonSets := opts.IgnoreDaemonSets
	if !opts.Force {
		ignoreDaemonSets = true
	}

	drainContext, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	if err := o.SetSchedulable(drainContext, clusterID, name, true); err != nil {
		return err
	}
	pods, err := client.CoreV1().Pods(metav1.NamespaceAll).List(drainContext, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", name).String(),
	})
	if err != nil {
		return nodeOperationError(err)
	}

	for index := range pods.Items {
		pod := &pods.Items[index]
		if nodeDrainSkipsPod(pod, ignoreDaemonSets) {
			continue
		}
		if err := evictAndWaitForNodePod(drainContext, client, pod, opts.Force, pollInterval, evictionBackoff, evictionMaxBackoff); err != nil {
			return err
		}
	}
	return nil
}

func (o *NodeOperations) CreateNamespace(ctx context.Context, clusterID uint64, name string, labels map[string]string) error {
	namespace := strings.TrimSpace(name)
	if namespace == "" {
		return kopsapp.ErrInvalidParams
	}
	if o == nil || o.transport == nil {
		return kopsapp.ErrConflict
	}
	client, err := o.transport.DynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	object := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]any{
			"name":   namespace,
			"labels": labels,
		},
	}}
	_, err = client.Resource(schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}).Create(ctx, object, metav1.CreateOptions{})
	return nodeOperationError(err)
}

func (o *NodeOperations) CheckHealth(ctx context.Context, clusterID uint64) (bool, int, int, string, error) {
	client, err := o.typedClient(ctx, clusterID)
	if err != nil {
		return false, 0, 0, "", err
	}
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return false, 0, 0, "", nodeOperationError(err)
	}
	kubernetesVersion := ""
	if version != nil {
		kubernetesVersion = version.GitVersion
	}
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return true, 0, 0, kubernetesVersion, nodeOperationError(err)
	}
	ready := 0
	for index := range nodes.Items {
		if nodeIsReady(&nodes.Items[index]) {
			ready++
		}
	}
	return true, ready, len(nodes.Items), kubernetesVersion, nil
}

func (o *NodeOperations) typedClient(ctx context.Context, clusterID uint64) (*kubernetes.Clientset, error) {
	if o == nil || o.transport == nil {
		return nil, kopsapp.ErrConflict
	}
	return o.transport.TypedClient(ctx, clusterID)
}

func evictAndWaitForNodePod(ctx context.Context, client *kubernetes.Clientset, pod *corev1.Pod, force bool, pollInterval, backoff, maxBackoff time.Duration) error {
	for {
		err := evictNodePod(ctx, client, pod.Namespace, pod.Name)
		if err == nil || apierrors.IsNotFound(err) {
			break
		}
		if apierrors.IsTooManyRequests(err) {
			select {
			case <-ctx.Done():
				return nodeOperationError(ctx.Err())
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		if !force {
			return nodeOperationError(err)
		}
		gracePeriod := int64(0)
		deleteErr := client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
			return nodeOperationError(deleteErr)
		}
		break
	}

	for {
		_, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return nodeOperationError(err)
		}
		select {
		case <-ctx.Done():
			return nodeOperationError(ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

func evictNodePod(ctx context.Context, client *kubernetes.Clientset, namespace, name string) error {
	if client == nil {
		return kopsapp.ErrRuntime
	}
	err := client.CoreV1().Pods(namespace).EvictV1(ctx, &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
	if err == nil || !nodeEvictionResourceMissing(err) {
		return err
	}
	return client.CoreV1().Pods(namespace).EvictV1beta1(ctx, &policyv1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
}

func nodeEvictionResourceMissing(err error) bool {
	if err == nil {
		return false
	}
	if apierrors.IsNotFound(err) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "the server could not find the requested resource") ||
		strings.Contains(lower, "could not find the requested resource") ||
		strings.Contains(lower, "the server doesn't have a resource type") ||
		strings.Contains(lower, "unable to recognize") ||
		strings.Contains(lower, "no matches for kind")
}

func nodeDrainSkipsPod(pod *corev1.Pod, ignoreDaemonSets bool) bool {
	if pod == nil || pod.DeletionTimestamp != nil || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return true
	}
	if _, mirror := pod.Annotations[corev1.MirrorPodAnnotationKey]; mirror {
		return true
	}
	owner := metav1.GetControllerOf(pod)
	return ignoreDaemonSets && owner != nil && owner.Kind == "DaemonSet"
}

func nodePodsToAnyList(pods []*corev1.Pod) []any {
	values := make([]any, 0, len(pods))
	for _, pod := range pods {
		if pod == nil {
			continue
		}
		value, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
		if err == nil {
			values = append(values, value)
		}
	}
	return values
}

func nodeEventsToAnyList(events []*corev1.Event) []any {
	values := make([]any, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		value, err := runtime.DefaultUnstructuredConverter.ToUnstructured(event)
		if err == nil {
			values = append(values, value)
		}
	}
	return values
}

func nodeIsReady(node *corev1.Node) bool {
	if node == nil {
		return false
	}
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func nodeOperationError(err error) error {
	if err == nil {
		return err
	}
	for _, known := range []error{
		kopsapp.ErrInvalidParams, kopsapp.ErrNotFound, kopsapp.ErrConflict,
		kopsapp.ErrRuntime, kopsapp.ErrRuntimeNetwork, kopsapp.ErrRuntimeTimeout,
		kopsapp.ErrRuntimeUnauthorized, kopsapp.ErrRuntimeForbidden, kopsapp.ErrRuntimeTLS,
	} {
		if errors.Is(err, known) {
			return err
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return kopsapp.ErrRuntimeTimeout
	}
	switch {
	case apierrors.IsNotFound(err):
		return kopsapp.ErrNotFound
	case apierrors.IsAlreadyExists(err):
		return kopsapp.ErrConflict
	case apierrors.IsInvalid(err), apierrors.IsBadRequest(err):
		return kopsapp.ErrInvalidParams
	case apierrors.IsUnauthorized(err):
		return kopsapp.ErrRuntimeUnauthorized
	case apierrors.IsForbidden(err):
		return kopsapp.ErrRuntimeForbidden
	case apierrors.IsTimeout(err):
		return kopsapp.ErrRuntimeTimeout
	}
	var requestError *url.Error
	if errors.As(err, &requestError) && requestError != nil && requestError.Unwrap() != nil {
		err = requestError.Unwrap()
	}
	var unknownAuthority *x509.UnknownAuthorityError
	var hostnameError x509.HostnameError
	var invalidCertificate x509.CertificateInvalidError
	lower := strings.ToLower(err.Error())
	if errors.As(err, &unknownAuthority) || errors.As(err, &hostnameError) || errors.As(err, &invalidCertificate) || strings.Contains(lower, "x509:") {
		return kopsapp.ErrRuntimeTLS
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		if networkError.Timeout() {
			return kopsapp.ErrRuntimeTimeout
		}
		return kopsapp.ErrRuntimeNetwork
	}
	if strings.Contains(lower, "connection refused") || strings.Contains(lower, "no route to host") || strings.Contains(lower, "i/o timeout") || strings.Contains(lower, "timeout") {
		return kopsapp.ErrRuntimeNetwork
	}
	return kopsapp.ErrRuntime
}
