// metrics_server_provider.go 实现基于 metrics-server 的 MetricsProvider。
package service

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// metricsServerProvider 通过 Kubernetes metrics-server 获取实时指标。
type metricsServerProvider struct {
	svc *K8sService
}

// newMetricsServerProvider 创建 metrics-server Provider。
func newMetricsServerProvider(svc *K8sService) *metricsServerProvider {
	return &metricsServerProvider{svc: svc}
}

func (p *metricsServerProvider) Name() string { return string(MonitorSourceMetricsServer) }

func (p *metricsServerProvider) GetNodeMetrics(ctx context.Context, clusterID uint64) ([]NodeMetrics, error) {
	nodes, err := p.svc.List(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, "", "", "", nil)
	if err != nil {
		return nil, err
	}

	dc, err := p.svc.dynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	metricsGVR := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}
	metricsList, err := dc.Resource(metricsGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, normalizeK8sErr(err)
	}

	metricsMap := map[string]map[string]int64{}
	for _, item := range metricsList.Items {
		usage, _ := item.Object["usage"].(map[string]any)
		if usage == nil {
			continue
		}
		cpuStr, _ := usage["cpu"].(string)
		memStr, _ := usage["memory"].(string)
		metricsMap[item.GetName()] = map[string]int64{
			"cpu":    parseResourceQuantity(cpuStr),
			"memory": parseResourceQuantity(memStr),
		}
	}

	result := make([]NodeMetrics, 0, len(nodes))
	for _, node := range nodes {
		raw, ok := node.(map[string]any)
		if !ok {
			continue
		}
		meta, _ := raw["metadata"].(map[string]any)
		name, _ := meta["name"].(string)
		status, _ := raw["status"].(map[string]any)
		capacity, _ := status["capacity"].(map[string]any)
		cpuCapStr, _ := capacity["cpu"].(string)
		memCapStr, _ := capacity["memory"].(string)
		cpuCap := parseResourceQuantity(cpuCapStr)
		memCap := parseResourceQuantity(memCapStr)

		m, has := metricsMap[name]
		cpuUsage, memUsage := 0.0, 0.0
		var cpuUsed, memUsed int64
		if has {
			cpuUsed = m["cpu"]
			memUsed = m["memory"]
			if cpuCap > 0 {
				cpuUsage = float64(cpuUsed) / float64(cpuCap) * 100
			}
			if memCap > 0 {
				memUsage = float64(memUsed) / float64(memCap) * 100
			}
		}

		result = append(result, NodeMetrics{
			Name:           name,
			CPUUsage:       cpuUsage,
			MemoryUsage:    memUsage,
			CPUCapacity:    cpuCap,
			MemoryCapacity: memCap,
			CPUUsed:        cpuUsed,
			MemoryUsed:     memUsed,
		})
	}
	return result, nil
}

func (p *metricsServerProvider) GetPodMetrics(ctx context.Context, clusterID uint64, namespace string) ([]PodMetrics, error) {
	dc, err := p.svc.dynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	metricsGVR := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = dc.Resource(metricsGVR).Namespace(namespace)
	} else {
		ri = dc.Resource(metricsGVR)
	}

	metricsList, err := ri.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, normalizeK8sErr(err)
	}

	result := make([]PodMetrics, 0, len(metricsList.Items))
	for _, item := range metricsList.Items {
		meta, _ := item.Object["metadata"].(map[string]any)
		name, _ := meta["name"].(string)
		ns, _ := meta["namespace"].(string)
		containers, _ := item.Object["containers"].([]any)
		var cpuTotal, memTotal int64
		for _, c := range containers {
			cm, _ := c.(map[string]any)
			usage, _ := cm["usage"].(map[string]any)
			if usage == nil {
				continue
			}
			cpuStr, _ := usage["cpu"].(string)
			memStr, _ := usage["memory"].(string)
			cpuTotal += parseResourceQuantity(cpuStr)
			memTotal += parseResourceQuantity(memStr)
		}
		result = append(result, PodMetrics{
			Name:      name,
			Namespace: ns,
			CPU:       cpuTotal,
			Memory:    memTotal,
		})
	}
	return result, nil
}

func (p *metricsServerProvider) GetNodeMetricTrend(ctx context.Context, clusterID uint64, nodeName, metric string, start, end time.Time, step time.Duration) ([]MetricPoint, error) {
	return nil, fmt.Errorf("metrics-server 不支持历史趋势查询")
}

func (p *metricsServerProvider) GetPodMetricTrend(ctx context.Context, clusterID uint64, namespace, podName, metric string, start, end time.Time, step time.Duration) ([]MetricPoint, error) {
	return nil, fmt.Errorf("metrics-server 不支持历史趋势查询")
}

func (p *metricsServerProvider) Health(ctx context.Context, clusterID uint64) error {
	dc, err := p.svc.dynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	metricsGVR := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}
	_, err = dc.Resource(metricsGVR).List(ctx, metav1.ListOptions{Limit: 1})
	return normalizeK8sErr(err)
}
