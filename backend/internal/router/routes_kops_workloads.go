package router

import (
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/middleware"
)

func registerWorkloadRoutes(a k8sRouteArgs) {
	k8s, manifest, metrics, relationships, workloads, pods, inspection, creator, p := a.k8s, a.manifest, a.metrics, a.relationships, a.workloads, a.pods, a.inspection, a.creator, a.perm

	k8s.GET("/clusters/:id/pods", p.read, pods.List)
	k8s.GET("/clusters/:id/podmetrics", p.read, pods.Metrics)
	k8s.GET("/clusters/:id/pods/metrics", p.read, metrics.PodMetrics)
	k8s.GET("/clusters/:id/pods/:ns/:pod/inspection", p.read, inspection.Pod)
	k8s.GET("/clusters/:id/pods/:ns/:pod/yaml", p.read, pods.YAML)
	k8s.GET("/clusters/:id/pods/:ns/:pod/logs", p.read, pods.Logs)
	k8s.POST("/clusters/:id/pods/:ns/:pod/logs/session", p.read, pods.CreateLogSession)
	k8s.POST("/clusters/:id/pods/:ns/:pod/log-sessions", p.read, pods.CreateLogSession)
	k8s.DELETE("/clusters/:id/pods/:ns/:pod", p.write, pods.Delete)
	k8s.POST("/clusters/:id/pods/:ns/:pod/exec", p.exec, pods.CreateExecSession)
	k8s.POST("/clusters/:id/pods/:ns/:pod/exec-sessions", p.exec, pods.CreateExecSession)

	k8s.GET("/clusters/:id/manifests/records", p.write, manifest.List)
	k8s.GET("/clusters/:id/manifests/records/:recordId", p.write, manifest.Get)
	k8s.POST("/clusters/:id/manifests/apply", p.write, manifest.Apply)
	k8s.POST("/clusters/:id/manifest-applications", p.write, manifest.Apply)

	k8s.GET("/clusters/:id/workloads", p.read, workloads.List)
	k8s.GET("/clusters/:id/workloads/deployments/:ns/:name/rollout-history", p.read, workloads.History)
	k8s.POST("/clusters/:id/workloads/deployments/:ns/:name/rollout-undo", p.write, workloads.Undo)
	k8s.POST("/clusters/:id/workloads/deployments/:ns/:name/rollback-attempts", p.write, workloads.Undo)
	k8s.PATCH("/clusters/:id/workloads/scale", p.write, workloads.Scale)
	k8s.PATCH("/clusters/:id/workloads/restart", p.write, workloads.Restart)
	k8s.PATCH("/clusters/:id/workloads/image", p.write, workloads.Image)
	k8s.PATCH("/clusters/:id/workloads/rollout-pause", p.write, workloads.Pause)
	k8s.POST("/clusters/:id/workloads/:kind/:ns/:name/image-updates", p.write, workloads.Image)
	k8s.PATCH("/clusters/:id/workloads/:kind/:ns/:name/pause-state", p.write, workloads.Pause)
	k8s.POST("/clusters/:id/workloads/deployments", p.write, creator.CreateDeployment)
	k8s.POST("/clusters/:id/workloads/statefulsets", p.write, creator.CreateStatefulSet)
	k8s.POST("/clusters/:id/workloads/daemonsets", p.write, creator.CreateDaemonSet)
	k8s.PATCH("/clusters/:id/workloads/deployments/edit", p.write, workloads.Edit(kopsapp.WorkloadDeployment))
	k8s.PATCH("/clusters/:id/workloads/statefulsets/edit", p.write, workloads.Edit(kopsapp.WorkloadStatefulSet))
	k8s.PATCH("/clusters/:id/workloads/daemonsets/edit", p.write, workloads.Edit(kopsapp.WorkloadDaemonSet))
	k8s.PATCH("/clusters/:id/workloads/yaml/edit", p.write, workloads.ApplyYAML)
	k8s.DELETE("/clusters/:id/workloads/:kind/:ns/:name", p.write, workloads.Delete)
	k8s.GET("/clusters/:id/workloads/:kind/:ns/:name/yaml", p.read, workloads.YAML)

	k8s.GET("/clusters/:id/replicasets", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), relationships.List(kopsapp.RelationshipReplicaSet))
	k8s.PATCH("/clusters/:id/replicasets/edit", p.write, relationships.Apply(kopsapp.RelationshipReplicaSet))
	k8s.DELETE("/clusters/:id/replicasets/:ns/:name", p.write, relationships.Delete(kopsapp.RelationshipReplicaSet))
	k8s.GET("/clusters/:id/replicasets/:ns/:name/yaml", p.read, relationships.YAML(kopsapp.RelationshipReplicaSet))
}
