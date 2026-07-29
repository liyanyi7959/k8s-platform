package runtime

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodOperationsHealthHelpers(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "production"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "api", RestartCount: 2, State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
			}},
		},
	}
	if !podIsAbnormal(pod) {
		t.Fatal("pending pod must be abnormal")
	}
	item := podHealthItem(pod)
	if item["name"] != "api" || item["restart_count"] != 2 || item["reason"] != "ImagePullBackOff" {
		t.Fatalf("unexpected pod health item: %#v", item)
	}
}

func TestNamespaceEventItemUsesNewestEventTime(t *testing.T) {
	created := time.Date(2026, 7, 30, 1, 0, 0, 0, time.UTC)
	last := created.Add(time.Minute)
	event := &corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "warning", CreationTimestamp: metav1.NewTime(created)}, LastTimestamp: metav1.NewTime(last), Type: corev1.EventTypeWarning, Reason: "Failed", Message: "pull failed"}
	item := namespaceEventItem(event)
	if item["last_timestamp"] != last.Format(time.RFC3339) || item["reason"] != "Failed" {
		t.Fatalf("unexpected event item: %#v", item)
	}
}
