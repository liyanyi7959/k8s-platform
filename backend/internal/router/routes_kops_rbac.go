package router

import kopsapp "k8s-platform-backend/internal/kops/application"

func registerRBACRoutes(a k8sRouteArgs) {
	k8s, platform, p := a.k8s, a.platform, a.perm

	k8s.GET("/clusters/:id/roles", p.rbacRead, platform.List(kopsapp.PlatformRole))
	k8s.PATCH("/clusters/:id/roles/edit", p.rbacWrite, platform.Apply(kopsapp.PlatformRole))
	k8s.DELETE("/clusters/:id/roles/:ns/:name", p.rbacWrite, platform.Delete(kopsapp.PlatformRole))
	k8s.GET("/clusters/:id/roles/:ns/:name/yaml", p.rbacRead, platform.YAML(kopsapp.PlatformRole))

	k8s.GET("/clusters/:id/clusterroles", p.rbacRead, platform.List(kopsapp.PlatformClusterRole))
	k8s.PATCH("/clusters/:id/clusterroles/edit", p.rbacWrite, platform.Apply(kopsapp.PlatformClusterRole))
	k8s.DELETE("/clusters/:id/clusterroles/:name", p.rbacWrite, platform.Delete(kopsapp.PlatformClusterRole))
	k8s.GET("/clusters/:id/clusterroles/:name/yaml", p.rbacRead, platform.YAML(kopsapp.PlatformClusterRole))

	k8s.GET("/clusters/:id/rolebindings", p.rbacRead, platform.List(kopsapp.PlatformRoleBinding))
	k8s.PATCH("/clusters/:id/rolebindings/edit", p.rbacWrite, platform.Apply(kopsapp.PlatformRoleBinding))
	k8s.DELETE("/clusters/:id/rolebindings/:ns/:name", p.rbacWrite, platform.Delete(kopsapp.PlatformRoleBinding))
	k8s.GET("/clusters/:id/rolebindings/:ns/:name/yaml", p.rbacRead, platform.YAML(kopsapp.PlatformRoleBinding))

	k8s.GET("/clusters/:id/clusterrolebindings", p.rbacRead, platform.List(kopsapp.PlatformClusterRoleBinding))
	k8s.PATCH("/clusters/:id/clusterrolebindings/edit", p.rbacWrite, platform.Apply(kopsapp.PlatformClusterRoleBinding))
	k8s.DELETE("/clusters/:id/clusterrolebindings/:name", p.rbacWrite, platform.Delete(kopsapp.PlatformClusterRoleBinding))
	k8s.GET("/clusters/:id/clusterrolebindings/:name/yaml", p.rbacRead, platform.YAML(kopsapp.PlatformClusterRoleBinding))
}
