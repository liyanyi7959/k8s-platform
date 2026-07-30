package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

// PrometheusInfo is the persisted endpoint state a Prometheus provider needs.
type PrometheusInfo struct {
	URL    string
	Status kopsapp.PrometheusStatus
}

// PrometheusInfoResolver keeps provider selection outside this adapter while
// exposing only the Kops endpoint data required for Prometheus queries.
type PrometheusInfoResolver interface {
	GetPrometheusInfo(context.Context, uint64) (*PrometheusInfo, error)
}

type prometheusProvider struct {
	resolver PrometheusInfoResolver
}

func NewPrometheusProvider(resolver PrometheusInfoResolver) kopsapp.MetricsProvider {
	return &prometheusProvider{resolver: resolver}
}

func (p *prometheusProvider) Name() string { return string(kopsapp.MonitorSourcePrometheus) }

func (p *prometheusProvider) client(ctx context.Context, clusterID uint64) (*prometheusClient, error) {
	if p == nil || p.resolver == nil {
		return nil, kopsapp.ErrConflict
	}
	info, err := p.resolver.GetPrometheusInfo(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(info.URL) == "" {
		return nil, fmt.Errorf("集群未配置 Prometheus 地址")
	}
	return newPrometheusClient(info.URL), nil
}

func (p *prometheusProvider) GetNodeMetrics(ctx context.Context, clusterID uint64) ([]kopsapp.NodeMetrics, error) {
	client, err := p.client(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	type result struct {
		value prometheusQueryResult
		err   error
	}
	var cpuUsage, memoryUsage, cpuCapacity, memoryCapacity result
	var group sync.WaitGroup
	group.Add(4)
	go func() {
		defer group.Done()
		cpuUsage.value, cpuUsage.err = client.query(ctx, `100 * (1 - avg by (node) (irate(node_cpu_seconds_total{mode="idle"}[5m])))`)
	}()
	go func() {
		defer group.Done()
		memoryUsage.value, memoryUsage.err = client.query(ctx, `100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes))`)
	}()
	go func() {
		defer group.Done()
		cpuCapacity.value, cpuCapacity.err = client.query(ctx, `count by (node) (node_cpu_seconds_total{mode="idle"})`)
	}()
	go func() {
		defer group.Done()
		memoryCapacity.value, memoryCapacity.err = client.query(ctx, `node_memory_MemTotal_bytes`)
	}()
	group.Wait()
	for _, query := range []result{cpuUsage, memoryUsage, cpuCapacity, memoryCapacity} {
		if query.err != nil {
			return nil, query.err
		}
	}

	metrics := map[string]*kopsapp.NodeMetrics{}
	for _, item := range cpuUsage.value.Result {
		node := item.Metric["node"]
		if node == "" {
			continue
		}
		if metrics[node] == nil {
			metrics[node] = &kopsapp.NodeMetrics{Name: node}
		}
		metrics[node].CPUUsage = parsePrometheusValue(item.Value[1])
	}
	for _, item := range memoryUsage.value.Result {
		node := item.Metric["node"]
		if node == "" {
			continue
		}
		if metrics[node] == nil {
			metrics[node] = &kopsapp.NodeMetrics{Name: node}
		}
		metrics[node].MemoryUsage = parsePrometheusValue(item.Value[1])
	}
	for _, item := range cpuCapacity.value.Result {
		node := item.Metric["node"]
		if node == "" {
			continue
		}
		if metrics[node] == nil {
			metrics[node] = &kopsapp.NodeMetrics{Name: node}
		}
		metrics[node].CPUCapacity = int64(parsePrometheusValue(item.Value[1]) * 1e9)
	}
	for _, item := range memoryCapacity.value.Result {
		node := item.Metric["node"]
		if node == "" {
			continue
		}
		if metrics[node] == nil {
			metrics[node] = &kopsapp.NodeMetrics{Name: node}
		}
		metrics[node].MemoryCapacity = int64(parsePrometheusValue(item.Value[1]))
	}
	resultValues := make([]kopsapp.NodeMetrics, 0, len(metrics))
	for _, metric := range metrics {
		if metric.CPUCapacity > 0 {
			metric.CPUUsed = int64(float64(metric.CPUCapacity) * metric.CPUUsage / 100)
		}
		if metric.MemoryCapacity > 0 {
			metric.MemoryUsed = int64(float64(metric.MemoryCapacity) * metric.MemoryUsage / 100)
		}
		resultValues = append(resultValues, *metric)
	}
	return resultValues, nil
}

func (p *prometheusProvider) GetPodMetrics(ctx context.Context, clusterID uint64, namespace string) ([]kopsapp.PodMetrics, error) {
	client, err := p.client(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	namespaceFilter := ""
	if namespace != "" {
		namespaceFilter = fmt.Sprintf(`namespace="%s",`, namespace)
	}
	cpuExpression := fmt.Sprintf(`sum by (pod, namespace) (rate(container_cpu_usage_seconds_total{%s container!="", pod!=""}[5m]))`, namespaceFilter)
	memoryExpression := fmt.Sprintf(`sum by (pod, namespace) (container_memory_working_set_bytes{%s container!="", pod!=""})`, namespaceFilter)
	var cpu, memory prometheusQueryResult
	var cpuErr, memoryErr error
	var group sync.WaitGroup
	group.Add(2)
	go func() { defer group.Done(); cpu, cpuErr = client.query(ctx, cpuExpression) }()
	go func() { defer group.Done(); memory, memoryErr = client.query(ctx, memoryExpression) }()
	group.Wait()
	if cpuErr != nil {
		return nil, cpuErr
	}
	if memoryErr != nil {
		return nil, memoryErr
	}

	metrics := map[string]*kopsapp.PodMetrics{}
	for _, item := range cpu.Result {
		pod, namespace := item.Metric["pod"], item.Metric["namespace"]
		if pod == "" {
			continue
		}
		key := namespace + "/" + pod
		if metrics[key] == nil {
			metrics[key] = &kopsapp.PodMetrics{Name: pod, Namespace: namespace}
		}
		metrics[key].CPU = int64(parsePrometheusValue(item.Value[1]) * 1e9)
	}
	for _, item := range memory.Result {
		pod, namespace := item.Metric["pod"], item.Metric["namespace"]
		if pod == "" {
			continue
		}
		key := namespace + "/" + pod
		if metrics[key] == nil {
			metrics[key] = &kopsapp.PodMetrics{Name: pod, Namespace: namespace}
		}
		metrics[key].Memory = int64(parsePrometheusValue(item.Value[1]))
	}
	resultValues := make([]kopsapp.PodMetrics, 0, len(metrics))
	for _, metric := range metrics {
		resultValues = append(resultValues, *metric)
	}
	return resultValues, nil
}

func (p *prometheusProvider) GetNodeMetricTrend(ctx context.Context, clusterID uint64, nodeName, metric string, start, end time.Time, step time.Duration) ([]kopsapp.MetricPoint, error) {
	client, err := p.client(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	expression, err := p.nodeTrendExpression(nodeName, metric)
	if err != nil {
		return nil, err
	}
	result, err := client.queryRange(ctx, expression, start, end, step)
	if err != nil {
		return nil, err
	}
	return rangeResultToMetricPoints(result), nil
}

func (p *prometheusProvider) GetPodMetricTrend(ctx context.Context, clusterID uint64, namespace, podName, metric string, start, end time.Time, step time.Duration) ([]kopsapp.MetricPoint, error) {
	client, err := p.client(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	expression, err := p.podTrendExpression(namespace, podName, metric)
	if err != nil {
		return nil, err
	}
	result, err := client.queryRange(ctx, expression, start, end, step)
	if err != nil {
		return nil, err
	}
	return rangeResultToMetricPoints(result), nil
}

func (p *prometheusProvider) Health(ctx context.Context, clusterID uint64) error {
	client, err := p.client(ctx, clusterID)
	if err != nil {
		return err
	}
	return client.health(ctx)
}

func (p *prometheusProvider) nodeTrendExpression(nodeName, metric string) (string, error) {
	switch strings.ToLower(metric) {
	case "cpu":
		return fmt.Sprintf(`100 * (1 - avg by (node) (irate(node_cpu_seconds_total{mode="idle", node="%s"}[5m])))`, nodeName), nil
	case "memory":
		return fmt.Sprintf(`100 * (1 - (node_memory_MemAvailable_bytes{node="%s"} / node_memory_MemTotal_bytes{node="%s"}))`, nodeName, nodeName), nil
	default:
		return "", fmt.Errorf("不支持的节点指标: %s", metric)
	}
}

func (p *prometheusProvider) podTrendExpression(namespace, podName, metric string) (string, error) {
	switch strings.ToLower(metric) {
	case "cpu":
		return fmt.Sprintf(`sum (rate(container_cpu_usage_seconds_total{namespace="%s", pod="%s", container!=""}[5m]))`, namespace, podName), nil
	case "memory":
		return fmt.Sprintf(`sum (container_memory_working_set_bytes{namespace="%s", pod="%s", container!=""})`, namespace, podName), nil
	default:
		return "", fmt.Errorf("不支持的 Pod 指标: %s", metric)
	}
}

func rangeResultToMetricPoints(result prometheusQueryRangeResult) []kopsapp.MetricPoint {
	points := make([]kopsapp.MetricPoint, 0)
	for _, item := range result.Result {
		for _, value := range item.Values {
			if len(value) < 2 {
				continue
			}
			timestamp, _ := value[0].(float64)
			points = append(points, kopsapp.MetricPoint{Timestamp: int64(timestamp * 1000), Value: parsePrometheusValue(value[1])})
		}
	}
	return points
}
