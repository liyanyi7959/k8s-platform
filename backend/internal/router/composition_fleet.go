package router

import (
	fleethttp "k8s-platform-backend/internal/fleet/adapters/http"
	fleetkubernetes "k8s-platform-backend/internal/fleet/adapters/kubernetes"
)

func buildFleetModule(d Deps, runtime moduleRuntime) fleetModule {
	return fleetModule{
		clusters:  fleethttp.NewClusterController(runtime.clusterRegistry, fleetkubernetes.NewClusterRuntime(fleetKubernetesTransport{k8s: runtime.k8s})),
		dashboard: fleethttp.NewDashboardController(runtime.dashboard),
	}
}
