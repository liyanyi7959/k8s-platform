package router

import (
	cicdhttp "k8s-platform-backend/internal/cicd/adapters/http"
	cicdkube "k8s-platform-backend/internal/cicd/adapters/kubernetes"
	cicdapp "k8s-platform-backend/internal/cicd/application"
)

func buildCICDModule(d Deps, runtime moduleRuntime) cicdModule {
	executor := cicdkube.NewJobExecutor(runtime.k8s)
	return cicdModule{controller: cicdhttp.NewController(cicdapp.NewService(d.DB, executor))}
}
