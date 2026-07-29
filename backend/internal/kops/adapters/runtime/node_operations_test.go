package runtime

import (
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

func TestNodeOperationsConvertTypedResourcesToObjects(t *testing.T) {
	pods := nodePodsToAnyList([]*corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "production"}}})
	if len(pods) != 1 {
		t.Fatalf("pod count = %d, want 1", len(pods))
	}
	pod, ok := pods[0].(map[string]any)
	if !ok {
		t.Fatalf("pod type = %T, want map[string]any", pods[0])
	}
	metadata, ok := pod["metadata"].(map[string]any)
	if !ok || metadata["name"] != "api" || metadata["namespace"] != "production" {
		t.Fatalf("pod metadata = %#v", pod["metadata"])
	}

	events := nodeEventsToAnyList([]*corev1.Event{{ObjectMeta: metav1.ObjectMeta{Name: "node-ready"}}})
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
}

func TestNodeDrainSkipsNonEvictablePods(t *testing.T) {
	if !nodeDrainSkipsPod(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{corev1.MirrorPodAnnotationKey: "mirror"}}}, false) {
		t.Fatal("mirror pod must be skipped")
	}
	daemonSet := true
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{{Kind: "DaemonSet", Controller: &daemonSet}}}}
	if !nodeDrainSkipsPod(pod, true) {
		t.Fatal("daemon set pod must be skipped when requested")
	}
	if nodeDrainSkipsPod(pod, false) {
		t.Fatal("daemon set pod must be evictable when daemon sets are not ignored")
	}
}

func TestNodeOperationErrorNormalizesKubernetesErrors(t *testing.T) {
	err := nodeOperationError(apierrors.NewNotFound(schema.GroupResource{Resource: "nodes"}, "worker-01"))
	if !errors.Is(err, kopsapp.ErrNotFound) {
		t.Fatalf("error = %v, want not found", err)
	}
}
