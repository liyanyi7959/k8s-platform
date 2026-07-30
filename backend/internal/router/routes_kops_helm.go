package router

func registerHelmRoutes(a k8sRouteArgs) {
	k8s, helm, p := a.k8s, a.helm, a.perm
	k8s.GET("/clusters/:id/helm/releases", p.read, helm.Releases)
	k8s.GET("/clusters/:id/helm/releases/detail", p.read, helm.ReleaseDetail)
	k8s.GET("/clusters/:id/helm/releases/:ns/:name", p.read, helm.ReleaseDetail)
	k8s.POST("/clusters/:id/helm/preflight", p.write, helm.Preflight)
	k8s.POST("/clusters/:id/helm/install", p.write, helm.Install)
	k8s.POST("/clusters/:id/helm/releases/:ns/:name/upgrade", p.write, helm.Upgrade)
	k8s.POST("/clusters/:id/helm/releases/:ns/:name/rollback", p.write, helm.Rollback)
	k8s.POST("/clusters/:id/helm/preflight-checks", p.write, helm.Preflight)
	k8s.POST("/clusters/:id/helm/releases", p.write, helm.Install)
	k8s.POST("/clusters/:id/helm/releases/:ns/:name/upgrade-attempts", p.write, helm.Upgrade)
	k8s.POST("/clusters/:id/helm/releases/:ns/:name/rollback-attempts", p.write, helm.Rollback)
	k8s.DELETE("/clusters/:id/helm/releases/:ns/:name", p.write, helm.Uninstall)
	k8s.GET("/clusters/:id/helm/repos", p.read, helm.Repositories)
	k8s.POST("/clusters/:id/helm/repos", p.write, helm.AddRepository)
	k8s.DELETE("/clusters/:id/helm/repos/:name", p.write, helm.DeleteRepository)
	k8s.GET("/clusters/:id/helm/search", p.read, helm.Search)
}
