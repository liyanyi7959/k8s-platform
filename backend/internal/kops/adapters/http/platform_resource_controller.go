package http

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type PlatformResourceController struct {
	service *kopsapp.PlatformResourceService
}

func NewPlatformResourceController(service *kopsapp.PlatformResourceService) *PlatformResourceController {
	return &PlatformResourceController{service: service}
}

func (ctl *PlatformResourceController) List(resource kopsapp.PlatformResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.List(c.Request.Context(), resource, kopsapp.PlatformResourceListQuery{
			ClusterID: namespaceClusterID(c), Namespace: c.Query("namespace"), SortBy: c.Query("sort_by"), Order: c.Query("order"),
		})
		ctl.respond(c, value, err)
	}
}

func (ctl *PlatformResourceController) YAML(resource kopsapp.PlatformResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.YAML(c.Request.Context(), resource, platformResourceReference(c))
		ctl.respond(c, value, err)
	}
}

func (ctl *PlatformResourceController) Apply(resource kopsapp.PlatformResource) gin.HandlerFunc {
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
		err := ctl.service.Apply(c.Request.Context(), resource, kopsapp.PlatformResourceEdit{ClusterID: namespaceClusterID(c), Namespace: request.Namespace, YAML: request.YAML})
		ctl.respondEmpty(c, err)
	}
}

func (ctl *PlatformResourceController) Delete(resource kopsapp.PlatformResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		ctl.respondEmpty(c, ctl.service.Delete(c.Request.Context(), resource, platformResourceReference(c)))
	}
}

func (ctl *PlatformResourceController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}

func (ctl *PlatformResourceController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}

func (ctl *PlatformResourceController) respondEmpty(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func platformResourceReference(c *gin.Context) kopsapp.PlatformResourceReference {
	decode := func(value string) string {
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
		return strings.TrimSpace(value)
	}
	return kopsapp.PlatformResourceReference{ClusterID: namespaceClusterID(c), Namespace: decode(c.Param("ns")), Name: decode(c.Param("name"))}
}
