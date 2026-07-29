package runtime

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

// PodOperations owns pod deletion and namespace-health aggregation.  It uses
// the same bounded client transport as node operations so Kops policy never
// depends on legacy service implementations.
type PodOperations struct{ transport NodeOperationsTransport }

func NewPodOperations(transport NodeOperationsTransport) *PodOperations {
	return &PodOperations{transport: transport}
}

func (o *PodOperations) Delete(ctx context.Context, clusterID uint64, namespace, name string, force bool) error {
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if namespace == "" || name == "" {
		return kopsapp.ErrInvalidParams
	}
	client, err := o.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}
	options := metav1.DeleteOptions{}
	if force {
		gracePeriod := int64(0)
		options.GracePeriodSeconds = &gracePeriod
	}
	return nodeOperationError(client.CoreV1().Pods(namespace).Delete(ctx, name, options))
}

func (o *PodOperations) NamespaceHealth(ctx context.Context, clusterID uint64, namespace string) (map[string]any, error) {
	namespace = strings.TrimSpace(namespace)
	if clusterID == 0 || namespace == "" {
		return nil, kopsapp.ErrInvalidParams
	}
	client, err := o.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	pods, err := listNamespacePods(ctx, client, namespace)
	if err != nil {
		return nil, err
	}

	abnormalPods := make([]map[string]any, 0, minPodOperationInt(len(pods), 20))
	counts := map[string]int{"total": len(pods), "running": 0, "pending": 0, "failed": 0, "succeeded": 0, "abnormal": 0}
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
		restarts := podRestartCount(pod)
		totalRestarts += restarts
		if podIsAbnormal(pod) {
			counts["abnormal"]++
			if len(abnormalPods) < 20 {
				abnormalPods = append(abnormalPods, podHealthItem(pod))
			}
		}
	}

	events, err := listNamespaceWarningEvents(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	eventItems := make([]map[string]any, 0, minPodOperationInt(len(events), 20))
	for _, event := range events {
		if event != nil && len(eventItems) < 20 {
			eventItems = append(eventItems, namespaceEventItem(event))
		}
	}
	return map[string]any{
		"namespace": namespace, "pod_counts": counts, "total_restarts": totalRestarts,
		"abnormal_pods": abnormalPods, "warning_event_count": len(events), "warning_events": eventItems,
	}, nil
}

func (o *PodOperations) typedClient(ctx context.Context, clusterID uint64) (*kubernetes.Clientset, error) {
	if o == nil || o.transport == nil {
		return nil, kopsapp.ErrRuntime
	}
	return o.transport.TypedClient(ctx, clusterID)
}

func listNamespacePods(ctx context.Context, client *kubernetes.Clientset, namespace string) ([]*corev1.Pod, error) {
	if client == nil {
		return nil, kopsapp.ErrRuntime
	}
	options := metav1.ListOptions{Limit: nodeOperationListPageLimit}
	pods := make([]*corev1.Pod, 0, 128)
	for {
		result, err := client.CoreV1().Pods(namespace).List(ctx, options)
		if err != nil {
			return nil, nodeOperationError(err)
		}
		for index := range result.Items {
			pods = append(pods, result.Items[index].DeepCopy())
		}
		if strings.TrimSpace(result.Continue) == "" {
			return pods, nil
		}
		options.Continue = result.Continue
	}
}

func listNamespaceWarningEvents(ctx context.Context, client *kubernetes.Clientset, namespace string) ([]*corev1.Event, error) {
	if client == nil {
		return nil, kopsapp.ErrRuntime
	}
	options := metav1.ListOptions{Limit: nodeOperationListPageLimit}
	events := make([]*corev1.Event, 0, 64)
	for {
		result, err := client.CoreV1().Events(namespace).List(ctx, options)
		if err != nil {
			return nil, nodeOperationError(err)
		}
		for index := range result.Items {
			event := result.Items[index].DeepCopy()
			if strings.EqualFold(strings.TrimSpace(event.Type), corev1.EventTypeWarning) {
				events = append(events, event)
			}
		}
		if strings.TrimSpace(result.Continue) == "" {
			break
		}
		options.Continue = result.Continue
	}
	sort.SliceStable(events, func(left, right int) bool {
		leftAt, rightAt := namespaceEventTimestamp(events[left]), namespaceEventTimestamp(events[right])
		if leftAt.Equal(rightAt) {
			if events[left] == nil || events[right] == nil {
				return left < right
			}
			return events[left].Name > events[right].Name
		}
		return leftAt.After(rightAt)
	})
	return events, nil
}

func podHealthItem(pod *corev1.Pod) map[string]any {
	ready, total := podReadyCount(pod)
	return map[string]any{
		"name": pod.Name, "namespace": pod.Namespace, "phase": string(pod.Status.Phase),
		"ready": fmt.Sprintf("%d/%d", ready, total), "restart_count": podRestartCount(pod), "reason": podReason(pod),
	}
}

func podReadyCount(pod *corev1.Pod) (int, int) {
	if pod == nil {
		return 0, 0
	}
	total, ready := len(pod.Status.ContainerStatuses), 0
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

func podRestartCount(pod *corev1.Pod) int {
	if pod == nil {
		return 0
	}
	total := 0
	for _, status := range pod.Status.ContainerStatuses {
		total += int(status.RestartCount)
	}
	return total
}

func podReason(pod *corev1.Pod) string {
	if pod == nil {
		return ""
	}
	if reason := strings.TrimSpace(pod.Status.Reason); reason != "" {
		return reason
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

func podIsAbnormal(pod *corev1.Pod) bool {
	if pod == nil || pod.DeletionTimestamp != nil {
		return false
	}
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodPending {
		return true
	}
	ready, total := podReadyCount(pod)
	return total > 0 && ready < total || podRestartCount(pod) > 0
}

func namespaceEventTimestamp(event *corev1.Event) time.Time {
	if event == nil {
		return time.Time{}
	}
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	return event.CreationTimestamp.Time
}

func namespaceEventItem(event *corev1.Event) map[string]any {
	return map[string]any{
		"type": strings.TrimSpace(event.Type), "reason": strings.TrimSpace(event.Reason), "message": strings.TrimSpace(event.Message),
		"involved_kind": strings.TrimSpace(event.InvolvedObject.Kind), "involved_name": strings.TrimSpace(event.InvolvedObject.Name),
		"count": event.Count, "last_timestamp": namespaceEventTimestamp(event).UTC().Format(time.RFC3339),
	}
}

func minPodOperationInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
