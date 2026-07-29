package http

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type RelationshipResourceController struct {
	service *kopsapp.RelationshipResourceService
}

func NewRelationshipResourceController(service *kopsapp.RelationshipResourceService) *RelationshipResourceController {
	return &RelationshipResourceController{service: service}
}

func (ctl *RelationshipResourceController) List(resource kopsapp.RelationshipResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.List(c.Request.Context(), resource, kopsapp.RelationshipResourceListQuery{
			ClusterID: namespaceClusterID(c), Namespace: c.Query("namespace"), SortBy: c.Query("sort_by"), Order: c.Query("order"),
		})
		ctl.respond(c, value, err)
	}
}

func (ctl *RelationshipResourceController) YAML(resource kopsapp.RelationshipResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.YAML(c.Request.Context(), resource, relationshipResourceReference(c))
		ctl.respond(c, value, err)
	}
}

func (ctl *RelationshipResourceController) Apply(resource kopsapp.RelationshipResource) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		err := ctl.service.Apply(c.Request.Context(), resource, kopsapp.RelationshipResourceEdit{ClusterID: namespaceClusterID(c), Namespace: request.Namespace, YAML: request.YAML})
		ctl.respondEmpty(c, err)
	}
}

func (ctl *RelationshipResourceController) Delete(resource kopsapp.RelationshipResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		ctl.respondEmpty(c, ctl.service.Delete(c.Request.Context(), resource, relationshipResourceReference(c)))
	}
}

func (ctl *RelationshipResourceController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}

func (ctl *RelationshipResourceController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}

func (ctl *RelationshipResourceController) respondEmpty(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func relationshipResourceReference(c *gin.Context) kopsapp.RelationshipResourceReference {
	decode := func(value string) string {
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
		return strings.TrimSpace(value)
	}
	return kopsapp.RelationshipResourceReference{ClusterID: namespaceClusterID(c), Namespace: decode(c.Param("ns")), Name: decode(c.Param("name"))}
}
