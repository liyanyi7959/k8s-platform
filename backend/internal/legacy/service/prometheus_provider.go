// prometheus_provider.go 实现基于 Prometheus 的 MetricsProvider。
package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

// prometheusProvider 通过 PromQL 从 Prometheus 获取指标。
type prometheusProvider struct {
	svc *K8sService
}

// newPrometheusProvider 创建 Prometheus Provider。
func newPrometheusProvider(svc *K8sService) *prometheusProvider {
	return &prometheusProvider{svc: svc}
}

func (p *prometheusProvider) Name() string { return string(kopsapp.MonitorSourcePrometheus) }

func (p *prometheusProvider) client(ctx context.Context, clusterID uint64) (*prometheusClient, error) {
	info, err := p.svc.GetPrometheusInfo(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if info.URL == "" {
		return nil, fmt.Errorf("集群未配置 Prometheus 地址")
	}
	return newPrometheusClient(info.URL), nil
}

func (p *prometheusProvider) GetNodeMetrics(ctx context.Context, clusterID uint64) ([]kopsapp.NodeMetrics, error) {
	pc, err := p.client(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	type queryResult struct {
		cpuUsage    prometheusQueryResult
		memUsage    prometheusQueryResult
		cpuCapacity prometheusQueryResult
		memCapacity prometheusQueryResult
		err         error
	}
	var res queryResult

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		res.cpuUsage, _ = pc.query(ctx, `100 * (1 - avg by (node) (irate(node_cpu_seconds_total{mode="idle"}[5m])))`)
	}()
	go func() {
		defer wg.Done()
		res.memUsage, _ = pc.query(ctx, `100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes))`)
	}()
	go func() {
		defer wg.Done()
		res.cpuCapacity, _ = pc.query(ctx, `count by (node) (node_cpu_seconds_total{mode="idle"})`)
	}()
	go func() {
		defer wg.Done()
		res.memCapacity, _ = pc.query(ctx, `node_memory_MemTotal_bytes`)
	}()
	wg.Wait()
	if res.err != nil {
		return nil, res.err
	}

	metricsMap := map[string]*kopsapp.NodeMetrics{}
	for _, r := range res.cpuUsage.Result {
		node := r.Metric["node"]
		if node == "" {
			continue
		}
		if metricsMap[node] == nil {
			metricsMap[node] = &kopsapp.NodeMetrics{Name: node}
		}
		metricsMap[node].CPUUsage = parsePrometheusValue(r.Value[1])
	}
	for _, r := range res.memUsage.Result {
		node := r.Metric["node"]
		if node == "" {
			continue
		}
		if metricsMap[node] == nil {
			metricsMap[node] = &kopsapp.NodeMetrics{Name: node}
		}
		metricsMap[node].MemoryUsage = parsePrometheusValue(r.Value[1])
	}
	for _, r := range res.cpuCapacity.Result {
		node := r.Metric["node"]
		if node == "" {
			continue
		}
		if metricsMap[node] == nil {
			metricsMap[node] = &kopsapp.NodeMetrics{Name: node}
		}
		metricsMap[node].CPUCapacity = int64(parsePrometheusValue(r.Value[1]) * 1e9)
	}
	for _, r := range res.memCapacity.Result {
		node := r.Metric["node"]
		if node == "" {
			continue
		}
		if metricsMap[node] == nil {
			metricsMap[node] = &kopsapp.NodeMetrics{Name: node}
		}
		metricsMap[node].MemoryCapacity = int64(parsePrometheusValue(r.Value[1]))
	}

	result := make([]kopsapp.NodeMetrics, 0, len(metricsMap))
	for _, m := range metricsMap {
		if m.CPUCapacity > 0 {
			m.CPUUsed = int64(float64(m.CPUCapacity) * m.CPUUsage / 100)
		}
		if m.MemoryCapacity > 0 {
			m.MemoryUsed = int64(float64(m.MemoryCapacity) * m.MemoryUsage / 100)
		}
		result = append(result, *m)
	}
	return result, nil
}

func (p *prometheusProvider) GetPodMetrics(ctx context.Context, clusterID uint64, namespace string) ([]kopsapp.PodMetrics, error) {
	pc, err := p.client(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	nsFilter := ""
	if namespace != "" {
		nsFilter = fmt.Sprintf(`namespace="%s",`, namespace)
	}
	cpuExpr := fmt.Sprintf(`sum by (pod, namespace) (rate(container_cpu_usage_seconds_total{%s container!="", pod!=""}[5m]))`, nsFilter)
	memExpr := fmt.Sprintf(`sum by (pod, namespace) (container_memory_working_set_bytes{%s container!="", pod!=""})`, nsFilter)

	type queryResult struct {
		cpu prometheusQueryResult
		mem prometheusQueryResult
	}
	var res queryResult
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		res.cpu, _ = pc.query(ctx, cpuExpr)
	}()
	go func() {
		defer wg.Done()
		res.mem, _ = pc.query(ctx, memExpr)
	}()
	wg.Wait()

	metricsMap := map[string]*kopsapp.PodMetrics{}
	for _, r := range res.cpu.Result {
		pod := r.Metric["pod"]
		ns := r.Metric["namespace"]
		if pod == "" {
			continue
		}
		key := ns + "/" + pod
		if metricsMap[key] == nil {
			metricsMap[key] = &kopsapp.PodMetrics{Name: pod, Namespace: ns}
		}
		metricsMap[key].CPU = int64(parsePrometheusValue(r.Value[1]) * 1e9)
	}
	for _, r := range res.mem.Result {
		pod := r.Metric["pod"]
		ns := r.Metric["namespace"]
		if pod == "" {
			continue
		}
		key := ns + "/" + pod
		if metricsMap[key] == nil {
			metricsMap[key] = &kopsapp.PodMetrics{Name: pod, Namespace: ns}
		}
		metricsMap[key].Memory = int64(parsePrometheusValue(r.Value[1]))
	}

	result := make([]kopsapp.PodMetrics, 0, len(metricsMap))
	for _, m := range metricsMap {
		result = append(result, *m)
	}
	return result, nil
}

func (p *prometheusProvider) GetNodeMetricTrend(ctx context.Context, clusterID uint64, nodeName, metric string, start, end time.Time, step time.Duration) ([]kopsapp.MetricPoint, error) {
	pc, err := p.client(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	expr, err := p.nodeTrendExpr(nodeName, metric)
	if err != nil {
		return nil, err
	}
	res, err := pc.queryRange(ctx, expr, start, end, step)
	if err != nil {
		return nil, err
	}
	return p.rangeResultToPoints(res), nil
}

func (p *prometheusProvider) GetPodMetricTrend(ctx context.Context, clusterID uint64, namespace, podName, metric string, start, end time.Time, step time.Duration) ([]kopsapp.MetricPoint, error) {
	pc, err := p.client(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	expr, err := p.podTrendExpr(namespace, podName, metric)
	if err != nil {
		return nil, err
	}
	res, err := pc.queryRange(ctx, expr, start, end, step)
	if err != nil {
		return nil, err
	}
	return p.rangeResultToPoints(res), nil
}

func (p *prometheusProvider) Health(ctx context.Context, clusterID uint64) error {
	pc, err := p.client(ctx, clusterID)
	if err != nil {
		return err
	}
	return pc.health(ctx)
}

func (p *prometheusProvider) nodeTrendExpr(nodeName, metric string) (string, error) {
	switch strings.ToLower(metric) {
	case "cpu":
		return fmt.Sprintf(`100 * (1 - avg by (node) (irate(node_cpu_seconds_total{mode="idle", node="%s"}[5m])))`, nodeName), nil
	case "memory":
		return fmt.Sprintf(`100 * (1 - (node_memory_MemAvailable_bytes{node="%s"} / node_memory_MemTotal_bytes{node="%s"}))`, nodeName, nodeName), nil
	default:
		return "", fmt.Errorf("不支持的节点指标: %s", metric)
	}
}

func (p *prometheusProvider) podTrendExpr(namespace, podName, metric string) (string, error) {
	switch strings.ToLower(metric) {
	case "cpu":
		return fmt.Sprintf(`sum (rate(container_cpu_usage_seconds_total{namespace="%s", pod="%s", container!=""}[5m]))`, namespace, podName), nil
	case "memory":
		return fmt.Sprintf(`sum (container_memory_working_set_bytes{namespace="%s", pod="%s", container!=""})`, namespace, podName), nil
	default:
		return "", fmt.Errorf("不支持的 Pod 指标: %s", metric)
	}
}

func (p *prometheusProvider) rangeResultToPoints(res prometheusQueryRangeResult) []kopsapp.MetricPoint {
	points := make([]kopsapp.MetricPoint, 0)
	for _, r := range res.Result {
		for _, v := range r.Values {
			if len(v) < 2 {
				continue
			}
			ts, _ := v[0].(float64)
			points = append(points, kopsapp.MetricPoint{
				Timestamp: int64(ts * 1000),
				Value:     parsePrometheusValue(v[1]),
			})
		}
	}
	return points
}
