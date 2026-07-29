package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"k8s-platform-backend/internal/ai/domain"
)

type ClusterReadPort interface {
	ClusterOverview(context.Context, uint64) (map[string]any, error)
	ClusterCertificateRisks(context.Context, uint64) ([]map[string]any, error)
}

type ClusterReadModelService struct{ port ClusterReadPort }

func NewClusterReadModelService(port ClusterReadPort) *ClusterReadModelService {
	return &ClusterReadModelService{port: port}
}

func (s *ClusterReadModelService) GetClusterHealth(ctx context.Context, clusterID uint64) (ToolResult, error) {
	if s == nil || s.port == nil {
		return ToolResult{}, ErrConflict
	}
	overview, err := s.port.ClusterOverview(ctx, clusterID)
	if err != nil {
		return ToolResult{}, err
	}
	return buildClusterHealthResult(clusterID, overview), nil
}

func (s *ClusterReadModelService) GetClusterInventory(ctx context.Context, clusterID uint64) (ToolResult, error) {
	return s.GetClusterOverview(ctx, clusterID)
}

func (s *ClusterReadModelService) GetClusterOverview(ctx context.Context, clusterID uint64) (ToolResult, error) {
	if s == nil || s.port == nil {
		return ToolResult{}, ErrConflict
	}
	overview, err := s.port.ClusterOverview(ctx, clusterID)
	if err != nil {
		return ToolResult{}, err
	}
	return buildClusterOverviewResult(clusterID, overview), nil
}

func (s *ClusterReadModelService) GetClusterCertificateRisks(ctx context.Context, clusterID uint64) (ToolResult, error) {
	if s == nil || s.port == nil {
		return ToolResult{}, ErrConflict
	}
	risks, err := s.port.ClusterCertificateRisks(ctx, clusterID)
	if err != nil {
		return ToolResult{}, err
	}
	return buildClusterCertificateRiskResult(clusterID, risks), nil
}

func buildClusterHealthResult(clusterID uint64, overview map[string]any) ToolResult {
	cluster, stats := readMap(overview, "cluster"), readMap(overview, "stats")
	nodes, pods, workloads := readMap(stats, "nodes"), readMap(stats, "pods"), readMap(stats, "workloads")
	version := strings.TrimSpace(fmt.Sprint(cluster["k8s_version"]))
	if version == "" {
		version = "unknown"
	}
	return ToolResult{Summary: fmt.Sprintf("API %t, nodes ready %d/%d, version %s, running pods %d/%d", readBool(cluster["api_ok"]), readInt(nodes["ready"]), readInt(nodes["total"]), version, readInt(pods["running"]), readInt(pods["total"])), Evidence: mapOf(
		"cluster", cluster, "nodes", nodes, "pods", pods, "workloads", workloads, "cpu", readMap(stats, "cpu"), "memory", readMap(stats, "memory"),
	), RawRef: mapOf("cluster_id", clusterID, "source", "cluster.health")}
}

func buildClusterOverviewResult(clusterID uint64, overview map[string]any) ToolResult {
	cluster, stats, charts, anomalies := readMap(overview, "cluster"), readMap(overview, "stats"), readMap(overview, "charts"), readMap(overview, "anomalies")
	pods, workloads, cpu, memory, nodes := readMap(stats, "pods"), readMap(stats, "workloads"), readMap(stats, "cpu"), readMap(stats, "memory"), readMap(stats, "nodes")
	return ToolResult{Summary: fmt.Sprintf("Cluster overview: nodes ready %d/%d, pods %d total (%d running, %d pending, %d failed), workloads %d deployments/%d statefulsets/%d daemonsets, CPU %d%%, memory %d%%", readInt(nodes["ready"]), readInt(nodes["total"]), readInt(pods["total"]), readInt(pods["running"]), readInt(pods["pending"]), readInt(pods["failed"]), readInt(workloads["deployments"]), readInt(workloads["statefulsets"]), readInt(workloads["daemonsets"]), readInt(cpu["used_percent"]), readInt(memory["used_percent"])), Evidence: mapOf(
		"cluster", cluster, "stats", stats, "pod_phase", charts["pod_phase"], "namespace_pods_top", charts["namespace_pods_top"], "node_ready", charts["node_ready"], "failed_pods", anomalies["failed_pods"], "certificate_endpoint", "cluster.certificate_risks",
	), RawRef: mapOf("cluster_id", clusterID, "source", "cluster.overview")}
}

func buildClusterCertificateRiskResult(clusterID uint64, risks []map[string]any) ToolResult {
	counts := mapOf("critical", 0, "warn", 0, "ok", 0, "unknown", 0)
	for _, item := range risks {
		status := strings.TrimSpace(fmt.Sprint(item["status"]))
		if _, ok := counts[status]; !ok {
			status = "unknown"
		}
		counts[status] = readInt(counts[status]) + 1
	}
	return ToolResult{Summary: fmt.Sprintf("Cluster certificate risks: %d critical, %d warning, %d ok, %d unknown", readInt(counts["critical"]), readInt(counts["warn"]), readInt(counts["ok"]), readInt(counts["unknown"])), Evidence: mapOf("total", len(risks), "by_status", counts, "items", risks), RawRef: mapOf("cluster_id", clusterID, "source", "cluster.certificate_risks")}
}

func mapOf(items ...any) domain.JSONMap {
	result := domain.JSONMap{}
	for index := 0; index+1 < len(items); index += 2 {
		key, _ := items[index].(string)
		result[key] = items[index+1]
	}
	return result
}
func readMap(source map[string]any, key string) map[string]any {
	value, _ := source[key].(map[string]any)
	return value
}
func readBool(value any) bool {
	if parsed, ok := value.(bool); ok {
		return parsed
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprint(value)))
	return err == nil && parsed
}
func readInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	default:
		return 0
	}
}
