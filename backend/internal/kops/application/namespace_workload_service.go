package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// NamespaceWorkloadKind identifies the workload resources that make up a
// namespace inventory. Kubernetes client details stay behind the runtime port.
type NamespaceWorkloadKind string

const (
	NamespaceWorkloadDeployment  NamespaceWorkloadKind = "deployment"
	NamespaceWorkloadStatefulSet NamespaceWorkloadKind = "statefulset"
	NamespaceWorkloadDaemonSet   NamespaceWorkloadKind = "daemonset"
)

// NamespaceWorkloadResult keeps the established read-model contract neutral
// so both the Kops HTTP endpoint and AI tool adapter can render it.
type NamespaceWorkloadResult struct {
	Summary  string         `json:"summary"`
	Evidence map[string]any `json:"evidence"`
	RawRef   map[string]any `json:"raw_ref"`
}

type NamespaceWorkloadRuntime interface {
	ListNamespaceWorkloads(context.Context, uint64, NamespaceWorkloadKind, string) ([]map[string]any, error)
}

type NamespaceWorkloadReader interface {
	NamespaceWorkloadInventory(context.Context, uint64, string) (NamespaceWorkloadResult, error)
}

type NamespaceWorkloadService struct{ runtime NamespaceWorkloadRuntime }

func NewNamespaceWorkloadService(runtime NamespaceWorkloadRuntime) *NamespaceWorkloadService {
	return &NamespaceWorkloadService{runtime: runtime}
}

func (s *NamespaceWorkloadService) NamespaceWorkloadInventory(ctx context.Context, clusterID uint64, namespace string) (NamespaceWorkloadResult, error) {
	namespace = strings.TrimSpace(namespace)
	if clusterID == 0 || namespace == "" {
		return NamespaceWorkloadResult{}, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return NamespaceWorkloadResult{}, ErrConflict
	}

	type workloadSpec struct {
		kind  NamespaceWorkloadKind
		label string
		key   string
	}
	specs := []workloadSpec{
		{kind: NamespaceWorkloadDeployment, label: "Deployment", key: "deployments"},
		{kind: NamespaceWorkloadStatefulSet, label: "StatefulSet", key: "statefulsets"},
		{kind: NamespaceWorkloadDaemonSet, label: "DaemonSet", key: "daemonsets"},
	}

	counts := map[string]any{}
	byKind := map[string]any{}
	items := make([]map[string]any, 0, 32)
	totalDesired, totalReady := 0, 0
	for _, spec := range specs {
		objects, err := s.runtime.ListNamespaceWorkloads(ctx, clusterID, spec.kind, namespace)
		if err != nil {
			return NamespaceWorkloadResult{}, err
		}
		summaries := make([]map[string]any, 0, len(objects))
		for _, object := range objects {
			summary := BuildInspectionWorkloadSummary(spec.label, object)
			summaries = append(summaries, summary)
			items = append(items, summary)
			totalDesired += inspectionInt(summary["desired_replicas"])
			totalReady += inspectionInt(summary["ready_replicas"])
		}
		counts[spec.key] = len(summaries)
		byKind[spec.key] = summaries
	}

	sort.SliceStable(items, func(i, j int) bool {
		leftKind, rightKind := strings.TrimSpace(fmt.Sprint(items[i]["kind"])), strings.TrimSpace(fmt.Sprint(items[j]["kind"]))
		if leftKind == rightKind {
			return strings.TrimSpace(fmt.Sprint(items[i]["name"])) < strings.TrimSpace(fmt.Sprint(items[j]["name"]))
		}
		return leftKind < rightKind
	})

	summary := fmt.Sprintf(
		"Namespace %s workload inventory: %d workloads (%d Deployments, %d StatefulSets, %d DaemonSets), ready replicas %d/%d",
		namespace, len(items), inspectionInt(counts["deployments"]), inspectionInt(counts["statefulsets"]), inspectionInt(counts["daemonsets"]), totalReady, totalDesired,
	)
	return NamespaceWorkloadResult{
		Summary: summary,
		Evidence: map[string]any{
			"namespace": namespace,
			"counts":    counts,
			"items":     items,
			"by_kind":   byKind,
			"replicas":  map[string]any{"desired": totalDesired, "ready": totalReady},
		},
		RawRef: map[string]any{"cluster_id": clusterID, "namespace": namespace, "source": "namespace.workloads"},
	}, nil
}
