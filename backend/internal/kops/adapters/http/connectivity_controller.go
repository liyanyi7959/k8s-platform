package http

import (
	"github.com/gin-gonic/gin"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
	"net/url"
	"strconv"
	"strings"
)

type ConnectivityController struct{ service *kopsapp.ConnectivityService }

func NewConnectivityController(service *kopsapp.ConnectivityService) *ConnectivityController {
	return &ConnectivityController{service: service}
}
func (ctl *ConnectivityController) ListEndpoints(c *gin.Context) {
	ctl.list(c, kopsapp.ConnectivityEndpoints)
}
func (ctl *ConnectivityController) EndpointsYAML(c *gin.Context) {
	ctl.yaml(c, kopsapp.ConnectivityEndpoints)
}
func (ctl *ConnectivityController) EditEndpoints(c *gin.Context) {
	ctl.edit(c, kopsapp.ConnectivityEndpoints)
}
func (ctl *ConnectivityController) DeleteEndpoints(c *gin.Context) {
	ctl.delete(c, kopsapp.ConnectivityEndpoints)
}
func (ctl *ConnectivityController) ListEndpointSlices(c *gin.Context) {
	ctl.list(c, kopsapp.ConnectivityEndpointSlices)
}
func (ctl *ConnectivityController) EndpointSliceYAML(c *gin.Context) {
	ctl.yaml(c, kopsapp.ConnectivityEndpointSlices)
}
func (ctl *ConnectivityController) EditEndpointSlice(c *gin.Context) {
	ctl.edit(c, kopsapp.ConnectivityEndpointSlices)
}
func (ctl *ConnectivityController) DeleteEndpointSlice(c *gin.Context) {
	ctl.delete(c, kopsapp.ConnectivityEndpointSlices)
}
func (ctl *ConnectivityController) ListLeases(c *gin.Context) {
	ctl.list(c, kopsapp.ConnectivityLeases)
}
func (ctl *ConnectivityController) LeaseYAML(c *gin.Context) { ctl.yaml(c, kopsapp.ConnectivityLeases) }
func (ctl *ConnectivityController) EditLease(c *gin.Context) { ctl.edit(c, kopsapp.ConnectivityLeases) }
func (ctl *ConnectivityController) DeleteLease(c *gin.Context) {
	ctl.delete(c, kopsapp.ConnectivityLeases)
}
func (ctl *ConnectivityController) list(c *gin.Context, resource kopsapp.ConnectivityResource) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.List(c.Request.Context(), resource, kopsapp.ConnectivityListQuery{ClusterID: connectivityClusterID(c), Namespace: c.Query("namespace"), SortBy: c.Query("sort_by"), Order: c.Query("order")})
	ctl.respond(c, value, err)
}
func (ctl *ConnectivityController) yaml(c *gin.Context, resource kopsapp.ConnectivityResource) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.YAML(c.Request.Context(), resource, connectivityRef(c))
	ctl.respond(c, value, err)
}
func (ctl *ConnectivityController) edit(c *gin.Context, resource kopsapp.ConnectivityResource) {
	var request struct {
		Namespace string `json:"namespace"`
		YAML      string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if !ctl.ready(c) {
		return
	}
	err := ctl.service.Apply(c.Request.Context(), resource, kopsapp.ConnectivityEdit{ClusterID: connectivityClusterID(c), Namespace: request.Namespace, YAML: request.YAML})
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *ConnectivityController) delete(c *gin.Context, resource kopsapp.ConnectivityResource) {
	if !ctl.ready(c) {
		return
	}
	err := ctl.service.Delete(c.Request.Context(), resource, connectivityRef(c))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *ConnectivityController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}
func (ctl *ConnectivityController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}
func connectivityClusterID(c *gin.Context) uint64 {
	value, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil {
		return 0
	}
	return value
}
func connectivityRef(c *gin.Context) kopsapp.ConnectivityReference {
	decode := func(value string) string {
		if result, err := url.PathUnescape(value); err == nil {
			value = result
		}
		return strings.TrimSpace(value)
	}
	return kopsapp.ConnectivityReference{ClusterID: connectivityClusterID(c), Namespace: decode(c.Param("ns")), Name: decode(c.Param("name"))}
}
