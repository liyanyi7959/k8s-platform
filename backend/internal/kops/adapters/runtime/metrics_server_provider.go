package runtime

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

type MetricsServerTransport interface {
	TypedClient(context.Context, uint64) (*kubernetes.Clientset, error)
	DynamicClient(context.Context, uint64) (*dynamic.DynamicClient, error)
}

type metricsServerProvider struct{ transport MetricsServerTransport }

// NewMetricsServerProvider builds the Kops-owned metrics-server adapter.
// Cross-context provider selection may consume only this application port.
func NewMetricsServerProvider(transport MetricsServerTransport) kopsapp.MetricsProvider {
	return &metricsServerProvider{transport: transport}
}

func (p *metricsServerProvider) Name() string { return string(kopsapp.MonitorSourceMetricsServer) }

func (p *metricsServerProvider) GetNodeMetrics(ctx context.Context, clusterID uint64) ([]kopsapp.NodeMetrics, error) {
	if p == nil || p.transport == nil {
		return nil, kopsapp.ErrConflict
	}
	client, err := p.transport.TypedClient(ctx, clusterID)
	if err != nil {
		return nil, TranslateKopsRuntimeError(err)
	}
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, TranslateKopsRuntimeError(service.NormalizeKubernetesError(err))
	}
	dynamicClient, err := p.transport.DynamicClient(ctx, clusterID)
	if err != nil {
		return nil, TranslateKopsRuntimeError(err)
	}
	metricsGVR := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}
	metricsList, err := dynamicClient.Resource(metricsGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, TranslateKopsRuntimeError(service.NormalizeKubernetesError(err))
	}

	metricsMap := make(map[string]map[string]int64, len(metricsList.Items))
	for _, item := range metricsList.Items {
		usage, _ := item.Object["usage"].(map[string]any)
		if usage == nil {
			continue
		}
		cpu, _ := usage["cpu"].(string)
		memory, _ := usage["memory"].(string)
		metricsMap[item.GetName()] = map[string]int64{"cpu": parseMetricsQuantity(cpu), "memory": parseMetricsQuantity(memory)}
	}

	result := make([]kopsapp.NodeMetrics, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		cpuCapacity := parseMetricsQuantity(node.Status.Capacity.Cpu().String())
		memoryCapacity := parseMetricsQuantity(node.Status.Capacity.Memory().String())
		metric, found := metricsMap[node.Name]
		var cpuUsed, memoryUsed int64
		if found {
			cpuUsed, memoryUsed = metric["cpu"], metric["memory"]
		}
		cpuUsage, memoryUsage := 0.0, 0.0
		if cpuCapacity > 0 {
			cpuUsage = float64(cpuUsed) / float64(cpuCapacity) * 100
		}
		if memoryCapacity > 0 {
			memoryUsage = float64(memoryUsed) / float64(memoryCapacity) * 100
		}
		result = append(result, kopsapp.NodeMetrics{
			Name: node.Name, CPUUsage: cpuUsage, MemoryUsage: memoryUsage,
			CPUCapacity: cpuCapacity, MemoryCapacity: memoryCapacity, CPUUsed: cpuUsed, MemoryUsed: memoryUsed,
		})
	}
	return result, nil
}

func (p *metricsServerProvider) GetPodMetrics(ctx context.Context, clusterID uint64, namespace string) ([]kopsapp.PodMetrics, error) {
	if p == nil || p.transport == nil {
		return nil, kopsapp.ErrConflict
	}
	dynamicClient, err := p.transport.DynamicClient(ctx, clusterID)
	if err != nil {
		return nil, TranslateKopsRuntimeError(err)
	}
	metricsGVR := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
	var resource dynamic.ResourceInterface
	if namespace != "" {
		resource = dynamicClient.Resource(metricsGVR).Namespace(namespace)
	} else {
		resource = dynamicClient.Resource(metricsGVR)
	}
	metricsList, err := resource.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, TranslateKopsRuntimeError(service.NormalizeKubernetesError(err))
	}
	result := make([]kopsapp.PodMetrics, 0, len(metricsList.Items))
	for _, item := range metricsList.Items {
		var cpuTotal, memoryTotal int64
		containers, _ := item.Object["containers"].([]any)
		for _, container := range containers {
			values, _ := container.(map[string]any)
			usage, _ := values["usage"].(map[string]any)
			if usage == nil {
				continue
			}
			cpu, _ := usage["cpu"].(string)
			memory, _ := usage["memory"].(string)
			cpuTotal += parseMetricsQuantity(cpu)
			memoryTotal += parseMetricsQuantity(memory)
		}
		result = append(result, kopsapp.PodMetrics{Name: item.GetName(), Namespace: item.GetNamespace(), CPU: cpuTotal, Memory: memoryTotal})
	}
	return result, nil
}

func (p *metricsServerProvider) GetNodeMetricTrend(context.Context, uint64, string, string, time.Time, time.Time, time.Duration) ([]kopsapp.MetricPoint, error) {
	return nil, fmt.Errorf("metrics-server 不支持历史趋势查询")
}

func (p *metricsServerProvider) GetPodMetricTrend(context.Context, uint64, string, string, string, time.Time, time.Time, time.Duration) ([]kopsapp.MetricPoint, error) {
	return nil, fmt.Errorf("metrics-server 不支持历史趋势查询")
}

func (p *metricsServerProvider) Health(ctx context.Context, clusterID uint64) error {
	if p == nil || p.transport == nil {
		return kopsapp.ErrConflict
	}
	dynamicClient, err := p.transport.DynamicClient(ctx, clusterID)
	if err != nil {
		return TranslateKopsRuntimeError(err)
	}
	metricsGVR := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}
	_, err = dynamicClient.Resource(metricsGVR).List(ctx, metav1.ListOptions{Limit: 1})
	return TranslateKopsRuntimeError(service.NormalizeKubernetesError(err))
}

func parseMetricsQuantity(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if strings.HasSuffix(value, "m") {
		parsed, _ := strconv.ParseInt(value[:len(value)-1], 10, 64)
		return parsed * 1_000_000
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed * 1_000_000_000
	}
	for _, suffix := range []struct {
		value      string
		multiplier int64
	}{{"Ki", 1024}, {"Mi", 1024 * 1024}, {"Gi", 1024 * 1024 * 1024}} {
		if strings.HasSuffix(value, suffix.value) {
			parsed, _ := strconv.ParseInt(value[:len(value)-len(suffix.value)], 10, 64)
			return parsed * suffix.multiplier
		}
	}
	return 0
}
