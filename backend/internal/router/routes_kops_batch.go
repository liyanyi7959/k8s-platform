package router

import (
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/middleware"
)

func registerBatchRoutes(a k8sRouteArgs) {
	k8s, batch, p := a.k8s, a.batch, a.perm

	k8s.GET("/clusters/:id/jobs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), batch.List(kopsapp.BatchJob))
	k8s.POST("/clusters/:id/jobs/completed/deletion-requests", p.write, batch.DeleteCompletedJobs)
	k8s.DELETE("/clusters/:id/jobs/:ns/:name", p.write, batch.Delete(kopsapp.BatchJob))
	k8s.GET("/clusters/:id/jobs/:ns/:name/yaml", p.read, batch.YAML(kopsapp.BatchJob))

	k8s.GET("/clusters/:id/cronjobs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), batch.List(kopsapp.BatchCronJob))
	k8s.POST("/clusters/:id/cronjobs/:ns/:name/execution-requests", p.write, batch.TriggerCronJob)
	k8s.PATCH("/clusters/:id/cronjobs/:ns/:name/suspension-state", p.write, batch.SuspendCronJob)
	k8s.DELETE("/clusters/:id/cronjobs/:ns/:name", p.write, batch.Delete(kopsapp.BatchCronJob))
	k8s.GET("/clusters/:id/cronjobs/:ns/:name/yaml", p.read, batch.YAML(kopsapp.BatchCronJob))
}
