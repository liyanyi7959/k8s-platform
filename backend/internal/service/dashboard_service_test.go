package service

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractTopWorkloadsSupportsTypedInformerObjects(t *testing.T) {
	svc := &DashboardService{}
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "prod"}, Status: appsv1.DeploymentStatus{Replicas: 3, ReadyReplicas: 2}}
	daemonSet := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "ops"}, Status: appsv1.DaemonSetStatus{DesiredNumberScheduled: 5, NumberReady: 5}}
	items := svc.extractTopWorkloads([]any{deployment}, nil, []any{daemonSet})
	if len(items) != 2 || items[0]["name"] != "agent" || items[0]["replicas"] != int32(5) || items[1]["ready"] != int32(2) {
		t.Fatalf("unexpected workloads: %#v", items)
	}
}

func TestUnhealthyPodReasonDetectsRunningCrashLoop(t *testing.T) {
	pod := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}}}}}
	if got := unhealthyPodReason(pod); got != "CrashLoopBackOff" {
		t.Fatalf("reason = %q", got)
	}
}
