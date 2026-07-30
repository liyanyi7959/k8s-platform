package router

import (
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/middleware"
)

func registerNetworkingRoutes(a k8sRouteArgs) {
	k8s, network, connectivity, platform, creator, p := a.k8s, a.network, a.connectivity, a.platform, a.creator, a.perm

	k8s.GET("/clusters/:id/services", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), network.List(kopsapp.NetworkKubernetesService))
	k8s.POST("/clusters/:id/services", p.write, creator.CreateService)
	k8s.PATCH("/clusters/:id/services/edit", p.write, network.EditService)
	k8s.DELETE("/clusters/:id/services/:ns/:name", p.write, network.Delete(kopsapp.NetworkKubernetesService))
	k8s.GET("/clusters/:id/services/:ns/:name/yaml", p.read, network.YAML(kopsapp.NetworkKubernetesService))

	k8s.GET("/clusters/:id/ingresses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), network.List(kopsapp.NetworkIngress))
	k8s.POST("/clusters/:id/ingresses", p.write, creator.CreateIngress)
	k8s.PATCH("/clusters/:id/ingresses/edit", p.write, network.EditIngress)
	k8s.DELETE("/clusters/:id/ingresses/:ns/:name", p.write, network.Delete(kopsapp.NetworkIngress))
	k8s.GET("/clusters/:id/ingresses/:ns/:name/yaml", p.read, network.YAML(kopsapp.NetworkIngress))

	k8s.GET("/clusters/:id/networkpolicies", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformNetworkPolicy))
	k8s.PATCH("/clusters/:id/networkpolicies/edit", p.write, platform.Apply(kopsapp.PlatformNetworkPolicy))
	k8s.DELETE("/clusters/:id/networkpolicies/:ns/:name", p.write, platform.Delete(kopsapp.PlatformNetworkPolicy))
	k8s.GET("/clusters/:id/networkpolicies/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformNetworkPolicy))

	k8s.GET("/clusters/:id/ingressclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), network.List(kopsapp.NetworkIngressClass))
	k8s.PATCH("/clusters/:id/ingressclasses/edit", p.write, network.EditIngressClass)
	k8s.DELETE("/clusters/:id/ingressclasses/:name", p.write, network.Delete(kopsapp.NetworkIngressClass))
	k8s.GET("/clusters/:id/ingressclasses/:name/yaml", p.read, network.YAML(kopsapp.NetworkIngressClass))

	k8s.GET("/clusters/:id/endpoints", p.read, connectivity.ListEndpoints)
	k8s.PATCH("/clusters/:id/endpoints/edit", p.write, connectivity.EditEndpoints)
	k8s.DELETE("/clusters/:id/endpoints/:ns/:name", p.write, connectivity.DeleteEndpoints)
	k8s.GET("/clusters/:id/endpoints/:ns/:name/yaml", p.read, connectivity.EndpointsYAML)

	k8s.GET("/clusters/:id/endpointslices", p.read, connectivity.ListEndpointSlices)
	k8s.PATCH("/clusters/:id/endpointslices/edit", p.write, connectivity.EditEndpointSlice)
	k8s.DELETE("/clusters/:id/endpointslices/:ns/:name", p.write, connectivity.DeleteEndpointSlice)
	k8s.GET("/clusters/:id/endpointslices/:ns/:name/yaml", p.read, connectivity.EndpointSliceYAML)
}
