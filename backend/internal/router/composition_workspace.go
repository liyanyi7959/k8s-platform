package router

import (
	"context"

	legacykops "k8s-platform-backend/internal/integration/kops"
	kopsapp "k8s-platform-backend/internal/kops/application"
	workspacehttp "k8s-platform-backend/internal/workspace/adapters/http"
	workspacemysql "k8s-platform-backend/internal/workspace/adapters/mysql"
	workspaceapp "k8s-platform-backend/internal/workspace/application"
	workspaceports "k8s-platform-backend/internal/workspace/ports"
)

type namespaceResourceReader struct {
	summary kopsapp.NamespaceResourceSummaryReader
}

func (reader namespaceResourceReader) Summary(ctx context.Context, clusterID uint64, namespace string) ([]workspaceports.ResourceCount, int, error) {
	summary, err := reader.summary.Summary(ctx, clusterID, namespace)
	if err != nil {
		return nil, 0, err
	}
	result := make([]workspaceports.ResourceCount, 0, len(summary.Items))
	for _, item := range summary.Items {
		result = append(result, workspaceports.ResourceCount{Key: item.Key, Count: item.Count})
	}
	return result, summary.Total, nil
}

func buildWorkspaceModule(d Deps, runtime moduleRuntime) workspaceModule {
	applicationService := workspaceapp.NewService(workspacemysql.NewRepository(d.DB), namespaceResourceReader{summary: runtime.namespaceSummary})
	return workspaceModule{projects: workspacehttp.NewController(applicationService)}
}

// Keep this import local to the workspace composition file: workspace projects
// use the K8s namespace summary port, while the adapter remains wired at the root.
var _ = legacykops.NewNamespaceSummaryRuntime
