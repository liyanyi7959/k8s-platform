package service

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func TestAppendResourceTrendSampleKeepsOnlyReal24HourWindow(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	samples := []resourceTrendSample{
		{Timestamp: now.Add(-25 * time.Hour), CPU: 99, Memory: 99},
		{Timestamp: now.Add(-time.Hour), CPU: 12.3, Memory: 45.6},
	}
	got := appendResourceTrendSample(samples, resourceTrendSample{Timestamp: now, CPU: 13.4, Memory: 46.7})
	if len(got) != 2 || got[0].CPU != 12.3 || got[1].Memory != 46.7 {
		t.Fatalf("unexpected trend samples: %#v", got)
	}
}

func TestAppendResourceTrendSampleReplacesSameMinute(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 30, 0, time.UTC)
	samples := []resourceTrendSample{{Timestamp: now.Add(-30 * time.Second), CPU: 10, Memory: 20}}
	got := appendResourceTrendSample(samples, resourceTrendSample{Timestamp: now, CPU: 11, Memory: 21})
	if len(got) != 1 || got[0].CPU != 11 || got[0].Memory != 21 {
		t.Fatalf("unexpected trend samples: %#v", got)
	}
}

func TestRecordResourceTrendSeparatesClusterAndNodeScopes(t *testing.T) {
	svc := &DashboardService{trendCache: make(map[string][]resourceTrendSample)}
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	svc.recordResourceTrend(context.Background(), 3, "cluster", resourceTrendSample{Timestamp: now, CPU: 20, Memory: 30})
	svc.recordResourceTrend(context.Background(), 3, "node:worker-1", resourceTrendSample{Timestamp: now, CPU: 40, Memory: 50})
	if svc.trendCache["3:cluster"][0].CPU != 20 || svc.trendCache["3:node:worker-1"][0].CPU != 40 {
		t.Fatalf("trend scopes were mixed: %#v", svc.trendCache)
	}
}

func TestCalculateClusterUsagePercentUsesOnlySampledNodes(t *testing.T) {
	nodes := []corev1.Node{
		{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "10.0.0.11"}}, Allocatable: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("4Gi")}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "node-b"}, Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("4Gi")}}},
	}
	metrics := []any{map[string]any{
		"metadata": map[string]any{"name": "node-a"},
		"usage":    map[string]any{"cpu": "1", "memory": "2Gi"},
	}}
	usage, available := calculateClusterUsageSnapshot(nodes, metrics)
	if !available || usage.CPU != 50 || usage.Memory != 50 || len(usage.Nodes) != 1 || usage.Nodes[0].IP != "10.0.0.11" {
		t.Fatalf("usage = (%+v, %v), want aggregate 50/50 and one node", usage, available)
	}
}

func TestUnhealthyPodReasonDetectsRunningCrashLoop(t *testing.T) {
	pod := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}}}}}
	if got := unhealthyPodReason(pod); got != "CrashLoopBackOff" {
		t.Fatalf("reason = %q", got)
	}
}
