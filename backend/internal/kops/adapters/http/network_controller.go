package http

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type NetworkController struct{ service *kopsapp.NetworkService }

func NewNetworkController(service *kopsapp.NetworkService) *NetworkController {
	return &NetworkController{service: service}
}

func (ctl *NetworkController) List(resource kopsapp.NetworkResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.List(c.Request.Context(), resource, kopsapp.NetworkListQuery{ClusterID: namespaceClusterID(c), Namespace: c.Query("namespace"), SortBy: c.Query("sort_by"), Order: c.Query("order")})
		ctl.respond(c, value, err)
	}
}

func (ctl *NetworkController) YAML(resource kopsapp.NetworkResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.YAML(c.Request.Context(), resource, networkReference(c))
		ctl.respond(c, value, err)
	}
}

func (ctl *NetworkController) Delete(resource kopsapp.NetworkResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		if err := ctl.service.Delete(c.Request.Context(), resource, networkReference(c)); err != nil {
			writeKopsApplicationError(c, err)
			return
		}
		resp.OK(c, gin.H{})
	}
}

func (ctl *NetworkController) EditService(c *gin.Context) {
	var input kopsapp.ServiceEditInput
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.EditService(c.Request.Context(), input))
}

func (ctl *NetworkController) EditIngress(c *gin.Context) {
	var input kopsapp.IngressEditInput
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.EditIngress(c.Request.Context(), input))
}

func (ctl *NetworkController) EditIngressClass(c *gin.Context) {
	var input kopsapp.IngressClassEditInput
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.EditIngressClass(c.Request.Context(), input))
}

func (ctl *NetworkController) bind(c *gin.Context, value any) bool {
	if err := c.ShouldBindJSON(value); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return false
	}
	return ctl.ready(c)
}

func (ctl *NetworkController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}

func (ctl *NetworkController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}

func (ctl *NetworkController) respondOK(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

func networkReference(c *gin.Context) kopsapp.NetworkReference {
	decode := func(value string) string {
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
		return strings.TrimSpace(value)
	}
	return kopsapp.NetworkReference{ClusterID: namespaceClusterID(c), Namespace: decode(c.Param("ns")), Name: decode(c.Param("name"))}
}
