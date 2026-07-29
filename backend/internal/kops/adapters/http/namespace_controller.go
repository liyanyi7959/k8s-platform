package http

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type NamespaceController struct{ service *kopsapp.NamespaceService }

func NewNamespaceController(service *kopsapp.NamespaceService) *NamespaceController {
	return &NamespaceController{service: service}
}

type namespaceCreateRequest struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

func (ctl *NamespaceController) List(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.List(c.Request.Context(), namespaceClusterID(c), c.Query("sort_by"), c.Query("order"))
	ctl.respond(c, result, err)
}
func (ctl *NamespaceController) Create(c *gin.Context) {
	var request namespaceCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if !ctl.ready(c) {
		return
	}
	err := ctl.service.Create(c.Request.Context(), namespaceClusterID(c), kopsapp.NamespaceCreateInput{Name: request.Name, Labels: request.Labels})
	ctl.respond(c, gin.H{}, err)
}
func (ctl *NamespaceController) Delete(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	err := ctl.service.Delete(c.Request.Context(), namespaceClusterID(c), namespacePath(c))
	ctl.respond(c, gin.H{}, err)
}
func (ctl *NamespaceController) YAML(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.YAML(c.Request.Context(), namespaceClusterID(c), namespacePath(c))
	ctl.respond(c, result, err)
}
func (ctl *NamespaceController) Summary(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.Summary(c.Request.Context(), namespaceClusterID(c), namespacePath(c))
	ctl.respond(c, result, err)
}
func (ctl *NamespaceController) Events(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.Events(c.Request.Context(), kopsapp.EventListQuery{ClusterID: namespaceClusterID(c), Namespace: c.Query("namespace"), InvolvedObjectKind: c.Query("involved_object_kind"), InvolvedObjectName: c.Query("involved_object_name"), InvolvedObjectUID: c.Query("involved_object_uid"), SortBy: c.Query("sort_by"), Order: c.Query("order")})
	ctl.respond(c, result, err)
}
func (ctl *NamespaceController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}
func (ctl *NamespaceController) respond(c *gin.Context, result any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}
func namespaceClusterID(c *gin.Context) uint64 {
	value, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil {
		return 0
	}
	return value
}
func namespacePath(c *gin.Context) string {
	value := c.Param("ns")
	if decoded, err := url.PathUnescape(value); err == nil {
		value = decoded
	}
	return strings.TrimSpace(value)
}
